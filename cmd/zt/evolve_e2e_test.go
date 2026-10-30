package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	syncpkg "sync"
	"testing"
	"time"

	"github.com/zeithrold/tools/internal/githubauth"
)

// This file drives the whole reviewed loop through the real CLI commands against an
// in-process HTTP GitHub stand-in. The stand-in is faithful where the client compares
// values: blob ids come from real Git plumbing, and base listings come from the local
// upstream. Values the client only re-reads (tree and commit ids) are stable hashes.
const cliRemoteToken = "cli-test-token-not-a-credential"

const cliClientID = "Iv1.clitestonly"

type cliTreeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

type cliCommit struct {
	SHA       string            `json:"sha"`
	Message   string            `json:"message"`
	Tree      cliTreeEntry      `json:"tree"`
	Parents   []cliTreeEntry    `json:"parents"`
	Author    map[string]string `json:"author"`
	Committer map[string]string `json:"committer"`
}

type cliPullRef struct {
	Ref  string `json:"ref"`
	SHA  string `json:"sha"`
	Repo struct {
		FullName string `json:"full_name"`
	} `json:"repo"`
}

type cliPull struct {
	Number  int        `json:"number"`
	Title   string     `json:"title"`
	Body    string     `json:"body"`
	State   string     `json:"state"`
	Draft   bool       `json:"draft"`
	HTMLURL string     `json:"html_url"`
	Head    cliPullRef `json:"head"`
	Base    cliPullRef `json:"base"`
}

// cliRemote is the GitHub stand-in. It holds one repository, its refs, the object
// ids it served, and the pulls it was asked to create.
type cliRemote struct {
	server *httptest.Server

	mu         syncpkg.Mutex
	upstream   string
	fullName   string
	baseCommit string
	baseTree   string
	base       map[string]cliTreeEntry
	trees      map[string]map[string]cliTreeEntry
	commits    map[string]cliCommit
	refs       map[string]string
	pulls      []cliPull
	writes     int
	authorized int
}

func newCLIRemote(t *testing.T, upstream, commit string) *cliRemote {
	t.Helper()
	remote := &cliRemote{
		upstream:   upstream,
		fullName:   "example/fixture",
		baseCommit: commit,
		baseTree:   fixtureGit(t, upstream, "rev-parse", commit+"^{tree}"),
		base:       map[string]cliTreeEntry{},
		trees:      map[string]map[string]cliTreeEntry{},
		commits:    map[string]cliCommit{},
		refs:       map[string]string{"refs/heads/main": commit},
	}
	for _, record := range strings.Split(fixtureGit(t, upstream, "ls-tree", "-r", "-z", "--full-tree", commit), "\x00") {
		if record == "" {
			continue
		}
		meta, name, ok := strings.Cut(record, "\t")
		if !ok {
			t.Fatalf("unexpected ls-tree record %q", record)
		}
		fields := strings.Fields(meta)
		if len(fields) != 3 {
			t.Fatalf("unexpected ls-tree metadata %q", meta)
		}
		remote.base[name] = cliTreeEntry{Path: name, Mode: fields[0], Type: fields[1], SHA: fields[2]}
	}
	remote.trees[remote.baseTree] = remote.base
	// The base commit object is served so the client can confirm the remote tree is
	// the pinned one; only its id and tree are compared for a base commit.
	remote.commits[commit] = cliCommit{
		SHA:       commit,
		Message:   "fixture",
		Tree:      cliTreeEntry{Path: "", Mode: "040000", Type: "tree", SHA: remote.baseTree},
		Parents:   []cliTreeEntry{},
		Author:    map[string]string{"name": "zt-fixture", "email": "zt@example.invalid"},
		Committer: map[string]string{"name": "zt-fixture", "email": "zt@example.invalid"},
	}
	remote.server = httptest.NewServer(http.HandlerFunc(remote.serve))
	t.Cleanup(remote.server.Close)
	return remote
}

func (r *cliRemote) writeCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writes
}

func (r *cliRemote) authorizedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.authorized
}

// cliBearer is the test twin of the githubauth transport: the production seam adds
// the real token inside that package.
type cliBearer struct{}

func (cliBearer) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header.Set("Authorization", "Bearer "+cliRemoteToken)
	return http.DefaultTransport.RoundTrip(clone)
}

func (r *cliRemote) serve(w http.ResponseWriter, request *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if request.Header.Get("Authorization") != "Bearer "+cliRemoteToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	r.authorized++
	if request.Method != http.MethodGet {
		r.writes++
	}
	prefix := "/repos/" + r.fullName
	path := request.URL.Path
	switch {
	case request.Method == http.MethodGet && path == prefix:
		writeCLIJSON(w, http.StatusOK, map[string]any{"full_name": r.fullName, "default_branch": "main", "private": false})
	case request.Method == http.MethodGet && strings.HasPrefix(path, prefix+"/git/ref/heads/"):
		name := "refs/heads/" + strings.TrimPrefix(path, prefix+"/git/ref/heads/")
		sha, ok := r.refs[name]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeCLIJSON(w, http.StatusOK, map[string]any{"ref": name, "object": map[string]string{"type": "commit", "sha": sha}})
	case request.Method == http.MethodGet && strings.HasPrefix(path, prefix+"/git/commits/"):
		commit, ok := r.commits[strings.TrimPrefix(path, prefix+"/git/commits/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		presented := commit
		presented.Message = cliMessageView(commit.Message)
		writeCLIJSON(w, http.StatusOK, presented)
	case request.Method == http.MethodGet && strings.HasPrefix(path, prefix+"/git/trees/"):
		sha := strings.TrimPrefix(path, prefix+"/git/trees/")
		entries, ok := r.trees[sha]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		names := make([]string, 0, len(entries))
		for name := range entries {
			names = append(names, name)
		}
		sort.Strings(names)
		list := make([]cliTreeEntry, 0, len(names))
		for _, name := range names {
			list = append(list, entries[name])
		}
		writeCLIJSON(w, http.StatusOK, map[string]any{"sha": sha, "truncated": false, "tree": list})
	case request.Method == http.MethodPost && path == prefix+"/git/blobs":
		r.createBlob(w, request)
	case request.Method == http.MethodPost && path == prefix+"/git/trees":
		r.createTree(w, request)
	case request.Method == http.MethodPost && path == prefix+"/git/commits":
		r.createCommit(w, request)
	case request.Method == http.MethodPost && path == prefix+"/git/refs":
		r.createRef(w, request)
	case request.Method == http.MethodGet && path == prefix+"/pulls":
		r.listPulls(w, request)
	case request.Method == http.MethodPost && path == prefix+"/pulls":
		r.createPull(w, request)
	case request.Method == http.MethodGet && strings.HasPrefix(path, prefix+"/pulls/"):
		number, err := strconv.Atoi(strings.TrimPrefix(path, prefix+"/pulls/"))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		for _, pull := range r.pulls {
			if pull.Number == number {
				writeCLIJSON(w, http.StatusOK, pull)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	case request.Method == http.MethodGet && strings.HasSuffix(path, "/check-runs"):
		writeCLIJSON(w, http.StatusOK, map[string]any{
			"total_count": 1,
			"check_runs":  []map[string]string{{"name": "ci", "status": "completed", "conclusion": "success"}},
		})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (r *cliRemote) createBlob(w http.ResponseWriter, request *http.Request) {
	var input struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if !decodeCLIJSON(w, request, &input) {
		return
	}
	if input.Encoding != "utf-8" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	// The blob id is the real Git object id, computed by Git without writing an object.
	command := exec.Command("git", "-C", r.upstream, "hash-object", "-t", "blob", "--stdin")
	command.Stdin = strings.NewReader(input.Content)
	output, err := command.Output()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeCLIJSON(w, http.StatusCreated, map[string]string{"sha": strings.TrimSpace(string(output))})
}

func (r *cliRemote) createTree(w http.ResponseWriter, request *http.Request) {
	var input struct {
		BaseTree string         `json:"base_tree"`
		Tree     []cliTreeEntry `json:"tree"`
	}
	if !decodeCLIJSON(w, request, &input) {
		return
	}
	base, ok := r.trees[input.BaseTree]
	if !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	merged := make(map[string]cliTreeEntry, len(base)+len(input.Tree))
	for name, entry := range base {
		merged[name] = entry
	}
	for _, entry := range input.Tree {
		if entry.Path == "" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		merged[entry.Path] = entry
	}
	sha := cliTreeSHA(merged)
	r.trees[sha] = merged
	writeCLIJSON(w, http.StatusCreated, map[string]string{"sha": sha})
}

func (r *cliRemote) createCommit(w http.ResponseWriter, request *http.Request) {
	var input struct {
		Message   string            `json:"message"`
		Tree      string            `json:"tree"`
		Parents   []string          `json:"parents"`
		Author    map[string]string `json:"author"`
		Committer map[string]string `json:"committer"`
	}
	if !decodeCLIJSON(w, request, &input) {
		return
	}
	if _, ok := r.trees[input.Tree]; !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	sha, err := r.gitCommitObjectSHA(input.Message, input.Tree, input.Parents, input.Author, input.Committer)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	parents := make([]cliTreeEntry, len(input.Parents))
	for i, parent := range input.Parents {
		parents[i] = cliTreeEntry{SHA: parent}
	}
	commit := cliCommit{
		SHA:       sha,
		Message:   input.Message,
		Tree:      cliTreeEntry{SHA: input.Tree, Mode: "040000", Type: "tree"},
		Parents:   parents,
		Author:    input.Author,
		Committer: input.Committer,
	}
	r.commits[sha] = commit
	presented := commit
	presented.Message = cliMessageView(commit.Message)
	writeCLIJSON(w, http.StatusCreated, presented)
}

func (r *cliRemote) createRef(w http.ResponseWriter, request *http.Request) {
	var input struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	}
	if !decodeCLIJSON(w, request, &input) {
		return
	}
	if _, exists := r.refs[input.Ref]; exists {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	r.refs[input.Ref] = input.SHA
	writeCLIJSON(w, http.StatusCreated, map[string]any{"ref": input.Ref, "object": map[string]string{"type": "commit", "sha": input.SHA}})
}

func (r *cliRemote) listPulls(w http.ResponseWriter, request *http.Request) {
	head := request.URL.Query().Get("head")
	state := request.URL.Query().Get("state")
	matched := make([]cliPull, 0, len(r.pulls))
	for _, pull := range r.pulls {
		if head != "" && head != "example:"+pull.Head.Ref {
			continue
		}
		if state != "" && state != "all" && state != pull.State {
			continue
		}
		matched = append(matched, pull)
	}
	writeCLIJSON(w, http.StatusOK, matched)
}

func (r *cliRemote) createPull(w http.ResponseWriter, request *http.Request) {
	var input struct {
		Title string `json:"title"`
		Head  string `json:"head"`
		Base  string `json:"base"`
		Body  string `json:"body"`
		Draft bool   `json:"draft"`
	}
	if !decodeCLIJSON(w, request, &input) {
		return
	}
	headSHA, ok := r.refs["refs/heads/"+input.Head]
	if !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	baseSHA, ok := r.refs["refs/heads/"+input.Base]
	if !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	for _, existing := range r.pulls {
		if existing.State == "open" && existing.Head.Ref == input.Head {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
	}
	pull := cliPull{Number: len(r.pulls) + 1, Title: input.Title, Body: input.Body, State: "open", Draft: input.Draft}
	pull.HTMLURL = fmt.Sprintf("https://github.example/%s/pull/%d", r.fullName, pull.Number)
	pull.Head.Ref = input.Head
	pull.Head.SHA = headSHA
	pull.Head.Repo.FullName = r.fullName
	pull.Base.Ref = input.Base
	pull.Base.SHA = baseSHA
	pull.Base.Repo.FullName = r.fullName
	r.pulls = append(r.pulls, pull)
	writeCLIJSON(w, http.StatusCreated, pull)
}

func cliTreeSHA(entries map[string]cliTreeEntry) string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha1.New()
	io.WriteString(hash, "tree")
	for _, name := range names {
		entry := entries[name]
		fmt.Fprintf(hash, "\x00%s\x00%s\x00%s", name, entry.Mode, entry.SHA)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// gitCommitObjectSHA builds the exact Git commit object for the request and asks Git
// for its object id, so the stand-in behaves like a real server: the object keeps the
// exact message bytes while the API presents a trimmed view.
func (r *cliRemote) gitCommitObjectSHA(message, tree string, parents []string, author, committer map[string]string) (string, error) {
	instant, err := time.Parse(time.RFC3339, author["date"])
	if err != nil {
		return "", err
	}
	committed, err := time.Parse(time.RFC3339, committer["date"])
	if err != nil {
		return "", err
	}
	var object strings.Builder
	fmt.Fprintf(&object, "tree %s\n", tree)
	for _, parent := range parents {
		fmt.Fprintf(&object, "parent %s\n", parent)
	}
	fmt.Fprintf(&object, "author %s <%s> %d +0000\n", author["name"], author["email"], instant.Unix())
	fmt.Fprintf(&object, "committer %s <%s> %d +0000\n", committer["name"], committer["email"], committed.Unix())
	object.WriteByte('\n')
	object.WriteString(message)

	command := exec.Command("git", "-C", r.upstream, "hash-object", "-t", "commit", "--stdin")
	command.Stdin = strings.NewReader(object.String())
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// cliMessageView mirrors GitHub's presentation of a commit message.
func cliMessageView(message string) string {
	return strings.TrimSuffix(message, "\n")
}

func writeCLIJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeCLIJSON(w http.ResponseWriter, request *http.Request, value any) bool {
	body, err := io.ReadAll(io.LimitReader(request.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	if err := json.Unmarshal(body, value); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	return true
}

// cliValidationFixture writes a validation contract whose checks are this test
// binary re-executed with a fixed selector.
func cliValidationFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "validation.json")
	argv := []string{os.Args[0], "-test.run=^TestCLIValidationHelper$", "-test.timeout=60s"}
	writeFile(t, path, evolveTestJSON(t, map[string]any{
		"schemaVersion": 1,
		"checks": []map[string]any{
			{"name": "go-test", "argv": argv, "timeoutSeconds": 60},
			{"name": "repo-lint", "argv": argv, "timeoutSeconds": 60},
		},
	}))
	return path
}

// TestCLIValidationHelper is inert during a normal run and exits successfully when
// zt executes it as the fixture validation command.
func TestCLIValidationHelper(t *testing.T) {
	if os.Getenv("ZT_EVOLVE_CANDIDATE_ID") == "" {
		return
	}
}

func TestEvolveLifecycleEndToEndThroughTheCLI(t *testing.T) {
	fixture := newEvolveFixture(t)
	remote := newCLIRemote(t, fixture.upstream, fixture.commit)
	validation := cliValidationFixture(t)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.homeDir = func() (string, error) { t.Fatal("evolve resolved the home directory"); return "", nil }
	a.newAuth = func(string) (authService, error) { t.Fatal("evolve built the auth service"); return nil, nil }
	a.openAPI = func(_ context.Context, host, clientID, profile, repository string) (githubauth.API, error) {
		if host != "github.com" || repository != "example/fixture" || clientID != cliClientID {
			t.Fatalf("the credential seam was opened for %s/%s as %q", host, repository, clientID)
		}
		return githubauth.API{BaseURL: remote.server.URL, Transport: cliBearer{}}, nil
	}
	run := func(args ...string) string {
		t.Helper()
		stdout.Reset()
		stderr.Reset()
		if err := executeApp(a, context.Background(), args); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, stderr.String())
		}
		return stdout.String()
	}

	prepared := run(fixture.prepareArgs()...)
	var preparedResult struct {
		CandidatePath string `json:"candidatePath"`
	}
	if err := json.Unmarshal([]byte(prepared), &preparedResult); err != nil {
		t.Fatalf("prepare json: %v", err)
	}
	candidate := filepath.Join(fixture.root, filepath.FromSlash(preparedResult.CandidatePath))

	checked := run("evolve", "check", candidate, "--policy", fixture.policy, "--upstream", fixture.upstream,
		"--validation", validation, "--root", fixture.root, "--trusted-local-execution", "--json")
	var checkResult struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(checked), &checkResult); err != nil {
		t.Fatalf("check json: %v\n%s", err, checked)
	}
	if checkResult.Status != "passed" {
		t.Fatalf("verification did not pass: %s", checked)
	}

	preview := run("evolve", "submit", candidate, "--policy", fixture.policy, "--upstream", fixture.upstream,
		"--root", fixture.root, "--show-payload", "--json")
	var plan struct {
		CandidateID         string `json:"candidateId"`
		PublicPayloadDigest string `json:"publicPayloadDigest"`
	}
	if err := json.Unmarshal([]byte(preview), &plan); err != nil {
		t.Fatalf("preview json: %v\n%s", err, preview)
	}
	if plan.CandidateID == "" || plan.PublicPayloadDigest == "" {
		t.Fatalf("preview did not name the candidate and payload: %s", preview)
	}

	submitted := run("evolve", "submit", candidate, "--policy", fixture.policy, "--upstream", fixture.upstream,
		"--root", fixture.root, "--client-id", cliClientID, "--json")
	var submitResult struct {
		Action      string `json:"action"`
		Publication string `json:"publication"`
		Writes      int    `json:"writes"`
		PullRequest *struct {
			Number int  `json:"number"`
			Draft  bool `json:"draft"`
		} `json:"pullRequest"`
	}
	if err := json.Unmarshal([]byte(submitted), &submitResult); err != nil {
		t.Fatalf("submit json: %v\n%s", err, submitted)
	}
	if submitResult.Action != "created" || submitResult.PullRequest == nil || !submitResult.PullRequest.Draft {
		t.Fatalf("unexpected submission: %s", submitted)
	}
	if submitResult.Publication != "submitted" || submitResult.Writes == 0 {
		t.Fatalf("submission did not report a draft pull request: %s", submitted)
	}
	writes := remote.writeCount()

	replayed := run("evolve", "submit", candidate, "--policy", fixture.policy, "--upstream", fixture.upstream,
		"--root", fixture.root, "--client-id", cliClientID, "--json")
	var replayResult struct {
		Action string `json:"action"`
		Writes int    `json:"writes"`
	}
	if err := json.Unmarshal([]byte(replayed), &replayResult); err != nil {
		t.Fatalf("replay json: %v\n%s", err, replayed)
	}
	if replayResult.Action != "existing" || replayResult.Writes != 0 {
		t.Fatalf("replay was not idempotent: %s", replayed)
	}
	if remote.writeCount() != writes {
		t.Fatalf("replay wrote to the remote: %d -> %d", writes, remote.writeCount())
	}

	status := run("evolve", "status", candidate, "--policy", fixture.policy, "--upstream", fixture.upstream,
		"--validation", validation, "--root", fixture.root, "--remote", "--client-id", cliClientID)
	for _, want := range []string{"pull request #1", "draft true", "github check runs: 1 total, 1 success", "publication: submitted"} {
		if !strings.Contains(status, want) {
			t.Fatalf("status omits %q:\n%s", want, status)
		}
	}
	if strings.Contains(status, cliRemoteToken) {
		t.Fatal("the CLI printed the credential")
	}
	if remote.authorizedCount() == 0 {
		t.Fatal("the remote never received an authenticated request")
	}
}
