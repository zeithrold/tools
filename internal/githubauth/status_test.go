package githubauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestStatusLocalDoesNotUseNetwork(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	client := h.client(store, clock)
	options := h.options()
	resolved, err := client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	record := newRecord(t, resolved, "at-local", "rt-local", clock.Now().Add(time.Hour))
	record.Scopes = []string{"repo", "offline_access"}
	setRecord(t, store, resolved.key, record)
	client.httpClient = &http.Client{Transport: failingTransport{}}

	summary, err := client.Status(context.Background(), options, false)
	if err != nil {
		t.Fatalf("Status local: %v", err)
	}
	if summary.Probed || summary.Login != "octocat" || summary.Host != options.Host || summary.Profile != "tools" {
		t.Fatalf("summary = %+v", summary)
	}
	if !summary.CanRefresh || summary.InstallationID != 0 {
		t.Fatalf("summary refresh/installation = %+v", summary)
	}
	if len(h.requests) != 0 {
		t.Fatalf("local status made %d requests", len(h.requests))
	}
}

func TestStatusNotLoggedInAndIsolation(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	client := h.client(store, clock)
	options := h.options()
	if _, err := client.Status(context.Background(), options, false); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("empty status error = %v, want ErrNotLoggedIn", err)
	}

	resolved, err := client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	record := newRecord(t, resolved, "at", "rt", clock.Now().Add(time.Hour))
	setRecord(t, store, resolved.key, record)

	other := options
	other.ClientID = "Iv1.other-client"
	if _, err := client.Status(context.Background(), other, false); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("other client status error = %v, want ErrNotLoggedIn", err)
	}
	other = options
	other.Profile = "other-profile"
	if _, err := client.Status(context.Background(), other, false); !errors.Is(err, ErrNotLoggedIn) {
		t.Fatalf("other profile status error = %v, want ErrNotLoggedIn", err)
	}

	otherResolved, err := client.resolve(other)
	if err != nil {
		t.Fatal(err)
	}
	mismatched := record
	mismatched.Profile = "mismatch"
	setRecord(t, store, otherResolved.key, mismatched)
	if _, err := client.Status(context.Background(), other, false); !errors.Is(err, ErrCredentialMismatch) {
		t.Fatalf("mismatched record error = %v, want ErrCredentialMismatch", err)
	}
}

func TestStatusRejectsDifferentAccountOnProbe(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	client := h.client(store, clock)
	options := h.options()
	resolved, err := client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	record := newRecord(t, resolved, "at", "rt", clock.Now().Add(time.Hour))
	record.UserID = 99999
	setRecord(t, store, resolved.key, record)
	if _, err := client.Status(context.Background(), options, true); !errors.Is(err, ErrAccountMismatch) {
		t.Fatalf("probe error = %v, want ErrAccountMismatch", err)
	}
}

func TestStatusReportsRenamedLoginForSameUserID(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	client := h.client(store, clock)
	options := h.options()
	resolved, err := client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	record := newRecord(t, resolved, "at", "rt", clock.Now().Add(time.Hour))
	record.Login = "old-login"
	setRecord(t, store, resolved.key, record)
	h.user = userResponse{Login: "new-login", ID: 12345}
	summary, err := client.Status(context.Background(), options, true)
	if err != nil {
		t.Fatalf("Status probe: %v", err)
	}
	if summary.Login != "new-login" || summary.UserID != 12345 {
		t.Fatalf("summary after rename = %+v", summary)
	}
}

func TestLogoutIsIdempotentAndScoped(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	client := h.client(store, clock)
	options := h.options()
	if err := client.Logout(context.Background(), options); err != nil {
		t.Fatalf("Logout with no record: %v", err)
	}
	resolved, err := client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	other := options
	other.Profile = "other-profile"
	otherResolved, err := client.resolve(other)
	if err != nil {
		t.Fatal(err)
	}
	setRecord(t, store, resolved.key, newRecord(t, resolved, "at", "rt", clock.Now().Add(time.Hour)))
	setRecord(t, store, otherResolved.key, newRecord(t, otherResolved, "at-other", "rt-other", clock.Now().Add(time.Hour)))
	if err := client.Logout(context.Background(), options); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if store.count() != 1 {
		t.Fatalf("remaining records = %d, want the other profile", store.count())
	}
	if err := client.Logout(context.Background(), options); err != nil {
		t.Fatalf("second Logout: %v", err)
	}
	if len(h.requests) != 0 {
		t.Fatalf("Logout made %d network requests", len(h.requests))
	}
}

func TestVerifyTargetPaginationAndAppFilter(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	client := h.client(store, clock)
	options := h.options()
	options.TargetRepository = "owner/repo"
	options.AppID = 42
	resolved, err := client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	h.repos[88] = []repository{{FullName: "other/repo"}}
	h.repos[77] = []repository{{FullName: "owner/repo"}}
	h.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/user/installations" {
			return false
		}
		if r.URL.Query().Get("page") == "2" {
			writeJSON(w, http.StatusOK, installationsPage{TotalCount: 2, Installations: []installation{{ID: 77, AppID: 42}}})
			return true
		}
		w.Header().Set("Link", "<"+h.srv.URL+"/user/installations?page=2>; rel=\"next\"")
		writeJSON(w, http.StatusOK, installationsPage{TotalCount: 2, Installations: []installation{{ID: 88, AppID: 42}}})
		return true
	}
	target, err := client.verifyTarget(context.Background(), resolved, "at")
	if err != nil {
		t.Fatalf("verifyTarget: %v", err)
	}
	if target.installationID != 77 || target.repository != "owner/repo" {
		t.Fatalf("target = %+v", target)
	}
	if len(h.requestsFor("/user/installations")) != 2 {
		t.Fatalf("installation pages = %d, want 2", len(h.requestsFor("/user/installations")))
	}

	options.AppID = 99
	resolved, err = client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.verifyTarget(context.Background(), resolved, "at"); !errors.Is(err, ErrScopeDenied) {
		t.Fatalf("wrong App error = %v, want ErrScopeDenied", err)
	}
}

func TestRecordsDifferUsesStableUserID(t *testing.T) {
	base := credentialRecord{AccessToken: "at", RefreshToken: "rt", UserID: 12345, Login: "old-login"}
	renamed := base
	renamed.Login = "new-login"
	if recordsDiffer(base, renamed) {
		t.Fatal("login rename with the same user ID must not count as a concurrent credential change")
	}
	different := base
	different.UserID = 99999
	if !recordsDiffer(base, different) {
		t.Fatal("different user ID must count as a concurrent credential change")
	}
}

func TestRefreshRotatesBothTokensAndOmitsClientSecret(t *testing.T) {

	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	h.userScopes = "repo, offline_access"
	options := h.options()
	options.RequestOfflineAccess = true
	h.tokens = []tokenResponse{{AccessToken: "at-old", RefreshToken: "rt-old", ExpiresIn: 1, RefreshTokenExpiresIn: 100, Scope: "offline_access"}}
	if _, err := h.login(context.Background(), store, clock, options, nil); err != nil {
		t.Fatalf("login: %v", err)
	}
	clock.advance(2 * time.Second)
	h.tokens = []tokenResponse{{AccessToken: "at-new", RefreshToken: "rt-new", ExpiresIn: 3600, RefreshTokenExpiresIn: 7200, Scope: "offline_access"}}

	client := h.client(store, clock)
	summary, err := client.Status(context.Background(), options, true)
	if err != nil {
		t.Fatalf("Status probe: %v", err)
	}
	if !summary.Probed || summary.Login != "octocat" {
		t.Fatalf("summary = %+v", summary)
	}
	record, _ := store.onlyRecord(t)
	if record.AccessToken != "at-new" || record.RefreshToken != "rt-new" {
		t.Fatalf("rotated pair = %q/%q", record.AccessToken, record.RefreshToken)
	}
	requests := h.requestsFor("/login/oauth/access_token")
	if len(requests) != 2 {
		t.Fatalf("token requests = %d, want login + refresh", len(requests))
	}
	refresh := requests[1].Form
	if refresh.Get("grant_type") != "refresh_token" || refresh.Get("refresh_token") != "rt-old" || refresh.Get("client_id") != options.ClientID {
		t.Fatalf("refresh form = %v", refresh)
	}
	assertNoClientSecret(t, refresh)
	assertNoSecret(t, errorText(err), "at-new", "rt-new", "at-old", "rt-old")
}

func TestRefreshRevokedDeletesStaleRecord(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	h.userScopes = "offline_access"
	options := h.options()
	options.RequestOfflineAccess = true
	h.tokens = []tokenResponse{{AccessToken: "at-old", RefreshToken: "rt-old", ExpiresIn: 1, RefreshTokenExpiresIn: 100}}
	if _, err := h.login(context.Background(), store, clock, options, nil); err != nil {
		t.Fatalf("login: %v", err)
	}
	clock.advance(2 * time.Second)
	h.tokens = []tokenResponse{{Error: "bad_refresh_token"}}
	client := h.client(store, clock)
	_, err := client.Status(context.Background(), options, true)
	if !errors.Is(err, ErrReloginRequired) {
		t.Fatalf("error = %v, want ErrReloginRequired", err)
	}
	if store.count() != 0 {
		t.Fatal("stale record was not removed")
	}
	assertNoSecret(t, errorText(err), "at-old", "rt-old")
}

func TestRefreshUnknownErrorIsNotEchoed(t *testing.T) {
	const secret = "ghr_review_fake_secret"
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	options := h.options()
	options.RequestOfflineAccess = true
	h.tokens = []tokenResponse{{AccessToken: "at-old", RefreshToken: "rt-old", ExpiresIn: 1, RefreshTokenExpiresIn: 100}}
	if _, err := h.login(context.Background(), store, clock, options, nil); err != nil {
		t.Fatalf("login: %v", err)
	}
	clock.advance(2 * time.Second)
	h.tokens = []tokenResponse{{Error: secret}}
	client := h.client(store, clock)
	_, err := client.Status(context.Background(), options, true)
	if !errors.Is(err, ErrProtocol) {
		t.Fatalf("error = %v, want ErrProtocol", err)
	}
	assertNoSecret(t, errorText(err), secret)
	if store.count() != 1 {
		t.Fatal("unknown refresh error should keep the stored record")
	}
}

func TestRefreshTransientFailureKeepsRecord(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	options := h.options()
	options.RequestOfflineAccess = true
	h.tokens = []tokenResponse{{AccessToken: "at-old", RefreshToken: "rt-old", ExpiresIn: 1, RefreshTokenExpiresIn: 100}}
	if _, err := h.login(context.Background(), store, clock, options, nil); err != nil {
		t.Fatalf("login: %v", err)
	}
	clock.advance(2 * time.Second)
	h.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/login/oauth/access_token" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"access_token":"MUST-NOT-LEAK"}`))
			return true
		}
		return false
	}
	client := h.client(store, clock)
	_, err := client.Status(context.Background(), options, true)
	if !errors.Is(err, ErrRemote) {
		t.Fatalf("error = %v, want ErrRemote", err)
	}
	assertNoSecret(t, errorText(err), "MUST-NOT-LEAK", "at-old", "rt-old")
	record, _ := store.onlyRecord(t)
	if record.AccessToken != "at-old" || record.RefreshToken != "rt-old" {
		t.Fatalf("record changed after transient failure: %+v", record)
	}
}

func TestRefreshWithoutTokenRequiresLogin(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	client := h.client(store, clock)
	options := h.options()
	resolved, err := client.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	record := newRecord(t, resolved, "at-old", "", clock.Now().Add(-time.Minute))
	setRecord(t, store, resolved.key, record)
	if _, err := client.Status(context.Background(), options, true); !errors.Is(err, ErrReloginRequired) {
		t.Fatalf("error = %v, want ErrReloginRequired", err)
	}
	if store.count() != 1 {
		t.Fatal("record without a refresh token should not be silently deleted")
	}
}

func TestRefreshDoesNotClobberAnotherProcessRecord(t *testing.T) {
	t.Run("revoked refresh sees newer record", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		clock := newFakeClock()
		client := h.client(store, clock)
		options := h.options()
		resolved, err := client.resolve(options)
		if err != nil {
			t.Fatal(err)
		}
		old := newRecord(t, resolved, "at-old", "rt-old", clock.Now().Add(-time.Minute))
		setRecord(t, store, resolved.key, old)
		h.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.URL.Path == "/login/oauth/access_token" && r.Form.Get("grant_type") == "refresh_token" {
				newer := old
				newer.AccessToken = "at-other"
				newer.RefreshToken = "rt-other"
				newer.TokenExpiresAt = clock.Now().Add(time.Hour).Unix()
				putRecord(store, resolved.key, newer)
				writeJSON(w, http.StatusOK, tokenResponse{Error: "bad_refresh_token"})
				return true
			}
			return false
		}
		_, err = client.refresh(context.Background(), resolved, old)
		if !errors.Is(err, ErrConcurrentRefresh) {
			t.Fatalf("refresh error = %v, want ErrConcurrentRefresh", err)
		}
		record, _ := store.onlyRecord(t)
		if record.AccessToken != "at-other" || record.RefreshToken != "rt-other" {
			t.Fatalf("newer record was clobbered: %+v", record)
		}
		if store.deleteCalls != 0 {
			t.Fatalf("delete calls = %d, want 0", store.deleteCalls)
		}
	})

	t.Run("successful refresh sees newer record", func(t *testing.T) {
		h := newHarness(t)
		store := newFakeStore()
		clock := newFakeClock()
		client := h.client(store, clock)
		options := h.options()
		resolved, err := client.resolve(options)
		if err != nil {
			t.Fatal(err)
		}
		old := newRecord(t, resolved, "at-old", "rt-old", clock.Now().Add(-time.Minute))
		setRecord(t, store, resolved.key, old)
		h.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.URL.Path == "/login/oauth/access_token" && r.Form.Get("grant_type") == "refresh_token" {
				newer := old
				newer.AccessToken = "at-other"
				newer.RefreshToken = "rt-other"
				newer.TokenExpiresAt = clock.Now().Add(time.Hour).Unix()
				putRecord(store, resolved.key, newer)
				writeJSON(w, http.StatusOK, tokenResponse{AccessToken: "at-ours", RefreshToken: "rt-ours", ExpiresIn: 3600})
				return true
			}
			return false
		}
		_, err = client.refresh(context.Background(), resolved, old)
		if !errors.Is(err, ErrConcurrentRefresh) {
			t.Fatalf("refresh error = %v, want ErrConcurrentRefresh", err)
		}
		record, _ := store.onlyRecord(t)
		if record.AccessToken != "at-other" || record.RefreshToken != "rt-other" {
			t.Fatalf("newer record was overwritten: %+v", record)
		}
	})
}

func TestRefreshWithLockerSerializesAcrossClients(t *testing.T) {
	h := newHarness(t)
	store := newFakeStore()
	clock := newFakeClock()
	options := h.options()
	clientA := h.client(store, clock)
	clientB := h.client(store, clock)
	locker := newChannelLocker()
	clientA.Locker = locker
	clientB.Locker = locker
	resolved, err := clientA.resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	old := newRecord(t, resolved, "at-old", "rt-old", clock.Now().Add(-time.Minute))
	setRecord(t, store, resolved.key, old)

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	h.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/login/oauth/access_token" && r.Form.Get("grant_type") == "refresh_token" {
			once.Do(func() { close(started) })
			<-release
			writeJSON(w, http.StatusOK, tokenResponse{AccessToken: "at-new", RefreshToken: "rt-new", ExpiresIn: 3600, RefreshTokenExpiresIn: 7200})
			return true
		}
		return false
	}

	var wg sync.WaitGroup
	errorsSeen := make(chan error, 2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := clientA.refresh(context.Background(), resolved, old)
		errorsSeen <- err
	}()
	<-started
	<-locker.entered // drain A's attempt
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := clientB.refresh(context.Background(), resolved, old)
		errorsSeen <- err
	}()
	<-locker.entered // B has reached the lock and is waiting
	close(release)
	wg.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatalf("refresh error: %v", err)
		}
	}
	record, _ := store.onlyRecord(t)
	if record.AccessToken != "at-new" || record.RefreshToken != "rt-new" {
		t.Fatalf("record = %+v", record)
	}
	if len(h.requestsFor("/login/oauth/access_token")) != 1 {
		t.Fatalf("token requests = %d, want 1", len(h.requestsFor("/login/oauth/access_token")))
	}
	locker.mu.Lock()
	attempts := locker.count
	locker.mu.Unlock()
	if attempts != 2 {
		t.Fatalf("lock attempts = %d, want 2", attempts)
	}
}

type channelLocker struct {
	sem     chan struct{}
	entered chan struct{}
	mu      sync.Mutex
	count   int
}

func newChannelLocker() *channelLocker {
	locker := &channelLocker{sem: make(chan struct{}, 1), entered: make(chan struct{}, 4)}
	locker.sem <- struct{}{}
	return locker
}

func (l *channelLocker) Lock(ctx context.Context, _ string) (Unlocker, error) {
	l.mu.Lock()
	l.count++
	l.mu.Unlock()
	select {
	case l.entered <- struct{}{}:
	default:
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.sem:
		return &channelUnlock{locker: l}, nil
	}
}

type channelUnlock struct{ locker *channelLocker }

func (u *channelUnlock) Unlock() error {
	u.locker.sem <- struct{}{}
	return nil
}

func setRecord(t *testing.T, store *fakeStore, key string, record credentialRecord) {
	t.Helper()
	if _, err := json.Marshal(record); err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	putRecord(store, key, record)
}

func putRecord(store *fakeStore, key string, record credentialRecord) {
	data, _ := json.Marshal(record)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.values[key] = data
}
