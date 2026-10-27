package evolve

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeithrold/tools/internal/githubauth"
)

// testRemoteToken is the bearer token the fake service requires. It is a fixture
// value: no real credential is read, minted or written by any test here.
const testRemoteToken = "test-token-not-a-credential"

// fakeBearer is the test twin of the githubauth transport. The production seam
// adds the real token inside that package and never returns it.
type fakeBearer struct{ token string }

func (b fakeBearer) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(clone)
}

// The validation commands in these tests are this test binary, re-executed with a
// fixed -test.run selector. The helper only acts when zt's own metadata variable is
// present, so the selectors are inert during a normal test run.
func helperArgv(selector string) []string {
	return []string{os.Args[0], "-test.run=^" + selector + "$", "-test.timeout=60s"}
}

func TestEvolveHelperPass(t *testing.T) {
	if os.Getenv("ZT_EVOLVE_CANDIDATE_ID") == "" {
		return
	}
}

func TestEvolveHelperFail(t *testing.T) {
	if os.Getenv("ZT_EVOLVE_CANDIDATE_ID") == "" {
		return
	}
	t.Fatal("fixture validation failure")
}

func TestEvolveHelperMutate(t *testing.T) {
	if os.Getenv("ZT_EVOLVE_CANDIDATE_ID") == "" {
		return
	}
	if err := os.WriteFile(filepath.Join("skills", "fixture-skill", "SKILL.md"), []byte("rewritten by the check\n"), 0o644); err != nil {
		t.Fatalf("helper could not rewrite a tracked file: %v", err)
	}
}

// submitHarness wires a local upstream and a fake remote that describe the same base
// commit, then prepares one candidate and verifies it locally.
type submitHarness struct {
	*harness
	remote     *fakeGitHub
	validation string
	approval   string
}

// newSubmitHarness builds the fixture pair and runs prepare plus a passing check, so
// a test starts from a submission that is actually allowed to proceed.
func newSubmitHarness(t *testing.T, fullName string) *submitHarness {
	t.Helper()
	return newSubmitHarnessWithModes(t, fullName, nil)
}

// newSubmitHarnessWithModes is newSubmitHarness with explicit base file modes, so a
// fixture can pin an executable base entry that a candidate replaces by content only.
// The local upstream and the fake remote must agree on the mode, because their base
// commit object ids have to match.
func newSubmitHarnessWithModes(t *testing.T, fullName string, modes map[string]string) *submitHarness {
	t.Helper()
	files := baseFileSet()
	remote := newFakeGitHub(t, fakeGHOptions{
		FullName:      fullName,
		DefaultBranch: "main",
		BaseFiles:     files,
		BaseModes:     modes,
		Token:         testRemoteToken,
	})
	upstream, commit := upstreamMatchingRemote(t, files, modes, remote)
	h := &harness{
		t:         t,
		upstream:  upstream,
		commit:    commit,
		workspace: t.TempDir(),
		policy:    filepath.Join(t.TempDir(), "policy.json"),
		proposal:  filepath.Join(t.TempDir(), "proposal.json"),
	}
	h.writePolicy(policyDoc(commit))
	h.writeProposal(proposalDoc(commit, defaultProposalFiles()...))
	h.mustPrepare()
	validation := filepath.Join(t.TempDir(), "validation.json")
	writeJSONFile(t, validation, validationDoc(
		validationCheck("go-test", 60, helperArgv("TestEvolveHelperPass")...),
		validationCheck("repo-lint", 60, helperArgv("TestEvolveHelperPass")...),
	))
	return &submitHarness{harness: h, remote: remote, validation: validation}
}

// upstreamMatchingRemote builds a real local repository whose HEAD commit is exactly
// the fake remote's base commit, so prepare, plan and JSON reads describe one tree.
func upstreamMatchingRemote(t *testing.T, files, modes map[string]string, remote *fakeGitHub) (string, string) {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet")
	for name, content := range files {
		writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), content)
	}
	runGit(t, dir, "add", "--all")
	pinIndexModes(t, dir, modes)
	name, email := remote.BaseCommitIdentity()
	date := remote.BaseCommittedAt().UTC().Format("2006-01-02T15:04:05-07:00")
	command := exec.Command("git", "-C", dir,
		"-c", "user.name="+name, "-c", "user.email="+email, "-c", "commit.gpgsign=false",
		"commit", "--quiet", "-m", remote.BaseCommitMessage())
	command.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("fixture commit failed: %v\n%s", err, output)
	}
	commit := runGit(t, dir, "rev-parse", "HEAD")
	if commit != remote.BaseCommit() {
		t.Fatalf("local fixture commit %s does not match the fake remote base %s", commit, remote.BaseCommit())
	}
	return dir, commit
}

// pinIndexModes records a declared executable mode in the Git index explicitly.
// os.Chmod is not used: with core.filemode=false — the usual setting on Windows
// hosts — Git ignores the physical executable bit, so a chmod-based fixture would
// silently describe a 100644 base instead of the 100755 base a mode regression needs.
// Paths must already be in the index; this runs after git add.
func pinIndexModes(t *testing.T, dir string, modes map[string]string) {
	t.Helper()
	for name, mode := range modes {
		flag := "--chmod=-x"
		if mode == "100755" {
			flag = "--chmod=+x"
		}
		runGit(t, dir, "update-index", flag, "--", name)
	}
}

// candidateFile prepares again (idempotently) and returns the stored candidate path.
func (s *submitHarness) candidateFile(t *testing.T) string {
	t.Helper()
	return s.candidatePath(s.mustPrepare())
}

// showPayload previews the publication with the harness validation contract.
func (s *submitHarness) showPayload(t *testing.T) SubmitResult {
	t.Helper()
	result, err := Submit(context.Background(), SubmitOptions{
		Candidate:   s.candidateFile(t),
		Policy:      s.policy,
		Upstream:    s.upstream,
		Validation:  s.validation,
		Root:        s.workspace,
		ShowPayload: true,
		OpenAPI: func(context.Context, Target) (githubauth.API, error) {
			t.Fatal("--show-payload opened the credential seam")
			return githubauth.API{}, nil
		},
	})
	if err != nil {
		t.Fatalf("show payload: %v", err)
	}
	return result
}

// verify runs `evolve check --trusted-local-execution` for this harness.
func (s *submitHarness) verify(t *testing.T) CheckResult {
	t.Helper()
	result, err := Verify(context.Background(), CheckOptions{
		Candidate:             s.candidateFile(t),
		Policy:                s.policy,
		Upstream:              s.upstream,
		Validation:            s.validation,
		Root:                  s.workspace,
		TrustedLocalExecution: true,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	return result
}

// approve writes an approval naming the payload preview for the current inputs.
func (s *submitHarness) approve(t *testing.T) {
	t.Helper()
	preview := s.showPayload(t)
	s.approval = filepath.Join(t.TempDir(), "approval.json")
	writeJSONFile(t, s.approval, approvalFor(preview, nil))
}

func (s *submitHarness) submit(t *testing.T) (SubmitResult, error) {
	t.Helper()
	return Submit(context.Background(), SubmitOptions{
		Candidate:  s.candidateFile(t),
		Policy:     s.policy,
		Upstream:   s.upstream,
		Validation: s.validation,
		Approval:   s.approval,
		Root:       s.workspace,
		OpenAPI: func(context.Context, Target) (githubauth.API, error) {
			return githubauth.API{BaseURL: s.remote.Server.URL, Transport: fakeBearer{token: testRemoteToken}}, nil
		},
	})
}

func TestSubmitCreatesOneDraftPullRequest(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	if result := s.verify(t); result.Status != executionPassed {
		t.Fatalf("fixture verification did not pass: %+v", result)
	}
	s.approve(t)
	before := snapshotTree(t, s.workspace)

	result, err := s.submit(t)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if result.Action != "created" || result.State != "completed" {
		t.Fatalf("unexpected outcome: %+v", result)
	}
	if result.PullRequest == nil || !result.PullRequest.Draft || result.PullRequest.State != "open" {
		t.Fatalf("no open draft pull request: %+v", result.PullRequest)
	}
	pulls := s.remote.Pulls()
	if len(pulls) != 1 {
		t.Fatalf("expected one remote pull request, got %d", len(pulls))
	}
	if pulls[0].Head != result.Branch || pulls[0].Base != "main" || !pulls[0].Draft {
		t.Fatalf("unexpected remote pull request: %+v", pulls[0])
	}
	if head, ok := s.remote.Ref("refs/heads/" + result.Branch); !ok || head != result.Commit {
		t.Fatalf("remote branch does not hold the created commit: %s %v", head, ok)
	}
	if result.JournalPath == "" {
		t.Fatal("submit did not report its journal")
	}
	journal, found, err := loadJournal(mustStore(t, s.workspace), submitJournalRelative(result.CandidateID, result.PublicPayloadDigest), result.CandidateID, result.PublicPayloadDigest)
	if err != nil || !found {
		t.Fatalf("journal: %v %v", found, err)
	}
	if journal.State != journalCompleted || journal.PullRequest == nil {
		t.Fatalf("journal did not record completion: %+v", journal)
	}
	after := snapshotTree(t, s.workspace)
	for name := range after {
		if _, existed := before[name]; !existed {
			if strings.Contains(name, "candidate.json") {
				t.Fatalf("submit rewrote a candidate: %s", name)
			}
		}
	}
	if err := assertNoSecretInTree(s.workspace); err != nil {
		t.Fatal(err)
	}
}

func mustStore(t *testing.T, workspace string) *Store {
	t.Helper()
	store, err := openStore(workspace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// assertNoSecretInTree proves the fixture credential never reached an owned artifact.
func assertNoSecretInTree(dir string) error {
	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), testRemoteToken) {
			return fmt.Errorf("the credential appears in %s", path)
		}
		return nil
	})
}

func TestSubmitReplaysWithoutRewritingAnything(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	first, err := s.submit(t)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	writes := s.remote.Writes()

	second, err := s.submit(t)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if second.Action != "existing" || second.Writes != 0 {
		t.Fatalf("replay wrote or claimed a new submission: %+v", second)
	}
	if second.Commit != first.Commit || second.PullRequest == nil || second.PullRequest.Number != first.PullRequest.Number {
		t.Fatalf("replay returned a different pull request: %+v", second)
	}
	if s.remote.Writes() != writes {
		t.Fatalf("replay wrote to the remote: %d -> %d", writes, s.remote.Writes())
	}
}

func TestSubmitReturnsAnExactExistingPullRequestEvenAfterTheBaseMoved(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	first, err := s.submit(t)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	writes := s.remote.Writes()
	s.remote.BaseMoved()

	second, err := s.submit(t)
	if err != nil {
		t.Fatalf("submit after the base moved: %v", err)
	}
	if second.Action != "existing" || second.PullRequest == nil || second.PullRequest.Number != first.PullRequest.Number {
		t.Fatalf("a moved base changed the reported submission: %+v", second)
	}
	if s.remote.Writes() != writes {
		t.Fatal("a reconciled submission wrote to the remote")
	}
}

func TestSubmitRefusesANewWriteAgainstAMovedBase(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	s.remote.BaseMoved()

	_, err := s.submit(t)
	if err == nil || !strings.Contains(err.Error(), "has moved") {
		t.Fatalf("submit accepted a moved base: %v", err)
	}
	if s.remote.Writes() != 0 {
		t.Fatalf("submit wrote %d objects against a moved base", s.remote.Writes())
	}
}

func TestSubmitNeverReopensOrAdoptsAClosedPullRequest(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	preview := s.showPayload(t)
	s.remote.AttachPull(fakeGHPull{
		Number:  7,
		Title:   preview.Payload.PullRequest.Title,
		Head:    preview.Payload.Branch,
		HeadSHA: strings.Repeat("0", 40),
		Base:    "main",
		State:   "closed",
		Draft:   true,
	})

	_, err := s.submit(t)
	if err == nil || !strings.Contains(err.Error(), "closed pull request") {
		t.Fatalf("submit did not refuse a closed pull request: %v", err)
	}
	if s.remote.Writes() != 0 {
		t.Fatal("submit wrote while a closed pull request existed")
	}
}

func TestLostRefResponseIsRecoveredByCreatingOnlyThePullRequest(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	s.remote.DropAfterWrite("/git/refs")

	result, err := s.submit(t)
	if err == nil || !strings.Contains(err.Error(), "outcome unknown") {
		t.Fatalf("an interrupted write was not reported as uncertain: %v", err)
	}
	if result.State != "uncertain" {
		t.Fatalf("unexpected reported state: %+v", result)
	}
	store := mustStore(t, s.workspace)
	journal, found, err := loadJournal(store, submitJournalRelative(result.CandidateID, result.PublicPayloadDigest), result.CandidateID, result.PublicPayloadDigest)
	if err != nil || !found {
		t.Fatalf("no journal after an interrupted write: %v %v", found, err)
	}
	if journal.State != journalObjectsCreated && journal.State != journalStarted {
		t.Fatalf("journal state %q does not describe an interrupted write", journal.State)
	}
	if journal.PullRequest != nil {
		t.Fatal("an interrupted write recorded a pull request it never observed")
	}
	if len(s.remote.Pulls()) != 0 {
		t.Fatal("the interrupted run created a pull request")
	}

	// The branch really exists now. Recovery must verify it and create only the
	// missing draft pull request: no ref write, no force, no second branch.
	writes := s.remote.Writes()
	recovered, err := s.submit(t)
	if err != nil {
		t.Fatalf("the retry could not complete its own exact branch: %v", err)
	}
	if recovered.Action != "created" || recovered.PullRequest == nil {
		t.Fatalf("unexpected recovery result: %+v", recovered)
	}
	if pulls := s.remote.Pulls(); len(pulls) != 1 {
		t.Fatalf("recovery created %d pull requests", len(pulls))
	}
	if added := s.remote.Writes() - writes; added != 1 {
		t.Fatalf("recovery performed %d remote writes; only the pull request was missing", added)
	}
	if head, ok := s.remote.Ref("refs/heads/" + recovered.Branch); !ok || head != recovered.Commit {
		t.Fatalf("recovery changed the branch: %s %v", head, ok)
	}
	final, _, err := loadJournal(store, submitJournalRelative(recovered.CandidateID, recovered.PublicPayloadDigest), recovered.CandidateID, recovered.PublicPayloadDigest)
	if err != nil {
		t.Fatal(err)
	}
	if final.State != journalCompleted || final.PullRequest == nil {
		t.Fatalf("recovery did not complete the journal: %+v", final)
	}
}

func TestLostPullRequestResponseIsAdoptedWithoutADuplicate(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	s.remote.DropAfterWrite("/pulls")

	result, err := s.submit(t)
	if err == nil || !strings.Contains(err.Error(), "outcome unknown") {
		t.Fatalf("a lost pull request response was not reported as uncertain: %v", err)
	}
	if result.State != "uncertain" {
		t.Fatalf("unexpected reported state: %+v", result)
	}
	pulls := s.remote.Pulls()
	if len(pulls) != 1 {
		t.Fatalf("the interrupted run created %d pull requests", len(pulls))
	}
	writes := s.remote.Writes()

	adopted, err := s.submit(t)
	if err != nil {
		t.Fatalf("the retry could not adopt the pull request it created: %v", err)
	}
	if adopted.Action != "existing" || adopted.Writes != 0 {
		t.Fatalf("the retry did not adopt the existing pull request: %+v", adopted)
	}
	if len(s.remote.Pulls()) != 1 {
		t.Fatal("the retry created a duplicate pull request")
	}
	if s.remote.Writes() != writes {
		t.Fatal("the retry wrote to the remote")
	}
}

func TestPartialImmutableObjectsAreCompletedOnRetry(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	s.remote.DropAfterWrite("/git/commits")

	result, err := s.submit(t)
	if err == nil || !strings.Contains(err.Error(), "outcome unknown") {
		t.Fatalf("a lost commit response was not reported as uncertain: %v", err)
	}
	if result.State != "uncertain" {
		t.Fatalf("unexpected reported state: %+v", result)
	}
	if _, ok := s.remote.Ref("refs/heads/" + result.Branch); ok {
		t.Fatal("an interrupted commit created the branch")
	}

	// Only unreferenced objects exist, so a fresh, deterministic write completes the
	// loop without losing anything a reader could see.
	completed, err := s.submit(t)
	if err != nil {
		t.Fatalf("the retry could not complete the submission: %v", err)
	}
	if completed.Action != "created" || completed.PullRequest == nil {
		t.Fatalf("unexpected completion: %+v", completed)
	}
	if pulls := s.remote.Pulls(); len(pulls) != 1 {
		t.Fatalf("completion created %d pull requests", len(pulls))
	}
}

func TestSubmitRefusesAnUnavailableCredentialWithoutWriting(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	_, err := Submit(context.Background(), SubmitOptions{
		Candidate:  s.candidateFile(t),
		Policy:     s.policy,
		Upstream:   s.upstream,
		Validation: s.validation,
		Approval:   s.approval,
		Root:       s.workspace,
		OpenAPI: func(context.Context, Target) (githubauth.API, error) {
			return githubauth.API{}, fmt.Errorf("no stored credential")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "no stored credential") {
		t.Fatalf("submit ignored the credential failure: %v", err)
	}
	if s.remote.Writes() != 0 {
		t.Fatal("submit wrote without a credential")
	}
}

func TestSubmitRefusesAnotherRepositoryThanThePolicyNames(t *testing.T) {
	s := newSubmitHarness(t, "other/repository")
	s.verify(t)
	s.approve(t)
	_, err := s.submit(t)
	if err == nil || !strings.Contains(err.Error(), "cannot see repository") {
		t.Fatalf("submit accepted a different repository: %v", err)
	}
	if s.remote.Writes() != 0 {
		t.Fatal("submit wrote to a repository it could not confirm")
	}
}

func TestSubmitRefusesAFailedVerification(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	writeJSONFile(t, s.validation, validationDoc(
		validationCheck("go-test", 60, helperArgv("TestEvolveHelperFail")...),
		validationCheck("repo-lint", 60, helperArgv("TestEvolveHelperPass")...),
	))
	result, err := Verify(context.Background(), CheckOptions{
		Candidate:             s.candidateFile(t),
		Policy:                s.policy,
		Upstream:              s.upstream,
		Validation:            s.validation,
		Root:                  s.workspace,
		TrustedLocalExecution: true,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Status != executionFailed {
		t.Fatalf("a failing command was reported as %q", result.Status)
	}
	s.approve(t)
	if _, err := s.submit(t); err == nil || !strings.Contains(err.Error(), "not passed") {
		t.Fatalf("submit accepted a failed verification: %v", err)
	}
}

func TestAVerificationThatRewritesItsInputsIsNotReusable(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	writeJSONFile(t, s.validation, validationDoc(
		validationCheck("go-test", 60, helperArgv("TestEvolveHelperMutate")...),
		validationCheck("repo-lint", 60, helperArgv("TestEvolveHelperPass")...),
	))
	first, err := Verify(context.Background(), CheckOptions{
		Candidate:             s.candidateFile(t),
		Policy:                s.policy,
		Upstream:              s.upstream,
		Validation:            s.validation,
		Root:                  s.workspace,
		TrustedLocalExecution: true,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !first.WorkspaceChanged || first.Status != executionBlocked {
		t.Fatalf("a run that rewrote its inputs was reported as %+v", first)
	}
	second, err := Verify(context.Background(), CheckOptions{
		Candidate:             s.candidateFile(t),
		Policy:                s.policy,
		Upstream:              s.upstream,
		Validation:            s.validation,
		Root:                  s.workspace,
		TrustedLocalExecution: true,
	})
	if err != nil {
		t.Fatalf("second check: %v", err)
	}
	if second.Reused {
		t.Fatal("a blocked run was reused")
	}
	s.approve(t)
	if _, err := s.submit(t); err == nil {
		t.Fatal("submit accepted a blocked verification")
	}
}

func TestVerificationReusesAnIdenticalPassedRun(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	first := s.verify(t)
	if first.Reused {
		t.Fatal("the first run claimed to be reused")
	}
	second := s.verify(t)
	if !second.Reused || second.WorkspaceDigest != first.WorkspaceDigest {
		t.Fatalf("an identical run was not reused: %+v", second)
	}
}

func TestStatusObservesTheSubmittedPullRequestAndItsChecks(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	submitted, err := s.submit(t)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	s.remote.AddCheckRuns(submitted.Commit, []fakeGHCheckRun{
		{Name: "ci", Status: "completed", Conclusion: "success"},
		{Name: "lint", Status: "in_progress"},
	})

	remote := func(context.Context, Target) (githubauth.API, error) {
		return githubauth.API{BaseURL: s.remote.Server.URL, Transport: fakeBearer{token: testRemoteToken}}, nil
	}
	local, err := Status(context.Background(), StatusOptions{
		Candidate: s.candidateFile(t), Policy: s.policy, Upstream: s.upstream,
		Validation: s.validation, Root: s.workspace,
	})
	if err != nil {
		t.Fatalf("local status: %v", err)
	}
	if local.Publication != publicationSubmitted || local.Remote != nil {
		t.Fatalf("local status invented remote observations: %+v", local)
	}

	observed, err := Status(context.Background(), StatusOptions{
		Candidate: s.candidateFile(t), Policy: s.policy, Upstream: s.upstream,
		Validation: s.validation, Root: s.workspace, Remote: true, OpenAPI: remote,
	})
	if err != nil {
		t.Fatalf("remote status: %v", err)
	}
	if observed.Remote == nil || observed.Remote.PullRequest == nil {
		t.Fatalf("remote status did not observe the pull request: %+v", observed)
	}
	if observed.Remote.PullRequest.Number != submitted.PullRequest.Number || observed.Remote.HeadSHA != submitted.Commit {
		t.Fatalf("remote status reported another head: %+v", observed.Remote)
	}
	if observed.Remote.CheckRuns == nil || observed.Remote.CheckRuns.Total != 2 || observed.Remote.CheckRuns.Success != 1 || observed.Remote.CheckRuns.Pending != 1 {
		t.Fatalf("unexpected check observation: %+v", observed.Remote.CheckRuns)
	}
	if len(observed.Remote.Drift) != 0 {
		t.Fatalf("an exact submission reported drift: %v", observed.Remote.Drift)
	}
}

func TestStatusReportsDriftWithoutWriting(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	s.verify(t)
	s.approve(t)
	if _, err := s.submit(t); err != nil {
		t.Fatalf("submit: %v", err)
	}
	writes := s.remote.Writes()
	s.remote.BaseMoved()
	observed, err := Status(context.Background(), StatusOptions{
		Candidate: s.candidateFile(t), Policy: s.policy, Upstream: s.upstream,
		Validation: s.validation, Root: s.workspace, Remote: true,
		OpenAPI: func(context.Context, Target) (githubauth.API, error) {
			return githubauth.API{BaseURL: s.remote.Server.URL, Transport: fakeBearer{token: testRemoteToken}}, nil
		},
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if observed.Remote == nil || len(observed.Remote.Drift) == 0 {
		t.Fatalf("status reported no drift after the base moved: %+v", observed.Remote)
	}
	if s.remote.Writes() != writes {
		t.Fatal("status wrote to the remote")
	}
}
