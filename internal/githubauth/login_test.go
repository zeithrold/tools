package githubauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zeithrold/tools/internal/keystore"
)

func TestLoginDeviceFlowSuccess(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	h.userScopes = "repo, offline_access"
	h.tokens = []tokenResponse{
		{Error: "authorization_pending"},
		{Error: "slow_down", Interval: 8},
		{AccessToken: "at-1", TokenType: "bearer", Scope: "repo, offline_access", RefreshToken: "rt-1", ExpiresIn: 3600, RefreshTokenExpiresIn: 7200},
	}
	options := h.options()
	options.RequestOfflineAccess = true

	var prompt DevicePrompt
	summary, err := h.login(context.Background(), store, clock, options, &prompt)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if prompt.UserCode != "ABCD-1234" || prompt.VerificationURI == "" || prompt.VerificationURIComplete == "" {
		t.Fatalf("prompt = %+v", prompt)
	}
	if prompt.ExpiresIn != 15*time.Minute || prompt.Interval != 5*time.Second {
		t.Fatalf("prompt durations = %v/%v", prompt.ExpiresIn, prompt.Interval)
	}
	assertNoSecret(t, formatPrompt(prompt), "at-1", "rt-1", "device-code-1")

	if summary.Login != "octocat" || summary.UserID != 12345 || summary.Host != options.Host || summary.Profile != "tools" {
		t.Fatalf("summary identity = %+v", summary)
	}
	if !summary.Probed || !summary.CanRefresh || summary.InstallationID != 77 || summary.TargetRepository != "owner/repo" {
		t.Fatalf("summary state = %+v", summary)
	}
	if summary.TokenExpiresAt.IsZero() || summary.RefreshExpiresAt.IsZero() {
		t.Fatalf("summary expiry = %+v", summary)
	}
	if summary.VerifiedAt.IsZero() {
		t.Fatal("summary VerifiedAt is zero")
	}
	encoded, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}
	assertNoSecret(t, string(encoded), "at-1", "rt-1", "device-code-1")

	record, _ := store.onlyRecord(t)
	if record.AccessToken != "at-1" || record.RefreshToken != "rt-1" {
		t.Fatalf("stored tokens = %q/%q", record.AccessToken, record.RefreshToken)
	}
	if len(clock.slept()) != 3 || clock.slept()[0] != 5*time.Second || clock.slept()[1] != 5*time.Second || clock.slept()[2] != 10*time.Second {
		t.Fatalf("poll sleeps = %v, want [5s 5s 10s]", clock.slept())
	}

	deviceRequests := h.requestsFor("/login/device/code")
	if len(deviceRequests) != 1 {
		t.Fatalf("device requests = %d", len(deviceRequests))
	}
	assertNoClientSecret(t, deviceRequests[0].Form)
	if deviceRequests[0].Form.Get("client_id") != "Iv1.test-client" || deviceRequests[0].Form.Get("scope") != "offline_access" {
		t.Fatalf("device form = %v", deviceRequests[0].Form)
	}

	tokenRequests := h.requestsFor("/login/oauth/access_token")
	if len(tokenRequests) != 3 {
		t.Fatalf("token requests = %d", len(tokenRequests))
	}
	for _, request := range tokenRequests {
		assertNoClientSecret(t, request.Form)
		if request.Form.Get("client_id") != "Iv1.test-client" || request.Form.Get("device_code") != "device-code-1" {
			t.Fatalf("token form = %v", request.Form)
		}
		if request.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
			t.Fatalf("grant type = %q", request.Form.Get("grant_type"))
		}
	}

	userRequests := h.requestsFor("/user")
	if len(userRequests) != 1 || userRequests[0].Header.Get("Authorization") != "Bearer at-1" {
		t.Fatalf("user requests = %+v", userRequests)
	}
	if len(h.requestsFor("/user/installations")) != 1 || len(h.requestsFor("/user/installations/77/repositories")) != 1 {
		t.Fatalf("installation requests missing: %d/%d", len(h.requestsFor("/user/installations")), len(h.requestsFor("/user/installations/77/repositories")))
	}
}

func formatPrompt(prompt DevicePrompt) string {
	return prompt.UserCode + "|" + prompt.VerificationURI + "|" + prompt.VerificationURIComplete
}

func TestDeviceFlowErrors(t *testing.T) {
	tests := []struct {
		name   string
		tokens []tokenResponse
		want   error
		sleeps int
	}{
		{name: "expired", tokens: []tokenResponse{{Error: "authorization_pending"}, {Error: "expired_token"}}, want: ErrDeviceCodeExpired, sleeps: 2},
		{name: "denied", tokens: []tokenResponse{{Error: "access_denied"}}, want: ErrAccessDenied, sleeps: 1},
		{name: "client", tokens: []tokenResponse{{Error: "incorrect_client_credentials"}}, want: ErrAuthorizationRejected, sleeps: 1},
		{name: "grant", tokens: []tokenResponse{{Error: "unsupported_grant_type"}}, want: ErrAuthorizationRejected, sleeps: 1},
		{name: "device", tokens: []tokenResponse{{Error: "incorrect_device_code"}}, want: ErrAuthorizationRejected, sleeps: 1},
		{name: "disabled", tokens: []tokenResponse{{Error: "device_flow_disabled"}}, want: ErrAuthorizationRejected, sleeps: 1},
		{name: "unknown", tokens: []tokenResponse{{Error: "surprise_error"}}, want: ErrProtocol, sleeps: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHarness(t)
			store := newFakeStore()
			clock := newFakeClock()
			h.tokens = test.tokens
			_, err := h.login(context.Background(), store, clock, h.options(), nil)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if len(clock.slept()) != test.sleeps {
				t.Fatalf("sleeps = %v, want %d", clock.slept(), test.sleeps)
			}
			if store.count() != 0 {
				t.Fatalf("store contains %d records", store.count())
			}
			assertNoSecret(t, errorText(err), "at-1", "rt-1", "device-code-1")
		})
	}
}

func TestSlowDownUsesServerInterval(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	h.tokens = []tokenResponse{{Error: "slow_down", Interval: 20}, {AccessToken: "at", ExpiresIn: 3600}}
	if _, err := h.login(context.Background(), store, clock, h.options(), nil); err != nil {
		t.Fatalf("Login: %v", err)
	}
	sleeps := clock.slept()
	if len(sleeps) != 2 || sleeps[0] != 5*time.Second || sleeps[1] != 20*time.Second {
		t.Fatalf("sleeps = %v, want [5s 20s]", sleeps)
	}
}

func TestDeviceDeadlineStartsAtDeviceResponseNotCallback(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	h.device.ExpiresIn = 10
	client := h.client(store, clock)
	_, err := client.Login(context.Background(), h.options(), func(DevicePrompt) error {
		clock.advance(11 * time.Second)
		return nil
	})
	if !errors.Is(err, ErrDeviceCodeExpired) {
		t.Fatalf("error = %v, want ErrDeviceCodeExpired after callback exceeded expiry", err)
	}
	if len(h.requestsFor("/login/oauth/access_token")) != 0 {
		t.Fatalf("polled %d times after device code expired during callback", len(h.requestsFor("/login/oauth/access_token")))
	}
	if store.count() != 0 {
		t.Fatal("credential stored after expired device code")
	}
}

func TestDeviceCodeExpiresWhilePolling(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	h.device.ExpiresIn = 10
	h.tokens = []tokenResponse{{Error: "authorization_pending"}, {Error: "authorization_pending"}, {AccessToken: "at", ExpiresIn: 3600}}
	_, err := h.login(context.Background(), store, clock, h.options(), nil)
	if !errors.Is(err, ErrDeviceCodeExpired) {
		t.Fatalf("error = %v, want ErrDeviceCodeExpired", err)
	}
	if len(h.requestsFor("/login/oauth/access_token")) != 1 {
		t.Fatalf("token requests after expiry = %d, want 1", len(h.requestsFor("/login/oauth/access_token")))
	}
}

func TestDeviceCodeHTTPAndMalformedResponses(t *testing.T) {
	t.Run("safe actionable errors", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			status int
			code   string
			want   string
			kind   error
		}{
			{"disabled http", 400, "device_flow_disabled", "enable Device Flow in the GitHub App settings", ErrAuthorizationRejected},
			{"disabled success body", 200, "device_flow_disabled", "enable Device Flow in the GitHub App settings", ErrAuthorizationRejected},
			{"invalid client", 401, "incorrect_client_credentials", "check the public GitHub App Client ID", ErrAuthorizationRejected},
			{"server error", 500, "server_error", "HTTP 500 (server_error)", ErrRemote},
			{"rate limit", 429, "temporarily_unavailable", "HTTP 429 (temporarily_unavailable)", ErrRemote},
			{"other status", 403, "access_denied", "HTTP 403 (access_denied)", ErrProtocol},
			{"unknown http error", 400, "ghu_UNTRUSTED_ERROR", "HTTP 400", ErrAuthorizationRejected},
			{"unknown success body", 200, "ghu_UNTRUSTED_ERROR", "unrecognized error", ErrAuthorizationRejected},
		} {
			t.Run(tc.name, func(t *testing.T) {
				h := newHarness(t)
				store := newFakeStore()
				h.override = func(w http.ResponseWriter, r *http.Request) bool {
					if r.URL.Path != "/login/device/code" {
						return false
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.status)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error":             tc.code,
						"error_description": "ghu_PRIVATE_DESCRIPTION",
						"error_uri":         "https://evil.example/ghu_PRIVATE_URI",
						"access_token":      "ghu_PRIVATE_ACCESS",
						"refresh_token":     "ghr_PRIVATE_REFRESH",
						"device_code":       "ghu_PRIVATE_DEVICE",
					})
					return true
				}
				_, err := h.login(context.Background(), store, newFakeClock(), h.options(), nil)
				if !errors.Is(err, tc.kind) || !strings.Contains(errorText(err), tc.want) {
					t.Fatalf("error = %v, want %v containing %q", err, tc.kind, tc.want)
				}
				if sanitizeOAuthCode(tc.code) != "" && !strings.Contains(errorText(err), tc.code) {
					t.Fatalf("error omits safe OAuth code %q: %v", tc.code, err)
				}
				assertNoSecret(t, errorText(err), "ghu_UNTRUSTED_ERROR", "ghu_PRIVATE_DESCRIPTION",
					"ghu_PRIVATE_URI", "ghu_PRIVATE_ACCESS", "ghr_PRIVATE_REFRESH", "ghu_PRIVATE_DEVICE")
				if len(h.requestsFor("/login/oauth/access_token")) != 0 || store.count() != 0 {
					t.Fatal("device rejection polled for or stored credentials")
				}
			})
		}
	})

	t.Run("http error", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		clock := newFakeClock()
		h.deviceStatus = http.StatusInternalServerError
		_, err := h.login(context.Background(), store, clock, h.options(), nil)
		if !errors.Is(err, ErrRemote) {
			t.Fatalf("error = %v, want ErrRemote", err)
		}
		assertNoSecret(t, errorText(err), "ghu_FAIL_BODY", "ghr_FAIL_BODY")
		if store.count() != 0 {
			t.Fatal("credential stored after device error")
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		h := newHarness(t)
		h.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.URL.Path == "/login/device/code" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"device_code":"ghu_FAIL_BODY",`))
				return true
			}
			return false
		}
		_, err := h.login(context.Background(), newFakeStore(), newFakeClock(), h.options(), nil)
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		assertNoSecret(t, errorText(err), "ghu_FAIL_BODY")
	})

	t.Run("missing fields", func(t *testing.T) {
		h := newHarness(t)
		h.device.DeviceCode = ""
		_, err := h.login(context.Background(), newFakeStore(), newFakeClock(), h.options(), nil)
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
	})

	t.Run("unsafe verification uri", func(t *testing.T) {
		h := newHarness(t)
		h.device.VerificationURI = "https://evil.example/login/device"
		_, err := h.login(context.Background(), newFakeStore(), newFakeClock(), h.options(), nil)
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		if len(h.requestsFor("/login/oauth/access_token")) != 0 {
			t.Fatal("polled after rejecting verification URI")
		}
	})

	t.Run("cross host redirect", func(t *testing.T) {
		h := newHarness(t)
		h.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.URL.Path == "/login/device/code" {
				http.Redirect(w, r, "http://127.0.0.1:1/login/device/code?access_token=ghu_REDIRECT_SECRET", http.StatusFound)
				return true
			}
			return false
		}
		_, err := h.login(context.Background(), newFakeStore(), newFakeClock(), h.options(), nil)
		if err == nil {
			t.Fatal("expected redirect rejection")
		}
		assertNoSecret(t, errorText(err), "ghu_REDIRECT_SECRET")
		if len(h.requestsFor("/login/oauth/access_token")) != 0 {
			t.Fatal("polled after redirect rejection")
		}
	})
}

func TestUnknownOAuthCodesAreNotEchoed(t *testing.T) {
	const secret = "ghr_review_fake_secret"

	t.Run("device endpoint error field", func(t *testing.T) {
		h := newHarness(t)
		h.device = deviceCodeResponse{Error: secret}
		_, err := h.login(context.Background(), newFakeStore(), newFakeClock(), h.options(), nil)
		if !errors.Is(err, ErrAuthorizationRejected) {
			t.Fatalf("error = %v, want ErrAuthorizationRejected", err)
		}
		assertNoSecret(t, errorText(err), secret)
	})

	t.Run("token endpoint error field", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		h.tokens = []tokenResponse{{Error: secret}}
		_, err := h.login(context.Background(), store, newFakeClock(), h.options(), nil)
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		assertNoSecret(t, errorText(err), secret)
		if store.count() != 0 {
			t.Fatal("credential stored after unknown token error")
		}
	})
}

func TestClientIDAndHostRejectSecretsWithoutEcho(t *testing.T) {
	secrets := []string{
		"ghu_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"ghr_BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
		"ghp_CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
		"gho_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD",
		"ghs_EEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE",
		"github_pat_FFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
	}
	for _, secret := range secrets {
		client := &Client{Store: newFakeStore()}
		summary, err := client.Login(context.Background(), Options{ClientID: secret}, nil)
		if !errors.Is(err, ErrInvalidOptions) {
			t.Errorf("Login with token-like client ID error = %v, want ErrInvalidOptions", err)
		}
		if summary.Login != "" || summary.Host != "" || summary.Profile != "" || summary.UserID != 0 || summary.CanRefresh {
			t.Errorf("Login returned non-empty summary on option error: %+v", summary)
		}
		assertNoSecret(t, errorText(err), secret)

		_, err = client.resolve(Options{ClientID: "Iv1.test", Host: secret})
		if !errors.Is(err, ErrInvalidOptions) {
			t.Errorf("resolve with token-like host error = %v, want ErrInvalidOptions", err)
		}
		assertNoSecret(t, errorText(err), secret)

		_, err = client.resolve(Options{ClientID: "Iv1.test", TargetRepository: secret})
		if !errors.Is(err, ErrInvalidOptions) {
			t.Errorf("resolve with token-like repository error = %v, want ErrInvalidOptions", err)
		}
		assertNoSecret(t, errorText(err), secret)

		_, err = client.resolve(Options{ClientID: "Iv1.test", Profile: secret})
		if !errors.Is(err, ErrInvalidOptions) {
			t.Errorf("resolve with token-like profile error = %v, want ErrInvalidOptions", err)
		}
		assertNoSecret(t, errorText(err), secret)
	}

	trusted := &Client{Store: newFakeStore(), AllowedHosts: []string{"ghp_SECRET_ALLOWED_HOST"}}
	_, err := trusted.resolve(Options{ClientID: "Iv1.test", Host: "github.com"})
	if !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("invalid trusted host error = %v, want ErrInvalidOptions", err)
	}
	assertNoSecret(t, errorText(err), "ghp_SECRET_ALLOWED_HOST")
}

func TestFailureBodiesDoNotLeakTokens(t *testing.T) {
	t.Run("token endpoint non-2xx", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		h.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.URL.Path == "/login/oauth/access_token" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"server_error","error_description":"ghr_FAIL_BODY","access_token":"ghu_FAIL_BODY"}`))
				return true
			}
			return false
		}
		_, err := h.login(context.Background(), store, newFakeClock(), h.options(), nil)
		if !errors.Is(err, ErrRemote) {
			t.Fatalf("error = %v, want ErrRemote", err)
		}
		assertNoSecret(t, errorText(err), "ghu_FAIL_BODY", "ghr_FAIL_BODY")
		if store.count() != 0 {
			t.Fatal("credential stored after token endpoint failure")
		}
	})

	t.Run("token endpoint malformed body", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		h.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.URL.Path == "/login/oauth/access_token" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"access_token":"ghu_FAIL_BODY",`))
				return true
			}
			return false
		}
		_, err := h.login(context.Background(), store, newFakeClock(), h.options(), nil)
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("error = %v, want ErrProtocol", err)
		}
		assertNoSecret(t, errorText(err), "ghu_FAIL_BODY")
		if store.count() != 0 {
			t.Fatal("credential stored after malformed token response")
		}
	})

	t.Run("user endpoint non-2xx", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		h.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.URL.Path == "/user" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error_description":"ghr_FAIL_BODY","access_token":"ghu_FAIL_BODY"}`))
				return true
			}
			return false
		}
		_, err := h.login(context.Background(), store, newFakeClock(), h.options(), nil)
		if !errors.Is(err, ErrRemote) {
			t.Fatalf("error = %v, want ErrRemote", err)
		}
		assertNoSecret(t, errorText(err), "ghu_FAIL_BODY", "ghr_FAIL_BODY")
		if store.count() != 0 {
			t.Fatal("credential stored after user endpoint failure")
		}
	})
}

func TestStoreUnavailableBeforeAuthorization(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	store.availableErr = keystore.ErrLocked
	_, err := h.login(context.Background(), store, newFakeClock(), h.options(), nil)
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("error = %v, want ErrLocked", err)
	}
	if h.requestCount() != 0 {
		t.Fatalf("network requests = %d, want 0", h.requestCount())
	}
	if store.count() != 0 {
		t.Fatal("credential stored while unavailable")
	}
}

func TestStoreWriteFailureDoesNotReturnToken(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	store.setErr = keystore.ErrLocked
	h.tokens = []tokenResponse{{AccessToken: "secret-access", RefreshToken: "secret-refresh", ExpiresIn: 3600}}
	_, err := h.login(context.Background(), store, newFakeClock(), h.options(), nil)
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("error = %v, want ErrLocked", err)
	}
	if store.count() != 0 {
		t.Fatal("credential stored despite write failure")
	}
	assertNoSecret(t, errorText(err), "secret-access", "secret-refresh")
}

func TestLoginRejectsSameLoginDifferentUserID(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	options := h.options()
	h.tokens = []tokenResponse{{AccessToken: "a1", RefreshToken: "r1", ExpiresIn: 3600}}
	if _, err := h.login(context.Background(), store, clock, options, nil); err != nil {
		t.Fatalf("first login: %v", err)
	}
	h.user = userResponse{Login: "octocat", ID: 99999}
	h.tokens = []tokenResponse{{AccessToken: "a2", RefreshToken: "r2", ExpiresIn: 3600}}
	_, err := h.login(context.Background(), store, clock, options, nil)
	if !errors.Is(err, ErrAccountConflict) {
		t.Fatalf("error = %v, want ErrAccountConflict for same login/different UID", err)
	}
	record, _ := store.onlyRecord(t)
	if record.UserID != 12345 || record.AccessToken != "a1" {
		t.Fatalf("record replaced despite different UID: %+v", record)
	}
}

func TestLoginAllowsLoginRenameForSameUserID(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	options := h.options()
	h.tokens = []tokenResponse{{AccessToken: "a1", RefreshToken: "r1", ExpiresIn: 3600}}
	if _, err := h.login(context.Background(), store, clock, options, nil); err != nil {
		t.Fatalf("first login: %v", err)
	}
	h.user = userResponse{Login: "renamed-user", ID: 12345}
	h.tokens = []tokenResponse{{AccessToken: "a2", RefreshToken: "r2", ExpiresIn: 3600}}
	summary, err := h.login(context.Background(), store, clock, options, nil)
	if err != nil {
		t.Fatalf("rename login: %v", err)
	}
	if summary.Login != "renamed-user" || summary.UserID != 12345 {
		t.Fatalf("summary after rename = %+v", summary)
	}
	record, _ := store.onlyRecord(t)
	if record.Login != "renamed-user" || record.UserID != 12345 {
		t.Fatalf("stored rename = %+v", record)
	}
}

func TestLoginRejectsSilentAccountReplacement(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	options := h.options()
	h.tokens = []tokenResponse{{AccessToken: "a1", RefreshToken: "r1", ExpiresIn: 3600}}
	if _, err := h.login(context.Background(), store, clock, options, nil); err != nil {
		t.Fatalf("first login: %v", err)
	}

	h.user = userResponse{Login: "other-user", ID: 999}
	h.tokens = []tokenResponse{{AccessToken: "a2", RefreshToken: "r2", ExpiresIn: 3600}}
	_, err := h.login(context.Background(), store, clock, options, nil)
	if !errors.Is(err, ErrAccountConflict) {
		t.Fatalf("error = %v, want ErrAccountConflict", err)
	}
	record, _ := store.onlyRecord(t)
	if record.Login != "octocat" || record.AccessToken != "a1" {
		t.Fatalf("record replaced without confirmation: %+v", record)
	}
	assertNoSecret(t, errorText(err), "a2", "r2")

	options.ReplaceExistingAccount = true
	h.tokens = []tokenResponse{{AccessToken: "a3", RefreshToken: "r3", ExpiresIn: 3600}}
	summary, err := h.login(context.Background(), store, clock, options, nil)
	if err != nil {
		t.Fatalf("replace login: %v", err)
	}
	if summary.Login != "other-user" {
		t.Fatalf("summary login = %q", summary.Login)
	}
	record, _ = store.onlyRecord(t)
	if record.Login != "other-user" || record.AccessToken != "a3" || record.RefreshToken != "r3" {
		t.Fatalf("replacement record = %+v", record)
	}
}

func TestCredentialSizeLimitKeepsPairOrFails(t *testing.T) {
	t.Run("long tokens fit", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		access := strings.Repeat("a", 1000)
		refresh := strings.Repeat("r", 1000)
		h.tokens = []tokenResponse{{AccessToken: access, RefreshToken: refresh, ExpiresIn: 3600, RefreshTokenExpiresIn: 7200}}
		if _, err := h.login(context.Background(), store, newFakeClock(), h.options(), nil); err != nil {
			t.Fatalf("Login with long tokens: %v", err)
		}
		record, _ := store.onlyRecord(t)
		if record.AccessToken != access || record.RefreshToken != refresh {
			t.Fatal("long token pair was not stored together")
		}
	})

	t.Run("oversized pair fails closed", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		access := strings.Repeat("a", 1400)
		refresh := strings.Repeat("r", 1400)
		h.tokens = []tokenResponse{{AccessToken: access, RefreshToken: refresh, ExpiresIn: 3600, RefreshTokenExpiresIn: 7200}}
		_, err := h.login(context.Background(), store, newFakeClock(), h.options(), nil)
		if !errors.Is(err, ErrCredentialTooLarge) {
			t.Fatalf("error = %v, want ErrCredentialTooLarge", err)
		}
		if store.count() != 0 {
			t.Fatal("oversized credential was partially stored")
		}
		assertNoSecret(t, errorText(err), access, refresh)
		if !strings.Contains(errorText(err), "no file fallback") {
			t.Fatalf("error is not actionable: %v", err)
		}
	})
}

func TestLoginCancellation(t *testing.T) {
	t.Run("before request", func(t *testing.T) {
		h := newHarness(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := h.login(ctx, newFakeStore(), newFakeClock(), h.options(), nil)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		if h.requestCount() != 0 {
			t.Fatalf("network requests = %d", h.requestCount())
		}
	})

	t.Run("during poll", func(t *testing.T) {
		h := newHarness(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		clock := newFakeClock()
		clock.onSleep = func(context.Context, time.Duration) error {
			cancel()
			return context.Canceled
		}
		_, err := h.login(ctx, newFakeStore(), clock, h.options(), nil)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		if len(h.requestsFor("/login/oauth/access_token")) != 0 {
			t.Fatal("polled after cancellation")
		}
	})

	t.Run("callback error", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		client := h.client(store, newFakeClock())
		sentinel := errors.New("user cancelled")
		_, err := client.Login(context.Background(), h.options(), func(DevicePrompt) error { return sentinel })
		if !errors.Is(err, sentinel) {
			t.Fatalf("error = %v, want sentinel", err)
		}
		if len(h.requestsFor("/login/oauth/access_token")) != 0 {
			t.Fatal("polled after callback error")
		}
		if store.count() != 0 {
			t.Fatal("stored after callback error")
		}
	})
}

func TestOfflineAccessIsExplicit(t *testing.T) {
	h := newHarness(t)
	h.tokens = []tokenResponse{{AccessToken: "at", ExpiresIn: 3600}}
	options := h.options()
	options.RequestOfflineAccess = false
	options.TargetRepository = ""
	options.AppID = 0
	summary, err := h.login(context.Background(), newFakeStore(), newFakeClock(), options, nil)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if summary.CanRefresh {
		t.Fatal("CanRefresh is true without a refresh token")
	}
	form := h.requestsFor("/login/device/code")[0].Form
	if form.Has("scope") {
		t.Fatalf("scope was sent without offline_access: %q", form.Get("scope"))
	}
}

func TestOptionAndEndpointValidation(t *testing.T) {
	base := &Client{Store: newFakeStore()}
	tests := []Options{
		{Host: "evil.example", ClientID: "Iv1.test"},
		{Host: "github.com", ClientID: ""},
		{Host: "github.com", ClientID: "has space"},
		{Host: "github.com", ClientID: "Iv1.test", Profile: "bad profile"},
		{Host: "github.com", ClientID: "Iv1.test", TargetRepository: "owner/repo/extra"},
		{Host: "github.com", ClientID: "Iv1.test", AppID: -1},
		{Host: "https://github.com", ClientID: "Iv1.test"},
	}
	for _, options := range tests {
		if _, err := base.resolve(options); !errors.Is(err, ErrInvalidOptions) {
			t.Errorf("resolve(%+v) error = %v, want ErrInvalidOptions", options, err)
		}
	}

	badHosts := &Client{Store: newFakeStore(), AllowedHosts: []string{"bad host"}}
	if _, err := badHosts.resolve(Options{Host: "github.com", ClientID: "Iv1.test"}); !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("invalid trusted host error = %v", err)
	}

	badEndpoint := &Client{
		Store: newFakeStore(),
		endpoints: endpoints{
			DeviceCodeURL: "http://example.com/login/device/code",
			TokenURL:      "https://example.com/login/oauth/access_token",
			APIBaseURL:    "https://example.com/api/v3",
		},
	}
	if _, err := badEndpoint.resolve(Options{ClientID: "Iv1.test"}); !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("non-loopback HTTP endpoint error = %v", err)
	}
}
