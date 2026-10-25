package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zeithrold/tools/internal/githubauth"
)

// fakeAuth is a deterministic authService for CLI tests. It records the
// options each command passes and never touches a keystore or the network.
type fakeAuth struct {
	account   githubauth.AccountSummary
	prompt    githubauth.DevicePrompt
	loginErr  error
	statusErr error
	logoutErr error

	loginCalls  int
	statusCalls int
	logoutCalls int

	lastOptions githubauth.Options
	lastProbe   bool
	lastLogout  githubauth.Options
}

func (f *fakeAuth) Login(ctx context.Context, opts githubauth.Options, onDevice func(githubauth.DevicePrompt) error) (githubauth.AccountSummary, error) {
	f.loginCalls++
	f.lastOptions = opts
	if f.loginErr != nil {
		return githubauth.AccountSummary{}, f.loginErr
	}
	select {
	case <-ctx.Done():
		return githubauth.AccountSummary{}, ctx.Err()
	default:
	}
	if onDevice != nil {
		if err := onDevice(f.prompt); err != nil {
			return githubauth.AccountSummary{}, err
		}
	}
	return f.account, nil
}

func (f *fakeAuth) Status(ctx context.Context, opts githubauth.Options, probe bool) (githubauth.AccountSummary, error) {
	f.statusCalls++
	f.lastOptions = opts
	f.lastProbe = probe
	if f.statusErr != nil {
		return githubauth.AccountSummary{}, f.statusErr
	}
	return f.account, nil
}

func (f *fakeAuth) Logout(ctx context.Context, opts githubauth.Options) error {
	f.logoutCalls++
	f.lastLogout = opts
	return f.logoutErr
}

func runAuthCLI(t *testing.T, auth authService, args ...string) (string, string, error) {
	return runAuthCLIWith(t, auth, nil, args...)
}

func runAuthCLIWith(t *testing.T, auth authService, configure func(*app), args ...string) (string, string, error) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.getenv = func(string) string { return "" }
	a.newAuth = func(string) (authService, error) { return auth, nil }
	if configure != nil {
		configure(a)
	}
	err := executeApp(a, context.Background(), args)
	return stdout.String(), stderr.String(), err
}

func TestAuthHelpListsSubcommandsAndNoTokenFlag(t *testing.T) {
	stdout, _, err := runAuthCLI(t, &fakeAuth{}, "auth", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"login", "status", "logout"} {
		if !strings.Contains(stdout, name) {
			t.Fatalf("auth help omits %q:\n%s", name, stdout)
		}
	}
	loginHelp, _, err := runAuthCLI(t, &fakeAuth{}, "auth", "login", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--client-id", "--hostname", "--no-browser", "--offline-access", "--replace", "--profile"} {
		if !strings.Contains(loginHelp, flag) {
			t.Fatalf("login help omits %q:\n%s", flag, loginHelp)
		}
	}
	if strings.Contains(loginHelp, "--token") || strings.Contains(loginHelp, "--app-id") {
		t.Fatalf("login help advertises a forbidden flag:\n%s", loginHelp)
	}
}

func TestAuthLoginKeepsDeviceCodeOffStdout(t *testing.T) {
	auth := &fakeAuth{
		account: githubauth.AccountSummary{Host: "github.com", Profile: "default", Login: "octocat"},
		prompt:  githubauth.DevicePrompt{UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device", ExpiresIn: 15 * time.Minute},
	}
	stdout, stderr, err := runAuthCLI(t, auth, "auth", "login", "--client-id", "Iv1.test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "Logged in as octocat on github.com") {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if strings.Contains(stdout, "ABCD-1234") {
		t.Fatalf("device code leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "ABCD-1234") || !strings.Contains(stderr, "https://github.com/login/device") {
		t.Fatalf("device guidance missing from stderr: %q", stderr)
	}
	if auth.lastOptions.ClientID != "Iv1.test" || auth.lastOptions.Host != "github.com" || auth.lastOptions.Profile != "default" {
		t.Fatalf("unexpected login options: %+v", auth.lastOptions)
	}
}

func TestAuthLoginJSONStreamsEventsOnStdout(t *testing.T) {
	auth := &fakeAuth{
		account: githubauth.AccountSummary{Host: "github.com", Profile: "default", Login: "octocat", UserID: 7},
		prompt:  githubauth.DevicePrompt{UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device", ExpiresIn: 15 * time.Minute},
	}
	stdout, stderr, err := runAuthCLI(t, auth, "auth", "login", "--client-id", "Iv1.test", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Fatalf("JSON login wrote human text to stderr: %q", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two JSON events, got %d: %q", len(lines), stdout)
	}
	var device, authorized struct {
		Event    string `json:"event"`
		UserCode string `json:"userCode"`
		Account  struct {
			Login string `json:"login"`
		} `json:"account"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &device); err != nil {
		t.Fatalf("invalid device event %q: %v", lines[0], err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &authorized); err != nil {
		t.Fatalf("invalid authorized event %q: %v", lines[1], err)
	}
	if device.Event != "device_code" || device.UserCode != "ABCD-1234" {
		t.Fatalf("unexpected device event: %+v", device)
	}
	if authorized.Event != "authorized" || authorized.Account.Login != "octocat" {
		t.Fatalf("unexpected authorized event: %+v", authorized)
	}
	assertNoSecret(t, stdout)
}

func TestAuthStatusDefaultsToLocalMetadataOnly(t *testing.T) {
	auth := &fakeAuth{account: githubauth.AccountSummary{Host: "github.com", Profile: "default", Login: "octocat", Scopes: []string{"repo"}}}
	stdout, _, err := runAuthCLI(t, auth, "auth", "status", "--client-id", "Iv1.test")
	if err != nil {
		t.Fatal(err)
	}
	if auth.lastProbe {
		t.Fatal("status probed the network by default")
	}
	if !strings.Contains(stdout, "logged in") || !strings.Contains(stdout, "octocat") || !strings.Contains(stdout, "repo") {
		t.Fatalf("unexpected status output: %q", stdout)
	}

	_, _, err = runAuthCLI(t, auth, "auth", "status", "--client-id", "Iv1.test", "--probe")
	if err != nil {
		t.Fatal(err)
	}
	if !auth.lastProbe {
		t.Fatal("--probe did not reach the adapter")
	}
}

func TestAuthStatusReportsAbsentCredentialWithoutFailing(t *testing.T) {
	auth := &fakeAuth{statusErr: githubauth.ErrNotLoggedIn}
	stdout, _, err := runAuthCLI(t, auth, "auth", "status", "--client-id", "Iv1.test")
	if err != nil {
		t.Fatalf("absent credential should be a normal status: %v", err)
	}
	if !strings.Contains(stdout, "not logged in") {
		t.Fatalf("unexpected status output: %q", stdout)
	}

	stdout, _, err = runAuthCLI(t, auth, "auth", "status", "--client-id", "Iv1.test", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report statusReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid status json %q: %v", stdout, err)
	}
	if report.LoggedIn || report.State != "absent" {
		t.Fatalf("unexpected absent report: %+v", report)
	}
}

func TestAuthStatusJSONHasNoSecretFields(t *testing.T) {
	expires := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	auth := &fakeAuth{account: githubauth.AccountSummary{
		Host:           "github.com",
		Profile:        "default",
		Login:          "octocat",
		UserID:         7,
		Scopes:         []string{"repo"},
		TokenExpiresAt: expires,
		CanRefresh:     true,
	}}
	stdout, _, err := runAuthCLI(t, auth, "auth", "status", "--client-id", "Iv1.test", "--json")
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecret(t, stdout)
	var report statusReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid status json %q: %v", stdout, err)
	}
	if report.SchemaVersion != 1 || !report.LoggedIn || report.State != "logged_in" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Login != "octocat" || report.TokenExpiresAt != "2026-01-02T03:04:05Z" {
		t.Fatalf("unexpected account fields: %+v", report)
	}
}

func TestAuthLogoutRemovesOnlyLocalCredential(t *testing.T) {
	auth := &fakeAuth{}
	stdout, _, err := runAuthCLI(t, auth, "auth", "logout", "--client-id", "Iv1.test", "--profile", "work")
	if err != nil {
		t.Fatal(err)
	}
	if auth.lastLogout.Host != "github.com" || auth.lastLogout.Profile != "work" || auth.lastLogout.ClientID != "Iv1.test" {
		t.Fatalf("unexpected logout options: %+v", auth.lastLogout)
	}
	if !strings.Contains(stdout, "no remote revocation") {
		t.Fatalf("logout overclaims remote revocation: %q", stdout)
	}
}

func TestAuthLogoutIsIdempotentAndPropagatesStoreErrors(t *testing.T) {
	// A valid key with no stored item is success.
	auth := &fakeAuth{}
	if _, _, err := runAuthCLI(t, auth, "auth", "logout", "--client-id", "Iv1.test"); err != nil {
		t.Fatalf("idempotent logout failed: %v", err)
	}

	auth.logoutErr = errors.New("keystore locked")
	if _, _, err := runAuthCLI(t, auth, "auth", "logout", "--client-id", "Iv1.test"); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("store error was not propagated: %v", err)
	}
}

func TestAuthStatusAndLogoutRequireClientIDBeforeOpeningKeystore(t *testing.T) {
	for _, command := range []string{"status", "logout"} {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		a := newApp(stdout, stderr)
		a.getenv = func(string) string { return "" }
		a.newAuth = func(string) (authService, error) {
			t.Fatalf("auth %s opened the keystore without a client id", command)
			return nil, nil
		}
		err := executeApp(a, context.Background(), []string{"auth", command})
		if err == nil || !strings.Contains(err.Error(), "--client-id") {
			t.Fatalf("auth %s did not require a client id: %v", command, err)
		}
	}
}

func TestAuthRejectsTokenAppIDAndURLsBeforeOpeningKeystore(t *testing.T) {
	cases := [][]string{
		{"auth", "login", "--client-id", "Iv1.test", "--token", "ghp_x"},
		{"auth", "login", "--client-id", "123456"},
		{"auth", "login", "--client-id", "Ov23abc"},
		{"auth", "login", "--client-id", "Iv1.test", "--hostname", "https://evil.example"},
	}
	for _, args := range cases {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		a := newApp(stdout, stderr)
		a.getenv = func(string) string { return "" }
		a.newAuth = func(string) (authService, error) {
			t.Fatalf("opened the keystore for rejected input %v", args)
			return nil, nil
		}
		if err := executeApp(a, context.Background(), args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestAuthLoginWithoutClientIDFails(t *testing.T) {
	if githubClientID != "" {
		t.Skip("build embeds a client id")
	}
	auth := &fakeAuth{}
	_, _, err := runAuthCLI(t, auth, "auth", "login")
	if err == nil || !strings.Contains(err.Error(), "--client-id") {
		t.Fatalf("missing client id was accepted: %v", err)
	}
	if auth.loginCalls != 0 {
		t.Fatal("missing client id reached the adapter")
	}
}

func TestAuthClientIDComesFromEnvironmentWhenFlagIsAbsent(t *testing.T) {
	auth := &fakeAuth{account: githubauth.AccountSummary{Host: "github.com", Login: "octocat"}}
	_, _, err := runAuthCLIWith(t, auth, func(a *app) {
		a.getenv = func(key string) string {
			if key == "ZEITHROLD_GITHUB_CLIENT_ID" {
				return "Iv1.from-env"
			}
			return ""
		}
	}, "auth", "login")
	if err != nil {
		t.Fatal(err)
	}
	if auth.lastOptions.ClientID != "Iv1.from-env" {
		t.Fatalf("environment client id not used: %+v", auth.lastOptions)
	}
}

func TestAuthLoginOnlyOpensTrustedBrowserAndHonorsNoBrowser(t *testing.T) {
	auth := &fakeAuth{
		account: githubauth.AccountSummary{Host: "github.com", Login: "octocat"},
		prompt:  githubauth.DevicePrompt{UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device"},
	}
	var opened []string
	configure := func(a *app) {
		a.isTTY = func() bool { return true }
		a.openBrowser = func(url string) error {
			opened = append(opened, url)
			return nil
		}
	}
	if _, _, err := runAuthCLIWith(t, auth, configure, "auth", "login", "--client-id", "Iv1.test"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "https://github.com/login/device" {
		t.Fatalf("trusted verification page was not opened: %v", opened)
	}

	opened = nil
	if _, _, err := runAuthCLIWith(t, auth, configure, "auth", "login", "--client-id", "Iv1.test", "--no-browser"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 0 {
		t.Fatalf("--no-browser still opened %v", opened)
	}
}

func TestAuthLoginDoesNotOpenAnUntrustedVerificationPage(t *testing.T) {
	auth := &fakeAuth{
		account: githubauth.AccountSummary{Host: "github.com", Login: "octocat"},
		prompt:  githubauth.DevicePrompt{UserCode: "ABCD-1234", VerificationURI: "https://evil.example/login/device"},
	}
	var opened []string
	_, stderr, err := runAuthCLIWith(t, auth, func(a *app) {
		a.isTTY = func() bool { return true }
		a.openBrowser = func(url string) error {
			opened = append(opened, url)
			return nil
		}
	}, "auth", "login", "--client-id", "Iv1.test")
	if err != nil {
		t.Fatal(err)
	}
	if len(opened) != 0 {
		t.Fatalf("opened an untrusted URL: %v", opened)
	}
	if !strings.Contains(stderr, "ABCD-1234") {
		t.Fatalf("device guidance missing: %q", stderr)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("stdin must not be read") }

func TestAuthNoInputNeverReadsStdin(t *testing.T) {
	auth := &fakeAuth{
		account: githubauth.AccountSummary{Host: "github.com", Login: "octocat"},
		prompt:  githubauth.DevicePrompt{UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device"},
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.stdin = failingReader{}
	a.getenv = func(string) string { return "" }
	a.newAuth = func(string) (authService, error) { return auth, nil }
	if err := executeApp(a, context.Background(), []string{"auth", "login", "--client-id", "Iv1.test", "--no-input"}); err != nil {
		t.Fatal(err)
	}
}

func TestAuthUnavailableFailsClosed(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.newAuth = nil
	err := executeApp(a, context.Background(), []string{"auth", "status", "--client-id", "Iv1.test"})
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("nil factory did not fail closed: %v", err)
	}
}

func TestAuthStoreUnavailableIsReported(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.newAuth = func(string) (authService, error) {
		return nil, errors.New("keystore backend unavailable")
	}
	err := executeApp(a, context.Background(), []string{"auth", "status", "--client-id", "Iv1.test"})
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("store failure was not reported: %v", err)
	}
}

func TestAuthLoginHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.getenv = func(string) string { return "" }
	a.newAuth = func(string) (authService, error) { return &fakeAuth{}, nil }
	err := executeApp(a, ctx, []string{"auth", "login", "--client-id", "Iv1.test"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was not propagated: %v", err)
	}
}

// assertNoSecret rejects token-like material in machine output while allowing
// non-secret timestamps such as tokenExpiresAt.
func assertNoSecret(t *testing.T, output string) {
	t.Helper()
	for _, marker := range []string{"accessToken", "refreshToken", "ghp_", "gho_", "ghs_", "github_pat_"} {
		if strings.Contains(output, marker) {
			t.Fatalf("output contains secret marker %q: %s", marker, output)
		}
	}
}

func TestAuthLoginHintsReplaceOnAccountConflict(t *testing.T) {
	auth := &fakeAuth{loginErr: githubauth.ErrAccountConflict}
	_, _, err := runAuthCLI(t, auth, "auth", "login", "--client-id", "Iv1.test")
	if !errors.Is(err, githubauth.ErrAccountConflict) {
		t.Fatalf("conflict error identity was lost: %v", err)
	}
	if !strings.Contains(err.Error(), "--replace") {
		t.Fatalf("conflict error does not explain --replace: %v", err)
	}
}

func TestAuthLoginMapsOfflineAndReplace(t *testing.T) {
	auth := &fakeAuth{
		account: githubauth.AccountSummary{Host: "github.com", Login: "octocat"},
		prompt:  githubauth.DevicePrompt{UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device"},
	}
	if _, _, err := runAuthCLI(t, auth, "auth", "login", "--client-id", "Iv1.test", "--offline-access", "--replace"); err != nil {
		t.Fatal(err)
	}
	if !auth.lastOptions.RequestOfflineAccess || !auth.lastOptions.ReplaceExistingAccount {
		t.Fatalf("offline/replace flags were not mapped: %+v", auth.lastOptions)
	}
}

func TestAuthLogoutJSON(t *testing.T) {
	auth := &fakeAuth{}
	stdout, _, err := runAuthCLI(t, auth, "auth", "logout", "--client-id", "Iv1.test", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report logoutReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid logout json %q: %v", stdout, err)
	}
	if !report.LoggedOut || report.Host != "github.com" || report.Profile != "default" {
		t.Fatalf("unexpected logout report: %+v", report)
	}
}

func TestAuthNoInputSuppressesBrowser(t *testing.T) {
	auth := &fakeAuth{
		account: githubauth.AccountSummary{Host: "github.com", Login: "octocat"},
		prompt:  githubauth.DevicePrompt{UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device"},
	}
	var opened []string
	_, _, err := runAuthCLIWith(t, auth, func(a *app) {
		a.isTTY = func() bool { return true }
		a.openBrowser = func(url string) error {
			opened = append(opened, url)
			return nil
		}
	}, "auth", "login", "--client-id", "Iv1.test", "--no-input")
	if err != nil {
		t.Fatal(err)
	}
	if len(opened) != 0 {
		t.Fatalf("--no-input opened a browser: %v", opened)
	}
}

func TestAuthSurfacesConcurrentRefreshAsRetryable(t *testing.T) {
	auth := &fakeAuth{statusErr: githubauth.ErrConcurrentRefresh}
	_, _, err := runAuthCLI(t, auth, "auth", "status", "--client-id", "Iv1.test", "--probe")
	if !errors.Is(err, githubauth.ErrConcurrentRefresh) {
		t.Fatalf("concurrent refresh identity was lost: %v", err)
	}
	if !strings.Contains(err.Error(), "retry") {
		t.Fatalf("concurrent refresh is not retryable: %v", err)
	}
}

func TestAuthSurfacesLockedStoreAsActionable(t *testing.T) {
	auth := &fakeAuth{loginErr: githubauth.ErrLocked}
	_, _, err := runAuthCLI(t, auth, "auth", "login", "--client-id", "Iv1.test")
	if !errors.Is(err, githubauth.ErrLocked) {
		t.Fatalf("locked identity was lost: %v", err)
	}
	if !strings.Contains(err.Error(), "unlock") || !strings.Contains(err.Error(), "retry") {
		t.Fatalf("locked store message is not actionable: %v", err)
	}
}

func assertNoTokenEcho(t *testing.T, text, secret string) {
	t.Helper()
	if strings.Contains(text, secret) {
		t.Fatalf("output echoed token-like input %q: %q", secret, text)
	}
	for _, prefix := range []string{"ghu_", "ghr_", "ghp_", "gho_", "ghs_", "github_pat_"} {
		if strings.Contains(text, prefix) {
			t.Fatalf("output echoed token prefix %q: %q", prefix, text)
		}
	}
}

func TestAuthClientIDErrorsNeverEchoTokenLikeInput(t *testing.T) {
	secrets := []string{"ghu_abc123", "ghr_abc123", "ghp_abc123", "gho_abc123", "ghs_abc123", "github_pat_abc123", "123456789"}
	for _, secret := range secrets {
		for _, command := range []string{"login", "status", "logout"} {
			stdout, stderr, err := runAuthCLI(t, &fakeAuth{}, "auth", command, "--client-id", secret)
			if err == nil {
				t.Fatalf("auth %s accepted token-like client id %q", command, secret)
			}
			assertNoTokenEcho(t, err.Error(), secret)
			assertNoTokenEcho(t, stdout, secret)
			assertNoTokenEcho(t, stderr, secret)

			// A token pasted into --hostname must not be echoed either.
			_, _, err = runAuthCLI(t, &fakeAuth{}, "auth", command, "--client-id", "Iv1.test", "--hostname", secret)
			if err == nil {
				t.Fatalf("auth %s accepted token-like hostname %q", command, secret)
			}
			assertNoTokenEcho(t, err.Error(), secret)
		}
	}
}

func TestAuthEnvClientIDTokenLikeInputIsNotEchoed(t *testing.T) {
	secret := "ghp_envsecret"
	stdout, stderr, err := runAuthCLIWith(t, &fakeAuth{}, func(a *app) {
		a.getenv = func(key string) string {
			if key == "ZEITHROLD_GITHUB_CLIENT_ID" {
				return secret
			}
			return ""
		}
	}, "auth", "login")
	if err == nil {
		t.Fatal("environment token-like client id was accepted")
	}
	assertNoTokenEcho(t, err.Error(), secret)
	assertNoTokenEcho(t, stdout, secret)
	assertNoTokenEcho(t, stderr, secret)
}

func TestAuthJSONErrorDoesNotLeakTokenLikeClientID(t *testing.T) {
	secret := "ghp_jsonsecret"
	stdout, stderr, err := runAuthCLI(t, &fakeAuth{}, "auth", "login", "--client-id", secret, "--json")
	if err == nil {
		t.Fatal("JSON login accepted token-like client id")
	}
	assertNoTokenEcho(t, err.Error(), secret)
	assertNoTokenEcho(t, stdout, secret)
	assertNoTokenEcho(t, stderr, secret)
}

func TestIsTerminalRejectsNonTerminalCharDevices(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Fatal("bytes.Buffer treated as a terminal")
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		t.Skipf("cannot open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()
	if isTerminal(devNull) {
		t.Fatalf("%s treated as a terminal", os.DevNull)
	}
}

func TestAuthLoginDoesNotOpenBrowserWhenStreamsAreDevNull(t *testing.T) {
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		t.Skipf("cannot open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	auth := &fakeAuth{
		account: githubauth.AccountSummary{Host: "github.com", Login: "octocat"},
		prompt:  githubauth.DevicePrompt{UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device"},
	}
	a := newApp(devNull, devNull)
	a.getenv = func(string) string { return "" }
	a.newAuth = func(string) (authService, error) { return auth, nil }
	opened := false
	a.openBrowser = func(string) error {
		opened = true
		return nil
	}
	if err := executeApp(a, context.Background(), []string{"auth", "login", "--client-id", "Iv1.test"}); err != nil {
		t.Fatal(err)
	}
	if opened {
		t.Fatal("login opened a browser with /dev/null streams")
	}
}

func TestAuthLoginRequestsOfflineAccessByDefault(t *testing.T) {
	auth := &fakeAuth{
		account: githubauth.AccountSummary{Host: "github.com", Login: "octocat"},
		prompt:  githubauth.DevicePrompt{UserCode: "ABCD-1234", VerificationURI: "https://github.com/login/device"},
	}
	if _, _, err := runAuthCLI(t, auth, "auth", "login", "--client-id", "Iv1.test"); err != nil {
		t.Fatal(err)
	}
	if !auth.lastOptions.RequestOfflineAccess {
		t.Fatal("default login did not request offline_access")
	}

	if _, _, err := runAuthCLI(t, auth, "auth", "login", "--client-id", "Iv1.test", "--offline-access=false"); err != nil {
		t.Fatal(err)
	}
	if auth.lastOptions.RequestOfflineAccess {
		t.Fatal("--offline-access=false still requested offline_access")
	}
}

func TestAuthPositionalAndUnknownInputNeverEchoesTokenLikeValues(t *testing.T) {
	cases := []struct {
		args   []string
		secret string
	}{
		{[]string{"auth", "login", "--client-id", "Iv1.test", "ghp_pos_secret"}, "ghp_pos_secret"},
		{[]string{"auth", "login", "--client-id", "Iv1.test", "--json", "ghp_json_pos"}, "ghp_json_pos"},
		{[]string{"auth", "status", "ghp_status_pos"}, "ghp_status_pos"},
		{[]string{"auth", "logout", "ghp_logout_pos"}, "ghp_logout_pos"},
		{[]string{"auth", "status", "--json", "ghr_refresh_pos"}, "ghr_refresh_pos"},
		{[]string{"auth", "ghu_unknown_secret"}, "ghu_unknown_secret"},
		{[]string{"auth", "login", "--ghu_flag_secret"}, "ghu_flag_secret"},
		{[]string{"auth", "login", "--ghu_flag_secret=value"}, "ghu_flag_secret"},
		{[]string{"auth", "--ghp_group_flag_secret"}, "ghp_group_flag_secret"},
		{[]string{"--ghp_root_flag_secret", "auth", "login"}, "ghp_root_flag_secret"},
	}
	for _, c := range cases {
		stdout, stderr, err := runAuthCLI(t, &fakeAuth{}, c.args...)
		if err == nil {
			t.Fatalf("accepted %v", c.args)
		}
		assertNoTokenEcho(t, err.Error(), c.secret)
		assertNoTokenEcho(t, stdout, c.secret)
		assertNoTokenEcho(t, stderr, c.secret)
	}
}
