package evolve

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zeithrold/tools/internal/githubauth"
)

// hostFoldsNames reports whether dir stores a newly created "CaseProbe" as an
// existing "caseprobe", i.e. whether this volume can distinguish two names that
// differ only in case.
func hostFoldsNames(t *testing.T, dir string) bool {
	t.Helper()
	probe := filepath.Join(dir, "CaseProbe")
	if err := os.WriteFile(probe, []byte("probe"), 0o644); err != nil {
		t.Fatalf("case probe: %v", err)
	}
	defer os.Remove(probe)
	_, err := os.Stat(filepath.Join(dir, "caseprobe"))
	return err == nil
}

// addCaseCollidingBasePath puts a second path into the pinned tree that differs
// from the existing README.md only in case. Git stores both; the host filesystem may
// not.
func addCaseCollidingBasePath(t *testing.T, h *harness) {
	t.Helper()
	blobFile := filepath.Join(t.TempDir(), "blob")
	writeFile(t, blobFile, "lower-case sibling content\n")
	blob := runGit(t, h.upstream, "hash-object", "-w", blobFile)
	runGit(t, h.upstream, "-c", "core.ignorecase=false", "update-index", "--add", "--cacheinfo", "100644", blob, "readme.md")
	runGit(t, h.upstream, "commit", "--quiet", "-m", "add a case-colliding path")
	h.commit = runGit(t, h.upstream, "rev-parse", "HEAD")
}

// TestCaseCollidingBasePathsAreRefusedOrFaithfullyMaterialised pins both legal
// outcomes: a host that cannot distinguish the names must refuse before executing,
// and a host that can must run against a workspace holding both files.
func TestCaseCollidingBasePathsAreRefusedOrFaithfullyMaterialised(t *testing.T) {
	h := newHarness(t)
	addCaseCollidingBasePath(t, h)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	candidate := h.candidatePath(h.mustPrepare())
	validation := filepath.Join(t.TempDir(), "validation.json")
	writeJSONFile(t, validation, validationDoc(
		validationCheck("go-test", 60, helperArgv("TestEvolveHelperPass")...),
		validationCheck("repo-lint", 60, helperArgv("TestEvolveHelperPass")...),
	))
	folds := hostFoldsNames(t, t.TempDir())

	result, err := Verify(context.Background(), CheckOptions{
		Candidate: candidate, Policy: h.policy, Upstream: h.upstream,
		Validation: validation, Root: h.workspace, TrustedLocalExecution: true,
	})
	if folds {
		if err == nil {
			t.Fatalf("a filesystem that folds README.md and readme.md executed the contract anyway: %+v", result)
		}
		if !strings.Contains(err.Error(), "filesystem") {
			t.Fatalf("refusal %q does not explain the host filesystem limit", err)
		}
		_, candidateID, readErr := readCandidate(candidate)
		if readErr != nil {
			t.Fatal(readErr)
		}
		loaded, loadErr := LoadValidation(validation)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		validationDigest, digestErr := loaded.digest()
		if digestErr != nil {
			t.Fatal(digestErr)
		}
		if _, statErr := os.Lstat(filepath.Join(h.workspace, artifactRoot, "checks", candidateID, validationDigest, "result.json")); !os.IsNotExist(statErr) {
			t.Fatalf("a refused materialisation left a verification result behind: %v", statErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("a case-sensitive host refused a tree it can hold faithfully: %v", err)
	}
	if result.Status != executionPassed {
		t.Fatalf("unexpected status: %+v", result)
	}
	tree := filepath.Join(filepath.Dir(filepath.Join(h.workspace, filepath.FromSlash(result.ResultPath))), "tree")
	upper, err := os.ReadFile(filepath.Join(tree, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	lower, err := os.ReadFile(filepath.Join(tree, "readme.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(upper) != baseFileSet()["README.md"] {
		t.Fatalf("README.md holds %q", upper)
	}
	if string(lower) != "lower-case sibling content\n" {
		t.Fatalf("readme.md holds %q", lower)
	}
}

func TestReuseRechecksTheRetainedWorkspaceAndLogs(t *testing.T) {
	h := newHarness(t)
	opts, first := verifyFixture(t, h)
	run := filepath.Dir(filepath.Join(h.workspace, filepath.FromSlash(first.ResultPath)))
	writeFile(t, filepath.Join(run, "tree", "README.md"), "changed after verification\n")
	if err := os.Remove(filepath.Join(run, filepath.FromSlash(first.Checks[0].Log))); err != nil {
		t.Fatal(err)
	}

	again, err := Verify(context.Background(), opts)
	if err != nil {
		t.Fatalf("check after tampering: %v", err)
	}
	if again.Reused {
		t.Fatal("a passed result was reused after the workspace changed and a retained log was deleted")
	}
}

func TestReuseRejectsAContradictoryStepRecord(t *testing.T) {
	h := newHarness(t)
	opts, first := verifyFixture(t, h)
	location := filepath.Join(h.workspace, filepath.FromSlash(first.ResultPath))
	data, err := os.ReadFile(location)
	if err != nil {
		t.Fatal(err)
	}
	var record CheckResult
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	record.Checks[0].Status = executionFailed
	record.Checks[0].ExitCode = 17
	edited, err := canonicalJSON(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(location, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	again, err := Verify(context.Background(), opts)
	if err != nil {
		t.Fatalf("check after editing the record: %v", err)
	}
	if again.Reused {
		t.Fatalf("reused an overall %s record whose first step is %s exit %d", again.Status, again.Checks[0].Status, again.Checks[0].ExitCode)
	}
}

// verifyFixture prepares one candidate and runs a passing verification.
func verifyFixture(t *testing.T, h *harness) (CheckOptions, CheckResult) {
	t.Helper()
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	options := CheckOptions{
		Candidate:             h.candidatePath(h.mustPrepare()),
		Policy:                h.policy,
		Upstream:              h.upstream,
		Validation:            filepath.Join(t.TempDir(), "validation.json"),
		Root:                  h.workspace,
		TrustedLocalExecution: true,
	}
	writeJSONFile(t, options.Validation, validationDoc(
		validationCheck("go-test", 60, helperArgv("TestEvolveHelperPass")...),
		validationCheck("repo-lint", 60, helperArgv("TestEvolveHelperPass")...),
	))
	result, err := Verify(context.Background(), options)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Status != executionPassed {
		t.Fatalf("fixture verification did not pass: %+v", result)
	}
	return options, result
}

func TestMalformedJournalFailsBeforeTheCredentialSeam(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	preview := s.showPayload(t)
	journalPath := filepath.Join(s.workspace, artifactRoot, submitJournalRelative(preview.CandidateID, preview.PublicPayloadDigest))
	if err := os.MkdirAll(filepath.Dir(journalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(journalPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	_, err := Submit(context.Background(), SubmitOptions{
		Candidate: s.candidateFile(t), Policy: s.policy, Upstream: s.upstream,
		Root: s.workspace,
		OpenAPI: func(context.Context, Target) (githubauth.API, error) {
			called = true
			return githubauth.API{BaseURL: s.remote.Server.URL, Transport: fakeBearer{token: testRemoteToken}}, nil
		},
	})
	if err == nil {
		t.Fatal("a malformed journal was accepted")
	}
	if called {
		t.Fatalf("the credential seam was opened before the journal was rejected: %v", err)
	}
}

// TestEditedPullRequestIsStillRecognised pins the recognition rule: an operator who
// edits the pull request title or body, or marks the draft ready for review, has
// changed the review surface rather than the proposal. The submission is still this
// candidate's — the code identity decides — so it is recognised without a write and
// nobody's edit is rewritten.
func TestEditedPullRequestIsStillRecognised(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	first, err := s.submit(t)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	s.remote.mu.Lock()
	s.remote.pulls[0].Title = "Reworded by the reviewer"
	s.remote.pulls[0].Body = "    " + s.remote.pulls[0].Body
	s.remote.pulls[0].Draft = false
	s.remote.mu.Unlock()

	writes := s.remote.Writes()
	repeated, err := s.submit(t)
	if err != nil {
		t.Fatalf("submit did not recognise its own edited pull request: %v", err)
	}
	if repeated.Action != "existing" || repeated.Writes != 0 {
		t.Fatalf("an edited pull request was not recognised as this submission: %+v", repeated)
	}
	if repeated.PullRequest == nil || repeated.PullRequest.Number != first.PullRequest.Number {
		t.Fatalf("a different pull request was reported: %+v", repeated.PullRequest)
	}
	if repeated.PullRequest.Draft {
		t.Fatal("a ready pull request was reported as a draft")
	}
	if len(repeated.Notes) != 1 || repeated.Notes[0] != "the existing pull request was recognized; no remote writes were made" {
		t.Fatalf("recognition did not report the existing pull request accurately: %v", repeated.Notes)
	}
	if s.remote.Writes() != writes {
		t.Fatal("recognition rewrote the remote side")
	}
	s.remote.mu.Lock()
	title, draft := s.remote.pulls[0].Title, s.remote.pulls[0].Draft
	s.remote.mu.Unlock()
	if title != "Reworded by the reviewer" || draft {
		t.Fatalf("the operator's edit was rewritten: title=%q draft=%t", title, draft)
	}

	// Status reports the same object, and the edit is not drift.
	observed, err := Status(context.Background(), StatusOptions{
		Candidate: s.candidateFile(t), Policy: s.policy, Upstream: s.upstream,
		Root: s.workspace, Remote: true,
		OpenAPI: func(context.Context, Target) (githubauth.API, error) {
			return githubauth.API{BaseURL: s.remote.Server.URL, Transport: fakeBearer{token: testRemoteToken}}, nil
		},
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if observed.Remote == nil || len(observed.Remote.Drift) != 0 {
		t.Fatalf("an edited pull request was reported as drift: %+v", observed.Remote)
	}
	if s.remote.Writes() != writes {
		t.Fatal("status wrote to the remote")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestJournalRefusesToDowngradeACompletedSubmission(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	prepared := h.mustPrepare()
	store := mustStore(t, h.workspace)
	relative := submitJournalRelative(prepared.CandidateID, strings.Repeat("1", 64))
	journal := Journal{
		SchemaVersion: schemaVersion, CandidateID: prepared.CandidateID,
		PublicPayloadDigest: strings.Repeat("1", 64),
		Branch:              "evolve/fixture-skill-000000000000",
		BaseCommit:          h.commit,
	}
	if err := record(context.Background(), store, relative, &journal, journalRefCreated, "branch created"); err != nil {
		t.Fatal(err)
	}
	completed := journal
	completed.PullRequest = &JournalPullRequest{Number: 7, URL: "https://example.invalid/pull/7"}
	if err := record(context.Background(), store, relative, &completed, journalCompleted, "pull request created"); err != nil {
		t.Fatal(err)
	}

	stale := journal // the view a slower, older attempt still holds
	if err := advanceJournal(context.Background(), store, relative, &stale, journalStarted, "stale attempt"); err == nil {
		t.Fatal("a stale attempt overwrote a completed journal")
	}
	stored, found, err := loadJournal(store, relative, prepared.CandidateID, journal.PublicPayloadDigest)
	if err != nil || !found {
		t.Fatalf("journal: %v %v", found, err)
	}
	if stored.State != journalCompleted || stored.PullRequest == nil || stored.PullRequest.Number != 7 {
		t.Fatalf("completed journal was downgraded: %+v", stored)
	}
}

// gatedTransport pauses the first commit creation, so a test can run a second
// submission while the first holds the transaction, then fail the first one before
// its pull request exists.
type gatedTransport struct {
	reached chan struct{}
	resume  chan struct{}
	failPR  bool

	once   sync.Once
	mu     sync.Mutex
	paused bool
}

func (g *gatedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/git/commits") {
		g.mu.Lock()
		first := !g.paused
		g.paused = true
		g.mu.Unlock()
		if first {
			g.once.Do(func() { close(g.reached) })
			select {
			case <-g.resume:
			case <-request.Context().Done():
				return nil, request.Context().Err()
			}
		}
	}
	if request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/pulls") && g.failPR {
		return nil, errors.New("synthetic connection loss before the pull request response")
	}
	return (fakeBearer{token: testRemoteToken}).RoundTrip(request)
}

func TestOverlappingSubmissionsSerialiseAndKeepTheCompletedJournal(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	gate := &gatedTransport{reached: make(chan struct{}), resume: make(chan struct{}), failPR: true}

	second := make(chan error, 1)
	go func() {
		_, err := Submit(context.Background(), SubmitOptions{
			Candidate: s.candidateFile(t), Policy: s.policy, Upstream: s.upstream,
			Root: s.workspace,
			OpenAPI: func(context.Context, Target) (githubauth.API, error) {
				return githubauth.API{BaseURL: s.remote.Server.URL, Transport: gate}, nil
			},
		})
		second <- err
	}()
	select {
	case <-gate.reached:
	case err := <-second:
		t.Fatalf("the overlapping attempt failed before its gated write: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("the overlapping attempt never reached its gated write")
	}

	// A second submission starts now. It must wait for the transaction, then reconcile
	// instead of writing its own objects.
	first := make(chan error, 1)
	go func() {
		_, err := s.submit(t)
		first <- err
	}()
	time.Sleep(500 * time.Millisecond)
	close(gate.resume)

	if err := <-second; err == nil || !strings.Contains(err.Error(), "outcome unknown") {
		t.Fatalf("the interrupted attempt did not report an uncertain outcome: %v", err)
	}
	select {
	case err := <-first:
		if err != nil {
			t.Fatalf("the overlapping submission could not complete: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the overlapping submission never finished")
	}

	if pulls := s.remote.Pulls(); len(pulls) != 1 {
		t.Fatalf("overlapping submissions created %d pull requests", len(pulls))
	}
	store := mustStore(t, s.workspace)
	preview := s.showPayload(t)
	journal, found, err := loadJournal(store, submitJournalRelative(preview.CandidateID, preview.PublicPayloadDigest), preview.CandidateID, preview.PublicPayloadDigest)
	if err != nil || !found {
		t.Fatalf("journal: %v %v", found, err)
	}
	if journal.State != journalCompleted || journal.PullRequest == nil {
		t.Fatalf("the completed journal was downgraded by the overlapping attempt: %+v", journal)
	}
}

// TestTreeListingUsesTheGitHubWireKey pins the wire contract on a literal body: the
// listing key is GitHub's "tree", a truncated listing is refused, and the
// non-standard "entries" spelling is not silently accepted as an empty tree.
func TestTreeListingUsesTheGitHubWireKey(t *testing.T) {
	blob := strings.Repeat("b", 40)
	body := `{"sha":"` + strings.Repeat("a", 40) + `","tree":[{"path":"README.md","mode":"100644","type":"blob","sha":"` + blob + `","size":7}],"truncated":false}`
	endpoint := literalEndpoint(t, body)
	entries, err := endpoint.treeEntries(context.Background(), strings.Repeat("a", 40))
	if err != nil {
		t.Fatalf("literal GitHub tree response was refused: %v", err)
	}
	if len(entries) != 1 || entries["README.md"].SHA != blob || entries["README.md"].Mode != "100644" {
		t.Fatalf("literal GitHub tree response decoded as %+v", entries)
	}

	legacy := literalEndpoint(t, `{"sha":"`+strings.Repeat("a", 40)+`","entries":[{"path":"README.md","mode":"100644","type":"blob","sha":"`+blob+`"}]}`)
	legacyEntries, err := legacy.treeEntries(context.Background(), strings.Repeat("a", 40))
	if err != nil {
		t.Fatalf("legacy body: %v", err)
	}
	if len(legacyEntries) != 0 {
		t.Fatal("the non-standard entries key was still honoured")
	}

	truncated := literalEndpoint(t, `{"sha":"`+strings.Repeat("a", 40)+`","tree":[],"truncated":true}`)
	if _, err := truncated.treeEntries(context.Background(), strings.Repeat("a", 40)); err == nil {
		t.Fatal("a truncated tree listing was accepted")
	}
}

func literalEndpoint(t *testing.T, body string) *githubEndpoint {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	endpoint, err := newGitHubEndpoint(githubauth.API{BaseURL: server.URL, Transport: http.DefaultTransport}, Target{
		Host: "github.com", Repository: "example/fixture", BaseCommit: strings.Repeat("a", 40),
	})
	if err != nil {
		t.Fatal(err)
	}
	return endpoint
}

// rewriteBaseDigest replaces the recorded base-set digest with a well-formed but
// wrong value, keeping the record canonical.
func rewriteBaseDigest(t *testing.T, workspace string, record CheckResult) {
	t.Helper()
	location := filepath.Join(workspace, filepath.FromSlash(record.ResultPath))
	data, err := os.ReadFile(location)
	if err != nil {
		t.Fatal(err)
	}
	var decoded CheckResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded.BaseTreeDigest = strings.Repeat("0", 64)
	encoded, err := canonicalJSON(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(location, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRecordWithAForeignBaseDigestIsNotReused(t *testing.T) {
	h := newHarness(t)
	options, first := verifyFixture(t, h)
	rewriteBaseDigest(t, h.workspace, first)

	again, err := Verify(context.Background(), options)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if again.Reused {
		t.Fatal("a record naming a foreign pinned base was reused")
	}
	if again.BaseTreeDigest != first.BaseTreeDigest {
		t.Fatalf("the re-run recorded a different base: %s", again.BaseTreeDigest)
	}
}

// TestRecoveredBranchIsCompletedAfterTheDefaultBranchAdvanced pins that recovery from a
// lost ref response still completes when the default branch moved meanwhile: the branch
// already holds this candidate's exact commit, so only the pull request is missing and
// zt creates only that, against the current default branch.
func TestRecoveredBranchIsCompletedAfterTheDefaultBranchAdvanced(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.remote.DropAfterWrite("/git/refs")
	first, err := s.submit(t)
	if err == nil || first.State != "uncertain" {
		t.Fatalf("setup: %+v %v", first, err)
	}
	if _, ok := s.remote.Ref("refs/heads/" + first.Branch); !ok {
		t.Fatal("setup: the interrupted run did not create the branch")
	}
	s.remote.BaseMoved()
	writes := s.remote.Writes()

	recovered, err := s.submit(t)
	if err != nil {
		t.Fatalf("recovery could not complete its own branch: %v", err)
	}
	if recovered.Action != "created" || recovered.PullRequest == nil {
		t.Fatalf("unexpected recovery result: %+v", recovered)
	}
	if added := s.remote.Writes() - writes; added != 1 {
		t.Fatalf("recovery performed %d remote writes; only the pull request was missing", added)
	}
	if head, ok := s.remote.Ref("refs/heads/" + recovered.Branch); !ok || head != first.Commit {
		t.Fatalf("recovery changed the branch: %s %v", head, ok)
	}
	if pulls := s.remote.Pulls(); len(pulls) != 1 || pulls[0].Base != "main" {
		t.Fatalf("unexpected pull requests: %+v", pulls)
	}
}

// TestCommitVerificationComparesTheGitObjectNotTheRestMessage pins the live
// compatibility rule: GitHub's REST message view drops one trailing newline while the
// stored object keeps the exact bytes. Verification must rebuild the raw
// commit object and compare object ids, so a presentation difference passes only when
// the object really is the same one, and a substantive difference fails.
func TestCommitVerificationComparesTheGitObjectNotTheRestMessage(t *testing.T) {
	upstream, baseCommit := newUpstream(t, baseFileSet())
	repo, err := OpenRepo(context.Background(), upstream)
	if err != nil {
		t.Fatal(err)
	}
	payload := PublicPayload{Commit: PayloadCommit{
		Message:     "evolve(fixture-skill): candidate 0123456789ab\n\nCandidate-Id: abc\n",
		AuthorName:  evolveAuthorName,
		AuthorEmail: evolveAuthorEmail,
		CommittedAt: "2026-10-26T06:36:37Z",
	}}
	tree := strings.Repeat("d", 40)
	raw, err := rawCommitObject(tree, baseCommit, payload.Commit)
	if err != nil {
		t.Fatal(err)
	}
	approvedSHA, err := repo.CommitID(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	author := map[string]string{"name": payload.Commit.AuthorName, "email": payload.Commit.AuthorEmail, "date": payload.Commit.CommittedAt}
	serve := func(sha, message, treeSHA, parent string, identity map[string]string) *githubEndpoint {
		t.Helper()
		body, err := json.Marshal(map[string]any{
			"sha":       sha,
			"tree":      map[string]string{"sha": treeSHA},
			"message":   message,
			"parents":   []map[string]string{{"sha": parent}},
			"author":    identity,
			"committer": identity,
		})
		if err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(request.URL.Path, "/git/trees/") {
				_, _ = io.WriteString(w, `{"sha":"`+treeSHA+`","tree":[],"truncated":false}`)
				return
			}
			_, _ = w.Write(body)
		}))
		t.Cleanup(server.Close)
		endpoint, err := newGitHubEndpoint(githubauth.API{BaseURL: server.URL, Transport: http.DefaultTransport}, Target{
			Host: "github.com", Repository: "example/fixture", BaseCommit: baseCommit,
		})
		if err != nil {
			t.Fatal(err)
		}
		return endpoint
	}
	check := func(endpoint *githubEndpoint, sha string) error {
		t.Helper()
		observed, err := endpoint.commitObject(context.Background(), sha)
		if err != nil {
			return err
		}
		return endpoint.verifyRealizedCommit(context.Background(), repo, observed, payload, baseCommit, map[string]treeFile{})
	}

	// 1. The API view omits the final newline; the object id still matches the approved
	// raw bytes, so this is the approved commit.
	if err := check(serve(approvedSHA, strings.TrimSuffix(payload.Commit.Message, "\n"), tree, baseCommit, author), approvedSHA); err != nil {
		t.Fatalf("the real REST message presentation was refused: %v", err)
	}
	// 2. A substantively different message has a different object id; presenting it
	// with its own id must fail closed against the approved object.
	changedCommit := payload.Commit
	changedCommit.Message = "evolve(fixture-skill): candidate 0123456789ab\n\nCandidate-Id: OTHER\n"
	changedRaw, err := rawCommitObject(tree, baseCommit, changedCommit)
	if err != nil {
		t.Fatal(err)
	}
	changedSHA, err := repo.CommitID(context.Background(), changedRaw)
	if err != nil {
		t.Fatal(err)
	}
	if changedSHA == approvedSHA {
		t.Fatal("the fixture messages hash to the same object")
	}
	changed := serve(changedSHA, strings.TrimSuffix(changedCommit.Message, "\n"), tree, baseCommit, author)
	if err := check(changed, changedSHA); err == nil {
		t.Fatal("a different message was accepted as the approved commit")
	}
	// 3. A different parent.
	otherParent := strings.Repeat("e", 40)
	if err := check(serve(approvedSHA, payload.Commit.Message, tree, otherParent, author), approvedSHA); err == nil {
		t.Fatal("a different parent was accepted")
	}
	// 4. A different tree id (the object id no longer matches the approved object).
	otherTree := strings.Repeat("f", 40)
	if err := check(serve(approvedSHA, strings.TrimSuffix(payload.Commit.Message, "\n"), otherTree, baseCommit, author), approvedSHA); err == nil {
		t.Fatal("a different tree was accepted")
	}
	// 5. A different author time.
	laterAuthor := map[string]string{"name": payload.Commit.AuthorName, "email": payload.Commit.AuthorEmail, "date": "2026-10-26T06:36:38Z"}
	if err := check(serve(approvedSHA, payload.Commit.Message, tree, baseCommit, laterAuthor), approvedSHA); err == nil {
		t.Fatal("a different author time was accepted")
	}
	// 6. A different identity.
	otherIdentity := map[string]string{"name": "someone", "email": payload.Commit.AuthorEmail, "date": payload.Commit.CommittedAt}
	if err := check(serve(approvedSHA, payload.Commit.Message, tree, baseCommit, otherIdentity), approvedSHA); err == nil {
		t.Fatal("a different identity was accepted")
	}
	// 7. Unknown object drift: the reported id is not the approved commit.
	unknown := strings.Repeat("0", 40)
	if err := check(serve(unknown, payload.Commit.Message, tree, baseCommit, author), unknown); err == nil {
		t.Fatal("an unknown commit object was accepted")
	}
}
