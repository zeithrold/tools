package evolve

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	fixtureSkill   = "fixture-skill"
	fixtureSkillMD = "# Go testing\n\nRun the unit gate first.\n"
	fixtureLicense = "Apache License 2.0\n"
	fixtureNotes   = "# Notes\n\nKeep fixtures small.\n"
)

func baseFileSet() map[string]string {
	return map[string]string{
		"skills/" + fixtureSkill + "/SKILL.md":            fixtureSkillMD,
		"skills/" + fixtureSkill + "/LICENSE":             fixtureLicense,
		"skills/" + fixtureSkill + "/references/notes.md": fixtureNotes,
		"README.md": "a repository that is not only Skills\n",
	}
}

// runGit runs one fixture Git command with an isolated identity and config, so no
// test depends on the machine's global Git configuration.
func runGit(t *testing.T, dir string, args ...string) string {
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(data)+"\n")
}

// newUpstreamWith commits files into a fresh repository. extra may add entries the
// file map cannot express, such as symlinks, before the commit.
func newUpstreamWith(t *testing.T, files map[string]string, extra func(dir string)) (string, string) {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet")
	for name, content := range files {
		writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), content)
	}
	if extra != nil {
		extra(dir)
	}
	runGit(t, dir, "add", "--all")
	runGit(t, dir, "commit", "--quiet", "-m", "fixture")
	return dir, runGit(t, dir, "rev-parse", "HEAD")
}

func newUpstream(t *testing.T, files map[string]string) (string, string) {
	t.Helper()
	return newUpstreamWith(t, files, nil)
}

// harness is one synthetic prepare/plan scenario: an upstream repository, an
// existing workspace, and the two inputs that deliberately live outside it.
type harness struct {
	t         *testing.T
	upstream  string
	commit    string
	workspace string
	policy    string
	proposal  string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	upstream, commit := newUpstream(t, baseFileSet())
	inputs := t.TempDir()
	return &harness{
		t:         t,
		upstream:  upstream,
		commit:    commit,
		workspace: t.TempDir(),
		policy:    filepath.Join(inputs, "policy.json"),
		proposal:  filepath.Join(inputs, "proposal.json"),
	}
}

func (h *harness) useUpstream(dir, commit string) {
	h.upstream, h.commit = dir, commit
	h.writePolicy(policyDoc(commit))
}

func (h *harness) writePolicy(value any) {
	h.t.Helper()
	writeJSONFile(h.t, h.policy, value)
}

func (h *harness) writeProposal(value any) {
	h.t.Helper()
	writeJSONFile(h.t, h.proposal, value)
}

func (h *harness) prepare() (PrepareResult, error) {
	h.t.Helper()
	return Prepare(context.Background(), PrepareOptions{Proposal: h.proposal, Policy: h.policy, Upstream: h.upstream, Root: h.workspace})
}

func (h *harness) mustPrepare() PrepareResult {
	h.t.Helper()
	result, err := h.prepare()
	if err != nil {
		h.t.Fatalf("prepare: %v", err)
	}
	return result
}

func (h *harness) candidatePath(result PrepareResult) string {
	h.t.Helper()
	return filepath.Join(h.workspace, filepath.FromSlash(result.CandidatePath))
}

func (h *harness) plan(candidate string) (PlanResult, error) {
	h.t.Helper()
	return Plan(context.Background(), PlanOptions{Candidate: candidate, Policy: h.policy, Upstream: h.upstream})
}

func policyDoc(commit string) map[string]any {
	return map[string]any{
		"schemaVersion":  1,
		"host":           "github.com",
		"repository":     "example/fixture",
		"baseCommit":     commit,
		"allowedSkills":  []string{fixtureSkill},
		"requiredChecks": []string{"go-test", "repo-lint"},
	}
}

func proposalDoc(commit string, files ...map[string]any) map[string]any {
	return map[string]any{
		"schemaVersion": 1,
		"skill":         fixtureSkill,
		"provenance": map[string]any{
			"repository": "example/upstream",
			"commit":     commit,
			"license":    "Apache-2.0",
			"reason":     "Clarify the first gate for reviewers.",
		},
		"evidence": []map[string]any{{"label": "review-thread", "digest": strings.Repeat("ab", 32)}},
		"files":    files,
	}
}

func fileEntry(path, preimage, content string) map[string]any {
	return map[string]any{"path": path, "preimageSha256": preimage, "content": content}
}

func basePreimage() string {
	return sha256Hex([]byte(fixtureSkillMD))
}

func defaultProposalFiles() []map[string]any {
	return []map[string]any{
		fileEntry("skills/fixture-skill/SKILL.md", basePreimage(), "# Go testing\n\nRun the unit gate, then the focused package.\n"),
		fileEntry("skills/fixture-skill/references/guide.md", "", "# Guide\n\nA new reference.\n"),
	}
}

// snapshotTree maps every path below dir to its kind and content digest, so a
// read-only check also notices a created directory or a touched file. Symbolic
// links are recorded by target rather than followed.
func snapshotTree(t *testing.T, dir string) map[string]string {
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
		switch {
		case entry.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entries[relative] = "symlink:" + target
		case entry.IsDir():
			entries[relative+"/"] = "dir"
		default:
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entries[relative] = "file:" + sha256Hex(data)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", dir, err)
	}
	return entries
}

func equalSnapshots(left, right map[string]string) bool {
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

func assertUntouched(t *testing.T, what string, before, after map[string]string) {
	t.Helper()
	if equalSnapshots(before, after) {
		return
	}
	changed := make([]string, 0)
	for name, value := range after {
		if before[name] != value {
			changed = append(changed, name)
		}
	}
	for name := range before {
		if _, ok := after[name]; !ok {
			changed = append(changed, name+" (removed)")
		}
	}
	sort.Strings(changed)
	t.Fatalf("%s changed: %v", what, changed)
}

func assertNoArtifacts(t *testing.T, workspace string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(workspace, ".zt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("an invalid run created .zt: %v", err)
	}
}

// writeNamedCandidate stores data in the owned layout under its own content id,
// so a test can present bytes that never came from prepare.
func writeNamedCandidate(t *testing.T, workspace string, data []byte) string {
	t.Helper()
	path := filepath.Join(workspace, ".zt", "artifacts", "evolve", sha256Hex(data), candidateName)
	writeFile(t, path, string(data))
	return path
}

func TestPrepareWritesOneOwnedArtifactAndRepeatsCleanly(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))

	workspaceBefore := snapshotTree(t, h.workspace)
	repositoryBefore := snapshotTree(t, h.upstream)

	result := h.mustPrepare()
	if !result.Created {
		t.Fatal("prepare reported no new candidate")
	}
	if len(result.CandidateID) != 64 {
		t.Fatalf("candidate id is not a SHA-256: %q", result.CandidateID)
	}
	if result.Publication != publicationNotReady {
		t.Fatalf("prepare claimed publication %q", result.Publication)
	}
	if result.ProvenanceState != declaredUnverified {
		t.Fatalf("prepare claimed provenance state %q", result.ProvenanceState)
	}
	if len(result.Files) != 2 {
		t.Fatalf("unexpected replacement count: %d", len(result.Files))
	}
	for _, check := range result.Checks {
		if check.Status != checkNotRun {
			t.Fatalf("check %s reported %q", check.Name, check.Status)
		}
	}

	candidateFile := h.candidatePath(result)
	info, err := os.Lstat(candidateFile)
	if err != nil {
		t.Fatalf("candidate file missing: %v", err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("candidate is not a regular file: %v", info.Mode())
	}

	// Exactly the owned artifact tree appeared below the workspace.
	workspaceAfter := snapshotTree(t, h.workspace)
	owned := map[string]bool{
		".zt/":                  true,
		".zt/artifacts/":        true,
		".zt/artifacts/evolve/": true,
		".zt/artifacts/evolve/" + result.CandidateID + "/": true,
		result.CandidatePath: true,
	}
	for name := range workspaceBefore {
		if workspaceAfter[name] != workspaceBefore[name] {
			t.Fatalf("prepare changed %s", name)
		}
	}
	for name := range workspaceAfter {
		if _, existed := workspaceBefore[name]; existed {
			continue
		}
		if !owned[name] {
			t.Fatalf("prepare created an unexpected path: %s", name)
		}
	}
	assertUntouched(t, "the upstream repository", repositoryBefore, snapshotTree(t, h.upstream))

	repeat, err := h.prepare()
	if err != nil {
		t.Fatalf("repeat prepare: %v", err)
	}
	if repeat.Created {
		t.Fatal("repeat prepare reported a new candidate")
	}
	if repeat.CandidateID != result.CandidateID {
		t.Fatalf("candidate id changed between identical runs: %s vs %s", result.CandidateID, repeat.CandidateID)
	}
	assertUntouched(t, "the workspace", workspaceAfter, snapshotTree(t, h.workspace))
}

func TestPrepareIsDeterministicAcrossWorkspaces(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	first := h.mustPrepare()

	second := PrepareResult{}
	other := &harness{t: t, upstream: h.upstream, commit: h.commit, workspace: t.TempDir(), policy: h.policy, proposal: h.proposal}
	second = other.mustPrepare()
	if first.CandidateID != second.CandidateID {
		t.Fatalf("identical inputs produced %s and %s", first.CandidateID, second.CandidateID)
	}
	left, err := os.ReadFile(h.candidatePath(first))
	if err != nil {
		t.Fatal(err)
	}
	right, err := os.ReadFile(other.candidatePath(second))
	if err != nil {
		t.Fatal(err)
	}
	if string(left) != string(right) {
		t.Fatal("identical inputs produced different candidate bytes")
	}
	if strings.Contains(string(left), "time") || strings.Contains(string(left), "status") {
		t.Fatal("candidate bytes contain a mutable field")
	}
}

func TestPlanRevalidatesWithoutWriting(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	prepared := h.mustPrepare()
	candidateFile := h.candidatePath(prepared)

	workspaceBefore := snapshotTree(t, h.workspace)
	repositoryBefore := snapshotTree(t, h.upstream)

	planned, err := h.plan(candidateFile)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if planned.CandidateID != prepared.CandidateID {
		t.Fatalf("plan reported candidate %s, prepared %s", planned.CandidateID, prepared.CandidateID)
	}
	if planned.Publication != publicationNotReady {
		t.Fatalf("plan claimed publication %q", planned.Publication)
	}
	if planned.ProvenanceState != declaredUnverified || planned.EvidenceState != declaredUnverified {
		t.Fatalf("plan treated declarations as verified: %q/%q", planned.ProvenanceState, planned.EvidenceState)
	}
	if len(planned.MissingEvidence) != len(missingEvidence) {
		t.Fatalf("missing evidence is incomplete: %v", planned.MissingEvidence)
	}
	if len(planned.Checks) != 2 {
		t.Fatalf("plan reported %d checks", len(planned.Checks))
	}
	for _, check := range planned.Checks {
		if check.Status != checkNotRun {
			t.Fatalf("check %s reported %q", check.Name, check.Status)
		}
	}
	if len(planned.Files) != len(prepared.Files) {
		t.Fatalf("plan reported %d replacements, prepare reported %d", len(planned.Files), len(prepared.Files))
	}
	for i, file := range planned.Files {
		if file != prepared.Files[i] {
			t.Fatalf("replacement %d differs: %+v vs %+v", i, file, prepared.Files[i])
		}
	}

	assertUntouched(t, "the workspace", workspaceBefore, snapshotTree(t, h.workspace))
	assertUntouched(t, "the upstream repository", repositoryBefore, snapshotTree(t, h.upstream))
}

// TestPlanReadsARelocatedCandidateWithoutWriting pins the location rule plan
// states: a candidate is identified by the content id naming the bytes beside it,
// not by the workspace that prepared it. Copying a candidate out of its workspace
// therefore keeps it plannable, and plan still writes nothing there.
func TestPlanReadsARelocatedCandidateWithoutWriting(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	prepared := h.mustPrepare()
	data, err := os.ReadFile(h.candidatePath(prepared))
	if err != nil {
		t.Fatal(err)
	}

	elsewhere := t.TempDir()
	relocated := filepath.Join(elsewhere, prepared.CandidateID, candidateName)
	writeFile(t, relocated, string(data))
	workspaceBefore := snapshotTree(t, h.workspace)
	elsewhereBefore := snapshotTree(t, elsewhere)

	planned, err := h.plan(relocated)
	if err != nil {
		t.Fatalf("plan refused a candidate whose directory names its own digest: %v", err)
	}
	if planned.CandidateID != prepared.CandidateID {
		t.Fatalf("plan reported candidate %s, prepared %s", planned.CandidateID, prepared.CandidateID)
	}
	assertUntouched(t, "the relocation directory", elsewhereBefore, snapshotTree(t, elsewhere))
	assertUntouched(t, "the workspace", workspaceBefore, snapshotTree(t, h.workspace))
}

func TestChangedPolicyInvalidatesCandidate(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	candidateFile := h.candidatePath(h.mustPrepare())

	writeFile(t, filepath.Join(h.upstream, "later.txt"), "a later commit\n")
	runGit(t, h.upstream, "add", "--all")
	runGit(t, h.upstream, "commit", "--quiet", "-m", "later")
	laterCommit := runGit(t, h.upstream, "rev-parse", "HEAD")

	cases := []struct {
		name   string
		mutate func(policy map[string]any)
		want   string
	}{
		{"required checks", func(policy map[string]any) { policy["requiredChecks"] = []string{"go-test"} }, "policy"},
		{"allowed skills", func(policy map[string]any) { policy["allowedSkills"] = []string{"other-skill"} }, "not allowed"},
		{"base commit", func(policy map[string]any) { policy["baseCommit"] = laterCommit }, "target"},
		{"host", func(policy map[string]any) { policy["host"] = "gitlab.com" }, "target"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			policy := policyDoc(h.commit)
			testCase.mutate(policy)
			h.writePolicy(policy)
			_, err := h.plan(candidateFile)
			if err == nil {
				t.Fatal("plan accepted a candidate under a changed policy")
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPolicyFormattingDoesNotInvalidateCandidate(t *testing.T) {
	h := newHarness(t)
	preparedPolicy := policyDoc(h.commit)
	preparedPolicy["allowedSkills"] = []string{"fixture-skill", "other-skill"}
	preparedPolicy["requiredChecks"] = []string{"go-test", "repo-lint"}
	h.writePolicy(preparedPolicy)
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	candidateFile := h.candidatePath(h.mustPrepare())

	// The same policy written with reversed lists and padded strings has the same
	// meaning, so the candidate stays valid.
	reformatted := policyDoc(h.commit)
	reformatted["allowedSkills"] = []string{" other-skill ", "fixture-skill"}
	reformatted["requiredChecks"] = []string{"repo-lint", "go-test"}
	h.writePolicy(reformatted)
	if _, err := h.plan(candidateFile); err != nil {
		t.Fatalf("plan rejected a semantically identical policy: %v", err)
	}
}

func TestPlanRefusesTamperedCandidates(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	prepared := h.mustPrepare()
	canonical, err := os.ReadFile(h.candidatePath(prepared))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("edited content", func(t *testing.T) {
		tampered := strings.Replace(string(canonical), "focused package", "unreviewed package", 1)
		candidateFile := h.candidatePath(prepared)
		writeFile(t, candidateFile, tampered)
		_, err := h.plan(candidateFile)
		if err == nil || !strings.Contains(err.Error(), "modified after preparation") {
			t.Fatalf("plan did not refuse edited bytes: %v", err)
		}
	})

	t.Run("trailing whitespace", func(t *testing.T) {
		candidateFile := h.candidatePath(prepared)
		writeFile(t, candidateFile, string(canonical)+"\n")
		_, err := h.plan(candidateFile)
		if err == nil {
			t.Fatal("plan accepted non-canonical bytes")
		}
	})

	t.Run("symlinked file", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need elevated privileges on Windows")
		}
		directory := filepath.Join(h.workspace, ".zt", "artifacts", "evolve", prepared.CandidateID)
		real := filepath.Join(directory, "real.json")
		writeFile(t, real, string(canonical))
		candidateFile := filepath.Join(directory, candidateName)
		if err := os.Remove(candidateFile); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, candidateFile); err != nil {
			t.Fatal(err)
		}
		_, err := h.plan(candidateFile)
		if err == nil || !strings.Contains(err.Error(), "symlinked candidate file") {
			t.Fatalf("plan did not refuse a symlinked file: %v", err)
		}
	})

	t.Run("symlinked directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need elevated privileges on Windows")
		}
		directory := filepath.Join(h.workspace, ".zt", "artifacts", "evolve", prepared.CandidateID)
		link := filepath.Join(h.workspace, ".zt", "artifacts", "evolve", strings.Repeat("cd", 32))
		if err := os.Symlink(directory, link); err != nil {
			t.Fatal(err)
		}
		_, err := h.plan(filepath.Join(link, candidateName))
		if err == nil || !strings.Contains(err.Error(), "symlinked candidate directory") {
			t.Fatalf("plan did not refuse a symlinked directory: %v", err)
		}
	})

	t.Run("outside the owned layout", func(t *testing.T) {
		candidateFile := filepath.Join(t.TempDir(), candidateName)
		writeFile(t, candidateFile, string(canonical))
		_, err := h.plan(candidateFile)
		if err == nil || !strings.Contains(err.Error(), "must be stored as") {
			t.Fatalf("plan did not refuse a candidate outside its identity path: %v", err)
		}
	})

	t.Run("duplicate key", func(t *testing.T) {
		tampered := strings.Replace(string(canonical), `"skill": "fixture-skill",`, `"skill": "fixture-skill", "skill": "x",`, 1)
		_, err := h.plan(writeNamedCandidate(t, h.workspace, []byte(tampered)))
		if err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
			t.Fatalf("plan accepted a duplicate key: %v", err)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		tampered := strings.Replace(string(canonical), `"skill": "fixture-skill",`, `"skill": "fixture-skill",
  "status": "passed",`, 1)
		_, err := h.plan(writeNamedCandidate(t, h.workspace, []byte(tampered)))
		if err == nil || !strings.Contains(err.Error(), "documented schema") {
			t.Fatalf("plan accepted an unknown field: %v", err)
		}
	})

	t.Run("reindented", func(t *testing.T) {
		var document map[string]any
		if err := json.Unmarshal(canonical, &document); err != nil {
			t.Fatal(err)
		}
		tampered, err := json.MarshalIndent(document, "", "    ")
		if err != nil {
			t.Fatal(err)
		}
		_, err = h.plan(writeNamedCandidate(t, h.workspace, append(tampered, '\n')))
		if err == nil || !strings.Contains(err.Error(), "not canonical") {
			t.Fatalf("plan accepted a re-indented candidate: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := h.plan(writeNamedCandidate(t, h.workspace, []byte(`{"schemaVersion":`)))
		if err == nil {
			t.Fatal("plan accepted invalid JSON")
		}
	})
}

func TestPrepareRejectsUnsafeReplacementPaths(t *testing.T) {
	cases := []struct {
		name  string
		files []map[string]any
		want  string
	}{
		{
			name:  "traversal",
			files: []map[string]any{fileEntry("skills/fixture-skill/../other/SKILL.md", basePreimage(), "x\n")},
			want:  "not a plain file name",
		},
		{
			name:  "another skill",
			files: []map[string]any{fileEntry("skills/other-skill/SKILL.md", "", "x\n")},
			want:  "outside skills/fixture-skill",
		},
		{
			name:  "absolute",
			files: []map[string]any{fileEntry("/skills/fixture-skill/SKILL.md", basePreimage(), "x\n")},
			want:  "relative to the repository root",
		},
		{
			name:  "backslash",
			files: []map[string]any{fileEntry("skills\\fixture-skill\\SKILL.md", basePreimage(), "x\n")},
			want:  "forward slashes",
		},
		{
			name:  "script",
			files: []map[string]any{fileEntry("skills/fixture-skill/scripts/run.sh", "", "rm -rf /\n")},
			want:  "must be skills/fixture-skill/SKILL.md",
		},
		{
			name:  "non-markdown reference",
			files: []map[string]any{fileEntry("skills/fixture-skill/references/notes.txt", "", "x\n")},
			want:  "must be skills/fixture-skill/SKILL.md",
		},
		{
			name:  "control character in path",
			files: []map[string]any{fileEntry("skills/fixture-skill/references/no\nte.md", "", "x\n")},
			want:  "control character",
		},
		{
			name:  "trailing separator",
			files: []map[string]any{fileEntry("skills/fixture-skill/SKILL.md/", basePreimage(), "x\n")},
			want:  "not a plain file name",
		},
		{
			name: "file and directory",
			files: []map[string]any{
				fileEntry("skills/fixture-skill/references/a.md", "", "x\n"),
				fileEntry("skills/fixture-skill/references/a.md/b.md", "", "y\n"),
			},
			want: "both a file and a directory",
		},
		{
			name: "case collision",
			files: []map[string]any{
				fileEntry("skills/fixture-skill/references/A.md", "", "x\n"),
				fileEntry("skills/fixture-skill/references/a.md", "", "y\n"),
			},
			want: "differs only in case",
		},
		{
			name:  "wrong preimage",
			files: []map[string]any{fileEntry("skills/fixture-skill/SKILL.md", strings.Repeat("0", 64), "x\n")},
			want:  "preimage sha256",
		},
		{
			name:  "missing preimage for an existing file",
			files: []map[string]any{fileEntry("skills/fixture-skill/SKILL.md", "", "x\n")},
			want:  "preimageSha256 is empty",
		},
		{
			name:  "preimage for a new file",
			files: []map[string]any{fileEntry("skills/fixture-skill/references/new.md", basePreimage(), "x\n")},
			want:  "does not exist at the base commit",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			h := newHarness(t)
			h.writePolicy(policyDoc(h.commit))
			h.writeProposal(proposalDoc(h.commit, testCase.files...))
			if _, err := h.prepare(); err == nil {
				t.Fatal("prepare accepted an unsafe replacement")
			} else if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("unexpected error: %v", err)
			}
			assertNoArtifacts(t, h.workspace)
		})
	}
}

func TestPrepareRejectsUnsafeBaseTrees(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Run("symlink ancestor", func(t *testing.T) {
			files := baseFileSet()
			upstream, commit := newUpstreamWith(t, files, func(dir string) {
				if err := os.Symlink("notes.md", filepath.Join(dir, "skills", fixtureSkill, "references", "link")); err != nil {
					t.Fatal(err)
				}
			})
			h := newHarness(t)
			h.useUpstream(upstream, commit)
			h.writeProposal(proposalDoc(commit, fileEntry("skills/fixture-skill/references/link/inside.md", "", "x\n")))
			_, err := h.prepare()
			if err == nil || !strings.Contains(err.Error(), "not a directory") {
				t.Fatalf("prepare walked through a symlink ancestor: %v", err)
			}
		})
	}

	t.Run("file ancestor", func(t *testing.T) {
		files := baseFileSet()
		delete(files, "skills/"+fixtureSkill+"/references/notes.md")
		files["skills/"+fixtureSkill+"/references/plain"] = "a plain file\n"
		upstream, commit := newUpstream(t, files)
		h := newHarness(t)
		h.useUpstream(upstream, commit)
		h.writeProposal(proposalDoc(commit, fileEntry("skills/fixture-skill/references/plain/inside.md", "", "x\n")))
		_, err := h.prepare()
		if err == nil || !strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("prepare accepted a file as an ancestor: %v", err)
		}
	})

	t.Run("base path differing only in case", func(t *testing.T) {
		files := baseFileSet()
		delete(files, "skills/"+fixtureSkill+"/references/notes.md")
		files["skills/"+fixtureSkill+"/references/Notes.md"] = fixtureNotes
		upstream, commit := newUpstream(t, files)
		h := newHarness(t)
		h.useUpstream(upstream, commit)
		h.writeProposal(proposalDoc(commit, fileEntry("skills/fixture-skill/references/notes.md", "", "x\n")))
		_, err := h.prepare()
		if err == nil || !strings.Contains(err.Error(), "differs only in case") {
			t.Fatalf("prepare accepted a case collision with the base: %v", err)
		}
	})

	t.Run("replacement would drop base content", func(t *testing.T) {
		files := baseFileSet()
		delete(files, "skills/"+fixtureSkill+"/LICENSE")
		files["skills/"+fixtureSkill+"/LICENSE/inside.md"] = "nested\n"
		upstream, commit := newUpstream(t, files)
		h := newHarness(t)
		h.useUpstream(upstream, commit)
		h.writeProposal(proposalDoc(commit, fileEntry("skills/fixture-skill/LICENSE", "", "Apache License 2.0\n")))
		_, err := h.prepare()
		if err == nil || !strings.Contains(err.Error(), "below the replacement path") {
			t.Fatalf("prepare accepted a replacement that would drop base content: %v", err)
		}
	})
}

func TestPrepareRejectsInvalidDocuments(t *testing.T) {
	type testCase struct {
		name     string
		proposal func(commit string) any
		policy   func(commit string) any
		want     string
	}
	cases := []testCase{
		{
			name: "proposal unknown field",
			proposal: func(commit string) any {
				document := proposalDoc(commit, defaultProposalFiles()...)
				document["mode"] = 420
				return document
			},
			want: "documented schema",
		},
		{
			name: "proposal duplicate key",
			proposal: func(commit string) any {
				return rawJSON(`{"schemaVersion": 1, "skill": "fixture-skill", "skill": "fixture-skill", "provenance": {"repository": "example/upstream", "commit": "` + commit + `", "license": "MIT", "reason": "r"}, "files": []}`)
			},
			want: "duplicate JSON object key",
		},
		{
			name: "proposal trailing data",
			proposal: func(commit string) any {
				return rawJSON(mustMarshal(proposalDoc(commit, defaultProposalFiles()...)) + `{}`)
			},
			want: "trailing JSON data",
		},
		{
			name: "proposal invalid JSON",
			proposal: func(string) any {
				return rawJSON(`{"schemaVersion": 1,`)
			},
			want: "incomplete JSON",
		},
		{
			name: "proposal schema version",
			proposal: func(commit string) any {
				document := proposalDoc(commit, defaultProposalFiles()...)
				document["schemaVersion"] = 2
				return document
			},
			want: "schemaVersion",
		},
		{
			name: "proposal without replacements",
			proposal: func(commit string) any {
				document := proposalDoc(commit)
				document["files"] = []map[string]any{}
				return document
			},
			want: "at least one replacement",
		},
		{
			name: "proposal with a null replacement list",
			proposal: func(commit string) any {
				document := proposalDoc(commit)
				document["files"] = nil
				return document
			},
			want: "null values",
		},
		{
			name: "proposal too many replacements",
			proposal: func(commit string) any {
				files := make([]map[string]any, maxFiles+1)
				for i := range files {
					files[i] = fileEntry("skills/fixture-skill/references/doc-"+strings.Repeat("a", i)+".md", "", "x\n")
				}
				return proposalDoc(commit, files...)
			},
			want: "at most",
		},
		{
			name: "proposal oversized content",
			proposal: func(commit string) any {
				return proposalDoc(commit, fileEntry("skills/fixture-skill/references/big.md", "", strings.Repeat("a", maxFileBytes+1)))
			},
			want: "content exceeds",
		},
		{
			name: "proposal binary content",
			proposal: func(commit string) any {
				return rawJSON(strings.Replace(mustMarshal(proposalDoc(commit, fileEntry("skills/fixture-skill/references/bin.md", "", "placeholder\n"))), "placeholder", `\u0000\u0001`, 1))
			},
			want: "control character",
		},
		{
			name: "proposal unasserted license",
			proposal: func(commit string) any {
				document := proposalDoc(commit, defaultProposalFiles()...)
				document["provenance"].(map[string]any)["license"] = "NOASSERTION"
				return document
			},
			want: "redistribution license",
		},
		{
			name: "proposal without reason",
			proposal: func(commit string) any {
				document := proposalDoc(commit, defaultProposalFiles()...)
				document["provenance"].(map[string]any)["reason"] = "  "
				return document
			},
			want: "reason",
		},
		{
			name: "proposal without source commit",
			proposal: func(commit string) any {
				document := proposalDoc(commit, defaultProposalFiles()...)
				document["provenance"].(map[string]any)["commit"] = "abc"
				return document
			},
			want: "complete lowercase Git object id",
		},
		{
			name: "policy unknown field",
			policy: func(commit string) any {
				document := policyDoc(commit)
				document["commands"] = []string{"go", "test"}
				return document
			},
			want: "documented schema",
		},
		{
			name: "policy host with scheme",
			policy: func(commit string) any {
				document := policyDoc(commit)
				document["host"] = "https://github.com"
				return document
			},
			want: "host",
		},
		{
			name: "policy repository without owner",
			policy: func(commit string) any {
				document := policyDoc(commit)
				document["repository"] = "fixture"
				return document
			},
			want: "owner/name",
		},
		{
			name: "policy abbreviated base commit",
			policy: func(commit string) any {
				document := policyDoc(commit[:39])
				return document
			},
			want: "baseCommit",
		},
		{
			name: "policy without checks",
			policy: func(commit string) any {
				document := policyDoc(commit)
				document["requiredChecks"] = []string{}
				return document
			},
			want: "requiredChecks",
		},
		{
			name: "policy duplicate skill",
			policy: func(commit string) any {
				document := policyDoc(commit)
				document["allowedSkills"] = []string{fixtureSkill, fixtureSkill}
				return document
			},
			want: "repeats",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			h := newHarness(t)
			if testCase.policy != nil {
				writeRawOrValue(t, h.policy, testCase.policy(h.commit))
			} else {
				h.writePolicy(policyDoc(h.commit))
			}
			if testCase.proposal != nil {
				writeRawOrValue(t, h.proposal, testCase.proposal(h.commit))
			} else {
				h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
			}
			if _, err := h.prepare(); err == nil {
				t.Fatal("prepare accepted an invalid document")
			} else if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("unexpected error: %v", err)
			}
			assertNoArtifacts(t, h.workspace)
		})
	}
}

// rawJSON marks a pre-built document so a table entry can supply malformed JSON.
type rawJSON string

func writeRawOrValue(t *testing.T, path string, value any) {
	t.Helper()
	if raw, ok := value.(rawJSON); ok {
		writeFile(t, path, string(raw))
		return
	}
	writeJSONFile(t, path, value)
}

func mustMarshal(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

func TestValidateFilesRejectsWrongDigestsAndNonTextContent(t *testing.T) {
	valid := func() File {
		content := "readable\n"
		return File{Path: "skills/fixture-skill/references/doc.md", SHA256: sha256Hex([]byte(content)), Bytes: len(content), Content: content}
	}
	if err := validateFiles([]File{valid()}, fixtureSkill); err != nil {
		t.Fatalf("valid replacement rejected: %v", err)
	}

	cases := []struct {
		name string
		edit func(file *File)
		want string
	}{
		{"digest", func(file *File) { file.SHA256 = strings.Repeat("0", 64) }, "sha256 does not match"},
		{"bytes", func(file *File) { file.Bytes = 1 }, "does not match the content"},
		{"control character", func(file *File) {
			file.Content = "a\x00b"
			file.SHA256 = sha256Hex([]byte(file.Content))
			file.Bytes = len(file.Content)
		}, "control character"},
		{"invalid UTF-8", func(file *File) {
			file.Content = "a\xffb"
			file.SHA256 = sha256Hex([]byte(file.Content))
			file.Bytes = len(file.Content)
		}, "not valid UTF-8"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			file := valid()
			testCase.edit(&file)
			if err := validateFiles([]File{file}, fixtureSkill); err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("unexpected result: %v", err)
			}
		})
	}
}

func TestPrepareRejectsMissingOrAdversarialInputs(t *testing.T) {
	t.Run("missing base object", func(t *testing.T) {
		h := newHarness(t)
		policy := policyDoc(strings.Repeat("0", 40))
		h.writePolicy(policy)
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		_, err := h.prepare()
		if err == nil || !strings.Contains(err.Error(), "never fetches") {
			t.Fatalf("prepare accepted a missing base object: %v", err)
		}
		assertNoArtifacts(t, h.workspace)
	})

	t.Run("workspace does not exist", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		missing := filepath.Join(t.TempDir(), "absent")
		_, err := Prepare(context.Background(), PrepareOptions{Proposal: h.proposal, Policy: h.policy, Upstream: h.upstream, Root: missing})
		if err == nil {
			t.Fatal("prepare accepted a workspace that does not exist")
		}
	})

	t.Run("workspace is a file", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		file := filepath.Join(t.TempDir(), "workspace")
		writeFile(t, file, "not a directory\n")
		_, err := Prepare(context.Background(), PrepareOptions{Proposal: h.proposal, Policy: h.policy, Upstream: h.upstream, Root: file})
		if err == nil {
			t.Fatal("prepare accepted a file as the workspace")
		}
	})

	t.Run("symlinked artifact component", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need elevated privileges on Windows")
		}
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		elsewhere := t.TempDir()
		if err := os.Symlink(elsewhere, filepath.Join(h.workspace, ".zt")); err != nil {
			t.Fatal(err)
		}
		_, err := h.prepare()
		if err == nil || !strings.Contains(err.Error(), "symlinked artifact path") {
			t.Fatalf("prepare followed a symlinked artifact component: %v", err)
		}
		if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
			t.Fatalf("prepare wrote through the symlink: %v", entries)
		}
	})

	t.Run("artifact component is a file", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		writeFile(t, filepath.Join(h.workspace, ".zt", "artifacts", "evolve"), "not a directory\n")
		_, err := h.prepare()
		if err == nil || !strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("prepare accepted a file as an artifact component: %v", err)
		}
	})

	t.Run("base file above the replacement bound", func(t *testing.T) {
		files := baseFileSet()
		big := strings.Repeat("a", maxFileBytes+1)
		files["skills/"+fixtureSkill+"/SKILL.md"] = big
		upstream, commit := newUpstream(t, files)
		h := newHarness(t)
		h.useUpstream(upstream, commit)
		h.writeProposal(proposalDoc(commit, fileEntry("skills/fixture-skill/SKILL.md", sha256Hex([]byte(big)), "small\n")))
		_, err := h.prepare()
		if err == nil || !strings.Contains(err.Error(), "replacement bound") {
			t.Fatalf("prepare accepted an oversized base file: %v", err)
		}
	})

	t.Run("symlinked proposal", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlinks need elevated privileges on Windows")
		}
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		real := filepath.Join(t.TempDir(), "proposal.json")
		writeJSONFile(t, real, proposalDoc(h.commit, defaultProposalFiles()...))
		if err := os.Symlink(real, h.proposal); err != nil {
			t.Fatal(err)
		}
		if _, err := h.prepare(); err == nil || !strings.Contains(err.Error(), "refusing symlinked file") {
			t.Fatalf("prepare followed a symlinked proposal: %v", err)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := Prepare(ctx, PrepareOptions{Proposal: h.proposal, Policy: h.policy, Upstream: h.upstream, Root: h.workspace})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("prepare ignored cancellation: %v", err)
		}
		assertNoArtifacts(t, h.workspace)
	})
}

func TestPrepareKeepsLeftoverStageDirectories(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	// A run interrupted before the atomic install leaves an identifiable owned
	// stage directory. prepare must ignore it as a candidate and must not delete
	// content it cannot prove it owns.
	stage := filepath.Join(h.workspace, ".zt", "artifacts", "evolve", stagePrefix+"deadbeefdeadbeef")
	writeFile(t, filepath.Join(stage, candidateName), "interrupted")

	result := h.mustPrepare()
	if _, err := os.Stat(filepath.Join(stage, candidateName)); err != nil {
		t.Fatalf("prepare removed a leftover stage directory: %v", err)
	}
	if _, err := os.Stat(h.candidatePath(result)); err != nil {
		t.Fatalf("prepare did not install the candidate beside the leftover stage: %v", err)
	}
}

func TestCandidateDirectoriesAreImmutable(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	prepared := h.mustPrepare()

	store, err := openStore(h.workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.install(t.Context(), prepared.CandidateID, []byte("different bytes")); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("install replaced an existing candidate: %v", err)
	}

	// A changed proposal is a different candidate and never overwrites the first.
	th := newHarness(t)
	th.upstream, th.commit, th.workspace = h.upstream, h.commit, h.workspace
	th.policy, th.proposal = h.policy, filepath.Join(t.TempDir(), "proposal.json")
	th.writeProposal(proposalDoc(h.commit, fileEntry("skills/fixture-skill/SKILL.md", basePreimage(), "# Go testing\n\nA different edit.\n")))
	changed := th.mustPrepare()
	if changed.CandidateID == prepared.CandidateID {
		t.Fatal("different content produced the same candidate id")
	}
	for _, candidate := range []PrepareResult{prepared, changed} {
		if _, err := os.Stat(h.candidatePath(candidate)); err != nil {
			t.Fatalf("candidate %s is missing: %v", candidate.CandidateID, err)
		}
	}
}

func TestPlanHonoursCancellation(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	candidateFile := h.candidatePath(h.mustPrepare())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Plan(ctx, PlanOptions{Candidate: candidateFile, Policy: h.policy, Upstream: h.upstream}); !errors.Is(err, context.Canceled) {
		t.Fatalf("plan ignored cancellation: %v", err)
	}
}

// gitStartMarker puts a git wrapper first on PATH that records every invocation,
// so a test can assert that zt refused a repository layout before starting Git. A
// refused repository must not fail inside Git instead: Git is what would read the
// configuration zt promised to inspect first.
func gitStartMarker(t *testing.T) string {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	bin := t.TempDir()
	marker := filepath.Join(bin, "git-started")
	// The wrapper records the start and then behaves exactly like Git, so a layout
	// Git would accept still reaches Git if zt lets it through.
	script := "#!/bin/sh\nprintf 'started\\n' >> '" + marker + "'\nexec '" + realGit + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return marker
}

// appendConfig adds text to an existing Git config file, so a test can inject a
// directive a repository could carry.
func appendConfig(t *testing.T, path, text string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(data)+text)
}

// replaceIn replaces the first occurrence of marker in value, so a test can build
// a document the encoder would never produce.
func replaceIn(t *testing.T, value, marker, replacement string) string {
	t.Helper()
	if !strings.Contains(value, marker) {
		t.Fatalf("fixture marker %q not found", marker)
	}
	return strings.Replace(value, marker, replacement, 1)
}

func TestPrepareAndPlanRoundTripEscapeHeavyContent(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	// The canonical document used to HTML-escape '<', '>' and '&', growing these
	// two replacements past the size plan accepts. The proposal is written raw,
	// because those characters need no JSON escape, so the input stays inside the
	// proposal bound while the old canonical encoding reached several megabytes.
	content := strings.Repeat("&", maxFileBytes)
	other := strings.Repeat("<", maxFileBytes)
	writeFile(t, h.proposal, `{
  "schemaVersion": 1,
  "skill": "`+fixtureSkill+`",
  "provenance": {
    "repository": "example/upstream",
    "commit": "`+h.commit+`",
    "license": "Apache-2.0",
    "reason": "Bound the canonical document."
  },
  "files": [
    {"path": "skills/`+fixtureSkill+`/references/amp.md", "preimageSha256": "", "content": "`+content+`"},
    {"path": "skills/`+fixtureSkill+`/references/lt.md", "preimageSha256": "", "content": "`+other+`"}
  ]
}
`)
	result := h.mustPrepare()
	candidateFile := h.candidatePath(result)
	data, err := os.ReadFile(candidateFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > maxCandidateBytes {
		t.Fatalf("prepare wrote a %d byte candidate, above the %d byte reader bound", len(data), maxCandidateBytes)
	}
	if len(data) < maxFileBytes {
		t.Fatalf("candidate shrank unexpectedly: %d bytes", len(data))
	}
	if !bytes.Contains(data, []byte("&&&&&&&&&&")) || !bytes.Contains(data, []byte("<<<<<<<<<<")) {
		t.Fatal("canonical bytes still HTML-escape the replacement content")
	}
	if _, err := h.plan(candidateFile); err != nil {
		t.Fatalf("plan rejected the candidate prepare wrote: %v", err)
	}
}

func TestDocumentsRequireExactFieldSpellingAndReplacementContent(t *testing.T) {
	cases := []struct {
		name     string
		proposal func(commit string) any
		policy   func(commit string) any
		want     string
	}{
		{
			name: "case-variant replacement field",
			proposal: func(commit string) any {
				raw := mustMarshal(proposalDoc(commit, defaultProposalFiles()...))
				// encoding/json matches struct fields case-insensitively, so this
				// key would otherwise assign the same field as "path".
				return rawJSON(replaceIn(t, raw, `"path": "skills/fixture-skill/SKILL.md",`, `"path": "skills/fixture-skill/SKILL.md", "PATH": "skills/fixture-skill/LICENSE",`))
			},
			want: "not the documented spelling",
		},
		{
			name: "case-variant nested field",
			proposal: func(commit string) any {
				raw := mustMarshal(proposalDoc(commit, defaultProposalFiles()...))
				return rawJSON(replaceIn(t, raw, `"license": "Apache-2.0",`, `"license": "Apache-2.0", "License": "MIT",`))
			},
			want: "not the documented spelling",
		},
		{
			name: "case-variant policy field",
			policy: func(commit string) any {
				raw := mustMarshal(policyDoc(commit))
				return rawJSON(replaceIn(t, raw, `"allowedSkills": [`, `"ALLOWEDSKILLS": ["other-skill"], "allowedSkills": [`))
			},
			want: "not the documented spelling",
		},
		{
			name: "omitted replacement content",
			proposal: func(commit string) any {
				document := proposalDoc(commit, defaultProposalFiles()...)
				delete(document["files"].([]map[string]any)[0], "content")
				return document
			},
			want: "required field",
		},
		{
			name: "null replacement content",
			proposal: func(commit string) any {
				document := proposalDoc(commit, defaultProposalFiles()...)
				document["files"].([]map[string]any)[0]["content"] = nil
				return document
			},
			want: "null values",
		},
		{
			name: "omitted skill",
			proposal: func(commit string) any {
				document := proposalDoc(commit, defaultProposalFiles()...)
				delete(document, "skill")
				return document
			},
			want: "required field",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			h := newHarness(t)
			if testCase.policy != nil {
				writeRawOrValue(t, h.policy, testCase.policy(h.commit))
			} else {
				h.writePolicy(policyDoc(h.commit))
			}
			if testCase.proposal != nil {
				writeRawOrValue(t, h.proposal, testCase.proposal(h.commit))
			} else {
				h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
			}
			if _, err := h.prepare(); err == nil {
				t.Fatal("prepare accepted a document that is not the documented schema")
			} else if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("unexpected error: %v", err)
			}
			assertNoArtifacts(t, h.workspace)
		})
	}
}

func TestDocumentsRejectLossyUnicode(t *testing.T) {
	t.Run("invalid UTF-8 in the proposal", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		raw := mustMarshal(proposalDoc(h.commit, defaultProposalFiles()...))
		// A literal 0xff byte is not valid UTF-8; encoding/json would replace it
		// with U+FFFD and store bytes the proposal never named.
		raw = replaceIn(t, raw, "focused package", "focused \xffpackage")
		writeFile(t, h.proposal, raw)
		if _, err := h.prepare(); err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
			t.Fatalf("prepare repaired invalid UTF-8: %v", err)
		}
		assertNoArtifacts(t, h.workspace)
	})

	t.Run("unpaired surrogate escape in the proposal", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		raw := mustMarshal(proposalDoc(h.commit, defaultProposalFiles()...))
		raw = replaceIn(t, raw, "focused package", `focused \ud800package`)
		writeFile(t, h.proposal, raw)
		if _, err := h.prepare(); err == nil || !strings.Contains(err.Error(), "unpaired surrogate") {
			t.Fatalf("prepare repaired an unpaired surrogate: %v", err)
		}
		assertNoArtifacts(t, h.workspace)
	})

	t.Run("unpaired surrogate escape in a stored candidate", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		prepared := h.mustPrepare()
		data, err := os.ReadFile(h.candidatePath(prepared))
		if err != nil {
			t.Fatal(err)
		}
		tampered := replaceIn(t, string(data), "focused package", `focused \udc00package`)
		_, err = h.plan(writeNamedCandidate(t, h.workspace, []byte(tampered)))
		if err == nil || !strings.Contains(err.Error(), "unpaired surrogate") {
			t.Fatalf("plan repaired an unpaired surrogate: %v", err)
		}
	})
}

func TestPrepareRejectsNonDirectorySkillsAncestor(t *testing.T) {
	for _, kind := range []string{"file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			files := map[string]string{"README.md": "base\n"}
			if kind == "file" {
				files["skills"] = "not a directory\n"
			}
			upstream, commit := newUpstreamWith(t, files, func(dir string) {
				if kind != "symlink" {
					return
				}
				if runtime.GOOS == "windows" {
					t.Skip("symlinks need elevated privileges on Windows")
				}
				if err := os.Symlink("elsewhere", filepath.Join(dir, "skills")); err != nil {
					t.Fatal(err)
				}
			})
			h := newHarness(t)
			h.useUpstream(upstream, commit)
			// The scoped listing for skills/<id> is empty here, so only a check that
			// starts at the repository root can see the blocking entry.
			h.writeProposal(proposalDoc(commit, fileEntry("skills/fixture-skill/SKILL.md", "", "new\n")))
			_, err := h.prepare()
			if err == nil || !strings.Contains(err.Error(), "not a directory") {
				t.Fatalf("prepare accepted a replacement below a base %s: %v", kind, err)
			}
			assertNoArtifacts(t, h.workspace)
		})
	}
}

func TestPrepareRejectsCaseFoldedDirectoryCollisions(t *testing.T) {
	cases := []struct {
		name  string
		base  map[string]string
		files []map[string]any
	}{
		{
			"base directory versus proposed file",
			map[string]string{"skills/fixture-skill/references/Guide.md/child.md": "keep\n"},
			[]map[string]any{fileEntry("skills/fixture-skill/references/guide.md", "", "new\n")},
		},
		{
			"base parent directory casing",
			map[string]string{"skills/fixture-skill/references/Guide/child.md": "keep\n"},
			[]map[string]any{fileEntry("skills/fixture-skill/references/guide/new.md", "", "new\n")},
		},
		{
			"proposal file versus differently cased ancestor",
			map[string]string{"README.md": "base\n"},
			[]map[string]any{
				fileEntry("skills/fixture-skill/references/Guide.md", "", "a\n"),
				fileEntry("skills/fixture-skill/references/guide.md/child.md", "", "b\n"),
			},
		},
		{
			"proposal directories differing only in case",
			map[string]string{"README.md": "base\n"},
			[]map[string]any{
				fileEntry("skills/fixture-skill/references/Guide/a.md", "", "a\n"),
				fileEntry("skills/fixture-skill/references/guide/b.md", "", "b\n"),
			},
		},
		{
			"base skill directory casing",
			map[string]string{"skills/Fixture-Skill/SKILL.md": "keep\n"},
			[]map[string]any{fileEntry("skills/fixture-skill/SKILL.md", "", "new\n")},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			upstream, commit := newUpstream(t, testCase.base)
			h := newHarness(t)
			h.useUpstream(upstream, commit)
			h.writeProposal(proposalDoc(commit, testCase.files...))
			_, err := h.prepare()
			if err == nil {
				t.Fatalf("prepare accepted %v against base %v", testCase.files, testCase.base)
			}
			if !strings.Contains(err.Error(), "differs only in case") && !strings.Contains(err.Error(), "both a file and a directory") {
				t.Fatalf("unexpected error: %v", err)
			}
			assertNoArtifacts(t, h.workspace)
		})
	}
}

func TestDocumentErrorsDoNotEchoPayload(t *testing.T) {
	run := func(t *testing.T, build func(commit string) any) error {
		t.Helper()
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(build(h.commit))
		_, err := h.prepare()
		if err == nil {
			t.Fatal("prepare accepted a malformed proposal")
		}
		if len(err.Error()) > 300 {
			t.Fatalf("error is %d bytes long: %q", len(err.Error()), err.Error())
		}
		if strings.Contains(err.Error(), "PRIVATE_REVIEW_MARKER") {
			t.Fatalf("error echoes the document payload: %v", err)
		}
		assertNoArtifacts(t, h.workspace)
		return err
	}

	marker := "PRIVATE_REVIEW_MARKER_" + strings.Repeat("x", 4096)
	unknown := run(t, func(commit string) any {
		document := proposalDoc(commit, defaultProposalFiles()...)
		document[marker] = "unused"
		return document
	})
	if !strings.Contains(unknown.Error(), "documented schema") {
		t.Fatalf("unexpected error: %v", unknown)
	}

	typed := run(t, func(commit string) any {
		document := proposalDoc(commit, defaultProposalFiles()...)
		document["schemaVersion"] = marker
		return document
	})
	if !strings.Contains(typed.Error(), "wrong JSON type") {
		t.Fatalf("unexpected error: %v", typed)
	}
}

func TestGitTracingCannotWriteOutsideTheWorkspace(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	outside := t.TempDir()
	trace := filepath.Join(outside, "trace.log")
	t.Setenv("GIT_TRACE", trace)
	t.Setenv("GIT_TRACE2_EVENT", trace+".json")
	t.Setenv("GIT_TRACE_PERFORMANCE", trace+".perf")
	prepared := h.mustPrepare()
	if _, err := h.plan(h.candidatePath(prepared)); err != nil {
		t.Fatalf("plan: %v", err)
	}
	for _, path := range []string{trace, trace + ".json", trace + ".perf"} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("git tracing wrote outside the workspace: %s", path)
		}
	}
}

func TestPinnedBaseIgnoresGitReplacementRefs(t *testing.T) {
	h := newHarness(t)
	writeFile(t, filepath.Join(h.upstream, "skills", fixtureSkill, "SKILL.md"), "replacement ref content\n")
	runGit(t, h.upstream, "add", "--all")
	runGit(t, h.upstream, "commit", "--quiet", "-m", "replacement")
	replacement := runGit(t, h.upstream, "rev-parse", "HEAD")
	runGit(t, h.upstream, "replace", h.commit, replacement)
	t.Cleanup(func() { runGit(t, h.upstream, "replace", "-d", h.commit) })
	h.writePolicy(policyDoc(h.commit))

	// A preimage taken from the replacement object must not satisfy the pinned id.
	h.writeProposal(proposalDoc(h.commit, fileEntry("skills/fixture-skill/SKILL.md", sha256Hex([]byte("replacement ref content\n")), "new\n")))
	if _, err := h.prepare(); err == nil || !strings.Contains(err.Error(), "preimage sha256") {
		t.Fatalf("prepare followed a replace ref: %v", err)
	}

	// The pinned commit's own tree still works.
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	if _, err := h.prepare(); err != nil {
		t.Fatalf("prepare rejected the true base tree: %v", err)
	}
}

func TestRepositoryConfigIncludesAreRefused(t *testing.T) {
	const include = "\n[include]\n\tpath = /nonexistent/zt-review\n"

	t.Run("local config include", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		appendConfig(t, filepath.Join(h.upstream, ".git", "config"), include)
		if _, err := h.prepare(); err == nil || !strings.Contains(err.Error(), "include") {
			t.Fatalf("prepare accepted a repository with a config include: %v", err)
		}
		assertNoArtifacts(t, h.workspace)
	})

	t.Run("conditional include", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		appendConfig(t, filepath.Join(h.upstream, ".git", "config"), "\n[includeIf \"gitdir:~/elsewhere/\"]\n\tpath = /nonexistent/zt-review\n")
		if _, err := h.prepare(); err == nil || !strings.Contains(err.Error(), "includeif") {
			t.Fatalf("prepare accepted a conditional config include: %v", err)
		}
	})

	t.Run("commented include is allowed", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		appendConfig(t, filepath.Join(h.upstream, ".git", "config"), "\n# [include]\n#\tpath = /nonexistent/zt-review\n")
		if _, err := h.prepare(); err != nil {
			t.Fatalf("prepare refused a commented-out include: %v", err)
		}
	})

	t.Run("worktree config include", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		writeFile(t, filepath.Join(h.upstream, ".git", "config.worktree"), "[core]\n\trepositoryformatversion = 0\n"+include)
		if _, err := h.prepare(); err == nil || !strings.Contains(err.Error(), "include") {
			t.Fatalf("prepare accepted a worktree config include: %v", err)
		}
	})

	t.Run("ambiguous pointer layouts", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("a file name cannot end with a space on Windows")
		}
		// Git drops the line terminator from a gitdir or commondir pointer and keeps
		// every other byte, so a directory whose name ends in a space is a layout Git
		// opens. Trimming that space made zt inspect a different directory, miss the
		// include and then start the Git that read it. The marker pins the guarantee
		// the docs make: the refusal, not a Git failure, is what has to happen.
		layouts := []struct {
			name  string
			build func(t *testing.T, h *harness) (upstream, config string)
		}{
			{
				name: "gitdir pointer",
				build: func(t *testing.T, h *harness) (string, string) {
					moved := filepath.Join(t.TempDir(), "git dir ")
					if err := os.Rename(filepath.Join(h.upstream, ".git"), moved); err != nil {
						t.Fatal(err)
					}
					writeFile(t, filepath.Join(h.upstream, ".git"), "gitdir: "+moved+"\n")
					return h.upstream, filepath.Join(moved, "config")
				},
			},
			{
				name: "commondir pointer",
				build: func(t *testing.T, h *harness) (string, string) {
					common := filepath.Join(t.TempDir(), "common dir ")
					if err := os.Rename(filepath.Join(h.upstream, ".git"), common); err != nil {
						t.Fatal(err)
					}
					head, err := os.ReadFile(filepath.Join(common, "HEAD"))
					if err != nil {
						t.Fatal(err)
					}
					gitDir := filepath.Join(h.upstream, ".git")
					writeFile(t, filepath.Join(gitDir, "HEAD"), string(head))
					writeFile(t, filepath.Join(gitDir, "commondir"), common+"\n")
					return h.upstream, filepath.Join(common, "config")
				},
			},
		}
		for _, layout := range layouts {
			t.Run(layout.name, func(t *testing.T) {
				h := newHarness(t)
				h.writePolicy(policyDoc(h.commit))
				h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
				candidate := h.candidatePath(h.mustPrepare())
				upstream, config := layout.build(t, h)
				// The fixture only means something if real Git accepts the layout.
				if err := rawGitRead(t, upstream); err != nil {
					t.Fatalf("fixture layout is not one Git opens: %v", err)
				}

				marker := gitStartMarker(t)
				home := t.TempDir()
				t.Setenv("HOME", home)
				// Invalid configuration, so a Git start would name this file.
				writeFile(t, filepath.Join(home, "private-config"), "[invalid\n")
				appendConfig(t, config, "\n[include]\n\tpath = ~/private-config\n")
				before := snapshotTree(t, h.workspace)

				_, prepareErr := h.prepare()
				_, planErr := h.plan(candidate)
				if prepareErr == nil || !strings.Contains(prepareErr.Error(), "whitespace") {
					t.Fatalf("prepare did not refuse the ambiguous pointer: %v", prepareErr)
				}
				if planErr == nil || !strings.Contains(planErr.Error(), "whitespace") {
					t.Fatalf("plan did not refuse the ambiguous pointer: %v", planErr)
				}
				for _, err := range []error{prepareErr, planErr} {
					if strings.Contains(err.Error(), "private-config") || strings.Contains(err.Error(), "bad config") {
						t.Fatalf("zt read the included file instead of refusing first: %v", err)
					}
				}
				if _, err := os.Lstat(marker); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("zt started Git for an ambiguous pointer layout: %v", err)
				}
				assertUntouched(t, "the workspace", before, snapshotTree(t, h.workspace))
			})
		}
	})

	t.Run("linked worktree common config include", func(t *testing.T) {
		upstream, commit := newUpstream(t, baseFileSet())
		worktree := filepath.Join(t.TempDir(), "linked")
		runGit(t, upstream, "worktree", "add", "--quiet", worktree, "HEAD")
		appendConfig(t, filepath.Join(upstream, ".git", "config"), include)
		h := newHarness(t)
		h.useUpstream(worktree, commit)
		h.writeProposal(proposalDoc(commit, defaultProposalFiles()...))
		if _, err := h.prepare(); err == nil || !strings.Contains(err.Error(), "include") {
			t.Fatalf("prepare accepted a common config include through a linked worktree: %v", err)
		}
		assertNoArtifacts(t, h.workspace)
	})

	t.Run("bare repository", func(t *testing.T) {
		source, commit := newUpstream(t, baseFileSet())
		bare := filepath.Join(t.TempDir(), "bare.git")
		if err := os.MkdirAll(bare, 0o755); err != nil {
			t.Fatal(err)
		}
		runGit(t, bare, "init", "--quiet", "--bare")
		runGit(t, bare, "fetch", "--quiet", source, "HEAD:refs/heads/main")
		h := newHarness(t)
		h.useUpstream(bare, commit)
		h.writeProposal(proposalDoc(commit, defaultProposalFiles()...))
		if _, err := h.prepare(); err != nil {
			t.Fatalf("prepare rejected a bare repository: %v", err)
		}
		appendConfig(t, filepath.Join(bare, "config"), include)
		if _, err := h.prepare(); err == nil || !strings.Contains(err.Error(), "include") {
			t.Fatalf("prepare accepted a bare repository config include: %v", err)
		}
	})

	t.Run("directory without a repository", func(t *testing.T) {
		h := newHarness(t)
		h.writePolicy(policyDoc(h.commit))
		h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
		_, err := Prepare(t.Context(), PrepareOptions{Proposal: h.proposal, Policy: h.policy, Upstream: t.TempDir(), Root: h.workspace})
		if err == nil || !strings.Contains(err.Error(), "not a local git repository") {
			t.Fatalf("prepare accepted a directory that is not a repository: %v", err)
		}
	})
}

func TestPrepareRefusesNonRegularExistingCandidate(t *testing.T) {
	for _, kind := range []string{"symlink", "fifo"} {
		t.Run(kind, func(t *testing.T) {
			h := newHarness(t)
			h.writePolicy(policyDoc(h.commit))
			h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
			prepared := h.mustPrepare()
			candidateFile := h.candidatePath(prepared)
			data, err := os.ReadFile(candidateFile)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(candidateFile); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "symlink":
				if runtime.GOOS == "windows" {
					t.Skip("symlinks need elevated privileges on Windows")
				}
				payload := filepath.Join(filepath.Dir(candidateFile), "payload.json")
				writeFile(t, payload, string(data))
				if err := os.Symlink("payload.json", candidateFile); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if _, err := exec.LookPath("mkfifo"); err != nil {
					t.Skip("mkfifo is not available")
				}
				if output, err := exec.Command("mkfifo", candidateFile).CombinedOutput(); err != nil {
					t.Fatalf("mkfifo: %v: %s", err, output)
				}
			}
			done := make(chan error, 1)
			go func() {
				_, err := h.prepare()
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil || !strings.Contains(err.Error(), "non-regular") {
					t.Fatalf("prepare accepted a %s candidate: %v", kind, err)
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("prepare blocked on a %s candidate instead of refusing it", kind)
			}
			// plan must reach the same verdict as the repeat prepare.
			if _, err := h.plan(candidateFile); err == nil {
				t.Fatalf("plan accepted a %s candidate", kind)
			}
		})
	}
}

func TestInstallHonoursCancellation(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	prepared := h.mustPrepare()
	data, err := os.ReadFile(h.candidatePath(prepared))
	if err != nil {
		t.Fatal(err)
	}
	store, err := openStore(h.workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.install(ctx, prepared.CandidateID, data); !errors.Is(err, context.Canceled) {
		t.Fatalf("install ignored cancellation: %v", err)
	}
}

// bareGitDirWithCommon builds a bare-shaped git directory with a commondir, the
// layout Git reads the common directory's configuration from.
func bareGitDirWithCommon(t *testing.T) (gitDir, common string) {
	t.Helper()
	root := t.TempDir()
	gitDir = filepath.Join(root, "gitdir")
	common = filepath.Join(root, "common")
	for _, dir := range []string{gitDir, common} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		runGit(t, dir, "init", "--quiet", "--bare")
	}
	writeFile(t, filepath.Join(gitDir, "commondir"), common+"\n")
	return gitDir, common
}

// rawGitRead runs one Git command the way zt does but without the config
// preflight, so a test can prove that a fixture really makes Git read an included
// file. It is sensitivity evidence for tests, never a product path.
func rawGitRead(t *testing.T, dir string) error {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	repo := &Repo{dir: dir, git: git, format: "sha1"}
	_, err = repo.output(t.Context(), 256, "rev-parse", "--show-object-format")
	return err
}

func TestPlanRefusesConfigIncludesWithoutReadingHome(t *testing.T) {
	h := newHarness(t)
	h.writePolicy(policyDoc(h.commit))
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	prepared := h.mustPrepare()
	candidate := h.candidatePath(prepared)

	home := t.TempDir()
	t.Setenv("HOME", home)
	// The included target is deliberately invalid configuration: if the preflight
	// missed the include, Git would parse this file and fail naming it, which is
	// exactly the read this test forbids.
	writeFile(t, filepath.Join(home, "private-config"), "[invalid\n")

	assertRefused := func(t *testing.T, upstream string) {
		t.Helper()
		_, err := Plan(t.Context(), PlanOptions{Candidate: candidate, Policy: h.policy, Upstream: upstream})
		if err == nil {
			t.Fatal("plan accepted a repository config include")
		}
		if !strings.Contains(err.Error(), "include") {
			t.Fatalf("plan did not refuse before starting Git: %v", err)
		}
		if strings.Contains(err.Error(), "bad config") || strings.Contains(err.Error(), "private-config") {
			t.Fatalf("plan read the included home file: %v", err)
		}
		// Without the preflight the same repository does read that file, so this
		// fixture is sensitive and the refusal above is not a coincidence.
		if err := rawGitRead(t, upstream); err == nil || !strings.Contains(err.Error(), "bad config") {
			t.Fatalf("fixture is not sensitive to the include: %v", err)
		}
	}

	t.Run("byte order mark", func(t *testing.T) {
		config := filepath.Join(h.upstream, ".git", "config")
		data, err := os.ReadFile(config)
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, config, "\ufeff[include]\n\tpath = ~/private-config\n"+string(data))
		assertRefused(t, h.upstream)
	})

	t.Run("bare gitdir with a commondir", func(t *testing.T) {
		gitDir, common := bareGitDirWithCommon(t)
		writeFile(t, filepath.Join(common, "config"), "[core]\n\trepositoryformatversion = 0\n\n[include]\n\tpath = ~/private-config\n")
		assertRefused(t, gitDir)
		if _, err := Prepare(t.Context(), PrepareOptions{Proposal: h.proposal, Policy: h.policy, Upstream: gitDir, Root: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "include") {
			t.Fatalf("prepare accepted a commondir include: %v", err)
		}
	})
}

func TestConfigIncludesHiddenByAcceptedSyntax(t *testing.T) {
	cases := []struct {
		name  string
		build func(t *testing.T, h *harness) string // returns the upstream to use
	}{
		{
			name: "byte order mark before the first header",
			build: func(t *testing.T, h *harness) string {
				config := filepath.Join(h.upstream, ".git", "config")
				data, err := os.ReadFile(config)
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, config, "\ufeff[include]\n\tpath = ~/private-config\n"+string(data))
				return h.upstream
			},
		},
		{
			name: "bare gitdir with a commondir",
			build: func(t *testing.T, h *harness) string {
				gitDir, common := bareGitDirWithCommon(t)
				writeFile(t, filepath.Join(common, "config"), "[core]\n\trepositoryformatversion = 0\n\n[include]\n\tpath = ~/private-config\n")
				return gitDir
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			h := newHarness(t)
			h.writePolicy(policyDoc(h.commit))
			h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
			upstream := testCase.build(t, h)
			t.Setenv("HOME", t.TempDir())
			_, err := Prepare(t.Context(), PrepareOptions{Proposal: h.proposal, Policy: h.policy, Upstream: upstream, Root: t.TempDir()})
			if err == nil || !strings.Contains(err.Error(), "include") {
				t.Fatalf("prepare accepted an include hidden by %s: %v", testCase.name, err)
			}
		})
	}
}
