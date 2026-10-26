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
	if opts.TargetRepository == "" {
		return API{}, fmt.Errorf("%w: target repository is required (OWNER/NAME)", ErrInvalidOptions)
	}
	options, err := c.resolve(opts)
	if err != nil {
		return API{}, err
	}
	if c.Store == nil {
		return API{}, fmt.Errorf("%w", ErrStoreUnavailable)
	}
	record, found, err := c.load(ctx, options)
	if err != nil {
		return API{}, err
	}
	if !found {
		return API{}, fmt.Errorf("%w", ErrNotLoggedIn)
	}
	if record.TargetRepository != "" && !strings.EqualFold(record.TargetRepository, options.targetRepository) {
		return API{}, fmt.Errorf("%w: stored target repository does not match the requested repository", ErrAccountMismatch)
	}
	if !tokenValid(c.now(), timeFromUnix(record.TokenExpiresAt)) {
		record, err = c.refresh(ctx, options, record)
		if err != nil {
			return API{}, err
		}
	}

	baseURL, err := url.Parse(options.endpoints.APIBaseURL)
	if err != nil {
		return API{}, fmt.Errorf("%w: configured GitHub API base URL is invalid", ErrInvalidOptions)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/")
	baseURL.RawPath = ""
	baseURL.RawQuery = ""
	baseURL.Fragment = ""

	var baseTransport http.RoundTripper = http.DefaultTransport
	if c.httpClient != nil && c.httpClient.Transport != nil {
		baseTransport = c.httpClient.Transport
	}
	transport := &apiTransport{
		baseURL:    baseURL,
		target:     options.targetRepository,
		transport:  baseTransport,
		client:     c,
		options:    options,
		credential: record,
	}
	return API{BaseURL: strings.TrimRight(baseURL.String(), "/"), Transport: transport}, nil
}

type apiTransport struct {
	baseURL   *url.URL
	target    string
	transport http.RoundTripper
	client    *Client
	options   resolvedOptions

	mu         sync.RWMutex
	credential credentialRecord
}

func (t *apiTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := t.validateRequest(request); err != nil {
		return nil, err
	}

	credential := t.currentCredential()
	if err := t.validateCredentialTarget(credential); err != nil {
		return nil, err
	}
	response, err := t.send(request, credential.AccessToken, nil)
	if err != nil {
		return nil, remoteTransportError()
	}
	if response == nil {
		return nil, remoteTransportError()
	}
	if response.StatusCode != http.StatusUnauthorized {
		return response, nil
	}
	closeResponse(response)

	credential, err = t.client.refresh(request.Context(), t.options, credential)
	if err != nil {
		return nil, err
	}
	if err := t.validateCredentialTarget(credential); err != nil {
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
	if err := t.validateCredentialTarget(credential); err != nil {
		return nil, err
	}
	response, err = t.send(request, credential.AccessToken, retryBody)
	if err != nil || response == nil {
		return nil, remoteTransportError()
	}
	if response.StatusCode == http.StatusUnauthorized {
		closeResponse(response)
		return nil, fmt.Errorf("%w: GitHub rejected the refreshed access token", ErrReloginRequired)
	}
	return response, nil
}

func (t *apiTransport) send(request *http.Request, accessToken string, body io.ReadCloser) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	if clone.Header == nil {
		clone.Header = make(http.Header)
	}
	clone.Header.Set("Authorization", "Bearer "+accessToken)
	clone.Header.Set("Accept", "application/vnd.github+json")
	clone.Header.Set("User-Agent", "zt-github-auth")
	clone.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		clone.Body = body
	}
	response, err := t.transport.RoundTrip(clone)
	sanitizeResponseRequest(response)
	return response, err
}

func (t *apiTransport) validateCredentialTarget(credential credentialRecord) error {
	if credential.TargetRepository != "" && !strings.EqualFold(credential.TargetRepository, t.target) {
		return fmt.Errorf("%w: stored target repository does not match the requested repository", ErrAccountMismatch)
	}
	return nil
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

func remoteTransportError() error {
	return fmt.Errorf("%w: GitHub API transport failed (details omitted)", ErrRemote)
}

func closeResponse(response *http.Response) {
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
}

func (t *apiTransport) currentCredential() credentialRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.credential
}

func (t *apiTransport) setCredential(record credentialRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.credential = record
}
