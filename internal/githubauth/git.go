package githubauth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GitEndpoint is an authenticated Git smart HTTP endpoint for one verified
// repository. URL is the canonical remote to clone or fetch from and never
// contains a token; Transport adds the stored credential as GitHub's
// x-access-token Basic credential, and only to that repository's upload-pack
// discovery (info/refs) and negotiation (git-upload-pack) requests.
//
// Transport serves an in-process Git client. It is not meant for the native git
// executable, and no token ever goes into a remote URL, configuration, or
// environment variable.
type GitEndpoint struct {
	URL       string
	Transport http.RoundTripper
}

// OpenGit loads and target-binds the stored credential exactly as OpenAPI does
// (Options.TargetRepository, OWNER/NAME, is required) and returns the read-only
// Git smart HTTP endpoint for that repository: the canonical
// https://<host>/OWNER/NAME.git remote together with a transport that only
// upload-pack requests reach.
//
// The caller is expected to install Transport in an HTTP client that refuses
// redirects: Transport pins every request to the selected host and repository
// path, so a redirect target can never receive the credential, and refusing the
// redirect turns it into a plain error instead of a refusal on the next hop.
func (c *Client) OpenGit(ctx context.Context, opts Options) (GitEndpoint, error) {
	options, record, err := c.openCredential(ctx, opts)
	if err != nil {
		return GitEndpoint{}, err
	}
	baseURL, err := gitBaseURL(options)
	if err != nil {
		return GitEndpoint{}, err
	}
	transport := &gitTransport{
		tokenTransport: c.newTokenTransport(options, record, "GitHub Git"),
		baseURL:        baseURL,
		target:         options.targetRepository,
	}
	return GitEndpoint{URL: baseURL.String() + "/" + options.targetRepository + ".git", Transport: transport}, nil
}

// gitBaseURL is the scheme and host that serve Git smart HTTP for a resolved
// target: the web host of the validated login endpoints, which is github.com for
// github.com and the instance host for GitHub Enterprise Server. It is never the
// API host (api.github.com), which does not serve Git.
func gitBaseURL(options resolvedOptions) (*url.URL, error) {
	login, err := url.Parse(options.endpoints.TokenURL)
	if err != nil || login.Host == "" || login.User != nil {
		return nil, fmt.Errorf("%w: configured GitHub endpoints are invalid", ErrInvalidOptions)
	}
	return &url.URL{Scheme: login.Scheme, Host: login.Host}, nil
}

// gitTransport is the Git half of tokenTransport. It accepts only the two
// upload-pack requests of the selected repository, and adds the credential as
// GitHub's x-access-token Basic credential, leaving the Git protocol media types
// (Accept, Content-Type, Git-Protocol) that the client set untouched.
type gitTransport struct {
	tokenTransport
	baseURL *url.URL
	target  string
}

func (t *gitTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return t.roundTrip(request, t)
}

func (t *gitTransport) authorizeRequest(clone *http.Request, accessToken string) {
	clone.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("x-access-token:"+accessToken)))
	clone.Header.Set("User-Agent", "zt-github-auth")
}

func (t *gitTransport) validateCredentialTarget(credential credentialRecord) error {
	return credentialBoundToTarget(credential, t.target)
}

func (t *gitTransport) validateRequest(request *http.Request) error {
	if request == nil || request.URL == nil || request.URL.User != nil || request.URL.Fragment != "" || request.URL.Opaque != "" ||
		!strings.EqualFold(request.URL.Scheme, t.baseURL.Scheme) ||
		!strings.EqualFold(request.URL.Host, t.baseURL.Host) ||
		(request.Host != "" && !strings.EqualFold(request.Host, t.baseURL.Host)) {
		return invalidGitRequest()
	}

	// The same decoding rules as the REST endpoint: a path that hides a
	// traversal, an encoded separator, or a NUL byte is refused before it can be
	// compared with the selected route.
	decodedPath, err := decodeSafeAPIPath(request.URL.EscapedPath())
	if err != nil {
		return invalidGitRequest()
	}
	repositoryPath := "/" + t.target + ".git"

	switch request.Method {
	case http.MethodGet:
		if decodedPath != repositoryPath+"/info/refs" || !isUploadPackServiceQuery(request.URL.RawQuery) {
			return invalidGitRequest()
		}
	case http.MethodPost:
		if decodedPath != repositoryPath+"/git-upload-pack" || request.URL.RawQuery != "" || request.URL.ForceQuery {
			return invalidGitRequest()
		}
	default:
		return invalidGitRequest()
	}
	return nil
}

// isUploadPackServiceQuery accepts exactly the smart HTTP discovery query of a
// fetch: the single parameter service=git-upload-pack. receive-pack, extra
// parameters, and unparsable or repeated values are refused, so a request that
// reaches the transport can only read the selected repository.
func isUploadPackServiceQuery(rawQuery string) bool {
	if rawQuery == "" {
		return false
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return false
	}
	services := values["service"]
	return len(values) == 1 && len(services) == 1 && services[0] == "git-upload-pack"
}

func invalidGitRequest() error {
	return fmt.Errorf("%w: request is outside the selected GitHub repository Git upload-pack route", ErrInvalidOptions)
}
