package githubauth

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func openTestGit(t *testing.T, client *Client, options Options) GitEndpoint {
	t.Helper()
	endpoint, err := client.OpenGit(context.Background(), options)
	if err != nil {
		t.Fatalf("OpenGit: %v", err)
	}
	return endpoint
}

func gitRequest(t *testing.T, method, target string, body io.Reader) *http.Request {
	t.Helper()
	request, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

// gitBasicCredential is the credential GitHub accepts for an App installation
// token used as a Git password.
func gitBasicCredential(token string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+token))
}

// gitWebURL is the scheme and host of a Git endpoint, used to build requests
// that must be refused.
func gitWebURL(t *testing.T, endpoint GitEndpoint) (scheme, host string) {
	t.Helper()
	parsed, err := url.Parse(endpoint.URL)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Scheme, parsed.Host
}

func TestOpenGitUsesWebHostAndCanonicalRemote(t *testing.T) {
	store := newFakeStore()
	clock := newFakeClock()
	client := &Client{Store: store, AllowedHosts: []string{"github.com"}, clock: clock}
	options := Options{ClientID: "Iv1.test-client", Profile: "tools", TargetRepository: "owner/repo"}
	resolved, err := client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	record := newRecord(t, resolved, "canonical-remote-token", "canonical-remote-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)

	endpoint := openTestGit(t, client, options)
	if endpoint.URL != "https://github.com/owner/repo.git" {
		t.Fatalf("Git endpoint URL = %q, want the canonical upload-pack remote on the web host", endpoint.URL)
	}
	if endpoint.Transport == nil {
		t.Fatal("Git endpoint has no transport")
	}
}

func TestOpenGitRequiresAndBindsTargetBeforeNetwork(t *testing.T) {
	h, store, clock, client, options, resolved := apiTestClient(t)

	emptyTarget := options
	emptyTarget.TargetRepository = ""
	if _, err := client.OpenGit(context.Background(), emptyTarget); !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("empty target error = %v, want ErrInvalidOptions", err)
	}

	record := newRecord(t, resolved, "bound-access-token", "bound-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = "someone/else"
	store.put(t, resolved.key, record)
	if _, err := client.OpenGit(context.Background(), options); !errors.Is(err, ErrAccountMismatch) {
		t.Fatalf("mismatched target error = %v, want ErrAccountMismatch", err)
	}
	if h.requestCount() != 0 {
		t.Fatalf("refused Git endpoints made %d network requests, want none", h.requestCount())
	}
}

func TestOpenGitRefusesRequestsOutsideUploadPack(t *testing.T) {
	_, _, _, client, options, _ := apiTestClient(t)
	stub := &apiStubTransport{}
	client.httpClient = &http.Client{Transport: stub}
	endpoint := openTestGit(t, client, options)
	scheme, host := gitWebURL(t, endpoint)
	repository := endpoint.URL

	tests := []struct {
		name   string
		method string
		target string
	}{
		{"another repository", http.MethodGet, scheme + "://" + host + "/other/name.git/info/refs?service=git-upload-pack"},
		{"repository without .git suffix", http.MethodGet, scheme + "://" + host + "/owner/repo/info/refs?service=git-upload-pack"},
		{"discovery without service", http.MethodGet, repository + "/info/refs"},
		{"receive-pack discovery", http.MethodGet, repository + "/info/refs?service=git-receive-pack"},
		{"extra discovery parameter", http.MethodGet, repository + "/info/refs?service=git-upload-pack&depth=1"},
		{"repeated service", http.MethodGet, repository + "/info/refs?service=git-upload-pack&service=git-upload-pack"},
		{"receive-pack negotiation", http.MethodPost, repository + "/git-receive-pack"},
		{"negotiation with query", http.MethodPost, repository + "/git-upload-pack?service=git-upload-pack"},
		{"GET on negotiation", http.MethodGet, repository + "/git-upload-pack"},
		{"POST on discovery", http.MethodPost, repository + "/info/refs?service=git-upload-pack"},
		{"unsupported method", http.MethodPut, repository + "/git-upload-pack"},
		{"raw traversal", http.MethodGet, repository + "/../other/name.git/info/refs?service=git-upload-pack"},
		{"encoded traversal", http.MethodGet, repository + "/%2e%2e/other/name.git/info/refs?service=git-upload-pack"},
		{"encoded separator", http.MethodGet, scheme + "://" + host + "/owner%2frepo.git/info/refs?service=git-upload-pack"},
		{"userinfo", http.MethodGet, scheme + "://credential@" + host + "/owner/repo.git/info/refs?service=git-upload-pack"},
		{"foreign host", http.MethodGet, "http://example.invalid/owner/repo.git/info/refs?service=git-upload-pack"},
		{"fragment", http.MethodGet, repository + "/info/refs?service=git-upload-pack#main"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := gitRequest(t, test.method, test.target, strings.NewReader(""))
			response, err := endpoint.Transport.RoundTrip(request)
			if response != nil {
				_ = response.Body.Close()
			}
			if !errors.Is(err, ErrInvalidOptions) {
				t.Fatalf("RoundTrip error = %v, want ErrInvalidOptions", err)
			}
			assertNoSecret(t, err.Error(), "old-access-token")
			if strings.Contains(err.Error(), "service=") {
				t.Fatalf("refusal error includes query details: %v", err)
			}
		})
	}
	if stub.count() != 0 {
		t.Fatalf("refused requests reached the Git transport %d times, want zero", stub.count())
	}
}

func TestOpenGitRejectsOpaqueRoutesBeforeCredentials(t *testing.T) {
	_, _, _, client, options, _ := apiTestClient(t)
	stub := &apiStubTransport{}
	client.httpClient = &http.Client{Transport: stub}
	endpoint := openTestGit(t, client, options)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			suffix := "/info/refs?service=git-upload-pack"
			if method == http.MethodPost {
				suffix = "/git-upload-pack"
			}
			request := gitRequest(t, method, endpoint.URL+suffix, nil)
			request.URL.Opaque = "/someone/else.git/git-receive-pack"
			response, err := endpoint.Transport.RoundTrip(request)
			if response != nil {
				response.Body.Close()
			}
			if !errors.Is(err, ErrInvalidOptions) {
				t.Fatalf("opaque %s request error = %v, want ErrInvalidOptions", method, err)
			}
		})
	}
	if stub.count() != 0 {
		t.Fatalf("opaque route attached credential to %d requests", stub.count())
	}
}

func TestOpenGitSendsUploadPackHeadersAndPreservesMediaTypes(t *testing.T) {
	const token = "git-wire-access-token"
	h, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, token, "git-wire-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)

	var discoveryQuery string
	h.override = func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/owner/repo.git/info/refs":
			discoveryQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
			w.WriteHeader(http.StatusOK)
			return true
		case "/owner/repo.git/git-upload-pack":
			w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
			w.WriteHeader(http.StatusOK)
			return true
		}
		return false
	}

	endpoint := openTestGit(t, client, options)
	if endpoint.URL != h.srv.URL+"/owner/repo.git" {
		t.Fatalf("Git endpoint URL = %q, want the fixture repository remote", endpoint.URL)
	}

	discovery := gitRequest(t, http.MethodGet, endpoint.URL+"/info/refs?service=git-upload-pack", nil)
	discovery.Header.Set("Accept", "*/*")
	discovery.Header.Set("Git-Protocol", "version=2")
	response, err := endpoint.Transport.RoundTrip(discovery)
	if err != nil {
		assertNoSecret(t, err.Error(), token)
		t.Fatalf("discovery RoundTrip: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/x-git-upload-pack-advertisement" {
		t.Fatalf("discovery response = %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	if got := assertSanitizedResponseRequest(t, response, token).Header.Get("Authorization"); got != "" {
		t.Fatalf("discovery response request Authorization = %q, want none", got)
	}
	if discovery.Header.Get("Authorization") != "" {
		t.Fatal("RoundTrip mutated the caller's request headers")
	}
	if discoveryQuery != "service=git-upload-pack" {
		t.Fatalf("discovery query = %q, want service=git-upload-pack", discoveryQuery)
	}

	const negotiation = "0032want 1f0c3a9d5b\n0000"
	fetch := gitRequest(t, http.MethodPost, endpoint.URL+"/git-upload-pack", strings.NewReader(negotiation))
	fetch.Header.Set("Content-Type", "application/x-git-upload-pack-request")
	fetch.Header.Set("Accept", "application/x-git-upload-pack-result")
	response, err = endpoint.Transport.RoundTrip(fetch)
	if err != nil {
		assertNoSecret(t, err.Error(), token)
		t.Fatalf("fetch RoundTrip: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/x-git-upload-pack-result" {
		t.Fatalf("fetch response = %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}

	discoveries := h.requestsFor("/owner/repo.git/info/refs")
	if len(discoveries) != 1 {
		t.Fatalf("discovery requests = %d, want 1", len(discoveries))
	}
	for name, want := range map[string]string{
		"Authorization": gitBasicCredential(token),
		"Accept":        "*/*",
		"Git-Protocol":  "version=2",
		"User-Agent":    "zt-github-auth",
	} {
		if got := discoveries[0].Header.Get(name); got != want {
			t.Errorf("discovery %s = %q, want %q", name, got, want)
		}
	}
	if strings.Contains(discoveries[0].Header.Get("Authorization"), "Bearer") {
		t.Fatal("Git discovery used the REST bearer scheme instead of the x-access-token Basic credential")
	}

	fetches := h.requestsFor("/owner/repo.git/git-upload-pack")
	if len(fetches) != 1 {
		t.Fatalf("negotiation requests = %d, want 1", len(fetches))
	}
	for name, want := range map[string]string{
		"Authorization": gitBasicCredential(token),
		"Accept":        "application/x-git-upload-pack-result",
		"Content-Type":  "application/x-git-upload-pack-request",
	} {
		if got := fetches[0].Header.Get(name); got != want {
			t.Errorf("negotiation %s = %q, want %q", name, got, want)
		}
	}
	if fetches[0].Body != negotiation {
		t.Fatalf("negotiation body = %q, want %q", fetches[0].Body, negotiation)
	}
}

func TestOpenGitRetriesOnceAfter401WithRefreshedCredential(t *testing.T) {
	const oldToken = "git-retry-old-access-token"
	const newToken = "git-retry-new-access-token"
	h, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, oldToken, "git-retry-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	h.tokens = []tokenResponse{{
		AccessToken:           newToken,
		RefreshToken:          "git-retry-new-refresh-token",
		TokenType:             "bearer",
		Scope:                 "offline_access",
		ExpiresIn:             3600,
		RefreshTokenExpiresIn: 7200,
	}}
	var negotiations int
	h.override = func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/owner/repo.git/info/refs":
			w.WriteHeader(http.StatusOK)
			return true
		case "/owner/repo.git/git-upload-pack":
			negotiations++
			if negotiations == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return true
			}
			w.WriteHeader(http.StatusOK)
			return true
		}
		return false
	}

	endpoint := openTestGit(t, client, options)
	const negotiation = "0032want 4b825dc642cb6eb9a060e54bf8d69288fbee4904\n0000"
	request := gitRequest(t, http.MethodPost, endpoint.URL+"/git-upload-pack", strings.NewReader(negotiation))
	request.Header.Set("Content-Type", "application/x-git-upload-pack-request")
	response, err := endpoint.Transport.RoundTrip(request)
	if err != nil {
		assertNoSecret(t, err.Error(), oldToken, newToken)
		t.Fatalf("RoundTrip after refresh: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || negotiations != 2 {
		t.Fatalf("negotiations = %d with final status %d, want one 401 and one retry", negotiations, response.StatusCode)
	}

	fetches := h.requestsFor("/owner/repo.git/git-upload-pack")
	if len(fetches) != 2 {
		t.Fatalf("negotiation requests = %d, want 2", len(fetches))
	}
	if got := fetches[0].Header.Get("Authorization"); got != gitBasicCredential(oldToken) {
		t.Fatalf("initial negotiation authorization = %q, want the old credential", got)
	}
	if got := fetches[1].Header.Get("Authorization"); got != gitBasicCredential(newToken) {
		t.Fatalf("retried negotiation authorization = %q, want the refreshed credential", got)
	}
	for index, fetch := range fetches {
		if fetch.Body != negotiation {
			t.Fatalf("negotiation %d replayed body = %q, want %q", index, fetch.Body, negotiation)
		}
	}
	if len(h.requestsFor("/login/oauth/access_token")) != 1 {
		t.Fatal("401 did not cause exactly one token refresh")
	}
	updated, _ := store.onlyRecord(t)
	if updated.AccessToken != newToken || updated.TargetRepository != options.TargetRepository {
		t.Fatalf("stored credential after refresh = %+v, want the refreshed token for the same repository", updated)
	}
}

func TestOpenGitRefreshesExpiredCredentialBeforeFirstRequest(t *testing.T) {
	h, store, clock, client, options, resolved := apiTestClient(t)
	h.tokens = []tokenResponse{{
		AccessToken:           "git-expired-new-access-token",
		RefreshToken:          "git-expired-new-refresh-token",
		TokenType:             "bearer",
		Scope:                 "offline_access",
		ExpiresIn:             3600,
		RefreshTokenExpiresIn: 7200,
	}}
	record := newRecord(t, resolved, "git-expired-access-token", "git-expired-refresh-token", clock.Now().Add(-time.Minute))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	h.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/owner/repo.git/info/refs" {
			return false
		}
		w.WriteHeader(http.StatusOK)
		return true
	}

	endpoint := openTestGit(t, client, options)
	response, err := endpoint.Transport.RoundTrip(
		gitRequest(t, http.MethodGet, endpoint.URL+"/info/refs?service=git-upload-pack", nil))
	if err != nil {
		t.Fatalf("first Git request after refresh: %v", err)
	}
	_ = response.Body.Close()

	discoveries := h.requestsFor("/owner/repo.git/info/refs")
	if len(discoveries) != 1 || discoveries[0].Header.Get("Authorization") != gitBasicCredential("git-expired-new-access-token") {
		t.Fatalf("discovery requests = %+v, want the refreshed credential", discoveries)
	}
	if len(h.requests) < 2 || h.requests[0].Path != "/login/oauth/access_token" || h.requests[1].Path != "/owner/repo.git/info/refs" {
		t.Fatalf("request order = %+v, wanted refresh before the Git request", h.requests)
	}
}

func TestOpenGitSecond401RequiresRelogin(t *testing.T) {
	const token = "git-relogin-access-token"
	h, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, token, "git-relogin-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	h.tokens = []tokenResponse{{
		AccessToken:           "git-relogin-new-access-token",
		RefreshToken:          "git-relogin-new-refresh-token",
		TokenType:             "bearer",
		Scope:                 "offline_access",
		ExpiresIn:             3600,
		RefreshTokenExpiresIn: 7200,
	}}
	h.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/owner/repo.git/info/refs" {
			return false
		}
		w.WriteHeader(http.StatusUnauthorized)
		return true
	}

	endpoint := openTestGit(t, client, options)
	_, err := endpoint.Transport.RoundTrip(
		gitRequest(t, http.MethodGet, endpoint.URL+"/info/refs?service=git-upload-pack", nil))
	if !errors.Is(err, ErrReloginRequired) {
		t.Fatalf("repeated 401 error = %v, want ErrReloginRequired", err)
	}
	assertNoSecret(t, err.Error(), token, "git-relogin-new-access-token")
	if len(h.requestsFor("/owner/repo.git/info/refs")) != 2 || len(h.requestsFor("/login/oauth/access_token")) != 1 {
		t.Fatalf("requests after repeated 401 = %+v", h.requests)
	}
}

func TestOpenGitRefusesToRetryNonReplayableBody(t *testing.T) {
	h, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, "streamed-access-token", "streamed-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	h.tokens = []tokenResponse{{
		AccessToken:           "streamed-new-access-token",
		RefreshToken:          "streamed-new-refresh-token",
		TokenType:             "bearer",
		Scope:                 "offline_access",
		ExpiresIn:             3600,
		RefreshTokenExpiresIn: 7200,
	}}
	var negotiations int
	h.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/owner/repo.git/git-upload-pack" {
			return false
		}
		negotiations++
		w.WriteHeader(http.StatusUnauthorized)
		return true
	}

	endpoint := openTestGit(t, client, options)
	request := gitRequest(t, http.MethodPost, endpoint.URL+"/git-upload-pack", io.LimitReader(strings.NewReader("0032want 1\n0000"), 1024))
	if request.GetBody != nil {
		t.Fatal("test request body is replayable; it cannot exercise the refusal")
	}
	_, err := endpoint.Transport.RoundTrip(request)
	if !errors.Is(err, ErrProtocol) {
		t.Fatalf("non-replayable 401 error = %v, want ErrProtocol", err)
	}
	assertNoSecret(t, err.Error(), "streamed-access-token", "streamed-new-access-token")
	if negotiations != 1 {
		t.Fatalf("negotiations = %d, want only the initial 401 and no replay", negotiations)
	}
	if len(h.requestsFor("/login/oauth/access_token")) != 1 {
		t.Fatal("the 401 did not refresh the credential before refusing the replay")
	}
}

func TestOpenGitRemoteAndRefusalErrorsHideToken(t *testing.T) {
	const token = "git-error-secret-access-token"
	_, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, token, "git-error-secret-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	client.httpClient = &http.Client{Transport: errorAPIStub{err: errors.New("transport exposed " + token)}}
	endpoint := openTestGit(t, client, options)

	_, err := endpoint.Transport.RoundTrip(
		gitRequest(t, http.MethodGet, endpoint.URL+"/info/refs?service=git-upload-pack", nil))
	if !errors.Is(err, ErrRemote) {
		t.Fatalf("transport error = %v, want ErrRemote", err)
	}
	assertNoSecret(t, err.Error(), token)

	scheme, host := gitWebURL(t, endpoint)
	_, err = endpoint.Transport.RoundTrip(
		gitRequest(t, http.MethodGet, scheme+"://"+host+"/user?access_token="+url.QueryEscape(token), nil))
	if !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("refusal error = %v, want ErrInvalidOptions", err)
	}
	assertNoSecret(t, err.Error(), token)
	if strings.Contains(err.Error(), "access_token=") {
		t.Fatalf("refusal error includes query details: %v", err)
	}
}
