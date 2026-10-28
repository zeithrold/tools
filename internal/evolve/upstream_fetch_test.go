package evolve

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/zeithrold/tools/internal/githubauth"
)

// The adapter is exercised against a real smart HTTP server: native git
// http-backend serves fixture repositories, so the code under test speaks the
// upload-pack protocol to Git itself instead of to a hand-written double. Only
// the bytes served in the refusal cases are synthesized.

// newUpstreamFetchBare clones one fixture worktree into the layout http-backend
// serves, root/owner/name.git, and returns the project root and bare path.
func newUpstreamFetchBare(t *testing.T, work string) (root, repo string) {
	t.Helper()
	root = t.TempDir()
	repo = filepath.Join(root, "owner", "name.git")
	if err := os.MkdirAll(filepath.Dir(repo), 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "clone", "--quiet", "--bare", work, repo)
	return root, repo
}

// newUpstreamFetchServer serves one http-backend project root. It also asserts the
// request shape the adapter must produce: the canonical repository route and no
// Authorization header of its own (the credential belongs to the endpoint
// transport, which these fixtures replace with the default transport).
func newUpstreamFetchServer(t *testing.T, root string) *httptest.Server {
	t.Helper()
	backend := newUpstreamFetchCGI(t, root)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/owner/name.git/info/refs", "/owner/name.git/git-upload-pack":
		default:
			t.Errorf("upstream adapter requested unexpected path %q", r.URL.Path)
		}
		if value := r.Header.Get("Authorization"); value != "" {
			t.Errorf("upstream adapter sent an Authorization header it must not set")
		}
		backend.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

// newUpstreamFetchCGI runs native git http-backend as the CGI program it is, so
// each HTTP request becomes one real upload-pack service call.
func newUpstreamFetchCGI(t *testing.T, root string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		command := exec.Command("git", "http-backend")
		command.Env = append(os.Environ(),
			"GIT_PROJECT_ROOT="+root,
			"GIT_HTTP_EXPORT_ALL=1",
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_CONFIG_GLOBAL="+os.DevNull,
			"REQUEST_METHOD="+r.Method,
			"QUERY_STRING="+r.URL.RawQuery,
			"PATH_INFO="+r.URL.Path,
			"CONTENT_TYPE="+r.Header.Get("Content-Type"),
			"SERVER_PROTOCOL=HTTP/1.1",
		)
		if r.ContentLength >= 0 {
			command.Env = append(command.Env, "CONTENT_LENGTH="+strconv.FormatInt(r.ContentLength, 10))
		}
		command.Stdin = r.Body
		var output, detail bytes.Buffer
		command.Stdout, command.Stderr = &output, &detail
		if err := command.Run(); err != nil {
			t.Errorf("git http-backend %s %s: %v\n%s", r.Method, r.URL.Path, err, detail.String())
			http.Error(w, "fixture backend failed", http.StatusInternalServerError)
			return
		}
		header, body, ok := cutUpstreamFetchCGIHeaders(output.Bytes())
		if !ok {
			t.Errorf("git http-backend returned no CGI header terminator")
			http.Error(w, "fixture backend returned no response", http.StatusInternalServerError)
			return
		}
		status := http.StatusOK
		for _, line := range strings.Split(string(header), "\n") {
			name, value, ok := strings.Cut(strings.TrimSuffix(line, "\r"), ":")
			if !ok {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(name)) {
			case "status":
				code, _, _ := strings.Cut(strings.TrimSpace(value), " ")
				parsed, err := strconv.Atoi(code)
				if err != nil {
					t.Errorf("git http-backend returned an invalid status %q", value)
					continue
				}
				status = parsed
			case "content-type":
				w.Header().Set("Content-Type", strings.TrimSpace(value))
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
}

func cutUpstreamFetchCGIHeaders(output []byte) (header, body []byte, ok bool) {
	if header, body, ok = bytes.Cut(output, []byte("\r\n\r\n")); ok {
		return header, body, true
	}
	return bytes.Cut(output, []byte("\n\n"))
}

// upstreamFetchEndpoint derives the adapter inputs from one running test
// server. The host comes from the server URL so the loopback HTTP allowance is
// the only reason plain HTTP is accepted.
func upstreamFetchEndpoint(t *testing.T, serverURL, commit string) (Target, githubauth.GitEndpoint) {
	t.Helper()
	parsed, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	return Target{Host: parsed.Hostname(), Repository: "owner/name", BaseCommit: commit},
		githubauth.GitEndpoint{URL: serverURL + "/owner/name.git", Transport: http.DefaultTransport}
}

// fetchUpstreamStaging calls the adapter the way the cache lifecycle does: a new
// directory below a private staging directory, never an existing repository.
func fetchUpstreamStaging(t *testing.T, ctx context.Context, target Target, endpoint githubauth.GitEndpoint) (string, error) {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "repo.git")
	return directory, fetchGitBaseline(ctx, directory, target, endpoint)
}

// upstreamFetchHashObject writes one raw object into a fixture repository.
func upstreamFetchHashObject(t *testing.T, dir, kind string, data []byte) string {
	t.Helper()
	command := exec.Command("git", "--no-pager", "-C", dir, "hash-object", "-t", kind, "-w", "--stdin")
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	command.Stdin = bytes.NewReader(data)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git hash-object: %v", err)
	}
	return strings.TrimSpace(string(output))
}

// upstreamFetchGitOutput returns a fixture command's stdout without trimming,
// so raw commit bytes can be compared exactly.
func upstreamFetchGitOutput(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	command := exec.Command("git", append([]string{"--no-pager", "-C", dir,
		"-c", "user.name=zt-fixture", "-c", "user.email=zt@example.invalid",
		"-c", "commit.gpgsign=false"}, args...)...)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return output
}

// upstreamFetchRawCommit builds commit bytes with a deliberately non-UTC offset
// and, when signedLike is set, a gpgsig header carrying bytes Git must preserve
// verbatim even though they are not a usable signature.
func upstreamFetchRawCommit(tree, parent string, signedLike bool) []byte {
	lines := []string{
		"tree " + tree,
		"parent " + parent,
		"author Fixture Author <author@example.invalid> 1700000000 +0530",
		"committer Fixture Committer <committer@example.invalid> 1700000100 -0700",
	}
	if signedLike {
		lines = append(lines,
			"gpgsig -----BEGIN PGP SIGNATURE-----",
			" ",
			" iQIzBAABCAAdFiEEfixturefixturefixturefixturefixturefixture",
			" =ABCD",
			" -----END PGP SIGNATURE-----",
		)
	}
	lines = append(lines, "", "fixture raw commit")
	return []byte(strings.Join(lines, "\n") + "\n")
}

// upstreamFetchSingleObjectPack builds a valid packfile that contains one blob
// and nothing else. Served as an upload-pack result it is protocol-correct but
// does not contain the advertised commit, which is the incomplete-data case.
func upstreamFetchSingleObjectPack(t *testing.T) []byte {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet", "--bare")
	sha := upstreamFetchHashObject(t, dir, "blob", []byte("unrelated fixture content\n"))
	command := exec.Command("git", "--no-pager", "-C", dir, "pack-objects", "--stdout")
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	command.Stdin = strings.NewReader(sha + "\n")
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git pack-objects: %v", err)
	}
	if !bytes.HasPrefix(output, []byte("PACK")) {
		t.Fatalf("fixture pack is not a packfile")
	}
	return output
}

// servedFetchUpstream is one fixture repository reachable over real smart HTTP.
type servedFetchUpstream struct {
	t      *testing.T
	root   string
	repo   string
	server *httptest.Server
}

func newServedFetchUpstream(t *testing.T, work string) *servedFetchUpstream {
	t.Helper()
	root, repo := newUpstreamFetchBare(t, work)
	return &servedFetchUpstream{t: t, root: root, repo: repo, server: newUpstreamFetchServer(t, root)}
}

func (s *servedFetchUpstream) fetch(ctx context.Context, commit string) (string, error) {
	s.t.Helper()
	target, endpoint := upstreamFetchEndpoint(s.t, s.server.URL, commit)
	return fetchUpstreamStaging(s.t, ctx, target, endpoint)
}

// TestFetchGitBaselinePreservesRawCommits fetches two commits whose bytes must
// survive exactly: one unsigned commit with non-UTC author and committer
// offsets, and one with a signed-like gpgsig block. It also checks that the
// fetched repository is native-readable, complete for its other files, and
// shallow at exactly the pinned commit.
func TestFetchGitBaselinePreservesRawCommits(t *testing.T) {
	work, base := newUpstream(t, map[string]string{
		"skills/fixture-skill/SKILL.md": fixtureSkillMD,
		"README.md":                     "a repository that is not only Skills\n",
		"docs/unrelated.md":             "# Unrelated\n\nFetched but never checked out.\n",
		"scripts/tool.sh":               "#!/bin/sh\necho fixture\n",
	})
	upstream := newServedFetchUpstream(t, work)
	tree := runGit(t, upstream.repo, "rev-parse", base+"^{tree}")

	cases := []struct {
		name   string
		signed bool
	}{
		{name: "unsigned-nonutc"},
		{name: "signed-like", signed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			object := upstreamFetchRawCommit(tree, base, tc.signed)
			commit := upstreamFetchHashObject(t, upstream.repo, "commit", object)
			runGit(t, upstream.repo, "update-ref", "refs/heads/"+tc.name, commit)

			directory, err := upstream.fetch(context.Background(), commit)
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			if got := upstreamFetchGitOutput(t, directory, "cat-file", "commit", commit); !bytes.Equal(got, object) {
				t.Fatalf("fetched commit bytes changed:\n got %q\nwant %q", got, object)
			}
			if got := runGit(t, directory, "rev-parse", "refs/zt/upstream"); got != commit {
				t.Fatalf("pinned ref = %s, want %s", got, commit)
			}
			shallow, err := os.ReadFile(filepath.Join(directory, "shallow"))
			if err != nil {
				t.Fatalf("read shallow: %v", err)
			}
			if strings.TrimSpace(string(shallow)) != commit {
				t.Fatalf("shallow = %q, want only %s", shallow, commit)
			}
			if got := runGit(t, directory, "rev-list", "--count", commit); got != "1" {
				t.Fatalf("fetched history has %s commits, want depth one", got)
			}
			// The whole tree, not only the Skill directory, must be present.
			if got, want := runGit(t, directory, "ls-tree", "-r", commit), runGit(t, upstream.repo, "ls-tree", "-r", commit); got != want {
				t.Fatalf("fetched tree differs:\n got %s\nwant %s", got, want)
			}
			if got := runGit(t, directory, "cat-file", "blob", commit+":docs/unrelated.md"); got != "# Unrelated\n\nFetched but never checked out." {
				t.Fatalf("unrelated file content = %q", got)
			}
			// Native integrity, the same read-only check the cache uses.
			runGit(t, directory, "fsck", "--strict", "--no-reflogs", "--no-dangling", commit)
		})
	}
}

// TestFetchGitBaselineRequiresAdvertisedOrAllowedCommit proves the want rule: a
// parent commit that is not advertised is refused before any pack request, and
// the same commit is accepted once the server advertises that it accepts a
// reachable unadvertised want.
func TestFetchGitBaselineRequiresAdvertisedOrAllowedCommit(t *testing.T) {
	work, base := newUpstream(t, baseFileSet())
	writeFile(t, filepath.Join(work, "README.md"), "a later commit\n")
	runGit(t, work, "add", "--all")
	runGit(t, work, "commit", "--quiet", "-m", "later")
	if later := runGit(t, work, "rev-parse", "HEAD"); later == base {
		t.Fatal("fixture did not advance")
	}
	root, _ := newUpstreamFetchBare(t, work)

	var posts atomic.Int64
	backend := newUpstreamFetchCGI(t, root)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
		}
		backend.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	target, endpoint := upstreamFetchEndpoint(t, server.URL, base)
	directory, err := fetchUpstreamStaging(t, context.Background(), target, endpoint)
	if err == nil || !strings.Contains(err.Error(), "does not advertise") {
		t.Fatalf("unadvertised commit: err = %v", err)
	}
	if posts.Load() != 0 {
		t.Fatalf("adapter sent %d pack requests for a commit the server did not offer", posts.Load())
	}
	if _, statErr := os.Stat(filepath.Join(directory, "refs", "zt", "upstream")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("refused fetch recorded a pinned ref: %v", statErr)
	}

	// The server now advertises the capability that makes the parent fetchable.
	runGit(t, filepath.Join(root, "owner", "name.git"), "config", "uploadpack.allowReachableSHA1InWant", "true")
	directory, err = fetchUpstreamStaging(t, context.Background(), target, endpoint)
	if err != nil {
		t.Fatalf("reachable want: %v", err)
	}
	if got := runGit(t, directory, "rev-parse", "refs/zt/upstream"); got != base {
		t.Fatalf("pinned ref = %s, want %s", got, base)
	}
	if posts.Load() == 0 {
		t.Fatal("adapter never requested the pack it accepted")
	}
}

// TestFetchGitBaselineRefusesBadTransfer covers the transport and payload
// refusals: a not-found repository, a redirect, a truncated pack, cancellation
// and a protocol-valid but incomplete response.
func TestFetchGitBaselineRefusesBadTransfer(t *testing.T) {
	// A second commit is the target, so the depth-one fetch has a real parent to
	// hide and the shallow-boundary refusal below is meaningful.
	work, _ := newUpstream(t, baseFileSet())
	writeFile(t, filepath.Join(work, "README.md"), "a parented commit\n")
	runGit(t, work, "add", "--all")
	runGit(t, work, "commit", "--quiet", "-m", "parented")
	commit := runGit(t, work, "rev-parse", "HEAD")
	root, _ := newUpstreamFetchBare(t, work)

	t.Run("not-found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "missing", http.StatusNotFound)
		}))
		t.Cleanup(server.Close)
		target, endpoint := upstreamFetchEndpoint(t, server.URL, commit)
		_, err := fetchUpstreamStaging(t, context.Background(), target, endpoint)
		if err == nil || !strings.Contains(err.Error(), "not found or is not readable") {
			t.Fatalf("404: err = %v", err)
		}
		if strings.Contains(err.Error(), server.URL) || strings.Contains(err.Error(), "missing") {
			t.Fatalf("error repeats endpoint or response content: %v", err)
		}
	})

	t.Run("server-error-payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "payload-secret", http.StatusInternalServerError)
		}))
		t.Cleanup(server.Close)
		target, endpoint := upstreamFetchEndpoint(t, server.URL, commit)
		_, err := fetchUpstreamStaging(t, context.Background(), target, endpoint)
		if err == nil || !strings.Contains(err.Error(), "transport message is not repeated") {
			t.Fatalf("server error: err = %v", err)
		}
		if strings.Contains(err.Error(), "payload-secret") || strings.Contains(err.Error(), server.URL) {
			t.Fatalf("error repeats endpoint or response content: %v", err)
		}
	})

	t.Run("redirect", func(t *testing.T) {
		var hops atomic.Int64
		elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hops.Add(1)
			http.Error(w, "should not be reached", http.StatusInternalServerError)
		}))
		t.Cleanup(elsewhere.Close)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, elsewhere.URL+r.URL.RequestURI(), http.StatusFound)
		}))
		t.Cleanup(server.Close)
		target, endpoint := upstreamFetchEndpoint(t, server.URL, commit)
		_, err := fetchUpstreamStaging(t, context.Background(), target, endpoint)
		if !errors.Is(err, errUpstreamRedirect) {
			t.Fatalf("redirect: err = %v", err)
		}
		if hops.Load() != 0 {
			t.Fatalf("adapter followed a redirect %d time(s)", hops.Load())
		}
	})

	t.Run("truncated-pack", func(t *testing.T) {
		backend := newUpstreamFetchCGI(t, root)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				backend.ServeHTTP(w, r)
				return
			}
			recorder := httptest.NewRecorder()
			backend.ServeHTTP(recorder, r)
			body := recorder.Body.Bytes()
			if recorder.Code != http.StatusOK || len(body) < 2 {
				t.Errorf("fixture backend returned %d with %d bytes", recorder.Code, len(body))
				http.Error(w, "fixture backend failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", recorder.Header().Get("Content-Type"))
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body[:len(body)/2])
		}))
		t.Cleanup(server.Close)
		target, endpoint := upstreamFetchEndpoint(t, server.URL, commit)
		directory, err := fetchUpstreamStaging(t, context.Background(), target, endpoint)
		if err == nil {
			t.Fatal("truncated pack was accepted")
		}
		if _, statErr := os.Stat(filepath.Join(directory, "shallow")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("truncated fetch recorded a shallow boundary")
		}
	})

	t.Run("missing-shallow-boundary", func(t *testing.T) {
		backend := newUpstreamFetchCGI(t, root)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				backend.ServeHTTP(w, r)
				return
			}
			recorder := httptest.NewRecorder()
			backend.ServeHTTP(recorder, r)
			body := recorder.Body.Bytes()
			// The depth-one pack keeps the parent objects out. Serving it without
			// the shallow boundary must be refused rather than accepted as complete.
			marker := []byte("0008NAK\n")
			index := bytes.Index(body, marker)
			if recorder.Code != http.StatusOK || index < 0 {
				t.Errorf("fixture backend returned %d without a pack", recorder.Code)
				http.Error(w, "fixture backend failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", recorder.Header().Get("Content-Type"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("0000")) // empty shallow-update section
			_, _ = w.Write(body[index:])
		}))
		t.Cleanup(server.Close)
		target, endpoint := upstreamFetchEndpoint(t, server.URL, commit)
		_, err := fetchUpstreamStaging(t, context.Background(), target, endpoint)
		if err == nil || !strings.Contains(err.Error(), "did not mark the pinned commit") {
			t.Fatalf("missing shallow boundary: err = %v", err)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		backend := newUpstreamFetchCGI(t, root)
		started := make(chan struct{})
		release := make(chan struct{})
		var once, releaseOnce sync.Once
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				backend.ServeHTTP(w, r)
				return
			}
			once.Do(func() { close(started) })
			select {
			case <-r.Context().Done():
			case <-release:
			}
		}))
		t.Cleanup(server.Close)
		// LIFO: the release runs before server.Close, so a connection the server
		// still considers active cannot make the fixture hang.
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		target, endpoint := upstreamFetchEndpoint(t, server.URL, commit)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			select {
			case <-started:
				cancel()
			case <-release:
			}
		}()
		_, err := fetchUpstreamStaging(t, ctx, target, endpoint)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel: err = %v", err)
		}
	})

	t.Run("incomplete-data", func(t *testing.T) {
		backend := newUpstreamFetchCGI(t, root)
		pack := upstreamFetchSingleObjectPack(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				backend.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("0000"))      // empty shallow-update section
			_, _ = w.Write([]byte("0008NAK\n")) // pack follows
			_, _ = w.Write(pack)
		}))
		t.Cleanup(server.Close)
		target, endpoint := upstreamFetchEndpoint(t, server.URL, commit)
		directory, err := fetchUpstreamStaging(t, context.Background(), target, endpoint)
		if err == nil || !strings.Contains(err.Error(), "did not deliver the pinned commit") {
			t.Fatalf("incomplete data: err = %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(directory, "refs", "zt", "upstream")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("incomplete fetch recorded a pinned ref: %v", statErr)
		}
	})
}

// TestFetchGitBaselineValidatesEndpoint covers the canonical-URL rule without
// any server: a differently shaped endpoint is refused before a connection.
func TestFetchGitBaselineValidatesEndpoint(t *testing.T) {
	commit := strings.Repeat("a", 40)
	cases := []struct {
		name     string
		target   Target
		endpoint githubauth.GitEndpoint
		want     string
	}{
		{
			name:     "no-transport",
			target:   Target{Host: "github.com", Repository: "owner/name", BaseCommit: commit},
			endpoint: githubauth.GitEndpoint{URL: "https://github.com/owner/name.git"},
			want:     "authenticated HTTP transport",
		},
		{
			name:     "plain-http-for-real-host",
			target:   Target{Host: "github.com", Repository: "owner/name", BaseCommit: commit},
			endpoint: githubauth.GitEndpoint{URL: "http://github.com/owner/name.git", Transport: http.DefaultTransport},
			want:     "must use HTTPS",
		},
		{
			name:     "other-repository",
			target:   Target{Host: "127.0.0.1", Repository: "owner/name", BaseCommit: commit},
			endpoint: githubauth.GitEndpoint{URL: "http://127.0.0.1:1/other/name.git", Transport: http.DefaultTransport},
			want:     "does not name the policy target repository",
		},
		{
			name:     "other-host",
			target:   Target{Host: "127.0.0.1", Repository: "owner/name", BaseCommit: commit},
			endpoint: githubauth.GitEndpoint{URL: "https://github.com/owner/name.git", Transport: http.DefaultTransport},
			want:     "host is not the policy target host",
		},
		{
			name:     "credentials-in-url",
			target:   Target{Host: "github.com", Repository: "owner/name", BaseCommit: commit},
			endpoint: githubauth.GitEndpoint{URL: "https://x-access-token:secret@github.com/owner/name.git", Transport: http.DefaultTransport},
			want:     "without credentials",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fetchUpstreamStaging(t, context.Background(), tc.target, tc.endpoint)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}

	t.Run("existing-directory", func(t *testing.T) {
		target, endpoint := upstreamFetchEndpoint(t, "http://127.0.0.1:1", commit)
		directory := t.TempDir()
		if err := fetchGitBaseline(context.Background(), directory, target, endpoint); err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("existing directory: err = %v", err)
		}
	})
}

// TestBoundedUpstreamPack checks the encoded-transfer bound itself: an exactly
// full read is accepted and one extra byte is refused.
func TestBoundedUpstreamPack(t *testing.T) {
	payload := bytes.Repeat([]byte{7}, 8)
	exact := &boundedUpstreamPack{reader: bytes.NewReader(payload[:4]), limit: 4}
	if data, err := io.ReadAll(exact); err != nil || !bytes.Equal(data, payload[:4]) {
		t.Fatalf("exact bound: %q, %v", data, err)
	}
	over := &boundedUpstreamPack{reader: bytes.NewReader(payload), limit: 4}
	if _, err := io.ReadAll(over); !errors.Is(err, errUpstreamPackBound) {
		t.Fatalf("over bound: err = %v", err)
	}
}

// TestFetchUpstreamCachesServedBaseline is the end-to-end cache path: acquire a
// pinned baseline from a served fixture, verify it natively, then resolve it
// from the owned cache for a read-only plan that names no explicit upstream.
func TestFetchUpstreamCachesServedBaseline(t *testing.T) {
	work, commit := newUpstream(t, baseFileSet())
	upstream := newServedFetchUpstream(t, work)
	_, endpoint := upstreamFetchEndpoint(t, upstream.server.URL, commit)

	workspace := t.TempDir()
	inputs := t.TempDir()
	policy := filepath.Join(inputs, "policy.json")
	writeJSONFile(t, policy, map[string]any{
		"schemaVersion":  1,
		"host":           "127.0.0.1",
		"repository":     "owner/name",
		"baseCommit":     commit,
		"allowedSkills":  []string{fixtureSkill},
		"requiredChecks": []string{"go-test"},
	})

	result, err := FetchUpstream(context.Background(), FetchUpstreamOptions{
		Policy:       policy,
		Root:         workspace,
		AllowNetwork: true,
		OpenGit: func(context.Context, Target) (githubauth.GitEndpoint, error) {
			return endpoint, nil
		},
	})
	if err != nil {
		t.Fatalf("fetch upstream: %v", err)
	}
	if result.Status != "ready" || !result.Created {
		t.Fatalf("fetch result = %+v", result)
	}

	ctx := context.Background()
	repo, err := OpenRepo(ctx, result.Upstream)
	if err != nil {
		t.Fatalf("open published baseline: %v", err)
	}
	if err := verifyUpstream(ctx, repo, commit); err != nil {
		t.Fatalf("verify published baseline: %v", err)
	}
	status, err := UpstreamStatus(ctx, policy, workspace)
	if err != nil {
		t.Fatalf("upstream status: %v", err)
	}
	if status.Status != "ready" {
		t.Fatalf("upstream status = %+v", status)
	}

	proposal := filepath.Join(inputs, "proposal.json")
	writeJSONFile(t, proposal, proposalDoc(commit, defaultProposalFiles()...))
	prepared, err := Prepare(ctx, PrepareOptions{Proposal: proposal, Policy: policy, Root: workspace})
	if err != nil {
		t.Fatalf("prepare from cache: %v", err)
	}
	candidate := filepath.Join(workspace, filepath.FromSlash(prepared.CandidatePath))
	planned, err := Plan(ctx, PlanOptions{Candidate: candidate, Policy: policy, Root: workspace})
	if err != nil {
		t.Fatalf("plan from cache: %v", err)
	}
	if planned.CandidateID != prepared.CandidateID || planned.Target.BaseCommit != commit {
		t.Fatalf("plan result = %+v", planned)
	}
}
