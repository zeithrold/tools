package githubauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
)

// API is an authenticated GitHub REST endpoint for one verified target. It never
// carries a token: the transport adds the credential per request, and refuses any
// request outside BaseURL's host and API path prefix or selected repository subtree.
type API struct {
	BaseURL   string            // e.g. "https://api.github.com", no trailing slash
	Transport http.RoundTripper // adds Authorization; host- and prefix-bound
}

// OpenAPI loads the stored credential for the selected profile, verifies that
// the requested target repository matches the stored one when the record has
// one (a mismatch returns ErrAccountMismatch), refreshes an expired token when
// possible, and returns a token-free endpoint. Options.TargetRepository
// (OWNER/NAME) is required.
func (c *Client) OpenAPI(ctx context.Context, opts Options) (API, error) {
	options, record, err := c.openCredential(ctx, opts)
	if err != nil {
		return API{}, err
	}

	baseURL, err := url.Parse(options.endpoints.APIBaseURL)
	if err != nil {
		return API{}, fmt.Errorf("%w: configured GitHub API base URL is invalid", ErrInvalidOptions)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/")
	baseURL.RawPath = ""
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	transport := &apiTransport{
		tokenTransport: c.newTokenTransport(options, record, "GitHub API"),
		baseURL:        baseURL,
		target:         options.targetRepository,
	}
	return API{BaseURL: strings.TrimRight(baseURL.String(), "/"), Transport: transport}, nil
}

// openCredential is the entry shared by the token-free endpoints (OpenAPI and
// OpenGit). It resolves the options, loads the stored credential, binds it to
// the requested repository, and refreshes it when it has expired. It makes no
// repository request of its own, so opening an endpoint never depends on GitHub
// being reachable.
func (c *Client) openCredential(ctx context.Context, opts Options) (resolvedOptions, credentialRecord, error) {
	if opts.TargetRepository == "" {
		return resolvedOptions{}, credentialRecord{}, fmt.Errorf("%w: target repository is required (OWNER/NAME)", ErrInvalidOptions)
	}
	options, err := c.resolve(opts)
	if err != nil {
		return resolvedOptions{}, credentialRecord{}, err
	}
	if c.Store == nil {
		return resolvedOptions{}, credentialRecord{}, fmt.Errorf("%w", ErrStoreUnavailable)
	}
	record, found, err := c.load(ctx, options)
	if err != nil {
		return resolvedOptions{}, credentialRecord{}, err
	}
	if !found {
		return resolvedOptions{}, credentialRecord{}, fmt.Errorf("%w", ErrNotLoggedIn)
	}
	if err := credentialBoundToTarget(record, options.targetRepository); err != nil {
		return resolvedOptions{}, credentialRecord{}, err
	}
	if !tokenValid(c.now(), timeFromUnix(record.TokenExpiresAt)) {
		record, err = c.refresh(ctx, options, record)
		if err != nil {
			return resolvedOptions{}, credentialRecord{}, err
		}
	}
	return options, record, nil
}

// credentialBoundToTarget refuses a credential stored for another repository, so
// neither a refresh nor a newer record can redirect the token.
func credentialBoundToTarget(credential credentialRecord, target string) error {
	if credential.TargetRepository != "" && !strings.EqualFold(credential.TargetRepository, target) {
		return fmt.Errorf("%w: stored target repository does not match the requested repository", ErrAccountMismatch)
	}
	return nil
}

// apiTransport is the REST half of tokenTransport: it only accepts requests on
// BaseURL's host within the selected repository's API subtree.
type apiTransport struct {
	tokenTransport
	baseURL *url.URL
	target  string
}

func (t *apiTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return t.roundTrip(request, t)
}

func (t *apiTransport) authorizeRequest(clone *http.Request, accessToken string) {
	clone.Header.Set("Authorization", "Bearer "+accessToken)
	clone.Header.Set("Accept", "application/vnd.github+json")
	clone.Header.Set("User-Agent", "zt-github-auth")
	clone.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (t *apiTransport) validateCredentialTarget(credential credentialRecord) error {
	return credentialBoundToTarget(credential, t.target)
}

// tokenTransport is the pipeline shared by the token-free endpoints. It keeps
// the credential in memory, attaches it only to requests its endpointRoute
// accepts, refreshes once when GitHub answers 401, replays a re-creatable body,
// and strips credential headers from the response's request.
type tokenTransport struct {
	label     string // endpoint name for transport errors, e.g. "GitHub API"
	transport http.RoundTripper
	client    *Client
	options   resolvedOptions

	mu         sync.RWMutex
	credential credentialRecord
}

// endpointRoute is the per-endpoint half of tokenTransport: which requests may
// carry the credential, and how that credential is presented on them.
type endpointRoute interface {
	// validateRequest refuses any request outside the endpoint's selected host
	// and route before the credential is attached.
	validateRequest(*http.Request) error
	// authorizeRequest adds the credential to a private clone of the request,
	// never to the caller's request.
	authorizeRequest(clone *http.Request, accessToken string)
	// validateCredentialTarget checks that a stored or refreshed credential still
	// belongs to the verified repository.
	validateCredentialTarget(credentialRecord) error
}

func (c *Client) newTokenTransport(options resolvedOptions, credential credentialRecord, label string) tokenTransport {
	var baseTransport http.RoundTripper = http.DefaultTransport
	if c.httpClient != nil && c.httpClient.Transport != nil {
		baseTransport = c.httpClient.Transport
	}
	return tokenTransport{
		label:      label,
		transport:  baseTransport,
		client:     c,
		options:    options,
		credential: credential,
	}
}

func (t *tokenTransport) roundTrip(request *http.Request, route endpointRoute) (*http.Response, error) {
	if err := route.validateRequest(request); err != nil {
		return nil, err
	}

	credential := t.currentCredential()
	if err := route.validateCredentialTarget(credential); err != nil {
		return nil, err
	}
	response, err := t.send(request, route, credential.AccessToken, nil)
	if err != nil {
		return nil, t.remoteError()
	}
	if response == nil {
		return nil, t.remoteError()
	}
	if response.StatusCode != http.StatusUnauthorized {
		return response, nil
	}
	closeResponse(response)

	credential, err = t.client.refresh(request.Context(), t.options, credential)
	if err != nil {
		return nil, err
	}
	if err := route.validateCredentialTarget(credential); err != nil {
		return nil, err
	}
	t.setCredential(credential)

	var retryBody io.ReadCloser
	if request.Body != nil && request.Body != http.NoBody {
		if request.GetBody == nil {
			return nil, fmt.Errorf("%w: cannot retry a request with a non-replayable body", ErrProtocol)
		}
		retryBody, err = request.GetBody()
		if err != nil {
			return nil, fmt.Errorf("%w: could not recreate the request body for retry", ErrProtocol)
		}
	}
	response, err = t.send(request, route, credential.AccessToken, retryBody)
	if err != nil || response == nil {
		return nil, t.remoteError()
	}
	if response.StatusCode == http.StatusUnauthorized {
		closeResponse(response)
		return nil, fmt.Errorf("%w: GitHub rejected the refreshed access token", ErrReloginRequired)
	}
	return response, nil
}

func (t *tokenTransport) send(request *http.Request, route endpointRoute, accessToken string, body io.ReadCloser) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	if clone.Header == nil {
		clone.Header = make(http.Header)
	}
	route.authorizeRequest(clone, accessToken)
	if body != nil {
		clone.Body = body
	}
	response, err := t.transport.RoundTrip(clone)
	sanitizeResponseRequest(response)
	return response, err
}

func (t *tokenTransport) remoteError() error {
	return fmt.Errorf("%w: %s transport failed (details omitted)", ErrRemote, t.label)
}

func sanitizeResponseRequest(response *http.Response) {
	if response == nil || response.Request == nil {
		return
	}
	request := response.Request.Clone(response.Request.Context())
	request.Header = response.Request.Header.Clone()
	for name := range request.Header {
		if strings.EqualFold(name, "Authorization") ||
			strings.EqualFold(name, "Proxy-Authorization") ||
			strings.EqualFold(name, "Cookie") {
			delete(request.Header, name)
		}
	}
	response.Request = request
}

func (t *apiTransport) validateRequest(request *http.Request) error {
	if request == nil || request.URL == nil || request.URL.User != nil ||
		!strings.EqualFold(request.URL.Scheme, t.baseURL.Scheme) ||
		!strings.EqualFold(request.URL.Host, t.baseURL.Host) ||
		(request.Host != "" && !strings.EqualFold(request.Host, t.baseURL.Host)) {
		return invalidAPIRequest()
	}

	escapedPath := request.URL.EscapedPath()
	decodedPath, err := decodeSafeAPIPath(escapedPath)
	if err != nil {
		return invalidAPIRequest()
	}
	baseEscapedPath := t.baseURL.EscapedPath()
	owner, repository, ok := strings.Cut(t.target, "/")
	if !ok {
		return invalidAPIRequest()
	}
	requiredEscapedPrefix := baseEscapedPath + "/repos/" + owner + "/" + repository
	if !hasPathPrefix(escapedPath, requiredEscapedPrefix) {
		return invalidAPIRequest()
	}

	cleanedPath := path.Clean(decodedPath)
	requiredPathPrefix := strings.TrimRight(t.baseURL.Path, "/") + "/repos/" + owner + "/" + repository
	if !hasPathPrefix(cleanedPath, requiredPathPrefix) {
		return invalidAPIRequest()
	}
	return nil
}

func decodeSafeAPIPath(escapedPath string) (string, error) {
	current := escapedPath
	for {
		decoded, err := url.PathUnescape(current)
		if err != nil {
			return "", err
		}
		if strings.Count(decoded, "/") > strings.Count(current, "/") ||
			strings.ContainsAny(decoded, "\\\x00") {
			return "", fmt.Errorf("unsafe encoded path separator")
		}
		for _, segment := range strings.Split(decoded, "/") {
			if segment == "." || segment == ".." {
				return "", fmt.Errorf("path traversal is not allowed")
			}
		}
		if !containsPercentEscape(decoded) {
			return decoded, nil
		}
		current = decoded
	}
}

func containsPercentEscape(value string) bool {
	for index := 0; index+2 < len(value); index++ {
		if value[index] == '%' && isHex(value[index+1]) && isHex(value[index+2]) {
			return true
		}
	}
	return false
}

func isHex(value byte) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}

func hasPathPrefix(value, prefix string) bool {
	if value == prefix {
		return true
	}
	return strings.HasPrefix(value, prefix+"/")
}

func invalidAPIRequest() error {
	return fmt.Errorf("%w: request is outside the selected GitHub repository API path", ErrInvalidOptions)
}

func closeResponse(response *http.Response) {
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
}

func (t *tokenTransport) currentCredential() credentialRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.credential
}

func (t *tokenTransport) setCredential(record credentialRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.credential = record
}
