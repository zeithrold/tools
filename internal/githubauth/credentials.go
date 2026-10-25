package githubauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zeithrold/tools/internal/keystore"
)

const credentialVersion = 1

// credentialRecord is the complete, opaque value written to the native store.
// Access and refresh tokens are always written together: a failed Set leaves the
// previous record untouched and never stores half of a rotated pair.
type credentialRecord struct {
	Version          int      `json:"v"`
	Host             string   `json:"host"`
	ClientID         string   `json:"client"`
	Profile          string   `json:"profile"`
	Login            string   `json:"login"`
	UserID           int64    `json:"uid"`
	AccessToken      string   `json:"at"`
	RefreshToken     string   `json:"rt,omitempty"`
	TokenExpiresAt   int64    `json:"atExp,omitempty"`
	RefreshExpiresAt int64    `json:"rtExp,omitempty"`
	Scopes           []string `json:"scopes,omitempty"`
	InstallationID   int64    `json:"inst,omitempty"`
	TargetRepository string   `json:"repo,omitempty"`
	VerifiedAt       int64    `json:"verified,omitempty"`
}

func (r credentialRecord) validate(options resolvedOptions) error {
	if r.Version != credentialVersion {
		return fmt.Errorf("%w: unsupported credential record version %d", ErrCredentialCorrupt, r.Version)
	}
	if r.Host != options.host || r.ClientID != options.clientID || r.Profile != options.profile {
		return fmt.Errorf("%w", ErrCredentialMismatch)
	}
	if r.Login == "" || r.UserID <= 0 || r.AccessToken == "" {
		return fmt.Errorf("%w: required identity or token fields are missing", ErrCredentialCorrupt)
	}
	return nil
}

func (r credentialRecord) summary(probed bool) AccountSummary {
	scopes := append([]string(nil), r.Scopes...)
	return AccountSummary{
		Host:             r.Host,
		Profile:          r.Profile,
		Login:            r.Login,
		UserID:           r.UserID,
		Scopes:           scopes,
		TokenExpiresAt:   timeFromUnix(r.TokenExpiresAt),
		RefreshExpiresAt: timeFromUnix(r.RefreshExpiresAt),
		CanRefresh:       r.RefreshToken != "",
		InstallationID:   r.InstallationID,
		TargetRepository: r.TargetRepository,
		Probed:           probed,
		VerifiedAt:       timeFromUnix(r.VerifiedAt),
	}
}

func (c *Client) load(ctx context.Context, options resolvedOptions) (credentialRecord, bool, error) {
	if c.Store == nil {
		return credentialRecord{}, false, fmt.Errorf("%w", ErrStoreUnavailable)
	}
	data, found, err := c.Store.Get(ctx, options.key)
	if err != nil {
		return credentialRecord{}, false, mapStoreError("read GitHub credentials", err)
	}
	if !found {
		return credentialRecord{}, false, nil
	}
	var record credentialRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return credentialRecord{}, true, fmt.Errorf("%w: stored record cannot be decoded", ErrCredentialCorrupt)
	}
	if err := record.validate(options); err != nil {
		return credentialRecord{}, true, err
	}
	return record, true, nil
}

func (c *Client) save(ctx context.Context, options resolvedOptions, record credentialRecord) error {
	if c.Store == nil {
		return fmt.Errorf("%w", ErrStoreUnavailable)
	}
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("%w: could not encode credentials", ErrProtocol)
	}
	if len(data) > keystore.MaxSecretSize {
		return fmt.Errorf("%w: %d bytes exceeds the %d byte native limit; no token or refresh token was written and there is no file fallback", ErrCredentialTooLarge, len(data), keystore.MaxSecretSize)
	}
	if err := c.Store.Set(ctx, options.key, data); err != nil {
		return mapStoreError("save GitHub credentials", err)
	}
	return nil
}

func mapStoreError(action string, err error) error {
	switch {
	case errors.Is(err, keystore.ErrUnavailable):
		return fmt.Errorf("%w: %s: %v", ErrStoreUnavailable, action, err)
	case errors.Is(err, keystore.ErrLocked):
		return fmt.Errorf("%w: %s: %v", ErrLocked, action, err)
	case errors.Is(err, keystore.ErrDenied):
		return fmt.Errorf("%w: %s: %v", ErrDenied, action, err)
	case errors.Is(err, keystore.ErrTooLarge):
		return fmt.Errorf("%w: %s: %v", ErrCredentialTooLarge, action, err)
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func (c *Client) now() time.Time {
	if c.clock == nil {
		return time.Now()
	}
	return c.clock.Now()
}

func (c *Client) sleep(ctx context.Context, duration time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.clock == nil {
		if duration <= 0 {
			return nil
		}
		timer := time.NewTimer(duration)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
	return c.clock.Sleep(ctx, duration)
}

type refreshError struct {
	definitive bool
	err        error
}

func (e *refreshError) Error() string { return e.err.Error() }
func (e *refreshError) Unwrap() error { return e.err }

func (c *Client) lock(ctx context.Context, key string) (func(), error) {
	if c.Locker == nil {
		return func() {}, nil
	}
	unlocker, err := c.Locker.Lock(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: could not acquire the cross-process credential lock; another zt process may be using this profile: %v", ErrLocked, err)
	}
	if unlocker == nil {
		return nil, fmt.Errorf("%w: credential locker returned no unlocker", ErrLocked)
	}
	return func() { _ = unlocker.Unlock() }, nil
}

func recordsDiffer(left, right credentialRecord) bool {
	return left.AccessToken != right.AccessToken || left.RefreshToken != right.RefreshToken || left.UserID != right.UserID
}

func (c *Client) refresh(ctx context.Context, options resolvedOptions, expected credentialRecord) (credentialRecord, error) {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	release, err := c.lock(ctx, options.key)
	if err != nil {
		return credentialRecord{}, err
	}
	defer release()

	current, found, err := c.load(ctx, options)
	if err != nil {
		return credentialRecord{}, err
	}
	if !found {
		return credentialRecord{}, fmt.Errorf("%w", ErrNotLoggedIn)
	}
	if current.UserID != expected.UserID {
		return credentialRecord{}, fmt.Errorf("%w: stored user ID %d does not match the account being refreshed (user ID %d)", ErrAccountMismatch, current.UserID, expected.UserID)
	}
	now := c.now()
	if current.AccessToken != expected.AccessToken && tokenValid(now, timeFromUnix(current.TokenExpiresAt)) {
		return current, nil
	}
	if current.RefreshToken == "" {
		return credentialRecord{}, fmt.Errorf("%w: no refresh token is stored; request offline_access on login", ErrReloginRequired)
	}

	response, err := c.exchangeRefresh(ctx, options, current.RefreshToken)
	if err != nil {
		var refreshErr *refreshError
		if errors.As(err, &refreshErr) && refreshErr.definitive {
			latest, latestFound, latestErr := c.load(ctx, options)
			if latestErr != nil {
				return credentialRecord{}, fmt.Errorf("%w: refresh was rejected but the stored record could not be re-read; no record was deleted: %v", ErrReloginRequired, latestErr)
			}
			if latestFound && recordsDiffer(current, latest) {
				return credentialRecord{}, fmt.Errorf("%w: another zt process refreshed these credentials; retry the operation", ErrConcurrentRefresh)
			}
			_ = c.Store.Delete(ctx, options.key)
			return credentialRecord{}, fmt.Errorf("%w: %v", ErrReloginRequired, refreshErr)
		}
		return credentialRecord{}, err
	}

	latest, latestFound, latestErr := c.load(ctx, options)
	if latestErr != nil {
		return credentialRecord{}, fmt.Errorf("%w: could not re-read the stored record before saving; no token was written: %v", ErrStoreUnavailable, latestErr)
	}
	if latestFound && recordsDiffer(current, latest) {
		return credentialRecord{}, fmt.Errorf("%w: another zt process refreshed these credentials; the newer record was left untouched", ErrConcurrentRefresh)
	}

	updated := current
	updated.AccessToken = response.AccessToken
	updated.TokenExpiresAt = unixExpiry(now, response.ExpiresIn)
	if response.RefreshToken != "" {
		updated.RefreshToken = response.RefreshToken
		updated.RefreshExpiresAt = unixExpiry(now, response.RefreshTokenExpiresIn)
	}
	if scopes := parseScopes(response.Scope); len(scopes) > 0 {
		updated.Scopes = scopes
	}
	if err := c.save(ctx, options, updated); err != nil {
		return credentialRecord{}, fmt.Errorf("%w: refreshed credentials could not be saved as one atomic record: %v", ErrReloginRequired, err)
	}
	return updated, nil
}
