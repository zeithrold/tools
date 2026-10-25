package githubauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseBytes = 1 << 20

type oauthErrorBody struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Interval         int64  `json:"interval"`
}

type statusError struct {
	status int
	code   string
}

func (e *statusError) Error() string {
	if e.code == "" {
		return fmt.Sprintf("HTTP %d", e.status)
	}
	return fmt.Sprintf("HTTP %d (%s)", e.status, e.code)
}

func (c *Client) postForm(ctx context.Context, endpoint string, form url.Values, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: could not build request", ErrProtocol)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "zt-github-auth")

	response, err := c.do(request, "GitHub authorization request")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := readLimited(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var body oauthErrorBody
		_ = json.Unmarshal(data, &body)
		return &statusError{status: response.StatusCode, code: sanitizeOAuthCode(body.Error)}
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%w: could not decode GitHub authorization response", ErrProtocol)
	}
	return nil
}

func (c *Client) getJSON(ctx context.Context, endpoint, accessToken string, out any) (http.Header, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: could not build GitHub API request", ErrProtocol)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "zt-github-auth")

	response, err := c.do(request, "GitHub API request")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := readLimited(response.Body)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		if len(data) == 0 {
			return response.Header, nil
		}
		if err := json.Unmarshal(data, out); err != nil {
			return nil, fmt.Errorf("%w: could not decode GitHub API response", ErrProtocol)
		}
		return response.Header, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("%w: GitHub rejected the stored access token", ErrReloginRequired)
	case http.StatusForbidden:
		if response.Header.Get("X-RateLimit-Remaining") == "0" || response.Header.Get("Retry-After") != "" {
			return nil, fmt.Errorf("%w: GitHub rate limit (HTTP 403)", ErrRemote)
		}
		return nil, fmt.Errorf("%w: GitHub denied access (HTTP 403)", ErrScopeDenied)
	case http.StatusNotFound:
		return nil, fmt.Errorf("%w: GitHub resource not found or not accessible (HTTP 404)", ErrScopeDenied)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: GitHub rate limit (HTTP 429)", ErrRemote)
	default:
		if response.StatusCode >= 500 {
			return nil, fmt.Errorf("%w: GitHub returned HTTP %d", ErrRemote, response.StatusCode)
		}
		return nil, fmt.Errorf("%w: GitHub returned HTTP %d", ErrProtocol, response.StatusCode)
	}
}

func (c *Client) do(request *http.Request, action string) (*http.Response, error) {
	response, err := c.newHTTPClient().Do(request)
	if err != nil {
		if contextError := request.Context().Err(); contextError != nil {
			return nil, fmt.Errorf("%s: %w", action, contextError)
		}
		return nil, fmt.Errorf("%w: %s failed (transport details omitted to avoid echoing URLs or responses)", ErrRemote, action)
	}
	return response, nil
}

func readLimited(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: could not read GitHub response", ErrRemote)
	}
	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("%w: GitHub response exceeded %d bytes", ErrProtocol, maxResponseBytes)
	}
	return data, nil
}

func (c *Client) newHTTPClient() *http.Client {
	base := c.httpClient
	if base == nil {
		base = &http.Client{Timeout: 30 * time.Second}
	}
	client := *base
	previous := base.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if previous != nil {
			if err := previous(request, via); err != nil {
				return err
			}
		}
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		if len(via) == 0 {
			return nil
		}
		if request.URL.User != nil {
			return errors.New("refusing redirect with URL user info")
		}
		first := via[0].URL
		if !sameURLHost(first, request.URL) {
			return fmt.Errorf("refusing cross-host redirect from %s to %s", safeURLHost(first), safeURLHost(request.URL))
		}
		return nil
	}
	return &client
}

func sameURLHost(left, right *url.URL) bool {
	if left == nil || right == nil {
		return false
	}
	if !strings.EqualFold(left.Scheme, right.Scheme) || !strings.EqualFold(left.Host, right.Host) {
		return false
	}
	return right.User == nil
}

func safeURLHost(value *url.URL) string {
	if value == nil {
		return "<none>"
	}
	return strings.ToLower(value.Scheme) + "://" + strings.ToLower(value.Host)
}

func parseNextLink(header string) string {
	for _, part := range strings.Split(header, ",") {
		sections := strings.Split(part, ";")
		if len(sections) < 2 {
			continue
		}
		target := strings.TrimSpace(sections[0])
		if !strings.HasPrefix(target, "<") || !strings.HasSuffix(target, ">") {
			continue
		}
		for _, parameter := range sections[1:] {
			parameter = strings.TrimSpace(parameter)
			if parameter == `rel="next"` || parameter == "rel=next" {
				return strings.TrimSuffix(strings.TrimPrefix(target, "<"), ">")
			}
		}
	}
	return ""
}

func resolveNextPage(current, link, apiBase string) (string, error) {
	reference, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("%w: GitHub pagination link is invalid", ErrProtocol)
	}
	base, err := url.Parse(current)
	if err != nil {
		return "", fmt.Errorf("%w: current GitHub URL is invalid", ErrProtocol)
	}
	target := base.ResolveReference(reference)
	if target.User != nil || target.Fragment != "" {
		return "", fmt.Errorf("%w: GitHub pagination link contains user info or a fragment", ErrProtocol)
	}
	if target.Scheme != "https" && !(target.Scheme == "http" && isLoopbackHost(target.Hostname())) {
		return "", fmt.Errorf("%w: GitHub pagination link does not use HTTPS", ErrProtocol)
	}
	api, err := url.Parse(apiBase)
	if err != nil || !strings.EqualFold(target.Host, api.Host) {
		return "", fmt.Errorf("%w: GitHub pagination link points to a different host", ErrProtocol)
	}
	basePath := strings.TrimRight(api.Path, "/")
	if basePath != "" && target.Path != basePath && !strings.HasPrefix(target.Path, basePath+"/") {
		return "", fmt.Errorf("%w: GitHub pagination link points outside the API base path", ErrProtocol)
	}
	return target.String(), nil
}

// safeOAuthCodes is an allowlist, not a character filter. GitHub error codes
// are the only server-provided strings allowed into errors; an unknown value
// (including a token placed in "error") is replaced with an empty code and a
// generic message.
var safeOAuthCodes = map[string]bool{
	"authorization_pending":        true,
	"slow_down":                    true,
	"expired_token":                true,
	"access_denied":                true,
	"incorrect_client_credentials": true,
	"unsupported_grant_type":       true,
	"incorrect_device_code":        true,
	"device_flow_disabled":         true,
	"bad_refresh_token":            true,
	"invalid_grant":                true,
	"unauthorized_client":          true,
	"invalid_request":              true,
	"server_error":                 true,
	"temporarily_unavailable":      true,
}

func sanitizeOAuthCode(code string) string {
	if safeOAuthCodes[code] {
		return code
	}
	return ""
}
