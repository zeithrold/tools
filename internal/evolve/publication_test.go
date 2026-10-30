package evolve

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeithrold/tools/internal/githubauth"
)

// validationDoc is a contract covering the fixture policy's two required checks.
func validationDoc(checks ...map[string]any) map[string]any {
	return map[string]any{"schemaVersion": 1, "checks": checks}
}

func validationCheck(name string, timeout int, argv ...string) map[string]any {
	return map[string]any{"name": name, "argv": argv, "timeoutSeconds": timeout}
}

func defaultValidationChecks() []map[string]any {
	return []map[string]any{
		validationCheck("go-test", 60, "internal-validator", "--profile", "go"),
		validationCheck("repo-lint", 30, "internal-validator", "--profile", "lint"),
	}
}

// writeValidationFile stores one contract outside the workspace.
func writeValidationFile(t *testing.T, doc map[string]any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "validation.json")
	writeJSONFile(t, path, doc)
	return path
}

// publicationHarness prepares one candidate, which is everything a submission
// needs: no approval document and no local verification record.
type publicationHarness struct {
	*harness
}

func newPublicationHarness(t *testing.T) *publicationHarness {
	t.Helper()
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	h.mustPrepare()
	return &publicationHarness{harness: h}
}

func (p *publicationHarness) candidateFile(t *testing.T) string {
	t.Helper()
	result := p.mustPrepare()
	return p.candidatePath(result)
}

func (p *publicationHarness) showPayload(t *testing.T) SubmitResult {
	t.Helper()
	result, err := Submit(context.Background(), SubmitOptions{
		Candidate:   p.candidateFile(t),
		Policy:      p.policy,
		Upstream:    p.upstream,
		Root:        p.workspace,
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

func TestShowPayloadPublishesTheExactReviewedDocument(t *testing.T) {
	p := newPublicationHarness(t)
	before := snapshotTree(t, p.workspace)
	result := p.showPayload(t)

	if result.State != "planned" || result.Action != "payload" {
		t.Fatalf("unexpected planned state: %+v", result)
	}
	if result.Payload == nil {
		t.Fatal("no payload was returned")
	}
	payload := *result.Payload
	if payload.Branch != branchFor("fixture-skill", result.CandidateID) {
		t.Fatalf("unexpected branch %q", payload.Branch)
	}
	// The publication has to carry the real text a reviewer reads, not a digest of it.
	contents := map[string]string{}
	for _, file := range payload.Files {
		contents[file.Path] = file.Content
	}
	if contents["skills/fixture-skill/SKILL.md"] != "# Go testing\n\nRun the unit gate, then the focused package.\n" {
		t.Fatalf("payload does not carry the replacement content: %q", contents["skills/fixture-skill/SKILL.md"])
	}
	if !strings.Contains(payload.Commit.Message, result.CandidateID) {
		t.Fatalf("commit message does not name the candidate: %q", payload.Commit.Message)
	}
	if !strings.Contains(payload.Commit.Message, payload.BaseCommit) {
		t.Fatalf("commit message does not name the pinned base: %q", payload.Commit.Message)
	}
	if payload.Commit.AuthorName != evolveAuthorName || payload.Commit.AuthorEmail != evolveAuthorEmail {
		t.Fatalf("commit identity is not the fixed zt identity: %+v", payload.Commit)
	}
	if payload.Commit.CommittedAt != payload.BaseCommittedAt {
		t.Fatalf("commit date %q is not the pinned base committer instant %q", payload.Commit.CommittedAt, payload.BaseCommittedAt)
	}
	if _, err := time.Parse(time.RFC3339, payload.BaseCommittedAt); err != nil {
		t.Fatalf("base committer instant is not RFC 3339: %v", err)
	}
	if !payload.PullRequest.Draft {
		t.Fatal("payload does not request a draft pull request")
	}
	if !strings.Contains(payload.PullRequest.Body, "not verified by zt") {
		t.Fatalf("pull request body does not label the declarations zt did not verify:\n%s", payload.PullRequest.Body)
	}
	// The body states the facts of the proposal. It carries no validation contract,
	// no approval narration and no local path.
	for _, forbidden := range []string{"alidation", "attestation", "passed", "sandbox", p.workspace} {
		if strings.Contains(payload.PullRequest.Body, forbidden) {
			t.Fatalf("pull request body contains %q, which it must not carry:\n%s", forbidden, payload.PullRequest.Body)
		}
	}
	canonical, err := payload.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	if hex.EncodeToString(sum[:]) != result.PublicPayloadDigest {
		t.Fatalf("digest %s does not cover the canonical payload", result.PublicPayloadDigest)
	}

	assertUntouched(t, "the workspace", before, snapshotTree(t, p.workspace))
	for _, name := range []string{"checks", "submit"} {
		if _, err := os.Lstat(filepath.Join(p.workspace, artifactRoot, name)); !os.IsNotExist(err) {
			t.Fatalf("show payload created %s: %v", name, err)
		}
	}
	if result.Writes != 0 {
		t.Fatalf("show payload reported %d writes", result.Writes)
	}
}

func TestPayloadDigestIsStableAndBindsContent(t *testing.T) {
	first := newPublicationHarness(t)
	one := first.showPayload(t)
	two := first.showPayload(t)
	if one.PublicPayloadDigest != two.PublicPayloadDigest || one.PublicPayloadDigest != first.showPayload(t).PublicPayloadDigest {
		t.Fatal("the same candidate produced different payload digests")
	}

	changed := newPublicationHarness(t)
	changed.writeProposal(proposalDoc(changed.commit, fileEntry("skills/fixture-skill/SKILL.md", basePreimage(), "# Go testing\n\nRun the focused package first.\n")))
	other := changed.showPayload(t)
	if other.PublicPayloadDigest == one.PublicPayloadDigest {
		t.Fatal("a changed replacement kept the payload digest")
	}
}

// TestPayloadDigestIgnoresLocalVerification pins that the publication binds the
// candidate and the pinned base, not the local verification an operator happened to
// run: `evolve check` is a separate utility, so its contract, its results and its
// failures cannot change what would be published.
func TestPayloadDigestIgnoresLocalVerification(t *testing.T) {
	s := newSubmitHarness(t, "example/fixture")
	before := s.showPayload(t)

	writeJSONFile(t, s.validation, validationDoc(
		validationCheck("go-test", 60, helperArgv("TestEvolveHelperFail")...),
		validationCheck("repo-lint", 60, helperArgv("TestEvolveHelperPass")...),
	))
	failed, err := Verify(context.Background(), CheckOptions{
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
	if failed.Status != executionFailed {
		t.Fatalf("the fixture check did not fail: %+v", failed)
	}

	after := s.showPayload(t)
	if after.PublicPayloadDigest != before.PublicPayloadDigest {
		t.Fatal("a local verification changed the public payload identity")
	}
}

// TestPolicyMayRequireNoChecks pins that a policy naming no required check is a
// valid policy rather than an error, and that a standalone contract may still run
// whatever the operator wants to check.
func TestPolicyMayRequireNoChecks(t *testing.T) {
	h := newHarness(t)
	doc := policyDoc(h.commit)
	doc["requiredChecks"] = []string{}
	h.writePolicy(doc)
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))

	prepared := h.mustPrepare()
	if len(prepared.Checks) != 0 {
		t.Fatalf("a policy with no required checks reported %v", prepared.Checks)
	}
	planned, err := h.plan(h.candidatePath(prepared))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(planned.Checks) != 0 {
		t.Fatalf("plan reported checks %v for a policy that requires none", planned.Checks)
	}

	policy := policyDocToPolicy(t, h)
	contract, err := LoadValidation(writeValidationFile(t, validationDoc(validationCheck("behaviour", 60, "true"))))
	if err != nil {
		t.Fatalf("load validation: %v", err)
	}
	if err := contract.covers(policy); err != nil {
		t.Fatalf("a contract the operator chose was rejected by a policy that requires no check: %v", err)
	}
}

// TestValidationContractMustIncludeThePolicyChecks pins the one retained rule: a
// standalone contract has to run a command for every check the policy requires, and
// may add checks of its own.
func TestValidationContractMustIncludeThePolicyChecks(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	policy := policyDocToPolicy(t, h)

	accepted := map[string]map[string]any{
		"exact": validationDoc(defaultValidationChecks()...),
		"extra": validationDoc(append(defaultValidationChecks(), validationCheck("behaviour", 60, "true"))...),
	}
	for name, doc := range accepted {
		t.Run("accepted-"+name, func(t *testing.T) {
			validation, err := LoadValidation(writeValidationFile(t, doc))
			if err != nil {
				t.Fatalf("load validation: %v", err)
			}
			if err := validation.covers(policy); err != nil {
				t.Fatalf("validation contract %s was refused: %v", name, err)
			}
		})
	}

	refused := map[string]map[string]any{
		"missing":     validationDoc(validationCheck("go-test", 60, "true")),
		"only-extras": validationDoc(validationCheck("behaviour", 60, "true")),
		"duplicate": validationDoc(
			validationCheck("go-test", 60, "true"),
			validationCheck("go-test", 60, "true"),
			validationCheck("repo-lint", 60, "true"),
		),
		"empty-argv": validationDoc(
			validationCheck("go-test", 60),
			validationCheck("repo-lint", 60, "true"),
		),
		"control-argv": validationDoc(
			validationCheck("go-test", 60, "true\nrm -rf /"),
			validationCheck("repo-lint", 60, "true"),
		),
		"zero-timeout": validationDoc(
			validationCheck("go-test", 0, "true"),
			validationCheck("repo-lint", 60, "true"),
		),
	}
	for name, doc := range refused {
		t.Run("refused-"+name, func(t *testing.T) {
			validation, err := LoadValidation(writeValidationFile(t, doc))
			if err != nil {
				return // already refused while loading
			}
			if err := validation.covers(policy); err == nil {
				t.Fatalf("validation contract %s was accepted", name)
			}
		})
	}
}

func policyDocToPolicy(t *testing.T, h *harness) Policy {
	t.Helper()
	policy, err := LoadPolicy(h.policy)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestValidationDigestDependsOnMeaningNotFormatting(t *testing.T) {
	first := filepath.Join(t.TempDir(), "a.json")
	writeFile(t, first, `{"schemaVersion":1,"checks":[{"name":"go-test","argv":["go","test"],"timeoutSeconds":60},{"name":"repo-lint","argv":["lint"],"timeoutSeconds":30}]}`)
	second := filepath.Join(t.TempDir(), "b.json")
	writeFile(t, second, "{\n  \"checks\": [\n    {\"timeoutSeconds\": 30, \"argv\": [\"lint\"], \"name\": \"repo-lint\"},\n    {\"name\": \"go-test\", \"timeoutSeconds\": 60, \"argv\": [\"go\", \"test\"]}\n  ],\n  \"schemaVersion\": 1\n}\n")
	one, err := LoadValidation(first)
	if err != nil {
		t.Fatal(err)
	}
	two, err := LoadValidation(second)
	if err != nil {
		t.Fatal(err)
	}
	digestOne, err := one.digest()
	if err != nil {
		t.Fatal(err)
	}
	digestTwo, err := two.digest()
	if err != nil {
		t.Fatal(err)
	}
	if digestOne != digestTwo {
		t.Fatal("formatting changed the validation digest")
	}
	two.Checks[0].TimeoutSeconds = 31
	changed, err := two.digest()
	if err != nil {
		t.Fatal(err)
	}
	if changed == digestOne {
		t.Fatal("a changed timeout kept the validation digest")
	}
}

func TestJournalIsBoundToItsOwnCandidateAndPayload(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	prepared := h.mustPrepare()
	store := mustStore(t, h.workspace)
	journal := Journal{
		SchemaVersion:       schemaVersion,
		CandidateID:         prepared.CandidateID,
		PublicPayloadDigest: strings.Repeat("1", 64),
		Branch:              "evolve/fixture-skill-000000000000",
		BaseCommit:          h.commit,
		State:               journalStarted,
		UpdatedAt:           "2026-02-01T00:00:00Z",
	}
	relative := submitJournalRelative(prepared.CandidateID, journal.PublicPayloadDigest)
	if err := record(context.Background(), store, relative, &journal, journalStarted, "test"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadJournal(store, relative, prepared.CandidateID, journal.PublicPayloadDigest); err != nil {
		t.Fatalf("the recorded journal was not readable: %v", err)
	}
	if _, _, err := loadJournal(store, relative, prepared.CandidateID, strings.Repeat("9", 64)); err == nil {
		t.Fatal("a journal for another payload was adopted")
	}
	data, found, err := store.readOwned(relative, maxJournalBytes)
	if err != nil || !found {
		t.Fatalf("journal missing: %v", err)
	}
	for _, secret := range []string{"Authorization", "Bearer", "ghu_"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatalf("the journal recorded %q", secret)
		}
	}
}

// TestJournalFromTheRemovedProtocolIsRejected pins that a record written by the
// removed approval protocol is refused rather than silently misread. Its path is
// derived from the payload digest too, so an old record cannot describe a new one.
func TestJournalFromTheRemovedProtocolIsRejected(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	prepared := h.mustPrepare()
	store := mustStore(t, h.workspace)
	payloadDigest := strings.Repeat("1", 64)
	relative := submitJournalRelative(prepared.CandidateID, payloadDigest)
	legacy := `{
  "schemaVersion": 1,
  "candidateId": "` + prepared.CandidateID + `",
  "publicPayloadDigest": "` + payloadDigest + `",
  "approvalDigest": "` + strings.Repeat("2", 64) + `",
  "validationDigest": "` + strings.Repeat("3", 64) + `",
  "branch": "evolve/fixture-skill-000000000000",
  "baseBranch": "main",
  "baseCommit": "` + h.commit + `",
  "state": "started",
  "updatedAt": "2026-02-01T00:00:00Z"
}
`
	if err := store.writeOwned(context.Background(), relative, []byte(legacy)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadJournal(store, relative, prepared.CandidateID, payloadDigest); err == nil {
		t.Fatal("a journal written by the removed approval protocol was accepted")
	}
}
