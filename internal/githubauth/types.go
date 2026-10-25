// Package githubauth logs a user in to a GitHub App with the OAuth 2.0 device
// authorization flow, verifies the authenticated user and target installation,
// and keeps the resulting user access and refresh tokens in the operating
// system credential store.
//
// The package has no CLI, Cobra, shell, gh, environment-variable, or file
// coupling. It never accepts a client secret, App private key, or personal
// access token: a GitHub App's public client ID is the only authorization
// client identifier. The App ID is not used as the client ID.
//
// Tokens are returned only to the Store. Public results and errors never
// contain a device code, access token, or refresh token.
package githubauth

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/zeithrold/tools/internal/keystore"
)

// Store is the native credential store contract from internal/keystore. zt
// never falls back to a file, environment variable, command argument, or PAT.
type Store = keystore.Store

// Locker and Unlocker are the cross-process credential-mutation lock from
// internal/keystore. Client only needs the interface; callers normally pass
// keystore.NewLocker(...). Without one, refresh still refuses to overwrite or
// delete a record changed by another process, but cannot close the final
// read-to-write race.
type Locker = keystore.Locker
type Unlocker = keystore.Unlocker

// clock lets package tests replace wall time and sleeping.
type clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

// endpoints overrides the URLs derived from Options.Host for httptest
// coverage. Every endpoint must use HTTPS (plain HTTP is accepted only for
// loopback tests) and the expected host; cross-host redirects are rejected.
type endpoints struct {
	DeviceCodeURL string
	TokenURL      string
	APIBaseURL    string
}

// Client is the GitHub authorization client. Store and AllowedHosts are
// supplied by the CLI; Locker is normally keystore.NewLocker(service) so Login,
// Logout, and refresh are serialized across zt processes. The zero value uses a
// 30-second HTTP timeout, the real clock, and no store (operations that need a
// store fail with ErrStoreUnavailable).
type Client struct {
	Store        Store
	Locker       Locker
	AllowedHosts []string

	httpClient *http.Client
	clock      clock
	endpoints  endpoints
	refreshMu  sync.Mutex
}

// Options selects one credential profile and the target to verify. Host
// defaults to github.com; Profile defaults to "default". ClientID is the
// GitHub App's public client ID. AppID, when set, is used only to select the
// matching installation; it is never sent as the device-flow client ID.
//
// Only the GitHub App scope offline_access is requested, and only when
// RequestOfflineAccess is set. This package does not accept arbitrary OAuth
// scopes.
type Options struct {
	Host                   string
	ClientID               string
	Profile                string
	RequestOfflineAccess   bool
	TargetRepository       string
	AppID                  int64
	ReplaceExistingAccount bool
}

// DevicePrompt is the safe subset of the device-code response shown to the
// user. It never contains the device code or a token. The package does not open
// either URI; it only validates that both use HTTPS on the configured host.
type DevicePrompt struct {
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               time.Duration
	Interval                time.Duration
}

// AccountSummary is the non-secret result of Login or Status. It never contains
// a token. Probed reports whether this result was checked against GitHub in this
// call. VerifiedAt is the last successful identity/target verification.
type AccountSummary struct {
	Host    string `json:"host"`
	Profile string `json:"profile"`
	Login   string `json:"login"`
	UserID  int64  `json:"userId"`
	// Scopes is observed from GitHub (X-OAuth-Scopes or the token response).
	// This package only ever requests offline_access; it never invents OAuth
	// scopes for a GitHub App.
	Scopes           []string  `json:"scopes,omitempty"`
	TokenExpiresAt   time.Time `json:"tokenExpiresAt,omitempty"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt,omitempty"`
	CanRefresh       bool      `json:"canRefresh"`
	InstallationID   int64     `json:"installationId,omitempty"`
	TargetRepository string    `json:"targetRepository,omitempty"`
	Probed           bool      `json:"probed"`
	VerifiedAt       time.Time `json:"verifiedAt,omitempty"`
}

// Summary is an alias kept for callers that prefer the shorter name.
type Summary = AccountSummary
