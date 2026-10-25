package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// syncHome points HOME at a temporary directory so no test can touch the real
// user-global tree.
func syncHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func sync(t *testing.T, args ...string) (string, error) {
	t.Helper()
	stdout, _, err := runCLI(t, append([]string{"sync"}, args...)...)
	return stdout, err
}

func TestSyncDefaultsToTheGlobalTreeAndIgnoresTheProject(t *testing.T) {
	home := syncHome(t)
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web","go-testing"]}`)
	project := t.TempDir()
	// A project config no longer selects Distribution targets, so even an
	// unreadable one must not change the outcome.
	writeFile(t, filepath.Join(project, "zt.json"), `{"schemaVersion":1,"unexpected":true,"skills":["ui-foundation"]}`)
	t.Chdir(project)

	out, err := sync(t)
	if err != nil {
		t.Fatalf("sync from a project directory failed: %v\n%s", err, out)
	}
	for _, name := range []string{"ui-web", "go-testing"} {
		if _, err := os.Stat(filepath.Join(home, ".agents", "skills", name, "SKILL.md")); err != nil {
			t.Fatalf("global Skill %s not installed: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "ui-foundation")); !os.IsNotExist(err) {
		t.Fatalf("project Skill selection was distributed: %v", err)
	}
	lock := filepath.Join(home, ".agents", "zt.lock.json")
	data, err := os.ReadFile(lock)
	if err != nil {
		t.Fatalf("global lock missing: %v", err)
	}
	for _, name := range []string{"ui-web", "go-testing"} {
		if !bytes.Contains(data, []byte(`"`+name+`"`)) {
			t.Fatalf("lock omits %s: %s", name, data)
		}
	}
	if _, err := os.Stat(filepath.Join(project, "zt.lock.json")); !os.IsNotExist(err) {
		t.Fatalf("global sync wrote a project lock: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("global sync wrote project Skills: %v", err)
	}

	out, err = sync(t, "--global")
	if err != nil {
		t.Fatalf("explicit --global failed: %v\n%s", err, out)
	}
	if strings.Count(out, "unchanged:") != 2 {
		t.Fatalf("second global sync is not idempotent:\n%s", out)
	}
}

func TestSyncFromDifferentWorkingDirectoriesIsIdentical(t *testing.T) {
	home := syncHome(t)
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web"]}`)
	first := t.TempDir()
	t.Chdir(first)
	out, err := sync(t)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "install: .agents/skills/ui-web") {
		t.Fatalf("unexpected first run:\n%s", out)
	}
	second := t.TempDir()
	writeFile(t, filepath.Join(second, "zt.json"), `{"schemaVersion":1,"skills":["go-testing"]}`)
	t.Chdir(second)
	out, err = sync(t)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unchanged: .agents/skills/ui-web") {
		t.Fatalf("second working directory changed the target:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "go-testing")); !os.IsNotExist(err) {
		t.Fatalf("second working directory contributed Skills: %v", err)
	}
}

func TestSyncExplicitSelectionNeedsNoConfigAndWritesNone(t *testing.T) {
	home := syncHome(t)
	out, err := sync(t, "--skills", "ui-web, go-testing")
	if err != nil {
		t.Fatalf("explicit selection failed: %v\n%s", err, out)
	}
	for _, name := range []string{"ui-web", "go-testing"} {
		if _, err := os.Stat(filepath.Join(home, ".agents", "skills", name, "SKILL.md")); err != nil {
			t.Fatalf("Skill %s not installed: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "zt.json")); !os.IsNotExist(err) {
		t.Fatalf("--skills wrote a global config: %v", err)
	}
	if _, err := sync(t, "--skills", "UI-Web"); err == nil || !strings.Contains(err.Error(), "invalid or duplicate") {
		t.Fatalf("invalid Skill id accepted: %v", err)
	}
}

func TestSyncWithoutConfigOrSelectionFails(t *testing.T) {
	home := syncHome(t)
	out, err := sync(t)
	if err == nil {
		t.Fatalf("sync selected Skills by default:\n%s", out)
	}
	for _, want := range []string{"zt.json", "--skills"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error does not mention %q: %v", want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("failed sync created the global tree: %v", err)
	}

	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":[]}`)
	if _, err := sync(t); err == nil || !strings.Contains(err.Error(), "selects no Skill") {
		t.Fatalf("empty cfg selection accepted: %v", err)
	}
}

func TestSyncRejectsProjectRootWithoutTouchingProject(t *testing.T) {
	home := syncHome(t)
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web"]}`)
	project := t.TempDir()
	projectSkill := filepath.Join(project, ".agents", "skills", "ui-web", "SKILL.md")
	writeFile(t, projectSkill, "project owns this")

	out, err := sync(t, "--root", project)
	if err == nil {
		t.Fatalf("sync --root succeeded:\n%s", out)
	}
	for _, want := range []string{"--root", "user-global"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error does not explain %q: %v", want, err)
		}
	}
	if data, err := os.ReadFile(projectSkill); err != nil || string(data) != "project owns this" {
		t.Fatalf("project Skill changed: %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(project, "zt.lock.json")); !os.IsNotExist(err) {
		t.Fatalf("rejected sync wrote a project lock: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills")); !os.IsNotExist(err) {
		t.Fatalf("rejected sync wrote global Skills: %v", err)
	}
}

func TestSyncPlanIsReadOnly(t *testing.T) {
	home := syncHome(t)
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web"]}`)
	out, err := sync(t, "--plan", "--json")
	if err != nil {
		t.Fatalf("plan failed: %v\n%s", err, out)
	}
	var result struct {
		Actions []struct{ Skill, Action, Target string }
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid plan json %s: %v", out, err)
	}
	if len(result.Actions) != 1 || result.Actions[0].Action != "install" || result.Actions[0].Target != ".agents/skills/ui-web" {
		t.Fatalf("unexpected plan: %s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills")); !os.IsNotExist(err) {
		t.Fatalf("plan created the global Skills directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "zt.lock.json")); !os.IsNotExist(err) {
		t.Fatalf("plan wrote the global lock: %v", err)
	}
}

func TestSyncConflictWritesNothing(t *testing.T) {
	home := syncHome(t)
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web","go-testing"]}`)
	unmanaged := filepath.Join(home, ".agents", "skills", "ui-web", "SKILL.md")
	writeFile(t, unmanaged, "local work")

	if _, err := sync(t); err == nil || !strings.Contains(err.Error(), "not managed") {
		t.Fatalf("unmanaged global Skill was overwritten: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "go-testing")); !os.IsNotExist(err) {
		t.Fatalf("conflicting run installed another Skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "zt.lock.json")); !os.IsNotExist(err) {
		t.Fatalf("conflicting run wrote the lock: %v", err)
	}
	if data, err := os.ReadFile(unmanaged); err != nil || string(data) != "local work" {
		t.Fatalf("unmanaged Skill changed: %q, %v", data, err)
	}

	// A managed Skill with local edits is a conflict too.
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web"]}`)
	if _, err := os.Stat(unmanaged); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(home, ".agents", "skills")); err != nil {
		t.Fatal(err)
	}
	if _, err := sync(t); err != nil {
		t.Fatal(err)
	}
	writeFile(t, unmanaged, "local edit")
	if _, err := sync(t); err == nil || !strings.Contains(err.Error(), "local edits") {
		t.Fatalf("local edit not detected: %v", err)
	}
	if data, err := os.ReadFile(unmanaged); err != nil || string(data) != "local edit" {
		t.Fatalf("local edit overwritten: %q, %v", data, err)
	}
}

func TestSyncRejectsSymlinkedGlobalTree(t *testing.T) {
	home := syncHome(t)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(home, ".agents", "skills")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := sync(t, "--skills", "ui-web"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked global tree accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "ui-web")); !os.IsNotExist(err) {
		t.Fatalf("wrote through the symlink: %v", err)
	}
}

func TestSyncRejectsProjectSectionsInTheGlobalConfig(t *testing.T) {
	home := syncHome(t)
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web"],"modules":[{"id":"web","path":".","stack":"js-ts"}]}`)
	if _, err := sync(t); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("global config accepted project sections: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills")); !os.IsNotExist(err) {
		t.Fatalf("rejected config still wrote Skills: %v", err)
	}
}

func TestSyncRejectsExplicitEmptyRoot(t *testing.T) {
	home := syncHome(t)
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web"]}`)
	out, err := sync(t, "--root=")
	if err == nil {
		t.Fatalf("sync --root= succeeded:\n%s", out)
	}
	for _, want := range []string{"--root", "user-global"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error does not explain %q: %v", want, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills")); !os.IsNotExist(err) {
		t.Fatalf("rejected --root= still wrote global Skills: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "zt.lock.json")); !os.IsNotExist(err) {
		t.Fatalf("rejected --root= still wrote the global lock: %v", err)
	}
}

func TestSyncExplicitEmptySkillsDoesNotFallBackToConfig(t *testing.T) {
	home := syncHome(t)
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web"]}`)
	out, err := sync(t, "--skills=")
	if err == nil {
		t.Fatalf("sync --skills= fell back to the global config:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--skills") || !strings.Contains(err.Error(), "zt.json") {
		t.Fatalf("error does not explain the explicit empty selection: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills")); !os.IsNotExist(err) {
		t.Fatalf("explicit empty selection wrote global Skills: %v", err)
	}
}

func TestSkillsSyncAliasesLegacySync(t *testing.T) {
	home := syncHome(t)
	writeFile(t, filepath.Join(home, ".agents", "zt.json"), `{"schemaVersion":1,"skills":["ui-web"]}`)
	out, _, err := runCLI(t, "skills", "sync", "--plan", "--json")
	if err != nil {
		t.Fatalf("skills sync failed: %v\n%s", err, out)
	}
	var result struct {
		Actions []struct{ Skill, Action, Target string }
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid plan json %s: %v", out, err)
	}
	if len(result.Actions) != 1 || result.Actions[0].Target != ".agents/skills/ui-web" {
		t.Fatalf("unexpected plan: %s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills")); !os.IsNotExist(err) {
		t.Fatalf("skills sync --plan wrote the global tree: %v", err)
	}
}
