package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const evolveSkillMarkdown = "# Go testing\n\nRun the unit gate first.\n"

// evolveFixture is a synthetic upstream repository with one prepared candidate
// input set. The policy and proposal live outside the workspace, so only the
// owned artifact tree can appear below it.
type evolveFixture struct {
	upstream string
	commit   string
	root     string
	policy   string
	proposal string
}

func fixtureGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"--no-pager", "-C", dir,
		"-c", "user.name=zt-fixture", "-c", "user.email=zt@example.invalid",
		"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)...)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func evolveSHA(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

func newEvolveFixture(t *testing.T) *evolveFixture {
	t.Helper()
	upstream := t.TempDir()
	fixtureGit(t, upstream, "init", "--quiet")
	writeFile(t, filepath.Join(upstream, "skills", "fixture-skill", "SKILL.md"), evolveSkillMarkdown)
	writeFile(t, filepath.Join(upstream, "skills", "fixture-skill", "references", "notes.md"), "# Notes\n")
	fixtureGit(t, upstream, "add", "--all")
	fixtureGit(t, upstream, "commit", "--quiet", "-m", "fixture")
	commit := fixtureGit(t, upstream, "rev-parse", "HEAD")

	inputs := t.TempDir()
	policy := filepath.Join(inputs, "policy.json")
	writeFile(t, policy, `{
  "schemaVersion": 1,
  "host": "github.com",
  "repository": "example/fixture",
  "baseCommit": "`+commit+`",
  "allowedSkills": ["fixture-skill"],
  "requiredChecks": ["go-test", "repo-lint"]
}
`)
	proposal := filepath.Join(inputs, "proposal.json")
	writeFile(t, proposal, `{
  "schemaVersion": 1,
  "skill": "fixture-skill",
  "provenance": {
    "repository": "example/upstream",
    "commit": "`+commit+`",
    "license": "Apache-2.0",
    "reason": "Clarify the first gate for reviewers."
  },
  "files": [
    {
      "path": "skills/fixture-skill/SKILL.md",
      "preimageSha256": "`+evolveSHA(evolveSkillMarkdown)+`",
      "content": "# Go testing\n\nRun the unit gate, then the focused package.\n"
    },
    {
      "path": "skills/fixture-skill/references/guide.md",
      "preimageSha256": "",
      "content": "# Guide\n\nA new reference.\n"
    }
  ]
}
`)
	return &evolveFixture{upstream: upstream, commit: commit, root: t.TempDir(), policy: policy, proposal: proposal}
}

func (f *evolveFixture) prepareArgs() []string {
	return []string{"evolve", "prepare", "--proposal", f.proposal, "--policy", f.policy, "--upstream", f.upstream, "--root", f.root, "--json"}
}

func (f *evolveFixture) planArgs(candidate string) []string {
	return []string{"evolve", "plan", candidate, "--policy", f.policy, "--upstream", f.upstream, "--json"}
}

// runEvolveCLI fails the test if a command resolves the home directory or builds
// an auth service: evolve is offline and must never acquire an identity.
func runEvolveCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.homeDir = func() (string, error) {
		t.Fatal("evolve resolved the home directory")
		return "", nil
	}
	a.newAuth = func(string) (authService, error) {
		t.Fatal("evolve constructed the auth service")
		return nil, nil
	}
	if err := executeApp(a, context.Background(), args); err != nil {
		return stdout.String(), err
	}
	return stdout.String(), nil
}

// snapshotFiles records every path below dir with its kind and content digest so
// a read-only check also notices a created directory.
func snapshotFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			entries[relative+"/"] = "dir"
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries[relative] = evolveSHA(string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", dir, err)
	}
	return entries
}

func TestEvolveHelpDescribesAStructuredProposalAndStopsAtPlan(t *testing.T) {
	stdout, _, err := runCLI(t, "evolve", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"structured replacement proposal", "not a unified diff", "prepare", "plan", "not_run", "not_ready"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("evolve help omits %q:\n%s", want, stdout)
		}
	}
	if !strings.Contains(stdout, "there is no --patch flag") {
		t.Fatalf("evolve help does not explain that no patch flag exists:\n%s", stdout)
	}
	for _, absent := range []string{"publish CANDIDATE", "status CANDIDATE", "submit CANDIDATE"} {
		if strings.Contains(stdout, absent) {
			t.Fatalf("evolve help advertises %q:\n%s", absent, stdout)
		}
	}
	if !strings.Contains(strings.ToLower(stdout), "no check, submit, publish or status command") {
		t.Fatalf("evolve help does not state which commands are absent:\n%s", stdout)
	}
}

func TestEvolveRegistersNoPatchFlag(t *testing.T) {
	root := newRootCommand(newApp(io.Discard, io.Discard))
	for _, path := range [][]string{{"evolve"}, {"evolve", "prepare"}, {"evolve", "plan"}} {
		command, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		if command.Flags().Lookup("patch") != nil || command.PersistentFlags().Lookup("patch") != nil {
			t.Fatalf("%v advertises a patch flag", path)
		}
	}
	prepared, _, err := root.Find([]string{"evolve", "prepare"})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"proposal", "policy", "upstream", "root"} {
		if prepared.Flags().Lookup(required) == nil {
			t.Fatalf("evolve prepare is missing --%s", required)
		}
	}
	planned, _, err := root.Find([]string{"evolve", "plan"})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"policy", "upstream"} {
		if planned.Flags().Lookup(required) == nil {
			t.Fatalf("evolve plan is missing --%s", required)
		}
	}
	if planned.Flags().Lookup("root") != nil {
		t.Fatal("evolve plan accepted --root; it must only read the candidate it is given")
	}
}

func TestEvolveGroupRejectsUnknownSubcommandsButHelpSucceeds(t *testing.T) {
	stdout, stderr, err := runCLI(t, "evolve")
	if err != nil {
		t.Fatalf("bare evolve should print help and succeed: %v", err)
	}
	if stdout == "" || stderr != "" {
		t.Fatalf("bare evolve output: stdout=%q stderr=%q", stdout, stderr)
	}
	for _, name := range []string{"prepare", "plan"} {
		if !strings.Contains(stdout, name) {
			t.Fatalf("help omits %q:\n%s", name, stdout)
		}
	}
	_, _, err = runCLI(t, "evolve", "submit")
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unknown subcommand was accepted: %v", err)
	}
}

func TestEvolvePrepareAndPlanEndToEnd(t *testing.T) {
	fixture := newEvolveFixture(t)
	preparedJSON, err := runEvolveCLI(t, fixture.prepareArgs()...)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	var prepared struct {
		SchemaVersion   int    `json:"schemaVersion"`
		CandidateID     string `json:"candidateId"`
		CandidatePath   string `json:"candidatePath"`
		Created         bool   `json:"created"`
		Publication     string `json:"publication"`
		MissingEvidence []string
		Checks          []struct{ Name, Status string }
	}
	if err := json.Unmarshal([]byte(preparedJSON), &prepared); err != nil {
		t.Fatalf("prepare stdout is not pure JSON %q: %v", preparedJSON, err)
	}
	if !prepared.Created || prepared.Publication != "not_ready" {
		t.Fatalf("unexpected prepare result: %+v", prepared)
	}
	if len(prepared.Checks) != 2 {
		t.Fatalf("prepare reported %d checks", len(prepared.Checks))
	}
	for _, check := range prepared.Checks {
		if check.Status != "not_run" {
			t.Fatalf("check %s reported %q", check.Name, check.Status)
		}
	}
	candidate := filepath.Join(fixture.root, filepath.FromSlash(prepared.CandidatePath))
	if _, err := os.Stat(candidate); err != nil {
		t.Fatalf("candidate is missing: %v", err)
	}

	workspaceBefore := snapshotFiles(t, fixture.root)
	repositoryBefore := snapshotFiles(t, fixture.upstream)
	plannedJSON, err := runEvolveCLI(t, fixture.planArgs(candidate)...)
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}
	var planned struct {
		CandidateID     string `json:"candidateId"`
		Publication     string `json:"publication"`
		ProvenanceState string `json:"provenanceState"`
		MissingEvidence []string
		Checks          []struct{ Name, Status string }
		Files           []struct{ Path, SHA256 string }
	}
	if err := json.Unmarshal([]byte(plannedJSON), &planned); err != nil {
		t.Fatalf("plan stdout is not pure JSON %q: %v", plannedJSON, err)
	}
	if planned.CandidateID != prepared.CandidateID {
		t.Fatalf("plan reported %s, prepare reported %s", planned.CandidateID, prepared.CandidateID)
	}
	if planned.Publication != "not_ready" || planned.ProvenanceState != "declared_not_verified" {
		t.Fatalf("plan overstated its evidence: %+v", planned)
	}
	if len(planned.MissingEvidence) != 5 {
		t.Fatalf("plan reported missing evidence %v", planned.MissingEvidence)
	}
	for _, check := range planned.Checks {
		if check.Status != "not_run" {
			t.Fatalf("check %s reported %q", check.Name, check.Status)
		}
	}
	if len(planned.Files) != 2 {
		t.Fatalf("plan reported %d replacements", len(planned.Files))
	}
	if after := snapshotFiles(t, fixture.root); !sameSnapshot(workspaceBefore, after) {
		t.Fatal("plan changed the workspace")
	}
	if after := snapshotFiles(t, fixture.upstream); !sameSnapshot(repositoryBefore, after) {
		t.Fatal("plan changed the upstream repository")
	}

	// A repeated prepare reports the same candidate and writes nothing.
	repeatJSON, err := runEvolveCLI(t, fixture.prepareArgs()...)
	if err != nil {
		t.Fatalf("repeat prepare failed: %v", err)
	}
	var repeated struct {
		CandidateID string `json:"candidateId"`
		Created     bool   `json:"created"`
	}
	// Parse the response: a substring check misses any spacing the encoder
	// chooses, which is how this assertion previously passed unconditionally.
	if err := json.Unmarshal([]byte(repeatJSON), &repeated); err != nil {
		t.Fatalf("repeat prepare output is not JSON %q: %v", repeatJSON, err)
	}
	if repeated.Created {
		t.Fatalf("repeat prepare claimed a new candidate: %s", repeatJSON)
	}
	if repeated.CandidateID != prepared.CandidateID {
		t.Fatalf("repeat prepare reported %s, first run reported %s", repeated.CandidateID, prepared.CandidateID)
	}
	if after := snapshotFiles(t, fixture.root); !sameSnapshot(workspaceBefore, after) {
		t.Fatal("repeat prepare changed the workspace")
	}
}

func TestEvolveDocumentErrorsDoNotEchoPayload(t *testing.T) {
	fixture := newEvolveFixture(t)
	marker := "PRIVATE_REVIEW_MARKER_" + strings.Repeat("x", 4096)
	data, err := os.ReadFile(fixture.proposal)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	document[marker] = "unused"
	malformed, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, fixture.proposal, string(malformed)+"\n")

	_, err = runEvolveCLI(t, fixture.prepareArgs()...)
	if err == nil {
		t.Fatal("prepare accepted an unknown proposal field")
	}
	if len(err.Error()) > 300 {
		t.Fatalf("error is %d bytes long: %q", len(err.Error()), err.Error())
	}
	if strings.Contains(err.Error(), marker) || strings.Contains(err.Error(), "PRIVATE_REVIEW_MARKER") {
		t.Fatalf("error echoes the document payload: %v", err)
	}
	if !strings.Contains(err.Error(), "documented schema") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvolvePlanDoesNotWriteGitTracesOutside(t *testing.T) {
	fixture := newEvolveFixture(t)
	outside := filepath.Join(t.TempDir(), "trace.log")
	t.Setenv("GIT_TRACE", outside)
	t.Setenv("GIT_TRACE2_EVENT", outside+".json")
	preparedJSON, err := runEvolveCLI(t, fixture.prepareArgs()...)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	var prepared struct{ CandidatePath string }
	if err := json.Unmarshal([]byte(preparedJSON), &prepared); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(fixture.root, filepath.FromSlash(prepared.CandidatePath))
	if _, err := runEvolveCLI(t, fixture.planArgs(candidate)...); err != nil {
		t.Fatalf("plan failed: %v", err)
	}
	for _, path := range []string{outside, outside + ".json"} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("git tracing wrote outside the workspace: %s", path)
		}
	}
}

func sameSnapshot(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for name, value := range left {
		if right[name] != value {
			return false
		}
	}
	return true
}

func TestEvolveRequiresExplicitInputs(t *testing.T) {
	_, _, err := runCLI(t, "evolve", "prepare")
	if err == nil || !strings.Contains(err.Error(), "--proposal") {
		t.Fatalf("prepare accepted a missing proposal: %v", err)
	}
	_, _, err = runCLI(t, "evolve", "prepare", "--proposal", "p.json")
	if err == nil || !strings.Contains(err.Error(), "--policy") {
		t.Fatalf("prepare accepted a missing policy: %v", err)
	}
	_, _, err = runCLI(t, "evolve", "plan")
	if err == nil || !strings.Contains(err.Error(), "candidate file") {
		t.Fatalf("plan accepted a missing candidate: %v", err)
	}
	_, _, err = runCLI(t, "evolve", "plan", "a.json", "b.json")
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("plan accepted extra arguments: %v", err)
	}
}

func TestEvolveHonoursCancellationWithoutWriting(t *testing.T) {
	fixture := newEvolveFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.homeDir = func() (string, error) {
		t.Fatal("evolve resolved the home directory")
		return "", nil
	}
	if err := executeApp(a, ctx, fixture.prepareArgs()); !errors.Is(err, context.Canceled) {
		t.Fatalf("prepare ignored cancellation: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(fixture.root, ".zt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled prepare created artifacts: %v", err)
	}
}

func TestEvolvePlanRejectsAChangedPolicy(t *testing.T) {
	fixture := newEvolveFixture(t)
	preparedJSON, err := runEvolveCLI(t, fixture.prepareArgs()...)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	var prepared struct{ CandidatePath string }
	if err := json.Unmarshal([]byte(preparedJSON), &prepared); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(fixture.root, filepath.FromSlash(prepared.CandidatePath))
	// Any semantic policy change invalidates the stored candidate.
	writeFile(t, fixture.policy, `{
  "schemaVersion": 1,
  "host": "github.com",
  "repository": "example/fixture",
  "baseCommit": "`+fixture.commit+`",
  "allowedSkills": ["fixture-skill"],
  "requiredChecks": ["go-test"]
}
`)
	if _, err := runEvolveCLI(t, fixture.planArgs(candidate)...); err == nil {
		t.Fatal("plan accepted a candidate under a changed policy")
	}
}
