package evolve

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeGHPull struct {
	Number  int
	Title   string
	Body    string
	Head    string
	HeadSHA string
	Base    string
	BaseSHA string
	Draft   bool
	State   string
	HTMLURL string
}

type fakeGHCheckRun struct {
	Name       string
	Status     string
	Conclusion string
}

type fakeGHOptions struct {
	FullName      string
	DefaultBranch string
	BaseFiles     map[string]string
	// BaseModes optionally declares the Git mode of a base file. An undeclared path
	// is 100644; only the regular-file modes zt reasons about are accepted, so a
	// fixture cannot describe a tree the command surface would refuse.
	BaseModes map[string]string
	Token     string
}

// The seeded root commit is "Initial commit" by zt fixture at this fixed UTC instant.
// Its Git commit message contains exactly one terminating newline; the accessor returns none.
const (
	fakeGHBaseIdentityName  = "zt fixture"
	fakeGHBaseIdentityEmail = "fixture@example.invalid"
	fakeGHBaseMessage       = "Initial commit"
	fakeGHBaseDate          = "2026-01-05T12:00:00Z"
)

type fakeGitHub struct {
	Server        *httptest.Server
	Owner, Repo   string
	DefaultBranch string

	mu             sync.Mutex
	closeOnce      sync.Once
	token          string
	baseCommit     string
	baseTree       string
	baseFiles      map[string]string
	truncateTrees  bool
	blobs          map[string][]byte
	trees          map[string]fakeGHStoredTree
	commits        map[string]fakeGHStoredCommit
	refs           map[string]string
	pulls          []fakeGHPull
	nextPull       int
	checkRuns      map[string][]fakeGHCheckRun
	requests       []string
	writes         int
	failWrite      *fakeGHWriteFailure
	dropAfterWrite string
}

type fakeGHStoredTree struct {
	Entries map[string]fakeGHTreeEntry
	Files   map[string]fakeGHTreeEntry
}

type fakeGHTreeEntry struct {
	Mode string
	Type string
	SHA  string
}

type fakeGHRESTTreeResponse struct {
	SHA       string                `json:"sha"`
	Tree      []fakeGHRESTTreeEntry `json:"tree"`
	Truncated bool                  `json:"truncated"`
}

type fakeGHRESTTreeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	Size int    `json:"size"`
}

type fakeGHStoredCommit struct {
	Tree      string
	Parents   []string
	Message   string
	Author    fakeGHIdentity
	Committer fakeGHIdentity
}

type fakeGHWriteFailure struct {
	Match  string
	Status int
	Body   string
}

type fakeGHIdentity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type fakeGHRESTPull struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	State   string `json:"state"`
	Draft   bool   `json:"draft"`
	HTMLURL string `json:"html_url"`
	Head    struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
}

func newFakeGitHub(t *testing.T, options fakeGHOptions) *fakeGitHub {
	t.Helper()

	parts := strings.Split(options.FullName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		t.Fatalf("invalid fake GitHub full name %q; want owner/name", options.FullName)
	}
	if options.DefaultBranch == "" {
		t.Fatal("fake GitHub default branch must not be empty")
	}

	g := &fakeGitHub{
		Owner:         parts[0],
		Repo:          parts[1],
		DefaultBranch: options.DefaultBranch,
		token:         options.Token,
		blobs:         make(map[string][]byte),
		trees:         make(map[string]fakeGHStoredTree),
		commits:       make(map[string]fakeGHStoredCommit),
		refs:          make(map[string]string),
		nextPull:      1,
		checkRuns:     make(map[string][]fakeGHCheckRun),
		baseFiles:     make(map[string]string, len(options.BaseFiles)),
	}

	for name := range options.BaseModes {
		if _, ok := options.BaseFiles[name]; !ok {
			t.Fatalf("fake GitHub base mode names %q, which is not a base file", name)
		}
	}
	baseFiles := make(map[string]fakeGHTreeEntry, len(options.BaseFiles))
	for name, content := range options.BaseFiles {
		if !fakeGHValidTreePath(name) {
			t.Fatalf("invalid fake GitHub base file path %q", name)
		}
		mode := "100644"
		if declared, ok := options.BaseModes[name]; ok {
			if declared != "100644" && declared != "100755" {
				t.Fatalf("fake GitHub base file %q has mode %q; only regular-file modes are supported", name, declared)
			}
			mode = declared
		}
		g.baseFiles[name] = content
		data := []byte(content)
		sha := fakeGHObjectSHA("blob", data)
		g.blobs[sha] = append([]byte(nil), data...)
		baseFiles[name] = fakeGHTreeEntry{Mode: mode, Type: "blob", SHA: sha}
	}

	treeSHA, trees, err := fakeGHBuildTrees(baseFiles)
	if err != nil {
		t.Fatalf("build fake GitHub base tree: %v", err)
	}
	for sha, tree := range trees {
		g.trees[sha] = tree
	}
	g.baseTree = treeSHA

	identity := fakeGHIdentity{
		Name:  fakeGHBaseIdentityName,
		Email: fakeGHBaseIdentityEmail,
		Date:  fakeGHBaseDate,
	}
	// git commit -m stores the message with one trailing newline; the seeded commit
	// mirrors that so a real local repository can reproduce this exact object.
	commitObject := fakeGHCommitObject(treeSHA, nil, fakeGHBaseMessage+"\n", identity, identity)
	g.baseCommit = fakeGHObjectSHA("commit", []byte(commitObject))
	g.commits[g.baseCommit] = fakeGHStoredCommit{
		Tree:      treeSHA,
		Parents:   []string{},
		Message:   fakeGHBaseMessage + "\n",
		Author:    identity,
		Committer: identity,
	}
	g.refs["refs/heads/"+g.DefaultBranch] = g.baseCommit

	g.Server = httptest.NewServer(http.HandlerFunc(g.serveHTTP))
	t.Cleanup(g.Close)
	return g
}

func (g *fakeGitHub) BaseCommit() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.baseCommit
}

func (g *fakeGitHub) BaseTree() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.baseTree
}

// BaseCommittedAt returns the fixed UTC instant encoded in the seeded root commit.
func (g *fakeGitHub) BaseCommittedAt() time.Time {
	instant, _ := time.Parse(time.RFC3339, fakeGHBaseDate)
	return instant.UTC()
}

func (g *fakeGitHub) BaseCommitIdentity() (name, email string) {
	return fakeGHBaseIdentityName, fakeGHBaseIdentityEmail
}

// BaseCommitMessage has no trailing newline; the Git commit object adds one.
func (g *fakeGitHub) BaseCommitMessage() string {
	return fakeGHBaseMessage
}

func (g *fakeGitHub) BaseFiles() map[string]string {
	g.mu.Lock()
	defer g.mu.Unlock()
	files := make(map[string]string, len(g.baseFiles))
	for name, content := range g.baseFiles {
		files[name] = content
	}
	return files
}

func (g *fakeGitHub) TreeSHA(commit string) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	record, ok := g.commits[commit]
	return record.Tree, ok
}

// TreeFiles returns the flattened entries of a stored tree, so a test can read the
// modes and object ids a submission actually sent instead of the expected set it
// compared itself with.
func (g *fakeGitHub) TreeFiles(treeSHA string) (map[string]fakeGHTreeEntry, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	tree, ok := g.trees[treeSHA]
	if !ok {
		return nil, false
	}
	files := make(map[string]fakeGHTreeEntry, len(tree.Files))
	for name, entry := range tree.Files {
		files[name] = entry
	}
	return files, true
}

func (g *fakeGitHub) CommitExists(sha string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.commits[sha]
	return ok
}

func (g *fakeGitHub) Ref(name string) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	sha, ok := g.refs[name]
	return sha, ok
}

func (g *fakeGitHub) SetRef(name, sha string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.refs[name] = sha
}

func (g *fakeGitHub) Pulls() []fakeGHPull {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]fakeGHPull(nil), g.pulls...)
}

func (g *fakeGitHub) AttachPull(pull fakeGHPull) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	if pull.Number <= 0 {
		pull.Number = g.nextPull
		g.nextPull++
	} else if pull.Number >= g.nextPull {
		g.nextPull = pull.Number + 1
	}
	if pull.State == "" {
		pull.State = "open"
	}
	g.pulls = append(g.pulls, pull)
	return pull.Number
}

func (g *fakeGitHub) AddCheckRuns(commitSHA string, runs []fakeGHCheckRun) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.checkRuns[commitSHA] = append(g.checkRuns[commitSHA], runs...)
}

func (g *fakeGitHub) BaseMoved() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	refName := "refs/heads/" + g.DefaultBranch
	parent := g.refs[refName]
	identity := fakeGHIdentity{
		Name:  fakeGHBaseIdentityName,
		Email: fakeGHBaseIdentityEmail,
		Date:  time.Now().UTC().Format(time.RFC3339),
	}
	message := "Move base"
	object := fakeGHCommitObject(g.baseTree, []string{parent}, message, identity, identity)
	sha := fakeGHObjectSHA("commit", []byte(object))
	g.commits[sha] = fakeGHStoredCommit{
		Tree:      g.baseTree,
		Parents:   []string{parent},
		Message:   message,
		Author:    identity,
		Committer: identity,
	}
	g.refs[refName] = sha
	return sha
}

func (g *fakeGitHub) Requests() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string(nil), g.requests...)
}

func (g *fakeGitHub) Writes() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.writes
}

func (g *fakeGitHub) FailWrite(methodAndPathSubstring string, status int, body string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.failWrite = &fakeGHWriteFailure{
		Match:  methodAndPathSubstring,
		Status: status,
		Body:   body,
	}
}

func (g *fakeGitHub) DropAfterWrite(methodAndPathSubstring string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.dropAfterWrite = methodAndPathSubstring
}

func (g *fakeGitHub) TruncateTrees() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.truncateTrees = true
}

func (g *fakeGitHub) Close() {
	g.closeOnce.Do(func() {
		if g.Server != nil {
			g.Server.Close()
		}
	})
}

func (g *fakeGitHub) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if g.token != "" && r.Header.Get("Authorization") != "Bearer "+g.token {
		fakeGHWriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "Bad credentials"})
		return
	}

	g.mu.Lock()
	g.requests = append(g.requests, r.Method+" "+r.URL.Path)
	if r.Method != http.MethodGet {
		g.writes++
	}
	g.mu.Unlock()

	if strings.TrimSpace(r.Header.Get("User-Agent")) == "" {
		fakeGHWriteJSON(w, http.StatusBadRequest, map[string]string{"message": "User-Agent required"})
		return
	}

	owner, repo, tail, hasTail, ok := fakeGHRepositoryPath(r.URL.Path)
	if !ok || owner != g.Owner || repo != g.Repo {
		fakeGHWriteJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}

	methodAndPath := r.Method + " " + r.URL.Path
	isWriteHandler := fakeGHIsWriteHandler(r.Method, tail, hasTail)
	var injectedFailure *fakeGHWriteFailure
	dropAfterWrite := false
	if isWriteHandler {
		g.mu.Lock()
		if g.failWrite != nil && fakeGHWritePatternMatches(methodAndPath, g.failWrite.Match) {
			copyFailure := *g.failWrite
			injectedFailure = &copyFailure
			g.failWrite = nil
		} else if g.dropAfterWrite != "" && fakeGHWritePatternMatches(methodAndPath, g.dropAfterWrite) {
			dropAfterWrite = true
			g.dropAfterWrite = ""
		}
		g.mu.Unlock()
	}
	if injectedFailure != nil {
		fakeGHWriteRawJSON(w, injectedFailure.Status, injectedFailure.Body)
		return
	}

	g.mu.Lock()
	status, payload, rawBody, mutated := g.handleRequestLocked(r, tail, hasTail)
	g.mu.Unlock()

	if dropAfterWrite && mutated && status >= 200 && status < 300 {
		if hijacker, ok := w.(http.Hijacker); ok {
			conn, _, err := hijacker.Hijack()
			if err == nil {
				_ = conn.Close()
				return
			}
		}
	}
	if rawBody != "" {
		fakeGHWriteRawJSON(w, status, rawBody)
		return
	}
	fakeGHWriteJSON(w, status, payload)
}

func (g *fakeGitHub) handleRequestLocked(r *http.Request, tail string, hasTail bool) (int, any, string, bool) {
	if !hasTail && r.Method == http.MethodGet {
		return http.StatusOK, map[string]any{
			"full_name":      g.Owner + "/" + g.Repo,
			"default_branch": g.DefaultBranch,
			"private":        false,
			"id":             123,
		}, "", false
	}

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(tail, "git/trees/"):
		sha := strings.TrimPrefix(tail, "git/trees/")
		if sha == "" || strings.Contains(sha, "/") {
			break
		}
		tree, ok := g.trees[sha]
		if !ok {
			return http.StatusNotFound, map[string]string{"message": "Not Found"}, "", false
		}
		names := make([]string, 0, len(tree.Files))
		for name := range tree.Files {
			names = append(names, name)
		}
		sort.Strings(names)
		response := fakeGHRESTTreeResponse{
			SHA:       sha,
			Tree:      make([]fakeGHRESTTreeEntry, 0, len(names)),
			Truncated: g.truncateTrees,
		}
		for _, name := range names {
			entry := tree.Files[name]
			response.Tree = append(response.Tree, fakeGHRESTTreeEntry{
				Path: name,
				Mode: entry.Mode,
				Type: entry.Type,
				SHA:  entry.SHA,
				Size: len(g.blobs[entry.SHA]),
			})
		}
		return http.StatusOK, response, "", false

	case r.Method == http.MethodGet && strings.HasPrefix(tail, "git/commits/"):
		sha := strings.TrimPrefix(tail, "git/commits/")
		if sha == "" || strings.Contains(sha, "/") {
			break
		}
		commit, ok := g.commits[sha]
		if !ok {
			return http.StatusNotFound, map[string]string{"message": "Not Found"}, "", false
		}
		parents := make([]map[string]string, 0, len(commit.Parents))
		for _, parent := range commit.Parents {
			parents = append(parents, map[string]string{"sha": parent})
		}
		return http.StatusOK, map[string]any{
			"sha":       sha,
			"tree":      map[string]string{"sha": commit.Tree},
			"message":   fakeGHMessageView(commit.Message),
			"parents":   parents,
			"author":    commit.Author,
			"committer": commit.Committer,
		}, "", false

	case r.Method == http.MethodPost && tail == "git/blobs":
		var input struct {
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
		}
		if !fakeGHDecodeBody(r, &input) {
			return http.StatusBadRequest, map[string]string{"message": "Invalid JSON"}, "", false
		}
		if input.Encoding != "utf-8" {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Unsupported encoding"}, "", false
		}
		data := []byte(input.Content)
		sha := fakeGHObjectSHA("blob", data)
		g.blobs[sha] = append([]byte(nil), data...)
		return http.StatusCreated, map[string]string{"sha": sha}, "", true

	case r.Method == http.MethodPost && tail == "git/trees":
		var input struct {
			BaseTree string `json:"base_tree"`
			Tree     []struct {
				Path string `json:"path"`
				Mode string `json:"mode"`
				Type string `json:"type"`
				SHA  string `json:"sha"`
			} `json:"tree"`
		}
		if !fakeGHDecodeBody(r, &input) {
			return http.StatusBadRequest, map[string]string{"message": "Invalid JSON"}, "", false
		}
		files := make(map[string]fakeGHTreeEntry)
		if input.BaseTree != "" {
			base, ok := g.trees[input.BaseTree]
			if !ok {
				return http.StatusUnprocessableEntity, map[string]string{"message": "Unknown base tree"}, "", false
			}
			for name, entry := range base.Files {
				files[name] = entry
			}
		}
		for _, entry := range input.Tree {
			if !fakeGHValidTreePath(entry.Path) {
				return http.StatusUnprocessableEntity, map[string]string{"message": "Invalid tree path"}, "", false
			}
			if entry.Type != "blob" || (entry.Mode != "100644" && entry.Mode != "100755" && entry.Mode != "120000") {
				return http.StatusUnprocessableEntity, map[string]string{"message": "Unsupported tree entry"}, "", false
			}
			if _, ok := g.blobs[entry.SHA]; !ok {
				return http.StatusUnprocessableEntity, map[string]string{"message": "Unknown object"}, "", false
			}
			files[entry.Path] = fakeGHTreeEntry{Mode: entry.Mode, Type: entry.Type, SHA: entry.SHA}
		}
		sha, trees, err := fakeGHBuildTrees(files)
		if err != nil {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Invalid tree"}, "", false
		}
		for treeSHA, tree := range trees {
			g.trees[treeSHA] = tree
		}
		return http.StatusCreated, map[string]string{"sha": sha}, "", true

	case r.Method == http.MethodPost && tail == "git/commits":
		var input struct {
			Message   string         `json:"message"`
			Tree      string         `json:"tree"`
			Parents   []string       `json:"parents"`
			Author    fakeGHIdentity `json:"author"`
			Committer fakeGHIdentity `json:"committer"`
		}
		if !fakeGHDecodeBody(r, &input) {
			return http.StatusBadRequest, map[string]string{"message": "Invalid JSON"}, "", false
		}
		if _, ok := g.trees[input.Tree]; !ok {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Unknown tree"}, "", false
		}
		for _, parent := range input.Parents {
			if _, ok := g.commits[parent]; !ok {
				return http.StatusUnprocessableEntity, map[string]string{"message": "Unknown parent"}, "", false
			}
		}
		if _, err := fakeGHIdentityTimestamp(input.Author); err != nil {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Invalid author date"}, "", false
		}
		if _, err := fakeGHIdentityTimestamp(input.Committer); err != nil {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Invalid committer date"}, "", false
		}
		object := fakeGHCommitObject(input.Tree, input.Parents, input.Message, input.Author, input.Committer)
		sha := fakeGHObjectSHA("commit", []byte(object))
		g.commits[sha] = fakeGHStoredCommit{
			Tree:      input.Tree,
			Parents:   append([]string(nil), input.Parents...),
			Message:   input.Message,
			Author:    input.Author,
			Committer: input.Committer,
		}
		parents := make([]map[string]string, 0, len(input.Parents))
		for _, parent := range input.Parents {
			parents = append(parents, map[string]string{"sha": parent})
		}
		return http.StatusCreated, map[string]any{
			"sha":       sha,
			"tree":      map[string]string{"sha": input.Tree},
			"parents":   parents,
			"message":   fakeGHMessageView(input.Message),
			"author":    input.Author,
			"committer": input.Committer,
		}, "", true

	case r.Method == http.MethodPost && tail == "git/refs":
		var input struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		}
		if !fakeGHDecodeBody(r, &input) {
			return http.StatusBadRequest, map[string]string{"message": "Invalid JSON"}, "", false
		}
		if !strings.HasPrefix(input.Ref, "refs/heads/") || len(input.Ref) == len("refs/heads/") {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Invalid reference"}, "", false
		}
		if _, exists := g.refs[input.Ref]; exists {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Reference already exists"}, "", false
		}
		if _, exists := g.commits[input.SHA]; !exists {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Unknown commit"}, "", false
		}
		g.refs[input.Ref] = input.SHA
		return http.StatusCreated, map[string]any{
			"ref":    input.Ref,
			"object": map[string]string{"type": "commit", "sha": input.SHA},
		}, "", true

	case r.Method == http.MethodGet && strings.HasPrefix(tail, "git/ref/heads/"):
		branch := strings.TrimPrefix(tail, "git/ref/heads/")
		if branch == "" {
			break
		}
		ref := "refs/heads/" + branch
		sha, ok := g.refs[ref]
		if !ok {
			return http.StatusNotFound, map[string]string{"message": "Not Found"}, "", false
		}
		return http.StatusOK, map[string]any{
			"ref":    ref,
			"object": map[string]string{"type": "commit", "sha": sha},
		}, "", false

	case r.Method == http.MethodGet && tail == "pulls":
		state := r.URL.Query().Get("state")
		if state == "" {
			state = "open"
		}
		head := r.URL.Query().Get("head")
		pulls := make([]fakeGHRESTPull, 0, len(g.pulls))
		for _, pull := range g.pulls {
			if head != "" && head != g.Owner+":"+pull.Head {
				continue
			}
			if state != "all" && state != pull.State {
				continue
			}
			pulls = append(pulls, fakeGHPullResponse(g, pull))
		}
		return http.StatusOK, pulls, "", false

	case r.Method == http.MethodPost && tail == "pulls":
		var input struct {
			Title string `json:"title"`
			Head  string `json:"head"`
			Base  string `json:"base"`
			Body  string `json:"body"`
			Draft bool   `json:"draft"`
		}
		if !fakeGHDecodeBody(r, &input) {
			return http.StatusBadRequest, map[string]string{"message": "Invalid JSON"}, "", false
		}
		for _, pull := range g.pulls {
			if pull.Head == input.Head && pull.State == "open" {
				return http.StatusUnprocessableEntity, map[string]string{"message": "A pull request already exists for this head"}, "", false
			}
		}
		headSHA, ok := g.refs["refs/heads/"+input.Head]
		if !ok {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Head branch not found"}, "", false
		}
		baseSHA, ok := g.refs["refs/heads/"+input.Base]
		if !ok {
			return http.StatusUnprocessableEntity, map[string]string{"message": "Base branch not found"}, "", false
		}
		pull := fakeGHPull{
			Number:  g.nextPull,
			Title:   input.Title,
			Body:    input.Body,
			Head:    input.Head,
			HeadSHA: headSHA,
			Base:    input.Base,
			BaseSHA: baseSHA,
			Draft:   input.Draft,
			State:   "open",
		}
		g.nextPull++
		pull.HTMLURL = fmt.Sprintf("%s/repos/%s/%s/pull/%d", g.Server.URL, g.Owner, g.Repo, pull.Number)
		g.pulls = append(g.pulls, pull)
		return http.StatusCreated, fakeGHPullResponse(g, pull), "", true

	case r.Method == http.MethodGet && strings.HasPrefix(tail, "pulls/"):
		numberPart := strings.TrimPrefix(tail, "pulls/")
		if numberPart == "" || strings.Contains(numberPart, "/") {
			break
		}
		number, err := strconv.Atoi(numberPart)
		if err != nil {
			break
		}
		for _, pull := range g.pulls {
			if pull.Number == number {
				return http.StatusOK, fakeGHPullResponse(g, pull), "", false
			}
		}
		return http.StatusNotFound, map[string]string{"message": "Not Found"}, "", false

	case r.Method == http.MethodGet && strings.HasPrefix(tail, "commits/"):
		parts := strings.Split(tail, "/")
		if len(parts) != 3 || parts[0] != "commits" || parts[2] != "check-runs" {
			break
		}
		runs := g.checkRuns[parts[1]]
		checks := make([]map[string]string, 0, len(runs))
		for _, run := range runs {
			checks = append(checks, map[string]string{
				"name":       run.Name,
				"status":     run.Status,
				"conclusion": run.Conclusion,
			})
		}
		return http.StatusOK, map[string]any{"total_count": len(checks), "check_runs": checks}, "", false
	}

	return http.StatusNotFound, map[string]string{"message": "Not Found"}, "", false
}

func fakeGHIsWriteHandler(method, tail string, hasTail bool) bool {
	if method != http.MethodPost || !hasTail {
		return false
	}
	switch tail {
	case "git/blobs", "git/trees", "git/commits", "git/refs", "pulls":
		return true
	default:
		return false
	}
}

func fakeGHWritePatternMatches(methodAndPath, pattern string) bool {
	if strings.Contains(methodAndPath, pattern) {
		return true
	}
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) != 2 || !strings.HasPrefix(methodAndPath, parts[0]+" ") {
		return false
	}
	actualPath := strings.TrimPrefix(methodAndPath, parts[0]+" ")
	return strings.Contains(actualPath, parts[1])
}

func fakeGHRepositoryPath(requestPath string) (string, string, string, bool, bool) {
	if !strings.HasPrefix(requestPath, "/repos/") {
		return "", "", "", false, false
	}
	parts := strings.SplitN(strings.TrimPrefix(requestPath, "/repos/"), "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", false, false
	}
	if len(parts) == 2 {
		return parts[0], parts[1], "", false, true
	}
	return parts[0], parts[1], parts[2], true, true
}

func fakeGHDecodeBody(r *http.Request, value any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(value) == nil
}

func fakeGHWriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func fakeGHWriteRawJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func fakeGHPullResponse(g *fakeGitHub, pull fakeGHPull) fakeGHRESTPull {
	var response fakeGHRESTPull
	response.Number = pull.Number
	response.Title = pull.Title
	response.Body = pull.Body
	response.State = pull.State
	response.Draft = pull.Draft
	response.HTMLURL = pull.HTMLURL
	response.Head.Ref = pull.Head
	response.Head.SHA = pull.HeadSHA
	response.Head.Repo.FullName = g.Owner + "/" + g.Repo
	response.Base.Ref = pull.Base
	response.Base.SHA = pull.BaseSHA
	response.Base.Repo.FullName = g.Owner + "/" + g.Repo
	if response.HTMLURL == "" {
		response.HTMLURL = fmt.Sprintf("%s/repos/%s/%s/pull/%d", g.Server.URL, g.Owner, g.Repo, pull.Number)
	}
	return response
}

func fakeGHValidTreePath(name string) bool {
	if name == "" || strings.HasPrefix(name, "/") || path.Clean(name) != name {
		return false
	}
	for _, component := range strings.Split(name, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}

func fakeGHObjectSHA(kind string, content []byte) string {
	header := []byte(fmt.Sprintf("%s %d\x00", kind, len(content)))
	object := make([]byte, 0, len(header)+len(content))
	object = append(object, header...)
	object = append(object, content...)
	sum := sha1.Sum(object)
	return hex.EncodeToString(sum[:])
}

func fakeGHBuildTrees(files map[string]fakeGHTreeEntry) (string, map[string]fakeGHStoredTree, error) {
	root := &fakeGHTreeNode{
		Files: make(map[string]fakeGHTreeEntry),
		Dirs:  make(map[string]*fakeGHTreeNode),
	}
	for fullName, entry := range files {
		if !fakeGHValidTreePath(fullName) {
			return "", nil, fmt.Errorf("invalid path %q", fullName)
		}
		components := strings.Split(fullName, "/")
		node := root
		for index, component := range components {
			if index == len(components)-1 {
				if _, exists := node.Dirs[component]; exists {
					return "", nil, fmt.Errorf("path %q conflicts with a directory", fullName)
				}
				node.Files[component] = entry
				continue
			}
			if _, exists := node.Files[component]; exists {
				return "", nil, fmt.Errorf("path %q conflicts with a file", fullName)
			}
			if node.Dirs[component] == nil {
				node.Dirs[component] = &fakeGHTreeNode{
					Files: make(map[string]fakeGHTreeEntry),
					Dirs:  make(map[string]*fakeGHTreeNode),
				}
			}
			node = node.Dirs[component]
		}
	}

	trees := make(map[string]fakeGHStoredTree)
	var build func(*fakeGHTreeNode) (string, map[string]fakeGHTreeEntry, error)
	build = func(node *fakeGHTreeNode) (string, map[string]fakeGHTreeEntry, error) {
		entries := make(map[string]fakeGHTreeEntry, len(node.Files)+len(node.Dirs))
		flatFiles := make(map[string]fakeGHTreeEntry, len(node.Files))
		for name, entry := range node.Files {
			entries[name] = entry
			flatFiles[name] = entry
		}
		for name, child := range node.Dirs {
			childSHA, childFiles, err := build(child)
			if err != nil {
				return "", nil, err
			}
			entries[name] = fakeGHTreeEntry{Mode: "40000", Type: "tree", SHA: childSHA}
			for childName, entry := range childFiles {
				flatFiles[name+"/"+childName] = entry
			}
		}

		names := make([]string, 0, len(entries))
		for name := range entries {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			left, right := names[i], names[j]
			if entries[left].Type == "tree" {
				left += "/"
			}
			if entries[right].Type == "tree" {
				right += "/"
			}
			return bytes.Compare([]byte(left), []byte(right)) < 0
		})

		var content bytes.Buffer
		for _, name := range names {
			entry := entries[name]
			rawSHA, err := hex.DecodeString(entry.SHA)
			if err != nil || len(rawSHA) != sha1.Size {
				return "", nil, fmt.Errorf("invalid object id %q", entry.SHA)
			}
			content.WriteString(entry.Mode)
			content.WriteByte(' ')
			content.WriteString(name)
			content.WriteByte(0)
			content.Write(rawSHA)
		}
		sha := fakeGHObjectSHA("tree", content.Bytes())
		trees[sha] = fakeGHStoredTree{Entries: entries, Files: flatFiles}
		return sha, flatFiles, nil
	}

	sha, _, err := build(root)
	if err != nil {
		return "", nil, err
	}
	return sha, trees, nil
}

type fakeGHTreeNode struct {
	Files map[string]fakeGHTreeEntry
	Dirs  map[string]*fakeGHTreeNode
}

func fakeGHIdentityTimestamp(identity fakeGHIdentity) (int64, error) {
	date, err := time.Parse(time.RFC3339, identity.Date)
	if err == nil {
		return date.Unix(), nil
	}
	seconds, parseErr := strconv.ParseInt(identity.Date, 10, 64)
	if parseErr == nil {
		return seconds, nil
	}
	return 0, err
}

// fakeGHMessageView mirrors GitHub's REST presentation: the stored Git object keeps
// the exact message bytes, while the API returns the message with one trailing
// newline removed. Verification compares the object id, never this view.
func fakeGHMessageView(message string) string {
	return strings.TrimSuffix(message, "\n")
}

func fakeGHCommitObject(tree string, parents []string, message string, author, committer fakeGHIdentity) string {
	authorTime, _ := fakeGHIdentityTimestamp(author)
	committerTime, _ := fakeGHIdentityTimestamp(committer)
	var object strings.Builder
	fmt.Fprintf(&object, "tree %s\n", tree)
	for _, parent := range parents {
		fmt.Fprintf(&object, "parent %s\n", parent)
	}
	fmt.Fprintf(&object, "author %s <%s> %d +0000\n", author.Name, author.Email, authorTime)
	fmt.Fprintf(&object, "committer %s <%s> %d +0000\n", committer.Name, committer.Email, committerTime)
	object.WriteByte('\n')
	// The stored object keeps the exact message bytes it was given; GitHub does not
	// append a newline, so a message that already ends with one has exactly one.
	object.WriteString(message)
	return object.String()
}

func fakeGHTestRequest(t *testing.T, g *fakeGitHub, method, requestPath, body, token string) (int, http.Header, []byte) {
	t.Helper()
	request, err := http.NewRequest(method, g.Server.URL+requestPath, strings.NewReader(body))
	if err != nil {
		t.Fatalf("create HTTP request: %v", err)
	}
	request.Header.Set("User-Agent", "fake-github-test")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send %s %s: %v", method, requestPath, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, requestPath, err)
	}
	return response.StatusCode, response.Header, responseBody
}

func fakeGHTestJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode JSON response %q: %v", body, err)
	}
}

func fakeGHRunGit(t *testing.T, directory string, args []string, stdin string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	command.Stdin = strings.NewReader(stdin)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func TestFakeGitHubGitObjectSHAsMatchGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	directory := t.TempDir()
	fakeGHRunGit(t, directory, []string{"init", "--quiet"}, "")

	g := newFakeGitHub(t, fakeGHOptions{
		FullName:      "example/project",
		DefaultBranch: "main",
	})
	content := "faithful fixture\n"
	status, headers, body := fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/git/blobs",
		`{"content":"faithful fixture\n","encoding":"utf-8"}`, "")
	if status != http.StatusCreated || headers.Get("Content-Type") != "application/json" {
		t.Fatalf("blob response = %d %q, want 201 application/json", status, headers.Get("Content-Type"))
	}
	var blobResponse struct {
		SHA string `json:"sha"`
	}
	fakeGHTestJSON(t, body, &blobResponse)
	gitBlobSHA := fakeGHRunGit(t, directory, []string{"hash-object", "-w", "--stdin"}, content)
	if blobResponse.SHA != gitBlobSHA {
		t.Fatalf("blob SHA = %s, git hash-object reported %s", blobResponse.SHA, gitBlobSHA)
	}

	treeBody := fmt.Sprintf(`{"tree":[{"path":"file.txt","mode":"100644","type":"blob","sha":%q}]}`, blobResponse.SHA)
	status, _, body = fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/git/trees", treeBody, "")
	if status != http.StatusCreated {
		t.Fatalf("create tree status = %d, body = %s", status, body)
	}
	var treeResponse struct {
		SHA string `json:"sha"`
	}
	fakeGHTestJSON(t, body, &treeResponse)
	gitTreeSHA := fakeGHRunGit(t, directory, []string{"mktree"}, "100644 blob "+blobResponse.SHA+"\tfile.txt\n")
	if treeResponse.SHA != gitTreeSHA {
		t.Fatalf("tree SHA = %s, git mktree reported %s", treeResponse.SHA, gitTreeSHA)
	}

	identity := fakeGHIdentity{Name: "Test User", Email: "test@example.invalid", Date: "2025-01-02T03:04:05Z"}
	commitRequest := map[string]any{
		"message": "faithful commit",
		"tree":    treeResponse.SHA,
		"parents": []string{},
		"author": map[string]string{
			"name": identity.Name, "email": identity.Email, "date": identity.Date,
		},
		"committer": map[string]string{
			"name": identity.Name, "email": identity.Email, "date": identity.Date,
		},
	}
	commitBody, err := json.Marshal(commitRequest)
	if err != nil {
		t.Fatal(err)
	}
	status, _, body = fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/git/commits", string(commitBody), "")
	if status != http.StatusCreated {
		t.Fatalf("create commit status = %d, body = %s", status, body)
	}
	var commitResponse struct {
		SHA string `json:"sha"`
	}
	fakeGHTestJSON(t, body, &commitResponse)
	commitObject := fakeGHCommitObject(treeResponse.SHA, nil, "faithful commit", identity, identity)
	gitCommitSHA := fakeGHRunGit(t, directory, []string{"hash-object", "-t", "commit", "-w", "--stdin"}, commitObject)
	if commitResponse.SHA != gitCommitSHA {
		t.Fatalf("commit SHA = %s, git hash-object reported %s", commitResponse.SHA, gitCommitSHA)
	}
	gitCommitObject := fakeGHRunGit(t, directory, []string{"cat-file", "commit", commitResponse.SHA}, "")
	if gitCommitObject != strings.TrimSuffix(commitObject, "\n") {
		t.Fatalf("git cat-file commit returned %q, want %q", gitCommitObject, strings.TrimSuffix(commitObject, "\n"))
	}

	status, _, body = fakeGHTestRequest(t, g, http.MethodGet, "/repos/example/project", "", "")
	if status != http.StatusOK {
		t.Fatalf("repository response status = %d, body = %s", status, body)
	}
	var repository struct {
		FullName      string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
	}
	fakeGHTestJSON(t, body, &repository)
	if repository.FullName != "example/project" || repository.DefaultBranch != "main" {
		t.Fatalf("repository response = %+v", repository)
	}

	status, _, body = fakeGHTestRequest(t, g, http.MethodGet, "/repos/example/project/git/commits/"+commitResponse.SHA, "", "")
	if status != http.StatusOK {
		t.Fatalf("get commit status = %d, body = %s", status, body)
	}
}

func TestFakeGitHubBaseCommitMatchesGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	g := newFakeGitHub(t, fakeGHOptions{
		FullName:      "example/project",
		DefaultBranch: "main",
		BaseFiles: map[string]string{
			"nested/file.md": "base content\n",
			"root.txt":       "another file\n",
		},
	})
	directory := t.TempDir()
	fakeGHRunGit(t, directory, []string{"init", "--quiet"}, "")
	for name, content := range g.BaseFiles() {
		filePath := filepath.Join(directory, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatalf("create directory for %q: %v", name, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("write base file %q: %v", name, err)
		}
	}
	fakeGHRunGit(t, directory, []string{"add", "--all"}, "")
	name, email := g.BaseCommitIdentity()
	date := g.BaseCommittedAt().Format("2006-01-02T15:04:05-07:00")
	command := exec.Command(
		"git", "-C", directory,
		"-c", "user.name="+name,
		"-c", "user.email="+email,
		"-c", "commit.gpgsign=false",
		"commit", "--quiet", "-m", g.BaseCommitMessage(),
	)
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+date,
		"GIT_COMMITTER_DATE="+date,
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, output)
	}

	if got := fakeGHRunGit(t, directory, []string{"rev-parse", "HEAD"}, ""); got != g.BaseCommit() {
		t.Fatalf("local root commit = %s, fake base commit = %s", got, g.BaseCommit())
	}
	if got := fakeGHRunGit(t, directory, []string{"rev-parse", "HEAD^{tree}"}, ""); got != g.BaseTree() {
		t.Fatalf("local root tree = %s, fake base tree = %s", got, g.BaseTree())
	}
	if g.BaseCommittedAt().Location() != time.UTC {
		t.Fatalf("BaseCommittedAt location = %v, want UTC", g.BaseCommittedAt().Location())
	}
}

func TestFakeGitHubTreeEndpointFlattensAndCanTruncate(t *testing.T) {
	g := newFakeGitHub(t, fakeGHOptions{
		FullName:      "example/project",
		DefaultBranch: "main",
		BaseFiles: map[string]string{
			"a/b.md": "nested base\n",
			"top.md": "top level\n",
		},
	})
	getTree := func(sha string) (int, []byte) {
		status, _, body := fakeGHTestRequest(
			t, g, http.MethodGet,
			"/repos/example/project/git/trees/"+sha+"?recursive=1",
			"",
			"",
		)
		return status, body
	}
	checkEntries := func(body []byte, want map[string]string, wantTruncated bool) {
		t.Helper()
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatalf("unmarshal raw tree response: %v", err)
		}
		if _, ok := raw["tree"]; !ok {
			t.Fatal("tree response is missing the tree key")
		}
		if _, ok := raw["entries"]; ok {
			t.Fatal("tree response contains the non-standard entries key")
		}
		truncated, ok := raw["truncated"].(bool)
		if !ok || truncated != wantTruncated {
			t.Fatalf("raw tree response truncated = %v, want %t", raw["truncated"], wantTruncated)
		}
		rawEntries, ok := raw["tree"].([]any)
		if !ok {
			t.Fatalf("raw tree response tree = %T, want array", raw["tree"])
		}
		for index, rawEntry := range rawEntries {
			entry, ok := rawEntry.(map[string]any)
			if !ok {
				t.Fatalf("raw tree entry %d = %T, want object", index, rawEntry)
			}
			if size, ok := entry["size"].(float64); !ok || size < 0 {
				t.Fatalf("raw tree entry %d size = %v, want non-negative number", index, entry["size"])
			}
		}

		var response fakeGHRESTTreeResponse
		fakeGHTestJSON(t, body, &response)
		if len(response.Tree) != len(want) {
			t.Fatalf("tree entries = %+v, want %d entries", response.Tree, len(want))
		}
		for _, entry := range response.Tree {
			if entry.Mode != "100644" || entry.Type != "blob" {
				t.Errorf("tree entry %q has mode/type %q/%q", entry.Path, entry.Mode, entry.Type)
			}
			if entry.Size < 0 {
				t.Errorf("tree entry %q has negative size %d", entry.Path, entry.Size)
			}
			if _, ok := want[entry.Path]; !ok {
				t.Errorf("unexpected tree entry %q", entry.Path)
			}
		}
	}

	status, body := getTree(g.BaseTree())
	if status != http.StatusOK {
		t.Fatalf("get base tree status = %d, body = %s", status, body)
	}
	checkEntries(body, map[string]string{"a/b.md": "", "top.md": ""}, false)

	status, _, blobBody := fakeGHTestRequest(
		t, g, http.MethodPost, "/repos/example/project/git/blobs",
		`{"content":"replacement\n","encoding":"utf-8"}`, "",
	)
	if status != http.StatusCreated {
		t.Fatalf("create blob status = %d, body = %s", status, blobBody)
	}
	var blob struct {
		SHA string `json:"sha"`
	}
	fakeGHTestJSON(t, blobBody, &blob)
	treeBody := fmt.Sprintf(
		`{"base_tree":%q,"tree":[{"path":"a/c.md","mode":"100644","type":"blob","sha":%q},{"path":"top.md","mode":"100644","type":"blob","sha":%q}]}`,
		g.BaseTree(), blob.SHA, blob.SHA,
	)
	status, _, treeBodyResponse := fakeGHTestRequest(
		t, g, http.MethodPost, "/repos/example/project/git/trees", treeBody, "",
	)
	if status != http.StatusCreated {
		t.Fatalf("create merged tree status = %d, body = %s", status, treeBodyResponse)
	}
	var tree struct {
		SHA string `json:"sha"`
	}
	fakeGHTestJSON(t, treeBodyResponse, &tree)
	status, body = getTree(tree.SHA)
	if status != http.StatusOK {
		t.Fatalf("get merged tree status = %d, body = %s", status, body)
	}
	checkEntries(body, map[string]string{"a/b.md": "", "a/c.md": "", "top.md": ""}, false)

	status, body = getTree(strings.Repeat("0", 40))
	if status != http.StatusNotFound || !bytes.Contains(body, []byte("Not Found")) {
		t.Fatalf("unknown tree response = %d %s", status, body)
	}
	g.TruncateTrees()
	status, body = getTree(tree.SHA)
	if status != http.StatusOK {
		t.Fatalf("get truncated tree status = %d, body = %s", status, body)
	}
	checkEntries(body, map[string]string{"a/b.md": "", "a/c.md": "", "top.md": ""}, true)
}

func TestFakeGitHubRefsPullsAndCheckRuns(t *testing.T) {
	g := newFakeGitHub(t, fakeGHOptions{
		FullName:      "example/project",
		DefaultBranch: "main",
		Token:         "fixture-token",
	})
	base := g.BaseCommit()

	refBody := fmt.Sprintf(`{"ref":"refs/heads/feature/topic","sha":%q}`, base)
	status, _, body := fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/git/refs", refBody, "fixture-token")
	if status != http.StatusCreated {
		t.Fatalf("create ref status = %d, body = %s", status, body)
	}
	status, _, body = fakeGHTestRequest(t, g, http.MethodGet, "/repos/example/project/git/ref/heads/feature/topic", "", "fixture-token")
	if status != http.StatusOK {
		t.Fatalf("get ref status = %d, body = %s", status, body)
	}
	var refResponse struct {
		Ref    string `json:"ref"`
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	fakeGHTestJSON(t, body, &refResponse)
	if refResponse.Ref != "refs/heads/feature/topic" || refResponse.Object.SHA != base {
		t.Fatalf("get ref response = %+v", refResponse)
	}
	status, _, body = fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/git/refs", refBody, "fixture-token")
	if status != http.StatusUnprocessableEntity || !bytes.Contains(body, []byte("Reference already exists")) {
		t.Fatalf("duplicate ref response = %d %s", status, body)
	}

	otherRefBody := fmt.Sprintf(`{"ref":"refs/heads/feature/other","sha":%q}`, base)
	status, _, body = fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/git/refs", otherRefBody, "fixture-token")
	if status != http.StatusCreated {
		t.Fatalf("create second ref status = %d, body = %s", status, body)
	}

	pullBody := `{"title":"First pull","head":"feature/topic","base":"main","body":"description","draft":true}`
	status, _, body = fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/pulls", pullBody, "fixture-token")
	if status != http.StatusCreated {
		t.Fatalf("create pull status = %d, body = %s", status, body)
	}
	var pullResponse fakeGHRESTPull
	fakeGHTestJSON(t, body, &pullResponse)
	if pullResponse.Number != 1 || pullResponse.State != "open" || !pullResponse.Draft ||
		pullResponse.Head.Ref != "feature/topic" || pullResponse.Head.SHA != base ||
		pullResponse.Base.Ref != "main" || pullResponse.Base.SHA != base ||
		pullResponse.HTMLURL == "" {
		t.Fatalf("created pull response = %+v", pullResponse)
	}

	secondPullBody := `{"title":"Second pull","head":"feature/other","base":"main","body":"","draft":false}`
	status, _, body = fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/pulls", secondPullBody, "fixture-token")
	if status != http.StatusCreated {
		t.Fatalf("create second pull status = %d, body = %s", status, body)
	}
	status, _, body = fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/pulls", pullBody, "fixture-token")
	if status != http.StatusUnprocessableEntity || !bytes.Contains(body, []byte("already exists")) {
		t.Fatalf("duplicate pull response = %d %s", status, body)
	}

	query := url.Values{"head": {"example:feature/topic"}, "state": {"all"}}
	status, _, body = fakeGHTestRequest(t, g, http.MethodGet, "/repos/example/project/pulls?"+query.Encode(), "", "fixture-token")
	if status != http.StatusOK {
		t.Fatalf("list pulls status = %d, body = %s", status, body)
	}
	var listedPulls []fakeGHRESTPull
	fakeGHTestJSON(t, body, &listedPulls)
	if len(listedPulls) != 1 || listedPulls[0].Number != pullResponse.Number {
		t.Fatalf("filtered pulls = %+v", listedPulls)
	}
	status, _, body = fakeGHTestRequest(t, g, http.MethodGet, "/repos/example/project/pulls?state=closed", "", "fixture-token")
	if status != http.StatusOK {
		t.Fatalf("list closed pulls status = %d, body = %s", status, body)
	}
	fakeGHTestJSON(t, body, &listedPulls)
	if len(listedPulls) != 0 {
		t.Fatalf("closed pulls = %+v, want none", listedPulls)
	}
	status, _, body = fakeGHTestRequest(t, g, http.MethodGet, "/repos/example/project/pulls/1", "", "fixture-token")
	if status != http.StatusOK {
		t.Fatalf("get pull status = %d, body = %s", status, body)
	}

	g.AddCheckRuns(base, []fakeGHCheckRun{
		{Name: "unit", Status: "completed", Conclusion: "success"},
		{Name: "lint", Status: "in_progress", Conclusion: ""},
	})
	status, _, body = fakeGHTestRequest(t, g, http.MethodGet, "/repos/example/project/commits/"+base+"/check-runs", "", "fixture-token")
	if status != http.StatusOK {
		t.Fatalf("get check runs status = %d, body = %s", status, body)
	}
	var checks struct {
		TotalCount int `json:"total_count"`
		CheckRuns  []struct {
			Name       string `json:"name"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
		} `json:"check_runs"`
	}
	fakeGHTestJSON(t, body, &checks)
	if checks.TotalCount != 2 || len(checks.CheckRuns) != 2 ||
		checks.CheckRuns[0].Name != "unit" || checks.CheckRuns[0].Conclusion != "success" ||
		checks.CheckRuns[1].Status != "in_progress" {
		t.Fatalf("check runs = %+v", checks)
	}
}

func TestFakeGitHubAuthAndWriteFailures(t *testing.T) {
	g := newFakeGitHub(t, fakeGHOptions{
		FullName:      "example/project",
		DefaultBranch: "main",
		Token:         "secret",
	})
	base := g.BaseCommit()
	initialRequests := len(g.Requests())

	status, _, body := fakeGHTestRequest(t, g, http.MethodGet, "/repos/example/project", "", "wrong-token")
	if status != http.StatusUnauthorized || !bytes.Contains(body, []byte("Bad credentials")) {
		t.Fatalf("unauthorized response = %d %s", status, body)
	}
	if len(g.Requests()) != initialRequests || g.Writes() != 0 {
		t.Fatalf("auth-rejected request was counted: requests=%v writes=%d", g.Requests(), g.Writes())
	}

	failedRef := fmt.Sprintf(`{"ref":"refs/heads/fail","sha":%q}`, base)
	g.FailWrite("POST /repos/example/project/git/refs", http.StatusUnprocessableEntity, `{"message":"injected failure"}`)
	status, _, body = fakeGHTestRequest(t, g, http.MethodPost, "/repos/example/project/git/refs", failedRef, "secret")
	if status != http.StatusUnprocessableEntity || !bytes.Contains(body, []byte("injected failure")) {
		t.Fatalf("injected failure response = %d %s", status, body)
	}
	if _, ok := g.Ref("refs/heads/fail"); ok {
		t.Fatal("FailWrite applied the ref mutation")
	}

	droppedRef := fmt.Sprintf(`{"ref":"refs/heads/dropped","sha":%q}`, base)
	g.DropAfterWrite("POST /git/refs")
	request, err := http.NewRequest(http.MethodPost, g.Server.URL+"/repos/example/project/git/refs", strings.NewReader(droppedRef))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("User-Agent", "fake-github-test")
	request.Header.Set("Authorization", "Bearer secret")
	response, err := http.DefaultClient.Do(request)
	if response != nil {
		_ = response.Body.Close()
	}
	if err == nil {
		t.Fatal("DropAfterWrite returned an HTTP response; want a transport error")
	}
	if sha, ok := g.Ref("refs/heads/dropped"); !ok || sha != base {
		t.Fatalf("DropAfterWrite did not apply ref mutation: sha=%q exists=%t", sha, ok)
	}
	if g.Writes() != 2 {
		t.Fatalf("Writes() = %d, want 2 accepted write requests", g.Writes())
	}
}

func TestFakeGitHubBaseMovedKeepsPinnedCommit(t *testing.T) {
	g := newFakeGitHub(t, fakeGHOptions{
		FullName:      "example/project",
		DefaultBranch: "main",
		BaseFiles: map[string]string{
			"nested/file.txt": "base content\n",
		},
	})
	baseCommit := g.BaseCommit()
	baseTree := g.BaseTree()
	before, ok := g.Ref("refs/heads/main")
	if !ok || before != baseCommit {
		t.Fatalf("initial default branch ref = %q, exists=%t; want %q", before, ok, baseCommit)
	}
	moved := g.BaseMoved()
	after, ok := g.Ref("refs/heads/main")
	if !ok || after != moved || moved == baseCommit {
		t.Fatalf("moved default branch ref = %q, exists=%t; BaseMoved() = %q", after, ok, moved)
	}
	for _, commit := range []string{baseCommit, moved} {
		tree, exists := g.TreeSHA(commit)
		if !exists || tree != baseTree {
			t.Fatalf("TreeSHA(%s) = %q, exists=%t; want %q", commit, tree, exists, baseTree)
		}
		if !g.CommitExists(commit) {
			t.Fatalf("commit %s became unreadable", commit)
		}
	}
}
