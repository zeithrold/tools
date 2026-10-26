package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zeithrold/tools/internal/githubauth"
)

// runEvolveOfflineCLI runs one evolve command and fails the test if the command
// builds an auth service, opens an authenticated endpoint, or resolves the home
// directory. Only the explicitly remote commands may do any of those.
func runEvolveOfflineCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.homeDir = func() (string, error) {
		t.Fatal("the command resolved the home directory")
		return "", nil
	}
	a.newAuth = func(string) (authService, error) {
		t.Fatal("the command constructed the auth service")
		return nil, nil
	}
	a.openAPI = func(context.Context, string, string, string, string) (githubauth.API, error) {
		t.Fatal("the command opened the authenticated GitHub endpoint")
		return githubauth.API{}, nil
	}
	if err := executeApp(a, context.Background(), args); err != nil {
		return stdout.String(), err
	}
	return stdout.String(), nil
}

// preparedCandidate runs prepare and returns the candidate path in the workspace.
func preparedCandidate(t *testing.T, f *evolveFixture) string {
	t.Helper()
	stdout, err := runEvolveOfflineCLI(t, f.prepareArgs()...)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	var result struct {
		CandidatePath string `json:"candidatePath"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("prepare json: %v\n%s", err, stdout)
	}
	return filepath.Join(f.root, filepath.FromSlash(result.CandidatePath))
}

func writeEvolveValidation(t *testing.T, checks ...map[string]any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "validation.json")
	writeFile(t, path, evolveTestJSON(t, map[string]any{"schemaVersion": 1, "checks": checks}))
	return path
}

func deterministicCheck(name string) map[string]any {
	return map[string]any{"name": name, "argv": []string{"internal-validator", "--strict"}, "timeoutSeconds": 30}
}

func TestEvolveCheckRefusesToExecuteWithoutTheExplicitFlag(t *testing.T) {
	f := newEvolveFixture(t)
	candidate := preparedCandidate(t, f)
	validation := writeEvolveValidation(t, deterministicCheck("go-test"), deterministicCheck("repo-lint"))
	before := snapshotFiles(t, f.root)

	_, err := runEvolveOfflineCLI(t, "evolve", "check", candidate, "--policy", f.policy, "--upstream", f.upstream, "--validation", validation, "--root", f.root)
	if err == nil {
		t.Fatal("check executed the validation contract without --trusted-local-execution")
	}
	for _, want := range []string{"--trusted-local-execution", "OS sandbox", "credential isolation"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not mention %q", err, want)
		}
	}
	assertEqualSnapshots(t, "the workspace", before, snapshotFiles(t, f.root))
	if _, err := os.Lstat(filepath.Join(f.root, ".zt", "artifacts", "evolve", "checks")); !os.IsNotExist(err) {
		t.Fatalf("a refused check created the checks tree: %v", err)
	}
}

func TestEvolveSubmitShowPayloadNeedsNoApprovalCredentialOrWrite(t *testing.T) {
	f := newEvolveFixture(t)
	candidate := preparedCandidate(t, f)
	validation := writeEvolveValidation(t, deterministicCheck("go-test"), deterministicCheck("repo-lint"))
	before := snapshotFiles(t, f.root)

	stdout, err := runEvolveOfflineCLI(t, "evolve", "submit", candidate, "--policy", f.policy, "--upstream", f.upstream, "--validation", validation, "--root", f.root, "--show-payload")
	if err != nil {
		t.Fatalf("show payload: %v", err)
	}
	for _, want := range []string{
		"\"content\": \"# Go testing\\n\\nRun the unit gate, then the focused package.\\n\"",
		"public payload digest:",
		"Nothing was read from the OS keystore",
		"\"draft\": true",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("preview omits %q:\n%s", want, stdout)
		}
	}
	assertEqualSnapshots(t, "the workspace", before, snapshotFiles(t, f.root))
}

func TestEvolveSubmitRequiresAnApprovalFileBeforeAnyCredential(t *testing.T) {
	f := newEvolveFixture(t)
	candidate := preparedCandidate(t, f)
	validation := writeEvolveValidation(t, deterministicCheck("go-test"), deterministicCheck("repo-lint"))

	_, err := runEvolveOfflineCLI(t, "evolve", "submit", candidate, "--policy", f.policy, "--upstream", f.upstream, "--validation", validation, "--root", f.root)
	if err == nil || !strings.Contains(err.Error(), "--approval") {
		t.Fatalf("submit did not require an approval: %v", err)
	}

	approval := filepath.Join(t.TempDir(), "approval.json")
	writeFile(t, approval, evolveTestJSON(t, map[string]any{
		"schemaVersion":       1,
		"candidateId":         strings.Repeat("0", 64),
		"target":              map[string]any{"host": "github.com", "repository": "example/fixture", "baseCommit": f.commit},
		"validationDigest":    strings.Repeat("1", 64),
		"publicPayloadDigest": strings.Repeat("2", 64),
		"reviewer":            "reviewer",
		"reviewedAt":          "2026-02-01T09:30:00Z",
		"attestations":        map[string]any{"humanReview": "approved", "privacy": "approved", "license": "approved"},
	}))
	_, err = runEvolveOfflineCLI(t, "evolve", "submit", candidate, "--policy", f.policy, "--upstream", f.upstream, "--validation", validation, "--approval", approval, "--root", f.root)
	if err == nil || !strings.Contains(err.Error(), "candidateId") {
		t.Fatalf("a mismatched approval was accepted: %v", err)
	}
}

func TestEvolveStatusIsLocalReadOnlyAndReportsNotReady(t *testing.T) {
	f := newEvolveFixture(t)
	candidate := preparedCandidate(t, f)
	validation := writeEvolveValidation(t, deterministicCheck("go-test"), deterministicCheck("repo-lint"))
	before := snapshotFiles(t, f.root)

	stdout, err := runEvolveOfflineCLI(t, "evolve", "status", candidate, "--policy", f.policy, "--upstream", f.upstream, "--validation", validation, "--root", f.root)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, want := range []string{"local verification: not run", "submission journal: none", "publication: not_ready"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("status omits %q:\n%s", want, stdout)
		}
	}
	assertEqualSnapshots(t, "the workspace", before, snapshotFiles(t, f.root))
}

func TestEvolveSubmitRequiresATrustedLocalVerification(t *testing.T) {
	f := newEvolveFixture(t)
	candidate := preparedCandidate(t, f)
	validation := writeEvolveValidation(t, deterministicCheck("go-test"), deterministicCheck("repo-lint"))

	preview, err := runEvolveOfflineCLI(t, "evolve", "submit", candidate, "--policy", f.policy, "--upstream", f.upstream, "--validation", validation, "--root", f.root, "--show-payload", "--json")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	var plan struct {
		CandidateID         string `json:"candidateId"`
		ValidationDigest    string `json:"validationDigest"`
		PublicPayloadDigest string `json:"publicPayloadDigest"`
	}
	if err := json.Unmarshal([]byte(preview), &plan); err != nil {
		t.Fatalf("preview json: %v", err)
	}
	approval := filepath.Join(t.TempDir(), "approval.json")
	writeFile(t, approval, evolveTestJSON(t, map[string]any{
		"schemaVersion":       1,
		"candidateId":         plan.CandidateID,
		"target":              map[string]any{"host": "github.com", "repository": "example/fixture", "baseCommit": f.commit},
		"validationDigest":    plan.ValidationDigest,
		"publicPayloadDigest": plan.PublicPayloadDigest,
		"reviewer":            "reviewer@example.invalid",
		"reviewedAt":          "2026-02-01T09:30:00Z",
		"attestations":        map[string]any{"humanReview": "approved", "privacy": "approved", "license": "approved"},
	}))

	_, err = runEvolveOfflineCLI(t, "evolve", "submit", candidate, "--policy", f.policy, "--upstream", f.upstream, "--validation", validation, "--approval", approval, "--root", f.root)
	if err == nil || !strings.Contains(err.Error(), "zt evolve check") {
		t.Fatalf("submit did not require a passed local verification: %v", err)
	}
}

// assertEqualSnapshots is the file-level twin of the evolve package assertion.
func assertEqualSnapshots(t *testing.T, what string, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("%s changed: %d paths became %d", what, len(before), len(after))
	}
	for name, value := range before {
		if after[name] != value {
			t.Fatalf("%s changed at %s", what, name)
		}
	}
}

// evolveTestJSON marshals one fixture document; a failure here is a test bug, not a
// CLI result.
func evolveTestJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return string(data) + "\n"
}

// TestEvolveCheckExitsNonZeroOnAFailedContract is the real-CLI regression for a
// validation command that runs and fails: the result is still reported, and the
// command exits non-zero like `zt check`, rather than reporting success for a failed
// verification.
func TestEvolveCheckExitsNonZeroOnAFailedContract(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixture command is /bin/sh")
	}
	f := newEvolveFixture(t)
	candidate := preparedCandidate(t, f)
	validation := writeEvolveValidation(t,
		map[string]any{"name": "go-test", "argv": []string{"/bin/sh", "-c", "exit 17"}, "timeoutSeconds": 30},
		map[string]any{"name": "repo-lint", "argv": []string{"/bin/sh", "-c", "exit 0"}, "timeoutSeconds": 30},
	)
	stdout, err := runEvolveOfflineCLI(t, "evolve", "check", candidate, "--policy", f.policy, "--upstream", f.upstream, "--validation", validation, "--root", f.root, "--trusted-local-execution", "--json")
	var result struct {
		Status string `json:"status"`
		Checks []struct {
			ExitCode int `json:"exitCode"`
		} `json:"checks"`
	}
	if decodeErr := json.Unmarshal([]byte(stdout), &result); decodeErr != nil {
		t.Fatalf("check json: %v\n%s", decodeErr, stdout)
	}
	if result.Status != "failed" || len(result.Checks) == 0 || result.Checks[0].ExitCode != 17 {
		t.Fatalf("unexpected actual execution: %s", stdout)
	}
	if err == nil {
		t.Fatalf("the CLI reported success for status=%s with a native exit of 17", result.Status)
	}
	if !strings.Contains(err.Error(), "did not pass") {
		t.Fatalf("the CLI failure %q does not explain the status", err)
	}
}
