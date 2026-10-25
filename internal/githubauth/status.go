package githubauth

import (
	"context"
	"errors"
	"fmt"
)

// Status reports the stored account. With probe=false it performs no network
// request and does not claim that the target is currently accessible. With
// probe=true it refreshes an expired or rejected token when possible, calls
// GET /user, and verifies the configured installation/repository. It never
// returns a token.
func (c *Client) Status(ctx context.Context, opts Options, probe bool) (AccountSummary, error) {
	options, err := c.resolve(opts)
	if err != nil {
		return AccountSummary{}, err
	}
	if c.Store == nil {
		return AccountSummary{}, fmt.Errorf("%w", ErrStoreUnavailable)
	}
	record, found, err := c.load(ctx, options)
	if err != nil {
		return AccountSummary{}, err
	}
	if !found {
		return AccountSummary{}, fmt.Errorf("%w", ErrNotLoggedIn)
	}
	if !probe {
		return record.summary(false), nil
	}

	if !tokenValid(c.now(), timeFromUnix(record.TokenExpiresAt)) {
		record, err = c.refresh(ctx, options, record)
		if err != nil {
			return AccountSummary{}, err
		}
	}
	identity, scopes, err := c.verifyUser(ctx, options, record.AccessToken, "")
	if errors.Is(err, ErrReloginRequired) {
		record, err = c.refresh(ctx, options, record)
		if err != nil {
			return AccountSummary{}, err
		}
		identity, scopes, err = c.verifyUser(ctx, options, record.AccessToken, "")
	}
	if err != nil {
		return AccountSummary{}, err
	}
	if identity.ID != record.UserID {
		return AccountSummary{}, fmt.Errorf("%w: token belongs to user ID %d but the stored record holds user ID %d", ErrAccountMismatch, identity.ID, record.UserID)
	}
	// The login can be renamed while the stable user ID stays the same. Report
	// the current login rather than the stale stored string.
	record.Login = identity.Login
	if len(scopes) > 0 {
		record.Scopes = scopes
	}

	target, err := c.verifyTarget(ctx, options, record.AccessToken)
	if err != nil {
		return AccountSummary{}, err
	}
	if options.targetRepository != "" {
		record.TargetRepository = target.repository
		record.InstallationID = target.installationID
	} else if target.installationID != 0 {
		record.InstallationID = target.installationID
	}
	record.VerifiedAt = c.now().Unix()
	return record.summary(true), nil
}

// Logout removes the stored record. It is idempotent and does not contact
// GitHub; remote token revocation requires a separate explicit operation.
func (c *Client) Logout(ctx context.Context, opts Options) error {
	options, err := c.resolve(opts)
	if err != nil {
		return err
	}
	if c.Store == nil {
		return fmt.Errorf("%w", ErrStoreUnavailable)
	}
	release, err := c.lock(ctx, options.key)
	if err != nil {
		return err
	}
	defer release()
	if err := c.Store.Delete(ctx, options.key); err != nil {
		return mapStoreError("delete GitHub credentials", err)
	}
	return nil
}
