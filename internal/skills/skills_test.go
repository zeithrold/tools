package skills

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tools "github.com/zeithrold/tools"
)

func newTarget(t *testing.T) (Target, string) {
	t.Helper()
	home := t.TempDir()
	return GlobalTarget(home), home
}

func TestGlobalTargetKeepsConfigAndLockSeparate(t *testing.T) {
	target, home := newTarget(t)
	if target.Root != home {
		t.Fatalf("root %q is not home %q", target.Root, home)
	}
	if want := filepath.Join(home, ".agents", "skills"); target.SkillsDir != want {
		t.Fatalf("skills directory %q, want %q", target.SkillsDir, want)
	}
	if want := filepath.Join(home, ".agents", "zt.lock.json"); target.LockPath != want {
		t.Fatalf("lock %q, want %q", target.LockPath, want)
	}
	if _, err := Plan(target, nil); err == nil || !strings.Contains(err.Error(), "no Skills selected") {
		t.Fatalf("empty selection was accepted: %v", err)
	}
	relative := Target{Root: "relative", SkillsDir: "a", LockPath: "b"}
	if _, err := Plan(relative, []string{"ui-design"}); err == nil {
		t.Fatal("relative target accepted")
	}
}

func TestPlanIsReadOnly(t *testing.T) {
	target, home := newTarget(t)
	plan, err := Plan(target, []string{"ui-design"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Action != "install" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if want := ".agents/skills/ui-design"; plan.Actions[0].Target != want {
		t.Fatalf("target %q, want %q", plan.Actions[0].Target, want)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("plan created the global tree: %v", err)
	}
	if _, err := Apply(target, []string{"ui-design"}); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(home, ".agents", "zt.lock.json")
	before, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		plan, err := Plan(target, []string{"ui-design"})
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.Actions) != 1 || plan.Actions[0].Action != "unchanged" {
			t.Fatalf("plan is not read-only or not idempotent: %+v", plan)
		}
	}
	after, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("plan rewrote the lock file")
	}
}

func TestEverySyncedSkillRetainsMITLicense(t *testing.T) {
	entries, err := fs.ReadDir(tools.SkillFiles, "skills")
	if err != nil {
		t.Fatal(err)
	}
	license, err := tools.LicenseFiles.ReadFile("LICENSE")
	if err != nil {
		t.Fatal(err)
	}
	target, home := newTarget(t)
	var selection []string
	for _, entry := range entries {
		if entry.IsDir() {
			selection = append(selection, entry.Name())
		}
	}
	if len(selection) == 0 {
		t.Fatal("no bundled Skills checked")
	}
	if _, err := Apply(target, selection); err != nil {
		t.Fatal(err)
	}
	for _, name := range selection {
		data, err := os.ReadFile(filepath.Join(target.SkillsDir, name, "LICENSE"))
		if err != nil || !bytes.Equal(data, license) {
			t.Fatalf("Skill %s lost the complete MIT notice: %v", name, err)
		}
	}
	plan, err := Plan(target, selection)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range plan.Actions {
		if action.Action != "unchanged" {
			t.Fatalf("licensed Skill sync is not idempotent: %+v", action)
		}
	}
	if _, err := os.Stat(filepath.Join(home, "zt.lock.json")); !os.IsNotExist(err) {
		t.Fatalf("lock was written outside .agents: %v", err)
	}
}

func TestSyncInstallsCompleteDirectoryAndPreservesUnmanagedSkills(t *testing.T) {
	target, _ := newTarget(t)
	unmanaged := filepath.Join(target.SkillsDir, "project-ui", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(unmanaged), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unmanaged, []byte("project owns this"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(target, []string{"ui-design"}); err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(target.SkillsDir, "ui-design")
	for _, name := range []string{"SKILL.md", "references/review-matrix.md"} {
		if _, err := os.Stat(filepath.Join(managed, name)); err != nil {
			t.Fatalf("missing bundled file %s: %v", name, err)
		}
	}
	if data, err := os.ReadFile(unmanaged); err != nil || string(data) != "project owns this" {
		t.Fatalf("unmanaged Skill changed: %q, %v", data, err)
	}
	custom := filepath.Join(managed, "SKILL.md")
	if err := os.WriteFile(custom, []byte("local edit"), 0644); err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(target, []string{"ui-design"})
	if err != nil || plan.Actions[0].Action != "conflict" {
		t.Fatalf("local edit not detected: %+v, %v", plan, err)
	}
	if _, err := Apply(target, []string{"ui-design"}); err == nil || !strings.Contains(err.Error(), "local edits") {
		t.Fatalf("expected conflict, got %v", err)
	}
	if data, err := os.ReadFile(custom); err != nil || string(data) != "local edit" {
		t.Fatalf("local edit overwritten: %q, %v", data, err)
	}
}

func TestConflictAbortsTheWholeRun(t *testing.T) {
	target, _ := newTarget(t)
	unmanaged := filepath.Join(target.SkillsDir, "ui-design", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(unmanaged), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unmanaged, []byte("not zt managed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(target, []string{"ui-design", "verification"}); err == nil || !strings.Contains(err.Error(), "not managed") {
		t.Fatalf("expected unmanaged conflict, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(target.SkillsDir, "verification")); !os.IsNotExist(err) {
		t.Fatalf("conflicting run installed another Skill: %v", err)
	}
	if _, err := os.Stat(target.LockPath); !os.IsNotExist(err) {
		t.Fatalf("conflicting run wrote a lock: %v", err)
	}
	if data, err := os.ReadFile(unmanaged); err != nil || string(data) != "not zt managed" {
		t.Fatalf("unmanaged Skill changed: %q, %v", data, err)
	}
}

func TestExistingIdenticalDirectoryIsAdoptedNotOverwritten(t *testing.T) {
	target, _ := newTarget(t)
	if _, err := Apply(target, []string{"ui-design"}); err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(target.SkillsDir, "ui-design")
	if err := os.Remove(target.LockPath); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(managed, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(target, []string{"ui-design"})
	if err != nil || plan.Actions[0].Action != "adopt" {
		t.Fatalf("expected adopt: %+v, %v", plan, err)
	}
	if _, err := Apply(target, []string{"ui-design"}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(managed, "SKILL.md"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("adoption rewrote the directory: %v", err)
	}
	plan, err = Plan(target, []string{"ui-design"})
	if err != nil || plan.Actions[0].Action != "unchanged" {
		t.Fatalf("adopted Skill is not managed afterwards: %+v, %v", plan, err)
	}
}

func TestSymlinkedTargetIsRejected(t *testing.T) {
	target, home := newTarget(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(home, ".agents")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Plan(target, []string{"ui-design"}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked .agents was accepted: %v", err)
	}
	if _, err := Apply(target, []string{"ui-design"}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked .agents was written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "skills")); !os.IsNotExist(err) {
		t.Fatalf("wrote through symlink: %v", err)
	}

	single, home := newTarget(t)
	if err := os.MkdirAll(filepath.Dir(single.SkillsDir), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, single.SkillsDir); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Apply(single, []string{"ui-design"}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked skills directory was accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "zt.lock.json")); !os.IsNotExist(err) {
		t.Fatalf("symlinked target wrote a lock: %v", err)
	}
}

func TestDeselectedManagedSkillIsNotRemoved(t *testing.T) {
	target, _ := newTarget(t)
	if _, err := Apply(target, []string{"ui-design", "verification"}); err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(target, []string{"ui-design"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "verification") {
		t.Fatalf("expected a removal warning: %+v", plan)
	}
	if _, err := Apply(target, []string{"ui-design"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target.SkillsDir, "verification")); err != nil {
		t.Fatalf("deselected Skill was removed: %v", err)
	}
}

func TestSkillBundlesIncludeReusableReferences(t *testing.T) {
	target, _ := newTarget(t)
	if _, err := Apply(target, []string{"ui-design", "frontend-engineering", "verification"}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"ui-design/references/design-contract.md",
		"ui-design/references/web-browser-review.md",
		"ui-design/references/flutter.md",
		"frontend-engineering/references/preferences-i18n.md",
		"verification/references/frontend-gates.md",
		"verification/references/frontend-evidence.md",
		"verification/references/go.md",
		"verification/references/js-ts-checks.md",
	} {
		data, err := os.ReadFile(filepath.Join(target.SkillsDir, name))
		if err != nil || len(data) == 0 {
			t.Fatalf("reference not distributed: %s %v", name, err)
		}
	}
}

// setHook installs the package test seam for one test and restores it. The hook
// receives the point it fired at, so a test can target one window.
func setHook(t *testing.T, hook func(point, skill, stage string)) {
	t.Helper()
	testHook = hook
	t.Cleanup(func() { testHook = nil })
}

// prepareManagedSkill creates a managed Skill directory whose lock digest is the
// digest of files, so the next sync plans an update of that directory.
func prepareManagedSkill(t *testing.T, target Target, skill string, files map[string][]byte) {
	t.Helper()
	for name, data := range files {
		path := filepath.Join(target.SkillsDir, skill, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	lock := Lock{SchemaVersion: 1, Skills: map[string]LockedSkill{skill: {Version: "test", SHA256: digest(files)}}}
	data, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target.LockPath, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func transientEntries(t *testing.T, target Target) []string {
	t.Helper()
	entries, err := os.ReadDir(target.SkillsDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var found []string
	for _, entry := range entries {
		for _, prefix := range []string{stagePrefix, backupPrefix, rollbackPrefix} {
			if strings.HasPrefix(entry.Name(), prefix) {
				found = append(found, entry.Name())
			}
		}
	}
	return found
}

func readLockFile(t *testing.T, target Target) (Lock, error) {
	t.Helper()
	data, err := os.ReadFile(target.LockPath)
	if err != nil {
		return Lock{}, err
	}
	var lock Lock
	err = json.Unmarshal(data, &lock)
	return lock, err
}

// Review case 1: swapping the Skills path for a symlink after staging must not
// redirect the exchange, because every rename is relative to the opened handle.
func TestStagedExchangeCannotBeRedirectedOutside(t *testing.T) {
	target, _ := newTarget(t)
	outside := t.TempDir()
	var hookErr error
	setHook(t, func(point, skill, stage string) {
		if point != "staged" || hookErr != nil {
			return
		}
		// A concurrent writer replaces the Skills path and moves the staged
		// directory next to the new symlink target.
		if hookErr = os.Rename(target.SkillsDir, target.SkillsDir+"-moved"); hookErr != nil {
			return
		}
		if hookErr = os.Symlink(outside, target.SkillsDir); hookErr != nil {
			return
		}
		hookErr = os.Rename(filepath.Join(target.SkillsDir+"-moved", stage), filepath.Join(outside, stage))
	})
	_, err := Apply(target, []string{"ui-design"})
	if hookErr != nil {
		t.Fatal(hookErr)
	}
	if err == nil {
		t.Fatal("exchange succeeded through a swapped Skills path")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "ui-design" || entry.Name() == "SKILL.md" {
			t.Fatalf("Skill content written outside the managed tree: %s", entry.Name())
		}
	}
	if _, err := os.Stat(filepath.Join(target.SkillsDir+"-moved", "ui-design")); !os.IsNotExist(err) {
		t.Fatalf("Skill directory was installed in the moved tree: %v", err)
	}
	if _, err := os.Stat(target.LockPath); !os.IsNotExist(err) {
		t.Fatalf("failed run wrote the lock: %v", err)
	}
}

// Review case 2: an edit made after staging must be refused, not overwritten.
func TestLocalEditDuringStagingIsRefused(t *testing.T) {
	target, _ := newTarget(t)
	old := map[string][]byte{"SKILL.md": []byte("old version")}
	prepareManagedSkill(t, target, "ui-design", old)
	lockBefore, err := os.ReadFile(target.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	var hookErr error
	setHook(t, func(point, skill, stage string) {
		if point != "staged" || hookErr != nil {
			return
		}
		hookErr = os.WriteFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"), []byte("LOCAL EDIT DURING STAGING"), 0644)
	})
	_, err = Apply(target, []string{"ui-design"})
	if hookErr != nil {
		t.Fatal(hookErr)
	}
	if err == nil || !strings.Contains(err.Error(), "changed after planning") {
		t.Fatalf("expected the stale plan to be refused, got %v", err)
	}
	data, err := os.ReadFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"))
	if err != nil || string(data) != "LOCAL EDIT DURING STAGING" {
		t.Fatalf("local edit was overwritten: %q, %v", data, err)
	}
	lockAfter, err := os.ReadFile(target.LockPath)
	if err != nil || !bytes.Equal(lockBefore, lockAfter) {
		t.Fatalf("refused run rewrote the lock: %v", err)
	}
	if left := transientEntries(t, target); len(left) != 0 {
		t.Fatalf("staging left content behind: %v", left)
	}
}

// Review case 3: a failure after a directory was installed must keep edits made
// in the meantime and restore the previous managed version.
func TestRollbackPreservesEditsMadeAfterTheExchange(t *testing.T) {
	target, _ := newTarget(t)
	prepareManagedSkill(t, target, "ui-design", map[string][]byte{"SKILL.md": []byte("old version")})
	lockBefore, err := os.ReadFile(target.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	agentsDir := filepath.Dir(target.SkillsDir)
	t.Cleanup(func() { _ = os.Chmod(agentsDir, 0755) })
	var hookErr error
	edited := false
	setHook(t, func(point, skill, stage string) {
		if point != "staged" || skill != "verification" || edited || hookErr != nil {
			return
		}
		edited = true
		if hookErr = os.WriteFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"), []byte("EDIT AFTER FIRST INSTALL"), 0644); hookErr != nil {
			return
		}
		// Make the final lock write fail.
		hookErr = os.Chmod(agentsDir, 0555)
	})
	_, applyErr := Apply(target, []string{"ui-design", "verification"})
	if hookErr != nil {
		t.Fatal(hookErr)
	}
	if !edited {
		t.Fatal("the concurrent edit step did not run")
	}
	if applyErr == nil {
		t.Fatal("the lock write should have failed")
	}
	data, readErr := os.ReadFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"))
	if readErr != nil || string(data) != "old version" {
		t.Fatalf("previous managed version was not restored: %q, %v", data, readErr)
	}
	if !strings.Contains(applyErr.Error(), "preserved content") {
		t.Fatalf("preserved content is not reported: %v", applyErr)
	}
	preserved := ""
	for _, name := range transientEntries(t, target) {
		if !strings.HasPrefix(name, rollbackPrefix) {
			continue
		}
		if data, err := os.ReadFile(filepath.Join(target.SkillsDir, name, "SKILL.md")); err == nil && string(data) == "EDIT AFTER FIRST INSTALL" {
			preserved = name
		}
	}
	if preserved == "" {
		t.Fatal("the edit made after the exchange was deleted instead of preserved")
	}
	if !strings.Contains(applyErr.Error(), preserved) {
		t.Fatalf("error does not name the preserved directory %s: %v", preserved, applyErr)
	}
	lockAfter, readErr := os.ReadFile(target.LockPath)
	if readErr != nil || !bytes.Equal(lockBefore, lockAfter) {
		t.Fatalf("failed run rewrote the lock: %v", readErr)
	}
}

// Review case 5: content changed before the guard is held must not be adopted
// from the plan that was made before the guard.
func TestContentChangedBeforeTheGuardIsNotAdopted(t *testing.T) {
	target, _ := newTarget(t)
	files, err := bundledFiles("ui-design")
	if err != nil {
		t.Fatal(err)
	}
	// The directory already equals the bundle, so an unlocked plan adopts it.
	for name, data := range files {
		path := filepath.Join(target.SkillsDir, "ui-design", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	var hookErr error
	hookRan := false
	setHook(t, func(point, skill, stage string) {
		if point != "transaction" || hookRan || hookErr != nil {
			return
		}
		hookRan = true
		hookErr = os.WriteFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"), []byte("LOCAL EDIT BEFORE THE GUARD"), 0644)
	})
	_, err = Apply(target, []string{"ui-design"})
	if hookErr != nil {
		t.Fatal(hookErr)
	}
	if !hookRan {
		t.Fatal("the pre-guard edit step did not run")
	}
	if err == nil || !strings.Contains(err.Error(), "not managed") {
		t.Fatalf("edited directory was adopted from the stale plan: %v", err)
	}
	if _, err := os.Stat(target.LockPath); !os.IsNotExist(err) {
		t.Fatalf("adoption was recorded despite the edit: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"))
	if err != nil || string(data) != "LOCAL EDIT BEFORE THE GUARD" {
		t.Fatalf("edit was overwritten: %q, %v", data, err)
	}
}

// Review case 6: an edit made between the recheck and the exchange must be
// restored, and the moved-aside copy must not be deleted as zt's own.
func TestEditBetweenRecheckAndExchangeIsPreserved(t *testing.T) {
	target, _ := newTarget(t)
	prepareManagedSkill(t, target, "ui-design", map[string][]byte{"SKILL.md": []byte("old version")})
	var hookErr error
	hookRan := false
	setHook(t, func(point, skill, stage string) {
		if point != "before-exchange" || skill != "ui-design" || hookRan || hookErr != nil {
			return
		}
		hookRan = true
		hookErr = os.WriteFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"), []byte("EDIT IN THE EXCHANGE WINDOW"), 0644)
	})
	_, applyErr := Apply(target, []string{"ui-design"})
	if hookErr != nil {
		t.Fatal(hookErr)
	}
	if !hookRan {
		t.Fatal("the exchange-window edit step did not run")
	}
	if applyErr == nil || !strings.Contains(applyErr.Error(), "changed during the exchange") {
		t.Fatalf("edit inside the exchange window was accepted: %v", applyErr)
	}
	data, readErr := os.ReadFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"))
	if readErr != nil || string(data) != "EDIT IN THE EXCHANGE WINDOW" {
		t.Fatalf("edit inside the exchange window was deleted: %q, %v", data, readErr)
	}
	installed := ""
	for _, name := range transientEntries(t, target) {
		if !strings.HasPrefix(name, rollbackPrefix) {
			continue
		}
		if data, err := os.ReadFile(filepath.Join(target.SkillsDir, name, "SKILL.md")); err == nil && !bytes.Equal(data, []byte("EDIT IN THE EXCHANGE WINDOW")) {
			installed = name
		}
	}
	if installed == "" {
		t.Fatal("the installed copy was deleted instead of preserved")
	}
	if !strings.Contains(applyErr.Error(), installed) {
		t.Fatalf("error does not name the preserved directory %s: %v", installed, applyErr)
	}
}

// Review case 4: two runs must not interleave their lock read/modify/write.
func TestConcurrentApplyIsRefusedByTheGuard(t *testing.T) {
	target, _ := newTarget(t)
	var hookErr error
	var innerErr error
	setHook(t, func(point, skill, stage string) {
		if point != "staged" || hookErr != nil || innerErr != nil {
			return
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, innerErr = Apply(target, []string{"verification"})
		}()
		<-done
	})
	if _, err := Apply(target, []string{"ui-design"}); err != nil {
		t.Fatalf("outer sync failed: %v", err)
	}
	if hookErr != nil {
		t.Fatal(hookErr)
	}
	if innerErr == nil || !strings.Contains(innerErr.Error(), "another zt sync is running") {
		t.Fatalf("concurrent run was not refused: %v", innerErr)
	}
	lock, err := readLockFile(t, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.Skills) != 1 || lock.Skills["ui-design"].SHA256 == "" {
		t.Fatalf("interleaved run changed the lock: %+v", lock)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(target.SkillsDir), guardName)); !os.IsNotExist(err) {
		t.Fatalf("guard was not released: %v", err)
	}
	if left := transientEntries(t, target); len(left) != 0 {
		t.Fatalf("successful run left content behind: %v", left)
	}
	if _, err := Apply(target, []string{"verification"}); err != nil {
		t.Fatalf("follow-up sync failed: %v", err)
	}
}
