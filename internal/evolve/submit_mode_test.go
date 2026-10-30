package evolve

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// executableReplacementModes declares the synthetic base this regression needs: the
// Markdown file the candidate replaces is executable, and one untouched sibling is
// executable too, so the submitted tree shows both a preserved replacement mode and
// an inherited base mode. Content-only replacements never name a mode; the mode a
// replacement takes therefore comes from the pinned base alone.
var executableReplacementModes = map[string]string{
	"skills/" + fixtureSkill + "/SKILL.md": "100755",
	"skills/" + fixtureSkill + "/LICENSE":  "100755",
}

// hostStoresExecutableBit reports whether this filesystem can represent a Git 100755
// entry as a physical executable file. `zt evolve check` compares the materialised
// workspace byte-for-byte with the pinned modes, so on a host that cannot store the
// bit (a Windows filesystem, for example, where Go neither sets nor observes it) the
// check refuses before a submission is possible. The Git-side rule is covered
// independently of this capability by TestExpectedTreePreservesReplacedBaseMode.
func hostStoresExecutableBit(t *testing.T) bool {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "probe")
	if err := os.WriteFile(probe, []byte("probe\n"), 0o755); err != nil {
		t.Fatalf("write executable probe: %v", err)
	}
	info, err := os.Lstat(probe)
	if err != nil {
		t.Fatalf("stat executable probe: %v", err)
	}
	return info.Mode()&0o111 != 0
}

// newModePinnedUpstream commits files, then records the declared Git modes in the
// index explicitly and checks the commit really holds them. The hook in
// newUpstreamWith runs before git add, so it cannot express an index mode; this is
// the mode-aware equivalent.
func newModePinnedUpstream(t *testing.T, files, modes map[string]string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet")
	for name, content := range files {
		writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), content)
	}
	runGit(t, dir, "add", "--all")
	pinIndexModes(t, dir, modes)
	runGit(t, dir, "commit", "--quiet", "-m", "fixture")
	commit := runGit(t, dir, "rev-parse", "HEAD")
	for name, mode := range modes {
		if got := treeModeFromGit(t, dir, commit, name); got != mode {
			t.Fatalf("fixture commit records %s as mode %s instead of %s, so this regression would assert nothing", name, got, mode)
		}
	}
	return dir, commit
}

// treeModeFromGit reads the mode the fixture commit really stores for one path.
func treeModeFromGit(t *testing.T, dir, commit, name string) string {
	t.Helper()
	fields := strings.Fields(runGit(t, dir, "ls-tree", commit, "--", name))
	if len(fields) == 0 {
		t.Fatalf("fixture commit %s holds no tree entry for %s", commit, name)
	}
	return fields[0]
}

// indexModeFromGit reads the mode the index currently records for one path.
func indexModeFromGit(t *testing.T, dir, name string) string {
	t.Helper()
	fields := strings.Fields(runGit(t, dir, "ls-files", "--stage", "--", name))
	if len(fields) == 0 {
		t.Fatalf("index holds no entry for %s", name)
	}
	return fields[0]
}

// TestPinIndexModesRecordsExecutableWithoutPhysicalBit shows why the mode fixture
// pins the index instead of calling os.Chmod: with core.filemode disabled — the usual
// Windows setting, and what a filesystem without an executable bit behaves like — Git
// ignores the physical mode when adding, and the explicit pin is what records 100755.
// The assertion that the add produced 100644 first keeps this test from passing by
// accident on a host where the physical bit would have leaked into the index anyway.
func TestPinIndexModesRecordsExecutableWithoutPhysicalBit(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet")
	runGit(t, dir, "config", "core.filemode", "false")
	name := "skills/" + fixtureSkill + "/SKILL.md"
	target := filepath.Join(dir, filepath.FromSlash(name))
	writeFile(t, target, fixtureSkillMD)
	// A no-op where the host cannot store the bit; the point is that Git will not read
	// it either way with core.filemode disabled.
	if err := os.Chmod(target, 0o755); err != nil {
		t.Fatalf("chmod fixture file: %v", err)
	}
	runGit(t, dir, "add", "--all")
	if got := indexModeFromGit(t, dir, name); got != "100644" {
		t.Fatalf("the index records %s as %s after adding, so this test cannot show the pin acting alone", name, got)
	}
	pinIndexModes(t, dir, map[string]string{name: "100755"})
	if got := indexModeFromGit(t, dir, name); got != "100755" {
		t.Fatalf("the index records %s as %s after pinning", name, got)
	}
	runGit(t, dir, "commit", "--quiet", "-m", "fixture")
	if got := treeModeFromGit(t, dir, runGit(t, dir, "rev-parse", "HEAD"), name); got != "100755" {
		t.Fatalf("the commit records %s as %s after pinning, so the mode fixture is not platform independent", name, got)
	}
}

// TestExpectedTreePreservesReplacedBaseMode pins the tree rule a submission builds:
// a replacement carries content only, so it keeps the pinned base mode, an untouched
// base entry keeps the mode it was pinned with, and a new path keeps the existing
// 100644 default.
func TestExpectedTreePreservesReplacedBaseMode(t *testing.T) {
	ctx := context.Background()
	dir, commit := newModePinnedUpstream(t, baseFileSet(), executableReplacementModes)
	repo, err := OpenRepo(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := repo.CompleteTree(ctx, commit)
	if err != nil {
		t.Fatal(err)
	}
	candidate := Candidate{Files: []File{
		{Path: "skills/" + fixtureSkill + "/SKILL.md", Content: "# Go testing\n\nRun the unit gate, then the focused package.\n"},
		{Path: "skills/" + fixtureSkill + "/references/guide.md", Content: "# Guide\n\nA new reference.\n"},
	}}
	expected, err := expectedTree(ctx, repo, entries, candidate)
	if err != nil {
		t.Fatalf("expected tree: %v", err)
	}
	for path, mode := range map[string]string{
		"skills/" + fixtureSkill + "/SKILL.md":            "100755", // replaced: keeps the base mode
		"skills/" + fixtureSkill + "/LICENSE":             "100755", // untouched executable base entry
		"skills/" + fixtureSkill + "/references/notes.md": "100644",
		"skills/" + fixtureSkill + "/references/guide.md": "100644", // new: the existing default
		"README.md": "100644",
	} {
		if got := expected[path].Mode; got != mode {
			t.Errorf("expected tree holds %s with mode %s instead of %s", path, got, mode)
		}
	}
}

// TestSubmitPreservesReplacedBaseMode reproduces issue #36 through the real command
// path: the pinned base holds an executable Markdown file, the approved candidate
// replaces it by content only, and the tree the submission creates must hold the
// same mode the verification workspace materialised. The fake GitHub service serves
// both the pinned base and the submission, so no real remote is contacted.
func TestSubmitPreservesReplacedBaseMode(t *testing.T) {
	if !hostStoresExecutableBit(t) {
		t.Skipf("this host filesystem cannot store an executable bit, and zt evolve check compares the materialised workspace against the pinned 100755 mode, so the end-to-end submission path does not exist here; TestExpectedTreePreservesReplacedBaseMode covers the Git-side rule")
	}
	s := newSubmitHarnessWithModes(t, "example/fixture", executableReplacementModes)
	verified := s.verify(t)
	if verified.Status != executionPassed {
		t.Fatalf("fixture verification did not pass: %+v", verified)
	}
	// The recorded run directory is the only place the materialised overlay exists, so
	// read the mode the checks actually saw on disk rather than the expected set.
	workspaceTree := filepath.Join(filepath.Dir(filepath.Join(s.workspace, filepath.FromSlash(verified.ResultPath))), "tree")
	replaced := filepath.Join(workspaceTree, filepath.FromSlash("skills/"+fixtureSkill+"/SKILL.md"))
	info, err := os.Lstat(replaced)
	if err != nil {
		t.Fatalf("verification workspace: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("the verification workspace materialised the replaced Markdown file as %v, not executable", info.Mode())
	}

	result, err := s.submit(t)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	treeSHA, ok := s.remote.TreeSHA(result.Commit)
	if !ok {
		t.Fatalf("the fake remote holds no commit %s", result.Commit)
	}
	submitted, ok := s.remote.TreeFiles(treeSHA)
	if !ok {
		t.Fatalf("the fake remote holds no tree %s", treeSHA)
	}
	for path, mode := range map[string]string{
		"skills/" + fixtureSkill + "/SKILL.md":            "100755", // replaced by content only
		"skills/" + fixtureSkill + "/LICENSE":             "100755", // untouched base entry
		"skills/" + fixtureSkill + "/references/notes.md": "100644",
		"skills/" + fixtureSkill + "/references/guide.md": "100644", // new file
		"README.md": "100644",
	} {
		entry, ok := submitted[path]
		if !ok {
			t.Fatalf("the submitted tree is missing %s", path)
		}
		if entry.Mode != mode {
			t.Errorf("the submitted tree holds %s with mode %s instead of %s; a content-only replacement must keep the mode the verified workspace materialised", path, entry.Mode, mode)
		}
	}
	// The fixture proposal's first replacement is the Markdown file this test replaces.
	replacement := defaultProposalFiles()[0]["content"].(string)
	if entry := submitted["skills/"+fixtureSkill+"/SKILL.md"]; entry.SHA != fakeGHObjectSHA("blob", []byte(replacement)) {
		t.Errorf("the submitted tree does not hold the approved replacement content")
	}
}

// fakeGHObject creates one Git object through the fake's HTTP API and returns its
// object id, so a fixture can produce exactly the objects an earlier build wrote.
func fakeGHObject(t *testing.T, g *fakeGitHub, path string, body any) string {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode fake GitHub request %s: %v", path, err)
	}
	status, _, response := fakeGHTestRequest(t, g, http.MethodPost, "/repos/"+g.Owner+"/"+g.Repo+path, string(data), testRemoteToken)
	if status != http.StatusCreated {
		t.Fatalf("fake GitHub POST %s returned %d: %s", path, status, response)
	}
	var created struct {
		SHA    string            `json:"sha"`
		Object map[string]string `json:"object"`
	}
	fakeGHTestJSON(t, response, &created)
	if created.SHA != "" {
		return created.SHA
	}
	return created.Object["sha"]
}

// createLostModeSubmission writes the branch the earlier build left behind: the
// approved content, but the replaced Markdown file recorded as 100644 because the
// submitted tree lost the pinned base mode. withPull also attaches the open draft
// pull request that build would have created, so both recovery paths face an object
// that does not verify. It returns the commit it created.
func createLostModeSubmission(t *testing.T, s *submitHarness, payload PublicPayload, withPull bool) string {
	t.Helper()
	g := s.remote
	entries := make([]map[string]string, 0, len(payload.Files))
	for _, file := range payload.Files {
		blob := fakeGHObject(t, g, "/git/blobs", map[string]any{"content": file.Content, "encoding": "utf-8"})
		entries = append(entries, map[string]string{"path": file.Path, "mode": "100644", "type": "blob", "sha": blob})
	}
	tree := fakeGHObject(t, g, "/git/trees", map[string]any{"base_tree": g.BaseTree(), "tree": entries})
	identity := map[string]string{
		"name":  payload.Commit.AuthorName,
		"email": payload.Commit.AuthorEmail,
		"date":  payload.Commit.CommittedAt,
	}
	head := fakeGHObject(t, g, "/git/commits", map[string]any{
		"message":   payload.Commit.Message,
		"tree":      tree,
		"parents":   []string{payload.BaseCommit},
		"author":    identity,
		"committer": identity,
	})
	fakeGHObject(t, g, "/git/refs", map[string]string{"ref": "refs/heads/" + payload.Branch, "sha": head})
	if current, ok := g.Ref("refs/heads/" + payload.Branch); !ok || current != head {
		t.Fatalf("fixture branch is %s %v, want the created commit %s", current, ok, head)
	}
	if withPull {
		g.AttachPull(fakeGHPull{
			Title:   payload.PullRequest.Title,
			Body:    payload.PullRequest.Body,
			Head:    payload.Branch,
			HeadSHA: head,
			Base:    "main",
			BaseSHA: payload.BaseCommit,
			Draft:   true,
			State:   "open",
		})
	}
	return head
}

// TestReconcileRefusesALostModeSubmission pins the fail-closed recovery boundary
// this fix creates: a branch, or an open draft pull request, created by the earlier
// build holds the approved content with the replaced file at 100644. zt must report
// the mismatch and leave the remote alone: it never force-updates the ref and never
// adopts the mode-losing tree as this submission.
func TestReconcileRefusesALostModeSubmission(t *testing.T) {
	for _, withPull := range []bool{false, true} {
		name := "branch without pull request"
		if withPull {
			name = "open draft pull request"
		}
		t.Run(name, func(t *testing.T) {
			if !hostStoresExecutableBit(t) {
				t.Skipf("this host filesystem cannot store an executable bit, so zt evolve check refuses the 100755 base entry before any submission; TestExpectedTreePreservesReplacedBaseMode covers the Git-side rule")
			}
			s := newSubmitHarnessWithModes(t, "example/fixture", executableReplacementModes)
			if verified := s.verify(t); verified.Status != executionPassed {
				t.Fatalf("fixture verification did not pass: %+v", verified)
			}
			preview := s.showPayload(t)
			if preview.Payload == nil {
				t.Fatal("payload preview carries no payload")
			}
			payload := *preview.Payload
			head := createLostModeSubmission(t, s, payload, withPull)
			// Guard the fixture: the object this test faces really lost the mode.
			treeSHA, ok := s.remote.TreeSHA(head)
			if !ok {
				t.Fatalf("the fixture commit %s has no tree", head)
			}
			files, ok := s.remote.TreeFiles(treeSHA)
			if !ok {
				t.Fatalf("the fixture tree %s is not stored", treeSHA)
			}
			if got := files["skills/"+fixtureSkill+"/SKILL.md"].Mode; got != "100644" {
				t.Fatalf("the fixture tree holds the replaced file as %s, so this test no longer faces a lost mode", got)
			}

			writes := s.remote.Writes()
			result, err := s.submit(t)
			if err == nil {
				t.Fatalf("submit adopted the mode-losing %s as an exact submission: %+v", name, result)
			}
			if !strings.Contains(err.Error(), "the remote tree holds a different") {
				t.Fatalf("submit refused for a reason other than the mode difference: %v", err)
			}
			if current, ok := s.remote.Ref("refs/heads/" + payload.Branch); !ok || current != head {
				t.Fatalf("submit rewrote the refused branch: ref is %s %v, want %s", current, ok, head)
			}
			if after := s.remote.Writes(); after != writes {
				t.Fatalf("submit performed %d remote writes while refusing", after-writes)
			}
			wantPulls := 0
			if withPull {
				wantPulls = 1
			}
			if pulls := s.remote.Pulls(); len(pulls) != wantPulls {
				t.Fatalf("submit left %d pull requests, want the %d it found", len(pulls), wantPulls)
			}
		})
	}
}
