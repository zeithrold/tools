package githubauth

import "errors"

var (
	// ErrNotLoggedIn means no credential record is stored for the selected
	// host, client ID, and profile.
	ErrNotLoggedIn = errors.New("no stored GitHub credentials")
	// ErrStoreUnavailable means the native credential store cannot be used.
	// There is no plaintext fallback.
	ErrStoreUnavailable = errors.New("secure credential store unavailable")
	// ErrLocked means the native credential store is locked or needs
	// interactive approval.
	ErrLocked = errors.New("secure credential store locked")
	// ErrDenied means the native credential store refused access.
	ErrDenied = errors.New("secure credential store access denied")
	// ErrAccountConflict means a different GitHub account is already stored
	// for this profile and ReplaceExistingAccount was not explicitly set.
	ErrAccountConflict = errors.New("a different GitHub account is already stored")
	// ErrAccountMismatch means the authenticated user is not the account whose
	// credentials were stored.
	ErrAccountMismatch = errors.New("authenticated GitHub account does not match the stored account")
	// ErrCredentialMismatch means a stored record belongs to another host,
	// client ID, or profile than the requested one.
	ErrCredentialMismatch = errors.New("stored credentials belong to a different host, client, or profile")
	// ErrCredentialCorrupt means the stored record cannot be decoded or is
	// missing required fields.
	ErrCredentialCorrupt = errors.New("stored GitHub credential record is invalid")
	// ErrCredentialTooLarge means the complete credential record (access and
	// refresh tokens together with metadata) exceeds the native store limit.
	// Nothing is written in that case; callers must not drop the refresh token
	// or fall back to plaintext storage.
	ErrCredentialTooLarge = errors.New("GitHub credential record exceeds the secure credential store limit")
	// ErrReloginRequired means the access or refresh token was rejected and the
	// user must authorize again.
	ErrReloginRequired = errors.New("GitHub credentials are no longer valid; log in again")
	// ErrConcurrentRefresh means another zt process changed the stored credential
	// while this process was refreshing. The newer record is left untouched;
	// retry the operation or configure Client.Locker for cross-process exclusion.
	ErrConcurrentRefresh = errors.New("another zt process changed the stored GitHub credentials")
	// ErrAccessDenied means the user denied the device authorization.
	ErrAccessDenied = errors.New("GitHub authorization was denied")
	// ErrDeviceCodeExpired means the device code expired before authorization.
	ErrDeviceCodeExpired = errors.New("GitHub device code expired")
	// ErrAuthorizationRejected means GitHub rejected the device or refresh
	// request, for example because the public client ID is wrong.
	ErrAuthorizationRejected = errors.New("GitHub rejected the authorization request")
	// ErrScopeDenied means the requested repository or installation is not
	// available to the authenticated user and App installation.
	ErrScopeDenied = errors.New("target repository is not available to the authenticated account")
	// ErrProtocol means GitHub returned a response this package cannot safely
	// interpret.
	ErrProtocol = errors.New("unexpected GitHub response")
	// ErrInvalidOptions means the caller supplied an invalid host, client ID,
	// profile, scope, repository, or endpoint configuration.
	ErrInvalidOptions = errors.New("invalid GitHub authentication options")
	// ErrRemote means a GitHub request failed transiently. Stored credentials
	// are left in place for a later retry.
	ErrRemote = errors.New("GitHub request failed")
)
