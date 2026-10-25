package githubauth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type resolvedOptions struct {
	host             string
	clientID         string
	profile          string
	key              string
	scopeParam       string
	targetRepository string
	appID            int64
	replaceAccount   bool
	endpoints        endpoints
}

func (c *Client) resolve(opts Options) (resolvedOptions, error) {
	host, err := normalizeHost(opts.Host)
	if err != nil {
		return resolvedOptions{}, err
	}
	allowed, err := c.hostAllowed(host)
	if err != nil {
		return resolvedOptions{}, err
	}
	if !allowed {
		return resolvedOptions{}, fmt.Errorf("%w: host is not explicitly trusted in Client.AllowedHosts", ErrInvalidOptions)
	}

	if err := validateClientID(opts.ClientID); err != nil {
		return resolvedOptions{}, err
	}
	profile := opts.Profile
	if profile == "" {
		profile = "default"
	}
	if !profilePattern.MatchString(profile) || looksLikeToken(profile) {
		return resolvedOptions{}, fmt.Errorf("%w: profile must be a non-secret identifier matching %s", ErrInvalidOptions, profilePattern)
	}
	scopeParam := ""
	if opts.RequestOfflineAccess {
		scopeParam = "offline_access"
	}
	repository, err := normalizeRepository(opts.TargetRepository)
	if err != nil {
		return resolvedOptions{}, err
	}
	if opts.AppID < 0 {
		return resolvedOptions{}, fmt.Errorf("%w: App ID must not be negative", ErrInvalidOptions)
	}
	endpoints, err := c.resolveEndpoints(host)
	if err != nil {
		return resolvedOptions{}, err
	}
	return resolvedOptions{
		host:             host,
		clientID:         opts.ClientID,
		profile:          profile,
		key:              credentialKey(host, opts.ClientID, profile),
		scopeParam:       scopeParam,
		targetRepository: repository,
		appID:            opts.AppID,
		replaceAccount:   opts.ReplaceExistingAccount,
		endpoints:        endpoints,
	}, nil
}

func (c *Client) hostAllowed(host string) (bool, error) {
	if len(c.AllowedHosts) == 0 {
		return host == "github.com", nil
	}
	for _, raw := range c.AllowedHosts {
		allowed, err := normalizeHost(raw)
		if err != nil {
			return false, fmt.Errorf("%w: a trusted host is not valid", ErrInvalidOptions)
		}
		if allowed == host {
			return true, nil
		}
	}
	return false, nil
}

func normalizeHost(raw string) (string, error) {
	if raw == "" {
		raw = "github.com"
	}
	if raw != strings.TrimSpace(raw) || strings.ContainsAny(raw, "/@?#\\% \t\r\n") || strings.Contains(raw, "://") {
		return "", fmt.Errorf("%w: host must be a bare DNS name with an optional port, not a URL", ErrInvalidOptions)
	}
	parsed, err := url.Parse("https://" + raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: host is not a valid bare DNS name or IP address", ErrInvalidOptions)
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return "", fmt.Errorf("%w: host has no name", ErrInvalidOptions)
	}
	if ip := net.ParseIP(hostname); ip == nil && !validDNSName(hostname) {
		return "", fmt.Errorf("%w: host is not a valid DNS name or IP address", ErrInvalidOptions)
	}
	if ip := net.ParseIP(hostname); ip != nil && strings.Contains(hostname, ":") && !strings.HasPrefix(raw, "[") {
		return "", fmt.Errorf("%w: an IPv6 host must be enclosed in brackets", ErrInvalidOptions)
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return "", fmt.Errorf("%w: host port is not valid", ErrInvalidOptions)
		}
	}
	return strings.ToLower(parsed.Host), nil
}

func validDNSName(name string) bool {
	if len(name) > 253 {
		return false
	}
	for _, label := range strings.Split(name, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func (c *Client) resolveEndpoints(host string) (endpoints, error) {
	if c.endpoints == (endpoints{}) {
		return defaultEndpoints(host), nil
	}
	if c.endpoints.DeviceCodeURL == "" || c.endpoints.TokenURL == "" || c.endpoints.APIBaseURL == "" {
		return endpoints{}, fmt.Errorf("%w: all internal endpoints must be set together", ErrInvalidOptions)
	}
	device, err := validateEndpoint(c.endpoints.DeviceCodeURL, "device code endpoint", host)
	if err != nil {
		return endpoints{}, err
	}
	token, err := validateEndpoint(c.endpoints.TokenURL, "token endpoint", host)
	if err != nil {
		return endpoints{}, err
	}
	api, err := validateEndpoint(c.endpoints.APIBaseURL, "API base URL", apiHostFor(host))
	if err != nil {
		return endpoints{}, err
	}
	return endpoints{DeviceCodeURL: device, TokenURL: token, APIBaseURL: api}, nil
}

func defaultEndpoints(host string) endpoints {
	login := "https://" + host
	api := "https://" + apiHostFor(host) + apiPathFor(host)
	return endpoints{
		DeviceCodeURL: login + "/login/device/code",
		TokenURL:      login + "/login/oauth/access_token",
		APIBaseURL:    api,
	}
}

func apiHostFor(host string) string {
	if host == "github.com" {
		return "api.github.com"
	}
	return host
}

func apiPathFor(host string) string {
	if host == "github.com" {
		return ""
	}
	return "/api/v3"
}

func validateEndpoint(raw, label, expectedHost string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" {
		return "", fmt.Errorf("%w: %s must be an absolute URL without user info, query, or fragment", ErrInvalidOptions, label)
	}
	if parsed.Scheme != "https" {
		if parsed.Scheme != "http" || !isLoopbackHost(parsed.Hostname()) {
			return "", fmt.Errorf("%w: %s must use HTTPS (plain HTTP is accepted only for loopback tests)", ErrInvalidOptions, label)
		}
	}
	if !strings.EqualFold(parsed.Host, expectedHost) {
		return "", fmt.Errorf("%w: %s does not use the configured trusted host", ErrInvalidOptions, label)
	}
	return parsed.String(), nil
}

func isLoopbackHost(hostname string) bool {
	if strings.EqualFold(hostname, "localhost") {
		return true
	}
	ip := net.ParseIP(hostname)
	return ip != nil && ip.IsLoopback()
}

func joinAPI(base, path string, query url.Values) string {
	parsed, err := url.Parse(base)
	if err != nil {
		return base
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	if query == nil {
		parsed.RawQuery = ""
	} else {
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}

func credentialKey(host, clientID, profile string) string {
	sum := sha256.Sum256([]byte(host + "\x00" + clientID + "\x00" + profile))
	return "github." + hex.EncodeToString(sum[:])
}

var (
	clientIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,255}$`)
	tokenPrefixes   = []string{"ghu_", "ghp_", "gho_", "ghs_", "ghr_", "github_pat_"}
	profilePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	repoPartPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
)

func validateClientID(clientID string) error {
	if !clientIDPattern.MatchString(clientID) || looksLikeToken(clientID) {
		return fmt.Errorf("%w: client ID must be the GitHub App public client ID, not a GitHub token", ErrInvalidOptions)
	}
	return nil
}

func looksLikeToken(value string) bool {
	lower := strings.ToLower(value)
	for _, prefix := range tokenPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func normalizeRepository(repository string) (string, error) {
	if repository == "" {
		return "", nil
	}
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || !repoPartPattern.MatchString(parts[0]) || !repoPartPattern.MatchString(parts[1]) || parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
		return "", fmt.Errorf("%w: target repository must be OWNER/NAME with safe characters", ErrInvalidOptions)
	}
	return repository, nil
}

func secondsDuration(seconds int64) (time.Duration, error) {
	if seconds <= 0 {
		return 0, fmt.Errorf("%w: GitHub returned a non-positive duration", ErrProtocol)
	}
	if seconds > math.MaxInt64/int64(time.Second) {
		return 0, fmt.Errorf("%w: GitHub returned an unusable duration", ErrProtocol)
	}
	return time.Duration(seconds) * time.Second, nil
}

func unixExpiry(now time.Time, seconds int64) int64 {
	if seconds <= 0 {
		return 0
	}
	if seconds > math.MaxInt64-now.Unix() {
		return 0
	}
	return now.Unix() + seconds
}

func timeFromUnix(seconds int64) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0).UTC()
}

func tokenValid(now time.Time, expires time.Time) bool {
	if expires.IsZero() {
		return true
	}
	return now.Add(30 * time.Second).Before(expires)
}
