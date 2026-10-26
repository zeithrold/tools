package githubauth

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
	Error                   string `json:"error"`
	ErrorDescription        string `json:"error_description"`
}

type deviceAuthorization struct {
	deviceCode string
	prompt     DevicePrompt
	expiresAt  time.Time
	interval   time.Duration
}

type tokenResponse struct {
	AccessToken           string `json:"access_token"`
	TokenType             string `json:"token_type"`
	Scope                 string `json:"scope"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	Error                 string `json:"error"`
	ErrorDescription      string `json:"error_description"`
	Interval              int64  `json:"interval"`
}

type userResponse struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}

type installation struct {
	ID                  int64  `json:"id"`
	AppID               int64  `json:"app_id"`
	AppSlug             string `json:"app_slug"`
	RepositorySelection string `json:"repository_selection"`
}

type installationsPage struct {
	TotalCount    int            `json:"total_count"`
	Installations []installation `json:"installations"`
}

type repository struct {
	FullName string `json:"full_name"`
}

type repositoriesPage struct {
	TotalCount   int          `json:"total_count"`
	Repositories []repository `json:"repositories"`
}

type targetVerification struct {
	installationID int64
	repository     string
}

// Login runs the GitHub App device authorization flow and writes the resulting
// user access/refresh token pair to the native store. The callback receives no
// device code and no token. Nothing is saved when authorization, identity
// verification, target verification, or the atomic store write fails.
func (c *Client) Login(ctx context.Context, opts Options, onDevice func(DevicePrompt) error) (AccountSummary, error) {
	options, err := c.resolve(opts)
	if err != nil {
		return AccountSummary{}, err
	}
	if c.Store == nil {
		return AccountSummary{}, fmt.Errorf("%w", ErrStoreUnavailable)
	}
	if err := c.Store.Available(ctx); err != nil {
		return AccountSummary{}, mapStoreError("check secure credential store", err)
	}
	// Check for an existing record before starting the device flow so wrong
	// host/client/profile selections fail without a network round trip. The
	// record is checked again under the cross-process lock before saving.
	if _, _, err := c.load(ctx, options); err != nil && !(options.replaceAccount && errors.Is(err, ErrCredentialCorrupt)) {
		return AccountSummary{}, err
	}

	authorization, err := c.requestDeviceCode(ctx, options)
	if err != nil {
		return AccountSummary{}, err
	}
	if onDevice != nil {
		if err := onDevice(authorization.prompt); err != nil {
			return AccountSummary{}, fmt.Errorf("device authorization callback: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return AccountSummary{}, fmt.Errorf("device authorization: %w", err)
	}

	token, err := c.pollDeviceToken(ctx, options, authorization)
	if err != nil {
		return AccountSummary{}, err
	}
	identity, scopes, err := c.verifyUser(ctx, options, token.AccessToken, token.Scope)
	if err != nil {
		return AccountSummary{}, err
	}
	if !options.replaceAccount {
		if existing, found, loadErr := c.load(ctx, options); loadErr == nil && found && existing.UserID != identity.ID {
			return AccountSummary{}, fmt.Errorf("%w: this profile holds a different account (%q, user ID %d); set ReplaceExistingAccount only after explicit confirmation", ErrAccountConflict, existing.Login, existing.UserID)
		}
	}
	target, err := c.verifyTarget(ctx, options, token.AccessToken)
	if err != nil {
		return AccountSummary{}, err
	}
	if err := ctx.Err(); err != nil {
		return AccountSummary{}, fmt.Errorf("device authorization: %w", err)
	}

	release, err := c.lock(ctx, options.key)
	if err != nil {
		return AccountSummary{}, err
	}
	defer release()
	existing, found, err := c.load(ctx, options)
	if err != nil {
		if !(options.replaceAccount && errors.Is(err, ErrCredentialCorrupt)) {
			return AccountSummary{}, err
		}
		existing = credentialRecord{}
		found = false
	}
	if found && existing.UserID != identity.ID && !options.replaceAccount {
		return AccountSummary{}, fmt.Errorf("%w: this profile holds a different account (%q, user ID %d); set ReplaceExistingAccount only after explicit confirmation", ErrAccountConflict, existing.Login, existing.UserID)
	}
	if err := c.Store.Available(ctx); err != nil {
		return AccountSummary{}, fmt.Errorf("GitHub authorization succeeded but the secure credential store is not writable; no token was saved: %w", mapStoreError("check secure credential store", err))
	}

	now := c.now()
	record := credentialRecord{
		Version:          credentialVersion,
		Host:             options.host,
		ClientID:         options.clientID,
		Profile:          options.profile,
		Login:            identity.Login,
		UserID:           identity.ID,
		AccessToken:      token.AccessToken,
		RefreshToken:     token.RefreshToken,
		TokenExpiresAt:   unixExpiry(now, token.ExpiresIn),
		RefreshExpiresAt: unixExpiry(now, token.RefreshTokenExpiresIn),
		Scopes:           scopes,
		InstallationID:   target.installationID,
		TargetRepository: target.repository,
		VerifiedAt:       now.Unix(),
	}
	if err := c.save(ctx, options, record); err != nil {
		return AccountSummary{}, fmt.Errorf("GitHub authorization succeeded but credentials could not be stored; log in again: %w", err)
	}
	return record.summary(true), nil
}

func (c *Client) requestDeviceCode(ctx context.Context, options resolvedOptions) (deviceAuthorization, error) {
	form := url.Values{"client_id": {options.clientID}}
	if options.scopeParam != "" {
		form.Set("scope", options.scopeParam)
	}
	var response deviceCodeResponse
	if err := c.postForm(ctx, options.endpoints.DeviceCodeURL, form, &response); err != nil {
		var status *statusError
		if errors.As(err, &status) {
			kind := ErrProtocol
			switch {
			case status.status == 400 || status.status == 401:
				kind = ErrAuthorizationRejected
			case status.status == 429 || status.status >= 500:
				kind = ErrRemote
			}
			return deviceAuthorization{}, fmt.Errorf("%w: device code request returned %s%s", kind, status, deviceCodeHint(status.code))
		}
		return deviceAuthorization{}, err
	}
	if response.Error != "" {
		code := sanitizeOAuthCode(response.Error)
		if code == "" {
			return deviceAuthorization{}, fmt.Errorf("%w: device authorization endpoint returned an unrecognized error", ErrAuthorizationRejected)
		}
		return deviceAuthorization{}, fmt.Errorf("%w: GitHub returned %s%s", ErrAuthorizationRejected, code, deviceCodeHint(code))
	}
	if response.DeviceCode == "" || response.UserCode == "" || response.VerificationURI == "" {
		return deviceAuthorization{}, fmt.Errorf("%w: device code response is missing required fields", ErrProtocol)
	}
	expiresIn, err := secondsDuration(response.ExpiresIn)
	if err != nil {
		return deviceAuthorization{}, err
	}
	interval := 5 * time.Second
	if response.Interval > 0 {
		interval, err = secondsDuration(response.Interval)
		if err != nil {
			return deviceAuthorization{}, err
		}
	} else if response.Interval < 0 {
		return deviceAuthorization{}, fmt.Errorf("%w: device code interval is negative", ErrProtocol)
	}
	verificationURI, err := validateVerificationURI(response.VerificationURI, options.host)
	if err != nil {
		return deviceAuthorization{}, err
	}
	complete := ""
	if response.VerificationURIComplete != "" {
		complete, err = validateVerificationURI(response.VerificationURIComplete, options.host)
		if err != nil {
			return deviceAuthorization{}, err
		}
	}
	return deviceAuthorization{
		deviceCode: response.DeviceCode,
		prompt: DevicePrompt{
			UserCode:                response.UserCode,
			VerificationURI:         verificationURI,
			VerificationURIComplete: complete,
			ExpiresIn:               expiresIn,
			Interval:                interval,
		},
		expiresAt: c.now().Add(expiresIn),
		interval:  interval,
	}, nil
}

// Hints are local text, never the remote error_description or error_uri.
func deviceCodeHint(code string) string {
	switch code {
	case "device_flow_disabled":
		return "; enable Device Flow in the GitHub App settings, then retry zt auth login"
	case "incorrect_client_credentials":
		return "; check the public GitHub App Client ID, not the App ID or a client secret"
	default:
		return ""
	}
}

func (c *Client) pollDeviceToken(ctx context.Context, options resolvedOptions, authorization deviceAuthorization) (tokenResponse, error) {
	deadline := authorization.expiresAt
	interval := authorization.interval
	for {
		remaining := deadline.Sub(c.now())
		if remaining <= 0 {
			return tokenResponse{}, fmt.Errorf("%w", ErrDeviceCodeExpired)
		}
		sleepFor := interval
		if sleepFor > remaining {
			sleepFor = remaining
		}
		if err := c.sleep(ctx, sleepFor); err != nil {
			return tokenResponse{}, fmt.Errorf("waiting for GitHub device authorization: %w", err)
		}
		if !c.now().Before(deadline) {
			return tokenResponse{}, fmt.Errorf("%w", ErrDeviceCodeExpired)
		}
		response, err := c.exchangeDeviceToken(ctx, options, authorization.deviceCode)
		if err != nil {
			return tokenResponse{}, err
		}
		if response.Error == "" {
			if response.AccessToken == "" {
				return tokenResponse{}, fmt.Errorf("%w: token response had no access token", ErrProtocol)
			}
			return response, nil
		}
		switch code := sanitizeOAuthCode(response.Error); code {
		case "authorization_pending":
			continue
		case "slow_down":
			next := interval
			if next <= time.Duration(math.MaxInt64)-5*time.Second {
				next += 5 * time.Second
			}
			if response.Interval > 0 {
				serverInterval, err := secondsDuration(response.Interval)
				if err != nil {
					return tokenResponse{}, err
				}
				if serverInterval > next {
					next = serverInterval
				}
			}
			interval = next
			continue
		case "expired_token":
			return tokenResponse{}, fmt.Errorf("%w", ErrDeviceCodeExpired)
		case "access_denied":
			return tokenResponse{}, fmt.Errorf("%w", ErrAccessDenied)
		case "incorrect_client_credentials", "unsupported_grant_type", "incorrect_device_code", "device_flow_disabled":
			return tokenResponse{}, fmt.Errorf("%w: GitHub returned %s", ErrAuthorizationRejected, code)
		default:
			if code == "" {
				return tokenResponse{}, fmt.Errorf("%w: GitHub returned an unrecognized OAuth error", ErrProtocol)
			}
			return tokenResponse{}, fmt.Errorf("%w: GitHub returned OAuth error %q", ErrProtocol, code)
		}
	}
}

func (c *Client) exchangeDeviceToken(ctx context.Context, options resolvedOptions, deviceCode string) (tokenResponse, error) {
	form := url.Values{
		"client_id":   {options.clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	var response tokenResponse
	if err := c.postForm(ctx, options.endpoints.TokenURL, form, &response); err != nil {
		var status *statusError
		if errors.As(err, &status) {
			switch {
			case status.status == 400 || status.status == 401:
				return tokenResponse{}, fmt.Errorf("%w: token request returned HTTP %d", ErrAuthorizationRejected, status.status)
			case status.status == 429 || status.status >= 500:
				return tokenResponse{}, fmt.Errorf("%w: token request returned HTTP %d", ErrRemote, status.status)
			default:
				return tokenResponse{}, fmt.Errorf("%w: token request returned HTTP %d", ErrProtocol, status.status)
			}
		}
		return tokenResponse{}, err
	}
	if response.Error != "" {
		return response, nil
	}
	if response.AccessToken == "" {
		return tokenResponse{}, fmt.Errorf("%w: token response had no access token", ErrProtocol)
	}
	if response.ExpiresIn < 0 || response.RefreshTokenExpiresIn < 0 {
		return tokenResponse{}, fmt.Errorf("%w: token response had a negative duration", ErrProtocol)
	}
	return response, nil
}

func (c *Client) exchangeRefresh(ctx context.Context, options resolvedOptions, refreshToken string) (tokenResponse, error) {
	form := url.Values{
		"client_id":     {options.clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	var response tokenResponse
	if err := c.postForm(ctx, options.endpoints.TokenURL, form, &response); err != nil {
		var status *statusError
		if errors.As(err, &status) {
			switch {
			case status.status == 400 || status.status == 401:
				return tokenResponse{}, &refreshError{definitive: true, err: fmt.Errorf("%w: refresh request returned HTTP %d", ErrReloginRequired, status.status)}
			case status.status == 429 || status.status >= 500:
				return tokenResponse{}, fmt.Errorf("%w: refresh request returned HTTP %d", ErrRemote, status.status)
			default:
				return tokenResponse{}, fmt.Errorf("%w: refresh request returned HTTP %d", ErrProtocol, status.status)
			}
		}
		return tokenResponse{}, err
	}
	if response.Error != "" {
		switch code := sanitizeOAuthCode(response.Error); code {
		case "bad_refresh_token", "invalid_grant", "expired_token", "access_denied", "unauthorized_client":
			return tokenResponse{}, &refreshError{definitive: true, err: fmt.Errorf("%w: GitHub rejected the refresh token (%s)", ErrReloginRequired, code)}
		case "slow_down":
			return tokenResponse{}, fmt.Errorf("%w: GitHub rate limited the refresh request", ErrRemote)
		default:
			if code == "" {
				return tokenResponse{}, fmt.Errorf("%w: GitHub returned an unrecognized refresh error", ErrProtocol)
			}
			return tokenResponse{}, fmt.Errorf("%w: GitHub returned refresh error %q", ErrProtocol, code)
		}
	}
	if response.AccessToken == "" {
		return tokenResponse{}, fmt.Errorf("%w: refresh response had no access token", ErrProtocol)
	}
	if response.ExpiresIn < 0 || response.RefreshTokenExpiresIn < 0 {
		return tokenResponse{}, fmt.Errorf("%w: refresh response had a negative duration", ErrProtocol)
	}
	return response, nil
}

func (c *Client) verifyUser(ctx context.Context, options resolvedOptions, accessToken, fallbackScope string) (userResponse, []string, error) {
	var user userResponse
	headers, err := c.getJSON(ctx, joinAPI(options.endpoints.APIBaseURL, "/user", nil), accessToken, &user)
	if err != nil {
		if errors.Is(err, ErrScopeDenied) {
			return userResponse{}, nil, fmt.Errorf("%w: /user was not accessible with this token", ErrProtocol)
		}
		return userResponse{}, nil, err
	}
	if user.Login == "" || user.ID <= 0 {
		return userResponse{}, nil, fmt.Errorf("%w: /user response is missing identity fields", ErrProtocol)
	}
	scopes := parseScopes(headers.Get("X-OAuth-Scopes"))
	if len(scopes) == 0 {
		scopes = parseScopes(fallbackScope)
	}
	return user, scopes, nil
}

func (c *Client) verifyTarget(ctx context.Context, options resolvedOptions, accessToken string) (targetVerification, error) {
	if options.targetRepository == "" && options.appID == 0 {
		return targetVerification{}, nil
	}
	installations, err := c.listInstallations(ctx, options, accessToken)
	if err != nil {
		return targetVerification{}, err
	}
	candidates := make([]installation, 0, len(installations))
	for _, candidate := range installations {
		if candidate.ID <= 0 {
			continue
		}
		if options.appID != 0 && candidate.AppID != options.appID {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if options.appID != 0 && len(candidates) == 0 {
		return targetVerification{}, fmt.Errorf("%w: no installation for App ID %d is accessible", ErrScopeDenied, options.appID)
	}
	if options.targetRepository == "" {
		if len(candidates) == 0 {
			return targetVerification{}, nil
		}
		return targetVerification{installationID: candidates[0].ID}, nil
	}
	for _, candidate := range candidates {
		repositories, err := c.listRepositories(ctx, options, accessToken, candidate.ID)
		if err != nil {
			return targetVerification{}, err
		}
		for _, repository := range repositories {
			if strings.EqualFold(repository.FullName, options.targetRepository) {
				return targetVerification{installationID: candidate.ID, repository: options.targetRepository}, nil
			}
		}
	}
	return targetVerification{}, fmt.Errorf("%w: %s is not in the selected repositories of an accessible App installation", ErrScopeDenied, options.targetRepository)
}

func (c *Client) listInstallations(ctx context.Context, options resolvedOptions, accessToken string) ([]installation, error) {
	next := joinAPI(options.endpoints.APIBaseURL, "/user/installations", url.Values{"per_page": {"100"}})
	var installations []installation
	for page := 0; next != ""; page++ {
		if page >= 50 {
			return nil, fmt.Errorf("%w: installation pagination exceeded the safety limit", ErrProtocol)
		}
		var body installationsPage
		headers, err := c.getJSON(ctx, next, accessToken, &body)
		if err != nil {
			return nil, err
		}
		installations = append(installations, body.Installations...)
		link := parseNextLink(headers.Get("Link"))
		if link == "" {
			break
		}
		next, err = resolveNextPage(next, link, options.endpoints.APIBaseURL)
		if err != nil {
			return nil, err
		}
	}
	return installations, nil
}

func (c *Client) listRepositories(ctx context.Context, options resolvedOptions, accessToken string, installationID int64) ([]repository, error) {
	path := "/user/installations/" + strconv.FormatInt(installationID, 10) + "/repositories"
	next := joinAPI(options.endpoints.APIBaseURL, path, url.Values{"per_page": {"100"}})
	var repositories []repository
	for page := 0; next != ""; page++ {
		if page >= 50 {
			return nil, fmt.Errorf("%w: repository pagination exceeded the safety limit", ErrProtocol)
		}
		var body repositoriesPage
		headers, err := c.getJSON(ctx, next, accessToken, &body)
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, body.Repositories...)
		link := parseNextLink(headers.Get("Link"))
		if link == "" {
			break
		}
		next, err = resolveNextPage(next, link, options.endpoints.APIBaseURL)
		if err != nil {
			return nil, err
		}
	}
	return repositories, nil
}

func parseScopes(value string) []string {
	seen := make(map[string]bool)
	var scopes []string
	for _, part := range strings.Split(value, ",") {
		scope := strings.TrimSpace(part)
		if scope == "" || seen[scope] {
			continue
		}
		seen[scope] = true
		scopes = append(scopes, scope)
	}
	return scopes
}

func validateVerificationURI(raw, host string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Host == "" || parsed.Path == "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: GitHub returned an unsafe verification URI", ErrProtocol)
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return "", fmt.Errorf("%w: GitHub returned a non-HTTPS verification URI", ErrProtocol)
	}
	if !strings.EqualFold(parsed.Host, host) {
		return "", fmt.Errorf("%w: GitHub returned a verification URI for a different host", ErrProtocol)
	}
	return parsed.String(), nil
}
