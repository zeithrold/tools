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

// publicationHarness prepares one candidate and writes an operator validation
// contract outside the workspace.
type publicationHarness struct {
	*harness
	validation string
}

func newPublicationHarness(t *testing.T) *publicationHarness {
	t.Helper()
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	h.mustPrepare()
	validation := filepath.Join(t.TempDir(), "validation.json")
	writeJSONFile(t, validation, validationDoc(defaultValidationChecks()...))
	return &publicationHarness{harness: h, validation: validation}
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
		Validation:  p.validation,
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

// approvalFor builds a correct approval document for one previewed payload.
func approvalFor(result SubmitResult, mutate func(map[string]any)) map[string]any {
	doc := map[string]any{
		"schemaVersion":       1,
		"candidateId":         result.CandidateID,
		"target":              map[string]any{"host": result.Target.Host, "repository": result.Target.Repository, "baseCommit": result.Target.BaseCommit},
		"validationDigest":    result.ValidationDigest,
		"publicPayloadDigest": result.PublicPayloadDigest,
		"reviewer":            "reviewer@example.invalid",
		"reviewedAt":          "2026-02-01T09:30:00Z",
		"attestations":        map[string]any{"humanReview": "approved", "privacy": "approved", "license": "approved"},
	}
	if mutate != nil {
		mutate(doc)
	}
	return doc
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
	// The publication has to carry the real text a reviewer approves, not a digest
	// of it.
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
		t.Fatalf("pull request body does not state what zt did not verify:\n%s", payload.PullRequest.Body)
	}
	for _, forbidden := range []string{"passed", p.workspace} {
		if strings.Contains(payload.PullRequest.Body, forbidden) {
			t.Fatalf("pull request body contains %q, which the payload must not bind:\n%s", forbidden, payload.PullRequest.Body)
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
		t.Fatal("the same candidate and contract produced different payload digests")
	}

	changed := newPublicationHarness(t)
	changed.writeProposal(proposalDoc(changed.commit, fileEntry("skills/fixture-skill/SKILL.md", basePreimage(), "# Go testing\n\nRun the focused package first.\n")))
	other := changed.showPayload(t)
	if other.PublicPayloadDigest == one.PublicPayloadDigest {
		t.Fatal("a changed replacement kept the payload digest")
	}
}

func TestSubmitRefusesWithoutAPassedLocalVerification(t *testing.T) {
	p := newPublicationHarness(t)
	preview := p.showPayload(t)
	approval := filepath.Join(t.TempDir(), "approval.json")
	writeJSONFile(t, approval, approvalFor(preview, nil))
	_, err := Submit(context.Background(), SubmitOptions{
		Candidate:  p.candidateFile(t),
		Policy:     p.policy,
		Upstream:   p.upstream,
		Validation: p.validation,
		Approval:   approval,
		Root:       p.workspace,
		OpenAPI: func(context.Context, Target) (githubauth.API, error) {
			t.Fatal("submit opened the credential seam without a passed verification")
			return githubauth.API{}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "zt evolve check") {
		t.Fatalf("submit did not require a local verification: %v", err)
	}
	for _, name := range []string{"checks", "submit"} {
		if _, statErr := os.Lstat(filepath.Join(p.workspace, artifactRoot, name)); !os.IsNotExist(statErr) {
			t.Fatalf("a refused submit created %s", name)
		}
	}
}

func TestApprovalMustNameTheExactCandidateContractAndPayload(t *testing.T) {
	p := newPublicationHarness(t)
	preview := p.showPayload(t)
	cases := []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{"candidate", func(doc map[string]any) { doc["candidateId"] = strings.Repeat("0", 64) }, "candidateId"},
		{"base", func(doc map[string]any) {
			target := doc["target"].(map[string]any)
			target["baseCommit"] = strings.Repeat("a", 40)
		}, "target does not match"},
		{"host", func(doc map[string]any) {
			target := doc["target"].(map[string]any)
			target["host"] = "github.example"
		}, "target does not match"},
		{"validation", func(doc map[string]any) { doc["validationDigest"] = strings.Repeat("b", 64) }, "validationDigest"},
		{"payload", func(doc map[string]any) { doc["publicPayloadDigest"] = strings.Repeat("c", 64) }, "publicPayloadDigest"},
		{"privacy", func(doc map[string]any) {
			doc["attestations"].(map[string]any)["privacy"] = "pending"
		}, "attestations.privacy"},
		{"license", func(doc map[string]any) {
			doc["attestations"].(map[string]any)["license"] = "unknown"
		}, "attestations.license"},
		{"review-time-offset", func(doc map[string]any) { doc["reviewedAt"] = "2026-02-01T09:30:00+02:00" }, "reviewedAt"},
		{"reviewer", func(doc map[string]any) { doc["reviewer"] = "  " }, "reviewer"},
		{"unknown-field", func(doc map[string]any) { doc["approveEverything"] = true }, "approval file"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			approval := filepath.Join(t.TempDir(), "approval.json")
			writeJSONFile(t, approval, approvalFor(preview, testCase.mutate))
			_, err := Submit(context.Background(), SubmitOptions{
				Candidate:  p.candidateFile(t),
				Policy:     p.policy,
				Upstream:   p.upstream,
				Validation: p.validation,
				Approval:   approval,
				Root:       p.workspace,
				OpenAPI: func(context.Context, Target) (githubauth.API, error) {
					t.Fatal("an invalid approval reached the credential seam")
					return githubauth.API{}, nil
				},
			})
			if err == nil {
				t.Fatal("a mismatched approval was accepted")
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error %q does not mention %q", err, testCase.want)
			}
		})
	}
}

func TestValidationContractMustCoverThePolicyChecksExactly(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	policy := policyDocToPolicy(t, h)
	cases := map[string]map[string]any{
		"missing": validationDoc(validationCheck("go-test", 60, "true")),
		"extra": validationDoc(append(defaultValidationChecks(), []map[string]any{
			validationCheck("behaviour", 60, "true"),
		}...)...),
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
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "validation.json")
			writeJSONFile(t, path, doc)
			validation, err := LoadValidation(path)
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
	store, err := openStore(h.workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	journal := Journal{
		SchemaVersion:       schemaVersion,
		CandidateID:         prepared.CandidateID,
		PublicPayloadDigest: strings.Repeat("1", 64),
		ApprovalDigest:      strings.Repeat("2", 64),
		ValidationDigest:    strings.Repeat("3", 64),
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
