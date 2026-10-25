package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zeithrold/tools/internal/githubauth"
	"github.com/zeithrold/tools/internal/keystore"
)

// githubClientID is the public GitHub App client id used for device flow. It is
// not a secret and may be injected at build time with
// -ldflags "-X main.githubClientID=<id>". The ZEITHROLD_GITHUB_CLIENT_ID
// environment variable and --client-id override it. It is never an App ID and
// never a token or PAT.
var githubClientID string

const (
	authServiceName    = "dev.zeithrold.zt"
	defaultAuthProfile = "default"
)

// authService is the narrow command-facing view of *githubauth.Client. Tests
// substitute a fake; production uses the real client, which never exposes a
// token through this port.
type authService interface {
	Login(ctx context.Context, opts githubauth.Options, onDevice func(githubauth.DevicePrompt) error) (githubauth.AccountSummary, error)
	Status(ctx context.Context, opts githubauth.Options, probe bool) (githubauth.AccountSummary, error)
	Logout(ctx context.Context, opts githubauth.Options) error
}

// The real client must satisfy the command port; this fails at compile time if
// the auth adapter's method set changes.
var _ authService = (*githubauth.Client)(nil)

// newAuthService opens the OS keystore and a cross-process credential lock for
// one host. keystore.Open is lazy: it selects a native backend but does not
// call Keychain/Credential Manager/Secret Service until a credential operation
// runs. NewLocker only computes the cache path; it creates the lock directory
// on first use. The factory is deferred to auth subcommands, so no other
// command constructs either dependency.
func newAuthService(host string) (authService, error) {
	store, err := keystore.Open(keystore.Options{Service: authServiceName})
	if err != nil {
		return nil, fmt.Errorf("open OS keystore: %w", err)
	}
	locker, err := keystore.NewLocker(keystore.LockerOptions{Service: authServiceName})
	if err != nil {
		return nil, fmt.Errorf("create credential lock: %w", err)
	}
	return &githubauth.Client{
		Store:        store,
		Locker:       locker,
		AllowedHosts: []string{host},
	}, nil
}

func newAuthCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage GitHub device-flow credentials stored in the OS keystore",
		Long:  "auth signs in with the GitHub device flow and stores the resulting credential only in the OS keystore. It never accepts a token, PAT, or environment token fallback, and it never prints a secret. Only login and status --probe access the network.",
		Args:  cobra.ArbitraryArgs,
		RunE:  authUnknownSubcommand,
	}
	// Auth errors must not echo unmatched input: a pasted token can appear in a
	// stray positional or an unknown flag name.
	cmd.SetFlagErrorFunc(sanitizeAuthFlagError)
	cmd.AddCommand(newAuthLoginCommand(a), newAuthStatusCommand(a), newAuthLogoutCommand(a))
	return cmd
}

func newAuthLoginCommand(a *app) *cobra.Command {
	var clientID, hostname, profile string
	var noBrowser, offlineAccess, replace, asJSON bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in with the GitHub device flow",
		Long:  "login starts a GitHub App device-flow authorization. It prints the short user code to stderr (or streams JSON events with --json) and never reads a form from stdin. By default it requests offline_access so the stored credential can refresh; pass --offline-access=false to opt out. The token stays in the OS keystore and is never printed.",
		Args:  noEchoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := a.resolveClientID(clientID)
			if err := validateClientID(id); err != nil {
				return err
			}
			host, err := normalizeHostname(hostname)
			if err != nil {
				return err
			}
			service, err := a.requireAuth(host)
			if err != nil {
				return err
			}
			summary, err := service.Login(cmd.Context(), githubauth.Options{
				Host:                   host,
				ClientID:               id,
				Profile:                normalizeProfile(profile),
				RequestOfflineAccess:   offlineAccess,
				ReplaceExistingAccount: replace,
			}, func(prompt githubauth.DevicePrompt) error {
				if asJSON {
					return writeJSON(a.stdout, loginEvent{
						SchemaVersion:           1,
						Event:                   "device_code",
						UserCode:                prompt.UserCode,
						VerificationURI:         prompt.VerificationURI,
						VerificationURIComplete: prompt.VerificationURIComplete,
						ExpiresInSecond:         int64(prompt.ExpiresIn / time.Second),
					})
				}
				if _, err := fmt.Fprintf(a.stderr, "Open %s and enter code %s\n", prompt.VerificationURI, prompt.UserCode); err != nil {
					return err
				}
				// Only launch a trusted HTTPS page on the selected host, and never
				// from a non-interactive session.
				if noBrowser || a.noInput || !a.isTTY() {
					return nil
				}
				target := prompt.VerificationURI
				if prompt.VerificationURIComplete != "" {
					target = prompt.VerificationURIComplete
				}
				if trusted, err := trustedVerificationURL(target, host); err == nil {
					_ = a.openBrowser(trusted)
				}
				return nil
			})
			if err != nil {
				if errors.Is(err, githubauth.ErrAccountConflict) {
					return fmt.Errorf("%w; pass --replace to replace the existing credential for this profile", err)
				}
				return wrapAuthError(err)
			}
			if asJSON {
				report := accountFromSummary(summary)
				if report.Host == "" {
					report.Host = host
				}
				return writeJSON(a.stdout, loginEvent{SchemaVersion: 1, Event: "authorized", Account: &report})
			}
			_, err = fmt.Fprintf(a.stdout, "Logged in as %s on %s\n", accountLabel(summary), summaryHost(summary, host))
			return err
		},
	}
	cmd.Flags().StringVar(&clientID, "client-id", "", "public GitHub App client id (default: ZEITHROLD_GITHUB_CLIENT_ID or build value)")
	cmd.Flags().StringVar(&hostname, "hostname", "github.com", "GitHub host name")
	cmd.Flags().StringVar(&profile, "profile", defaultAuthProfile, "stored credential profile (account)")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "do not try to open a browser")
	cmd.Flags().BoolVar(&offlineAccess, "offline-access", true, "request offline_access for expiring tokens plus a refresh token; set false to opt out")
	cmd.Flags().BoolVar(&replace, "replace", false, "replace an existing credential for this profile")
	cmd.Flags().BoolVar(&asJSON, "json", false, "stream machine-readable device and completion events")
	return cmd
}

type loginEvent struct {
	SchemaVersion           int            `json:"schemaVersion"`
	Event                   string         `json:"event"`
	UserCode                string         `json:"userCode,omitempty"`
	VerificationURI         string         `json:"verificationUri,omitempty"`
	VerificationURIComplete string         `json:"verificationUriComplete,omitempty"`
	ExpiresInSecond         int64          `json:"expiresInSeconds,omitempty"`
	Account                 *accountReport `json:"account,omitempty"`
}

// accountReport is the CLI's token-free JSON view of an account summary. Times
// are RFC 3339 strings so a zero time is omitted rather than encoded as year 1.
type accountReport struct {
	Host             string   `json:"host"`
	Profile          string   `json:"profile,omitempty"`
	Login            string   `json:"login,omitempty"`
	UserID           int64    `json:"userId,omitempty"`
	Scopes           []string `json:"scopes,omitempty"`
	TokenExpiresAt   string   `json:"tokenExpiresAt,omitempty"`
	RefreshExpiresAt string   `json:"refreshExpiresAt,omitempty"`
	CanRefresh       bool     `json:"canRefresh"`
	InstallationID   int64    `json:"installationId,omitempty"`
	TargetRepository string   `json:"targetRepository,omitempty"`
	Probed           bool     `json:"probed"`
	VerifiedAt       string   `json:"verifiedAt,omitempty"`
}

func accountFromSummary(summary githubauth.AccountSummary) accountReport {
	return accountReport{
		Host:             summary.Host,
		Profile:          summary.Profile,
		Login:            summary.Login,
		UserID:           summary.UserID,
		Scopes:           summary.Scopes,
		TokenExpiresAt:   formatTime(summary.TokenExpiresAt),
		RefreshExpiresAt: formatTime(summary.RefreshExpiresAt),
		CanRefresh:       summary.CanRefresh,
		InstallationID:   summary.InstallationID,
		TargetRepository: summary.TargetRepository,
		Probed:           summary.Probed,
		VerifiedAt:       formatTime(summary.VerifiedAt),
	}
}

func newAuthStatusCommand(a *app) *cobra.Command {
	var hostname, clientID, profile string
	var probe, asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report local credential metadata, optionally probing GitHub",
		Long:  "status reads only local keystore metadata by default and does not access the network. --probe explicitly asks GitHub to verify the stored identity and target access; the response reports only identity, scope, and expiry, never a token.",
		Args:  noEchoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			host, err := normalizeHostname(hostname)
			if err != nil {
				return err
			}
			id := a.resolveClientID(clientID)
			if err := validateClientIDFor("status", id); err != nil {
				return err
			}
			service, err := a.requireAuth(host)
			if err != nil {
				return err
			}
			summary, err := service.Status(cmd.Context(), githubauth.Options{
				Host:     host,
				ClientID: id,
				Profile:  normalizeProfile(profile),
			}, probe)
			if errors.Is(err, githubauth.ErrNotLoggedIn) {
				if asJSON {
					return writeJSON(a.stdout, statusReport{SchemaVersion: 1, LoggedIn: false, State: "absent", accountReport: accountReport{Host: host}})
				}
				_, err := fmt.Fprintf(a.stdout, "%s: not logged in (os-keystore)\n", host)
				return err
			}
			if err != nil {
				return wrapAuthError(err)
			}
			if asJSON {
				report := accountFromSummary(summary)
				if report.Host == "" {
					report.Host = host
				}
				return writeJSON(a.stdout, statusReport{SchemaVersion: 1, LoggedIn: true, State: "logged_in", accountReport: report})
			}
			return renderAuthStatus(a.stdout, host, summary)
		},
	}
	cmd.Flags().StringVar(&hostname, "hostname", "github.com", "GitHub host name")
	cmd.Flags().StringVar(&clientID, "client-id", "", "public GitHub App client id that keys the stored record")
	cmd.Flags().StringVar(&profile, "profile", defaultAuthProfile, "stored credential profile (account)")
	cmd.Flags().BoolVar(&probe, "probe", false, "contact GitHub to verify the stored identity (network access)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable status")
	return cmd
}

type statusReport struct {
	SchemaVersion int    `json:"schemaVersion"`
	LoggedIn      bool   `json:"loggedIn"`
	State         string `json:"state"` // logged_in or absent
	accountReport
}

func renderAuthStatus(output io.Writer, host string, summary githubauth.AccountSummary) error {
	if summary.Host == "" {
		summary.Host = host
	}
	if _, err := fmt.Fprintf(output, "%s: logged in (os-keystore)\n", summary.Host); err != nil {
		return err
	}
	if summary.Profile != "" {
		if _, err := fmt.Fprintf(output, "  profile: %s\n", summary.Profile); err != nil {
			return err
		}
	}
	if summary.Login != "" {
		label := summary.Login
		if summary.UserID != 0 {
			label = fmt.Sprintf("%s (id %d)", summary.Login, summary.UserID)
		}
		if _, err := fmt.Fprintf(output, "  account: %s\n", label); err != nil {
			return err
		}
	}
	if len(summary.Scopes) > 0 {
		if _, err := fmt.Fprintf(output, "  scopes: %s\n", strings.Join(summary.Scopes, ", ")); err != nil {
			return err
		}
	}
	if !summary.TokenExpiresAt.IsZero() {
		if _, err := fmt.Fprintf(output, "  token expires: %s\n", formatTime(summary.TokenExpiresAt)); err != nil {
			return err
		}
	}
	if summary.CanRefresh {
		line := "  token can refresh"
		if !summary.RefreshExpiresAt.IsZero() {
			line += " until " + formatTime(summary.RefreshExpiresAt)
		}
		if _, err := fmt.Fprintln(output, line); err != nil {
			return err
		}
	}
	if summary.InstallationID != 0 {
		if _, err := fmt.Fprintf(output, "  installation: %d\n", summary.InstallationID); err != nil {
			return err
		}
	}
	if summary.TargetRepository != "" {
		if _, err := fmt.Fprintf(output, "  target repository: %s\n", summary.TargetRepository); err != nil {
			return err
		}
	}
	if summary.Probed {
		line := "  probe: verified"
		if !summary.VerifiedAt.IsZero() {
			line += " at " + formatTime(summary.VerifiedAt)
		}
		if _, err := fmt.Fprintln(output, line); err != nil {
			return err
		}
	}
	return nil
}

func newAuthLogoutCommand(a *app) *cobra.Command {
	var hostname, clientID, profile string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove a locally stored credential without revoking it remotely",
		Long:  "logout removes only the local OS keystore credential for the selected host, client id, and profile. It does not revoke the token on GitHub and does not touch other hosts, clients, or accounts.",
		Args:  noEchoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			host, err := normalizeHostname(hostname)
			if err != nil {
				return err
			}
			id := a.resolveClientID(clientID)
			if err := validateClientIDFor("logout", id); err != nil {
				return err
			}
			service, err := a.requireAuth(host)
			if err != nil {
				return err
			}
			selectedProfile := normalizeProfile(profile)
			// Logout is idempotent: a valid key with no stored item is success.
			if err := service.Logout(cmd.Context(), githubauth.Options{
				Host:     host,
				ClientID: id,
				Profile:  selectedProfile,
			}); err != nil {
				return wrapAuthError(err)
			}
			if asJSON {
				return writeJSON(a.stdout, logoutReport{SchemaVersion: 1, Host: host, Profile: selectedProfile, LoggedOut: true})
			}
			_, err = fmt.Fprintf(a.stdout, "Logout complete for profile %q on %s: no local credential remains (no remote revocation was attempted)\n", selectedProfile, host)
			return err
		},
	}
	cmd.Flags().StringVar(&hostname, "hostname", "github.com", "GitHub host name")
	cmd.Flags().StringVar(&clientID, "client-id", "", "public GitHub App client id that keys the stored record")
	cmd.Flags().StringVar(&profile, "profile", defaultAuthProfile, "stored credential profile (account) to remove")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable result")
	return cmd
}

type logoutReport struct {
	SchemaVersion int    `json:"schemaVersion"`
	Host          string `json:"host"`
	Profile       string `json:"profile"`
	LoggedOut     bool   `json:"loggedOut"`
}

// wrapAuthError adds retryable guidance to cross-process coordination failures
// without losing errors.Is identity. It never retries automatically.
func wrapAuthError(err error) error {
	switch {
	case errors.Is(err, githubauth.ErrConcurrentRefresh):
		return fmt.Errorf("%w; another zt process changed the stored credentials, retry", err)
	case errors.Is(err, githubauth.ErrLocked):
		return fmt.Errorf("%w; unlock the OS credential store or retry after another zt process finishes", err)
	}
	return err
}

// requireAuth fails closed when the build or test provides no auth factory. It
// never falls back to a fake keystore or a token from the environment.
func (a *app) requireAuth(host string) (authService, error) {
	if a.newAuth == nil {
		return nil, fmt.Errorf("GitHub authentication is unavailable in this build: the githubauth/keystore adapter is not linked")
	}
	return a.newAuth(host)
}

// resolveClientID applies flag > ZEITHROLD_GITHUB_CLIENT_ID > build value. It
// reads only the public client-id override, never a token.
func (a *app) resolveClientID(flagValue string) string {
	if id := strings.TrimSpace(flagValue); id != "" {
		return id
	}
	if a.getenv != nil {
		if id := strings.TrimSpace(a.getenv("ZEITHROLD_GITHUB_CLIENT_ID")); id != "" {
			return id
		}
	}
	return strings.TrimSpace(githubClientID)
}

func normalizeProfile(profile string) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return defaultAuthProfile
	}
	return profile
}

func accountLabel(summary githubauth.AccountSummary) string {
	if summary.Login == "" {
		return "(unknown account)"
	}
	return summary.Login
}

func summaryHost(summary githubauth.AccountSummary, fallback string) string {
	if summary.Host == "" {
		return fallback
	}
	return summary.Host
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

var hostnamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$`)

// validateClientID requires the public client id of a GitHub App. A numeric App
// ID cannot start device flow, and an OAuth App client id (Ov...) would request
// user authorization for the wrong application. The same id keys the stored
// record, so status and logout require it too.
func validateClientID(id string) error {
	return validateClientIDFor("login", id)
}

func validateClientIDFor(command, id string) error {
	if id == "" {
		return fmt.Errorf("auth %s requires the public GitHub App client id that keys the stored credential: pass --client-id, set ZEITHROLD_GITHUB_CLIENT_ID, or use a build with an embedded client id", command)
	}
	if strings.ContainsAny(id, " \t\r\n") {
		return fmt.Errorf("auth %s --client-id must not contain whitespace", command)
	}
	numeric := true
	for _, r := range id {
		if r < '0' || r > '9' {
			numeric = false
			break
		}
	}
	if numeric {
		return fmt.Errorf("auth %s --client-id must be a GitHub App client id, not a numeric App ID", command)
	}
	if strings.HasPrefix(id, "Ov") {
		return fmt.Errorf("auth %s --client-id looks like an OAuth App client id; device flow requires a GitHub App client id", command)
	}
	if !strings.HasPrefix(id, "Iv1.") && !strings.HasPrefix(id, "Iv23") {
		return fmt.Errorf("auth %s --client-id is not a GitHub App client id (expected Iv1... or Iv23...)", command)
	}
	return nil
}

// normalizeHostname accepts a bare DNS host name and rejects URLs and paths so
// a caller cannot smuggle a scheme, credential, or redirect target through a
// flag. Errors never echo the raw input, which may contain a pasted token.
func normalizeHostname(raw string) (string, error) {
	host := strings.ToLower(strings.TrimSpace(raw))
	if host == "" {
		return "", fmt.Errorf("--hostname must not be empty")
	}
	if strings.Contains(host, "://") || strings.ContainsAny(host, "/\\@ \t") {
		return "", fmt.Errorf("--hostname must be a bare host name such as github.com, not a URL")
	}
	if !strings.Contains(host, ".") || !hostnamePattern.MatchString(host) {
		return "", fmt.Errorf("--hostname must be a bare host name such as github.com")
	}
	return host, nil
}

// trustedVerificationURL rejects a verification page that is not HTTPS on the
// host the user selected, so a server response cannot launch an arbitrary URL.
// Errors never echo the raw URI.
func trustedVerificationURL(raw, host string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("verification URI is not a valid URL")
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("verification URI is not HTTPS")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("verification URI contains credentials")
	}
	if !strings.EqualFold(parsed.Hostname(), host) {
		return "", fmt.Errorf("verification URI host does not match the selected host")
	}
	return parsed.String(), nil
}
