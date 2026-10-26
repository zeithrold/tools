package githubauth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type apiStubTransport struct {
	mu        sync.Mutex
	requests  []string
	roundTrip func(*http.Request) (*http.Response, error)
}

func (s *apiStubTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	s.mu.Lock()
	s.requests = append(s.requests, request.URL.String())
	s.mu.Unlock()
	if s.roundTrip != nil {
		return s.roundTrip(request)
	}
	return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
}

func (s *apiStubTransport) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func (s *apiStubTransport) requestURLs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func apiTestClient(t *testing.T) (*harness, *fakeStore, *fakeClock, *Client, Options, resolvedOptions) {
	t.Helper()
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	client := h.client(store, clock)
	options := h.options()
	resolved, err := client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	record := newRecord(t, resolved, "old-access-token", "old-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	return h, store, clock, client, options, resolved
}

func openTestAPI(t *testing.T, client *Client, options Options) API {
	t.Helper()
	api, err := client.OpenAPI(context.Background(), options)
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	return api
}

func apiRequest(t *testing.T, baseURL, path string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func assertSanitizedResponseRequest(t *testing.T, response *http.Response, secrets ...string) *http.Request {
	t.Helper()
	if response == nil || response.Request == nil {
		t.Fatal("response does not include its originating request")
	}
	request := response.Request
	for name, values := range request.Header {
		if strings.EqualFold(name, "Authorization") ||
			strings.EqualFold(name, "Proxy-Authorization") ||
			strings.EqualFold(name, "Cookie") {
			t.Errorf("response request retains credential header %q", name)
		}
		for _, value := range values {
			assertNoSecret(t, value, secrets...)
		}
	}
	return request
}

func TestOpenAPIRequiresAndBindsTargetBeforeNetwork(t *testing.T) {
	h, store, clock, client, options, resolved := apiTestClient(t)
	emptyTarget := options
	emptyTarget.TargetRepository = ""
	if _, err := client.OpenAPI(context.Background(), emptyTarget); !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("empty target error = %v, want ErrInvalidOptions", err)
	}
	if h.requestCount() != 0 {
		t.Fatalf("empty target made %d network requests", h.requestCount())
	}

	record := newRecord(t, resolved, "old-access-token", "old-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = "someone/else"
	store.put(t, resolved.key, record)
	if _, err := client.OpenAPI(context.Background(), options); !errors.Is(err, ErrAccountMismatch) {
		t.Fatalf("mismatched target error = %v, want ErrAccountMismatch", err)
	}
	if h.requestCount() != 0 {
		t.Fatalf("mismatched target made %d network requests", h.requestCount())
	}
}

func TestOpenAPIReturnsNotLoggedInAndStoreUnavailable(t *testing.T) {
	h := newHarness(t)
	clock := newFakeClock()
	options := h.options()
	client := h.client(newFakeStore(), clock)
	if _, err := client.OpenAPI(context.Background(), options); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("empty store error = %v, want ErrNotLoggedIn", err)
	}
	client.Store = nil
	if _, err := client.OpenAPI(context.Background(), options); !errors.Is(err, ErrStoreUnavailable) {
		t.Fatalf("missing store error = %v, want ErrStoreUnavailable", err)
	}
	if h.requestCount() != 0 {
		t.Fatalf("opening without credentials made %d network requests", h.requestCount())
	}
}

func TestOpenAPISendsRequiredHeadersToSelectedAPIAndRepository(t *testing.T) {
	h, _, _, client, options, _ := apiTestClient(t)
	h.override = func(w http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path != "/repos/owner/repo/git/ref/heads/main" &&
			request.URL.Path != "/repos/owner/repo/pulls" {
			return false
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
		return true
	}
	api := openTestAPI(t, client, options)
	if api.BaseURL != h.srv.URL || strings.HasSuffix(api.BaseURL, "/") {
		t.Fatalf("BaseURL = %q, want the API base without a trailing slash", api.BaseURL)
	}

	for _, path := range []string{"/repos/owner/repo/git/ref/heads/main", "/repos/owner/repo/pulls"} {
		request := apiRequest(t, api.BaseURL, path)
		response, err := api.Transport.RoundTrip(request)
		if err != nil {
			t.Fatalf("RoundTrip(%s): %v", path, err)
		}
		_ = response.Body.Close()
		if request.Header.Get("Authorization") != "" {
			t.Fatal("RoundTrip mutated the caller's request headers")
		}
	}

	requests := h.requestsFor("/repos/owner/repo/git/ref/heads/main")
	if len(requests) != 1 {
		t.Fatalf("Git-data requests = %d, want 1", len(requests))
	}
	headers := requests[0].Header
	for name, want := range map[string]string{
		"Authorization":        "Bearer old-access-token",
		"Accept":               "application/vnd.github+json",
		"User-Agent":           "zt-github-auth",
		"X-Github-Api-Version": "2022-11-28",
	} {
		if got := headers.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if len(h.requestsFor("/repos/owner/repo/pulls")) != 1 {
		t.Fatal("pull-request path did not reach the selected API base")
	}
}

func TestOpenAPIResponseRequestOmitsAccessToken(t *testing.T) {
	const token = "response-request-access-token"
	h, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, token, "response-request-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	client.httpClient.Transport = http.DefaultTransport
	h.override = func(w http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path != "/repos/owner/repo/issues" {
			return false
		}
		w.WriteHeader(http.StatusOK)
		return true
	}

	api := openTestAPI(t, client, options)
	request := apiRequest(t, api.BaseURL, "/repos/owner/repo/issues")
	request.Header.Set("X-Request-Marker", "preserved")
	request.Header.Set("Proxy-Authorization", "Basic proxy-secret")
	request.Header.Set("Cookie", "session=cookie-secret")
	type contextKey struct{}
	request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "context-value"))
	response, err := api.Transport.RoundTrip(request)
	if err != nil {
		assertNoSecret(t, err.Error(), token)
		t.Fatalf("RoundTrip: %v", err)
	}
	defer response.Body.Close()

	responseRequest := assertSanitizedResponseRequest(t, response, token)
	if responseRequest.Method != request.Method || responseRequest.URL.String() != request.URL.String() {
		t.Fatalf("response request = %s %s, want %s %s", responseRequest.Method, responseRequest.URL, request.Method, request.URL)
	}
	if got := responseRequest.Header.Get("X-Request-Marker"); got != "preserved" {
		t.Fatalf("response request marker = %q, want preserved", got)
	}
	if got := responseRequest.Context().Value(contextKey{}); got != "context-value" {
		t.Fatalf("response request context value = %v, want context-value", got)
	}
	if got := responseRequest.Header.Get("Authorization"); got != "" {
		t.Fatalf("response request Authorization = %q, want no Authorization header", got)
	}
	requests := h.requestsFor("/repos/owner/repo/issues")
	if len(requests) != 1 || requests[0].Header.Get("Authorization") != "Bearer "+token {
		t.Fatalf("server API request authorizations = %+v, want Bearer token on the wire", requests)
	}
}

func TestOpenAPIRefreshRetryResponseRequestOmitsBothTokens(t *testing.T) {
	const oldToken = "retry-response-old-access-token"
	const newToken = "retry-response-new-access-token"
	h, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, oldToken, "retry-response-refresh-token", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	h.tokens = []tokenResponse{{
		AccessToken:           newToken,
		RefreshToken:          "retry-response-new-refresh-token",
		TokenType:             "bearer",
		Scope:                 "offline_access",
		ExpiresIn:             3600,
		RefreshTokenExpiresIn: 7200,
	}}
	client.httpClient.Transport = http.DefaultTransport
	var apiCalls int
	h.override = func(w http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path != "/repos/owner/repo/issues" {
			return false
		}
		apiCalls++
		if apiCalls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return true
		}
		w.WriteHeader(http.StatusOK)
		return true
	}

	api := openTestAPI(t, client, options)
	request := apiRequest(t, api.BaseURL, "/repos/owner/repo/issues")
	request.Header.Set("X-Request-Marker", "preserved")
	type contextKey struct{}
	request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "context-value"))
	response, err := api.Transport.RoundTrip(request)
	if err != nil {
		assertNoSecret(t, err.Error(), oldToken, newToken)
		t.Fatalf("RoundTrip after refresh: %v", err)
	}
	defer response.Body.Close()

	responseRequest := assertSanitizedResponseRequest(t, response, oldToken, newToken)
	if responseRequest.Method != request.Method || responseRequest.URL.String() != request.URL.String() {
		t.Fatalf("retry response request = %s %s, want %s %s", responseRequest.Method, responseRequest.URL, request.Method, request.URL)
	}
	if got := responseRequest.Header.Get("X-Request-Marker"); got != "preserved" {
		t.Fatalf("retry response request marker = %q, want preserved", got)
	}
	if got := responseRequest.Context().Value(contextKey{}); got != "context-value" {
		t.Fatalf("retry response request context value = %v, want context-value", got)
	}
	if apiCalls != 2 {
		t.Fatalf("API calls = %d, want initial 401 and one retry", apiCalls)
	}
	requests := h.requestsFor("/repos/owner/repo/issues")
	if len(requests) != 2 || requests[0].Header.Get("Authorization") != "Bearer "+oldToken ||
		requests[1].Header.Get("Authorization") != "Bearer "+newToken {
		t.Fatalf("API request authorizations = %+v", requests)
	}
}

func TestOpenAPIRefusesPathsOutsideVerifiedRepository(t *testing.T) {
	_, _, _, client, options, _ := apiTestClient(t)
	stub := &apiStubTransport{}
	client.httpClient = &http.Client{Transport: stub}
	api := openTestAPI(t, client, options)
	base, err := url.Parse(api.BaseURL)
	if err != nil {
		t.Fatal(err)
	}
	host := base.Host
	tests := []struct {
		name string
		url  string
	}{
		{"another repository", api.BaseURL + "/repos/other/name/issues"},
		{"user endpoint", api.BaseURL + "/user"},
		{"bare API prefix", api.BaseURL + "/"},
		{"bare owner", api.BaseURL + "/repos/owner"},
		{"raw traversal", api.BaseURL + "/repos/owner/repo/../other/name/x"},
		{"encoded traversal", api.BaseURL + "/repos/owner/repo/%2e%2e/other/name/x"},
		{"encoded separator", api.BaseURL + "/repos/owner/repo%2fname/x"},
		{"encoded owner spelling", api.BaseURL + "/repos/%6fwner/repo/issues"},
		{"userinfo", "http://credential@" + host + "/repos/owner/repo/issues"},
		{"foreign host", "http://example.invalid/repos/owner/repo/issues"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, test.url, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := api.Transport.RoundTrip(request)
			if response != nil {
				_ = response.Body.Close()
			}
			if !errors.Is(err, ErrInvalidOptions) {
				t.Fatalf("RoundTrip error = %v, want ErrInvalidOptions", err)
			}
			if stub.count() != 0 {
				t.Fatalf("invalid request reached transport %d times", stub.count())
			}
		})
	}
}

func TestOpenAPIRefusesCrossHostRedirectBeforeTransport(t *testing.T) {
	_, _, _, client, options, _ := apiTestClient(t)
	stub := &apiStubTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://example.invalid/repos/owner/repo/issues"}},
			Body:       http.NoBody,
			Request:    request,
		}, nil
	}}
	client.httpClient = &http.Client{Transport: stub}
	api := openTestAPI(t, client, options)
	request, err := http.NewRequest(http.MethodGet, api.BaseURL+"/repos/owner/repo/issues", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&http.Client{Transport: api.Transport}).Do(request)
	if err == nil {
		t.Fatal("cross-host redirect was followed")
	}
	if stub.count() != 1 {
		t.Fatalf("stub transport received %d requests; only the initial selected-host request is allowed", stub.count())
	}
	for _, requestURL := range stub.requestURLs() {
		parsed, parseErr := url.Parse(requestURL)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		if parsed.Host != "example.invalid" {
			continue
		}
		t.Fatal("cross-host redirect target reached the stub transport")
	}
}

func TestOpenAPIRetriesOnceAfter401WithRefreshedToken(t *testing.T) {
	h, _, _, client, options, _ := apiTestClient(t)
	h.tokens = []tokenResponse{{
		AccessToken:           "new-access-token",
		RefreshToken:          "new-refresh-token",
		TokenType:             "bearer",
		Scope:                 "offline_access",
		ExpiresIn:             3600,
		RefreshTokenExpiresIn: 7200,
	}}
	var apiCalls int
	h.override = func(w http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path != "/repos/owner/repo/issues" {
			return false
		}
		apiCalls++
		if apiCalls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return true
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
		return true
	}
	api := openTestAPI(t, client, options)
	response, err := api.Transport.RoundTrip(apiRequest(t, api.BaseURL, "/repos/owner/repo/issues"))
	if err != nil {
		t.Fatalf("RoundTrip after refresh: %v", err)
	}
	_ = response.Body.Close()
	if apiCalls != 2 {
		t.Fatalf("API calls = %d, want exactly 2", apiCalls)
	}
	requests := h.requestsFor("/repos/owner/repo/issues")
	if len(requests) != 2 || requests[0].Header.Get("Authorization") != "Bearer old-access-token" ||
		requests[1].Header.Get("Authorization") != "Bearer new-access-token" {
		t.Fatalf("API request authorizations = %+v", requests)
	}
	if len(h.requestsFor("/login/oauth/access_token")) != 1 {
		t.Fatal("401 did not cause exactly one token refresh")
	}
}

func TestOpenAPIRejectsChangedCredentialTargetBeforeFirstSend(t *testing.T) {
	_, _, _, client, options, _ := apiTestClient(t)
	stub := &apiStubTransport{}
	client.httpClient = &http.Client{Transport: stub}
	api := openTestAPI(t, client, options)
	transport := api.Transport.(*apiTransport)
	credential := transport.currentCredential()
	credential.TargetRepository = "owner/other"
	transport.setCredential(credential)

	_, err := api.Transport.RoundTrip(apiRequest(t, api.BaseURL, "/repos/owner/repo/issues"))
	if !errors.Is(err, ErrAccountMismatch) {
		t.Fatalf("RoundTrip error = %v, want ErrAccountMismatch", err)
	}
	assertNoSecret(t, err.Error(), credential.AccessToken)
	if stub.count() != 0 {
		t.Fatalf("mismatched credential reached API transport %d times, want zero", stub.count())
	}
}

func TestOpenAPIRejectsRefreshAdoptingCredentialForAnotherRepository(t *testing.T) {
	const adoptedToken = "adopted-token-for-another-repository"
	h, store, clock, client, options, resolved := apiTestClient(t)
	adopted := newRecord(t, resolved, adoptedToken, "adopted-refresh-token", clock.Now().Add(time.Hour))
	adopted.TargetRepository = "owner/other"
	var calls int
	stub := &apiStubTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			if got := request.Header.Get("Authorization"); got != "Bearer old-access-token" {
				t.Errorf("initial request Authorization = %q, want old credential", got)
			}
			store.put(t, resolved.key, adopted)
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Header:     make(http.Header),
				Body:       http.NoBody,
				Request:    request,
			}, nil
		}
		return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
	}}
	client.httpClient = &http.Client{Transport: stub}
	api := openTestAPI(t, client, options)

	_, err := api.Transport.RoundTrip(apiRequest(t, api.BaseURL, "/repos/owner/repo/issues"))
	if !errors.Is(err, ErrAccountMismatch) {
		t.Fatalf("RoundTrip error = %v, want ErrAccountMismatch", err)
	}
	assertNoSecret(t, err.Error(), "old-access-token", adoptedToken)
	if calls != 1 || stub.count() != 1 {
		t.Fatalf("API transport calls = %d (recorded %d), want only the initial 401 and zero retries", calls, stub.count())
	}
	if h.requestCount() != 0 {
		t.Fatalf("adopting another repository's credential made %d additional HTTP requests", h.requestCount())
	}
}

func TestOpenAPISecond401RequiresRelogin(t *testing.T) {
	h, _, _, client, options, _ := apiTestClient(t)
	h.tokens = []tokenResponse{{
		AccessToken:           "new-access-token",
		RefreshToken:          "new-refresh-token",
		TokenType:             "bearer",
		Scope:                 "offline_access",
		ExpiresIn:             3600,
		RefreshTokenExpiresIn: 7200,
	}}
	h.override = func(w http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path != "/repos/owner/repo/issues" {
			return false
		}
		w.WriteHeader(http.StatusUnauthorized)
		return true
	}
	api := openTestAPI(t, client, options)
	_, err := api.Transport.RoundTrip(apiRequest(t, api.BaseURL, "/repos/owner/repo/issues"))
	if !errors.Is(err, ErrReloginRequired) {
		t.Fatalf("second 401 error = %v, want ErrReloginRequired", err)
	}
	if len(h.requestsFor("/repos/owner/repo/issues")) != 2 || len(h.requestsFor("/login/oauth/access_token")) != 1 {
		t.Fatalf("requests after repeated 401 = %+v", h.requests)
	}
}

func TestOpenAPIReturns422WithoutTranslatingStatus(t *testing.T) {
	h, _, _, client, options, _ := apiTestClient(t)
	h.override = func(w http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path != "/repos/owner/repo/issues" {
			return false
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"message":"validation failed"}`)
		return true
	}
	api := openTestAPI(t, client, options)
	response, err := api.Transport.RoundTrip(apiRequest(t, api.BaseURL, "/repos/owner/repo/issues"))
	if err != nil {
		t.Fatalf("422 RoundTrip error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.StatusCode)
	}
}

func TestOpenAPIRemoteAndRefusalErrorsDoNotExposeTokenOrQuery(t *testing.T) {
	const token = "access-secret-for-error-test"
	_, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, token, "refresh-secret-for-error-test", clock.Now().Add(time.Hour))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	client.httpClient = &http.Client{Transport: errorAPIStub{err: errors.New("transport exposed " + token)}}
	api := openTestAPI(t, client, options)

	request := apiRequest(t, api.BaseURL, "/repos/owner/repo/issues")
	_, err := api.Transport.RoundTrip(request)
	if !errors.Is(err, ErrRemote) {
		t.Fatalf("transport error = %v, want ErrRemote", err)
	}
	assertNoSecret(t, err.Error(), token)

	request, err = http.NewRequest(http.MethodGet, api.BaseURL+"/user?access_token="+url.QueryEscape(token), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = api.Transport.RoundTrip(request)
	if !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("refusal error = %v, want ErrInvalidOptions", err)
	}
	assertNoSecret(t, err.Error(), token)
	if strings.Contains(err.Error(), "access_token=") {
		t.Fatalf("refusal error includes query details: %v", err)
	}
}

type errorAPIStub struct {
	err error
}

func (s errorAPIStub) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, s.err
}

func TestOpenAPIRefreshesExpiredTokenBeforeFirstAPIRequest(t *testing.T) {
	h, store, clock, client, options, resolved := apiTestClient(t)
	h.tokens = []tokenResponse{{
		AccessToken:           "new-access-token",
		RefreshToken:          "new-refresh-token",
		TokenType:             "bearer",
		Scope:                 "offline_access",
		ExpiresIn:             3600,
		RefreshTokenExpiresIn: 7200,
	}}
	record := newRecord(t, resolved, "expired-access-token", "expired-refresh-token", clock.Now().Add(-time.Minute))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	h.override = func(w http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path != "/repos/owner/repo/issues" {
			return false
		}
		w.WriteHeader(http.StatusOK)
		return true
	}
	api := openTestAPI(t, client, options)
	response, err := api.Transport.RoundTrip(apiRequest(t, api.BaseURL, "/repos/owner/repo/issues"))
	if err != nil {
		t.Fatalf("first API request after refresh: %v", err)
	}
	_ = response.Body.Close()

	if len(h.requestsFor("/login/oauth/access_token")) != 1 {
		t.Fatal("expired credential was not refreshed before the API request")
	}
	requests := h.requestsFor("/repos/owner/repo/issues")
	if len(requests) != 1 || requests[0].Header.Get("Authorization") != "Bearer new-access-token" {
		t.Fatalf("first API request = %+v", requests)
	}
	if len(h.requests) < 2 || h.requests[0].Path != "/login/oauth/access_token" || h.requests[1].Path != "/repos/owner/repo/issues" {
		t.Fatalf("request order = %+v, wanted refresh before API request", h.requests)
	}
}

func TestOpenAPIAcceptsAnUnboundStoredTarget(t *testing.T) {
	h, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, "old-access-token", "old-refresh-token", clock.Now().Add(time.Hour))
	store.put(t, resolved.key, record)
	h.override = func(w http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path == "/repos/owner/repo/issues" {
			w.WriteHeader(http.StatusOK)
			return true
		}
		return false
	}
	api := openTestAPI(t, client, options)
	response, err := api.Transport.RoundTrip(apiRequest(t, api.BaseURL, "/repos/owner/repo/issues"))
	if err != nil {
		t.Fatalf("target access request: %v", err)
	}
	_ = response.Body.Close()
	if len(h.requestsFor("/repos/owner/repo/issues")) != 1 {
		t.Fatal("requested target was not available to the first API request")
	}
}

func TestOpenAPIRefreshErrorsPreserveSentinelAndHideToken(t *testing.T) {
	const token = "secret-access-token-on-refresh-failure"
	h, store, clock, client, options, resolved := apiTestClient(t)
	record := newRecord(t, resolved, token, "refresh-token", clock.Now().Add(-time.Minute))
	record.TargetRepository = options.TargetRepository
	store.put(t, resolved.key, record)
	h.override = func(w http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path != "/login/oauth/access_token" {
			return false
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
		return true
	}
	_, err := client.OpenAPI(context.Background(), options)
	if !errors.Is(err, ErrReloginRequired) {
		t.Fatalf("expired-token refresh error = %v, want ErrReloginRequired", err)
	}
	assertNoSecret(t, err.Error(), token)
	if h.requestCount() != 1 {
		t.Fatalf("refresh failure made %d requests, want only refresh", h.requestCount())
	}
}
