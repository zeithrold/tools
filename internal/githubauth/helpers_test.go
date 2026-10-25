package githubauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	sleeps  []time.Duration
	onSleep func(context.Context, time.Duration) error
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.onSleep != nil {
		if err := c.onSleep(ctx, d); err != nil {
			return err
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sleeps = append(c.sleeps, d)
	c.now = c.now.Add(d)
	return nil
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func (c *fakeClock) slept() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.sleeps...)
}

type fakeStore struct {
	mu            sync.Mutex
	values        map[string][]byte
	availableErr  error
	getErr        error
	setErr        error
	deleteErr     error
	availableCall int
	setCalls      int
	deleteCalls   int
	onSet         func(key string, value []byte) error
}

func newFakeStore() *fakeStore {
	return &fakeStore{values: make(map[string][]byte)}
}

func (s *fakeStore) Available(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.availableCall++
	return s.availableErr
}

func (s *fakeStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return nil, false, s.getErr
	}
	value, found := s.values[key]
	return append([]byte(nil), value...), found, nil
}

func (s *fakeStore) Set(_ context.Context, key string, value []byte) error {
	if s.onSet != nil {
		if err := s.onSet(key, value); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setCalls++
	if s.setErr != nil {
		return s.setErr
	}
	s.values[key] = append([]byte(nil), value...)
	return nil
}

func (s *fakeStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteCalls++
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.values, key)
	return nil
}

func (s *fakeStore) onlyRecord(t *testing.T) (credentialRecord, string) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.values) != 1 {
		t.Fatalf("store has %d records, want 1", len(s.values))
	}
	for key, value := range s.values {
		var record credentialRecord
		if err := json.Unmarshal(value, &record); err != nil {
			t.Fatalf("decode stored record: %v", err)
		}
		return record, key
	}
	panic("unreachable")
}

func (s *fakeStore) put(t *testing.T, key string, record credentialRecord) {
	t.Helper()
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = data
}

func (s *fakeStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.values)
}

type capturedRequest struct {
	Method string
	Path   string
	Form   url.Values
	Header http.Header
	Body   string
}

type harness struct {
	t   *testing.T
	mu  sync.Mutex
	srv *httptest.Server

	requests []capturedRequest

	device        deviceCodeResponse
	deviceStatus  int
	tokens        []tokenResponse
	user          userResponse
	userStatus    int
	userScopes    string
	installations []installation
	installStatus int
	repos         map[int64][]repository

	override func(w http.ResponseWriter, r *http.Request) bool
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{
		t:          t,
		user:       userResponse{Login: "octocat", ID: 12345},
		userScopes: "repo",
		installations: []installation{
			{ID: 77, AppID: 42, AppSlug: "zt-test", RepositorySelection: "selected"},
		},
		repos: map[int64][]repository{
			77: {{FullName: "owner/repo"}},
		},
	}
	h.srv = httptest.NewServer(http.HandlerFunc(h.serve))
	t.Cleanup(h.srv.Close)
	h.device = deviceCodeResponse{
		DeviceCode:              "device-code-1",
		UserCode:                "ABCD-1234",
		VerificationURI:         h.srv.URL + "/login/device",
		VerificationURIComplete: h.srv.URL + "/login/device?user_code=ABCD-1234",
		ExpiresIn:               900,
		Interval:                5,
	}
	return h
}

func (h *harness) client(store Store, clock clock) *Client {
	host := strings.TrimPrefix(h.srv.URL, "http://")
	return &Client{
		httpClient:   h.srv.Client(),
		clock:        clock,
		Store:        store,
		AllowedHosts: []string{host},
		endpoints: endpoints{
			DeviceCodeURL: h.srv.URL + "/login/device/code",
			TokenURL:      h.srv.URL + "/login/oauth/access_token",
			APIBaseURL:    h.srv.URL,
		},
	}
}

func (h *harness) options() Options {
	return Options{
		Host:             strings.TrimPrefix(h.srv.URL, "http://"),
		ClientID:         "Iv1.test-client",
		Profile:          "tools",
		TargetRepository: "owner/repo",
		AppID:            42,
	}
}

func (h *harness) serve(w http.ResponseWriter, r *http.Request) {
	body := ""
	if r.Body != nil {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
	}
	form, _ := url.ParseQuery(body)
	r.Form = form
	r.PostForm = form
	h.mu.Lock()
	h.requests = append(h.requests, capturedRequest{Method: r.Method, Path: r.URL.Path, Form: form, Header: r.Header.Clone(), Body: body})
	override := h.override
	h.mu.Unlock()
	if override != nil && override(w, r) {
		return
	}

	switch {
	case r.URL.Path == "/login/device/code":
		if r.Method != http.MethodPost {
			h.t.Errorf("device code method = %s, want POST", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.mu.Lock()
		status := h.deviceStatus
		device := h.device
		h.mu.Unlock()
		if status != 0 {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"server_error","error_description":"ghu_FAIL_BODY","access_token":"ghu_FAIL_BODY","refresh_token":"ghr_FAIL_BODY"}`))
			return
		}
		writeJSON(w, http.StatusOK, device)
	case r.URL.Path == "/login/oauth/access_token":
		if r.Method != http.MethodPost {
			h.t.Errorf("token method = %s, want POST", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.mu.Lock()
		var response tokenResponse
		if len(h.tokens) > 0 {
			response = h.tokens[0]
			h.tokens = h.tokens[1:]
		} else {
			response = tokenResponse{AccessToken: "default-access-token", TokenType: "bearer", Scope: "repo", ExpiresIn: 3600}
		}
		h.mu.Unlock()
		writeJSON(w, http.StatusOK, response)
	case r.URL.Path == "/user":
		h.mu.Lock()
		status := h.userStatus
		user := h.user
		scopes := h.userScopes
		h.mu.Unlock()
		if scopes != "" {
			w.Header().Set("X-OAuth-Scopes", scopes)
		}
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		writeJSON(w, http.StatusOK, user)
	case r.URL.Path == "/user/installations":
		h.mu.Lock()
		status := h.installStatus
		installations := append([]installation(nil), h.installations...)
		h.mu.Unlock()
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		writeJSON(w, http.StatusOK, installationsPage{TotalCount: len(installations), Installations: installations})
	case strings.HasPrefix(r.URL.Path, "/user/installations/") && strings.HasSuffix(r.URL.Path, "/repositories"):
		var id int64
		middle := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/user/installations/"), "/repositories")
		if _, err := fmt.Sscanf(middle, "%d", &id); err != nil {
			h.t.Errorf("repository path id: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.mu.Lock()
		repos := append([]repository(nil), h.repos[id]...)
		h.mu.Unlock()
		writeJSON(w, http.StatusOK, repositoriesPage{TotalCount: len(repos), Repositories: repos})
	default:
		h.t.Errorf("unexpected GitHub path %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *harness) requestsFor(path string) []capturedRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	var matches []capturedRequest
	for _, request := range h.requests {
		if request.Path == path {
			matches = append(matches, request)
		}
	}
	return matches
}

func (h *harness) requestCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.requests)
}

func (h *harness) login(ctx context.Context, store Store, clock clock, options Options, prompt *DevicePrompt) (AccountSummary, error) {
	return h.client(store, clock).Login(ctx, options, func(value DevicePrompt) error {
		if prompt != nil {
			*prompt = value
		}
		return nil
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func assertNoSecret(t *testing.T, text string, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if secret != "" && strings.Contains(text, secret) {
			t.Fatalf("text contains a secret %q: %s", secret, text)
		}
	}
}

func assertNoClientSecret(t *testing.T, form url.Values) {
	t.Helper()
	for _, key := range []string{"client_secret", "access_token", "token", "private_key"} {
		if form.Has(key) {
			t.Fatalf("request unexpectedly contains %q", key)
		}
	}
}

func newRecord(t *testing.T, options resolvedOptions, access, refresh string, expires time.Time) credentialRecord {
	t.Helper()
	return credentialRecord{
		Version:        credentialVersion,
		Host:           options.host,
		ClientID:       options.clientID,
		Profile:        options.profile,
		Login:          "octocat",
		UserID:         12345,
		AccessToken:    access,
		RefreshToken:   refresh,
		TokenExpiresAt: expires.Unix(),
		VerifiedAt:     time.Now().Unix(),
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("unexpected network call")
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
