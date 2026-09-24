package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeithrold/tools/internal/project"
)

func TestSyncInstallsCompleteDirectoryAndPreservesProjectEdits(t *testing.T) {
	root := t.TempDir()
	config := &project.Config{SchemaVersion: 1, Skills: []string{"ui-foundation"}}
	projectSkill := filepath.Join(root, ".agents", "skills", "project-ui", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(projectSkill), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectSkill, []byte("project owns this"), 0644); err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(root, config)
	if err != nil || len(plan.Actions) != 1 || plan.Actions[0].Action != "install" {
		t.Fatalf("unexpected plan: %+v, %v", plan, err)
	}
	if _, err := os.Stat(filepath.Join(root, "zt.lock.json")); !os.IsNotExist(err) {
		t.Fatalf("plan wrote a lock: %v", err)
	}
	if _, err := Apply(root, config); err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(root, ".agents", "skills", "ui-foundation")
	for _, name := range []string{"SKILL.md", "references/review-matrix.md"} {
		if _, err := os.Stat(filepath.Join(managed, name)); err != nil {
			t.Fatalf("missing bundled file %s: %v", name, err)
		}
	}
	if data, err := os.ReadFile(projectSkill); err != nil || string(data) != "project owns this" {
		t.Fatalf("project skill changed: %q, %v", data, err)
	}
	plan, err = Plan(root, config)
	if err != nil || plan.Actions[0].Action != "unchanged" {
		t.Fatalf("sync is not idempotent: %+v, %v", plan, err)
	}
	custom := filepath.Join(managed, "SKILL.md")
	if err := os.WriteFile(custom, []byte("local edit"), 0644); err != nil {
		t.Fatal(err)
	}
	plan, err = Plan(root, config)
	if err != nil || plan.Actions[0].Action != "conflict" {
		t.Fatalf("local edit not detected: %+v, %v", plan, err)
	}
	if _, err := Apply(root, config); err == nil || !strings.Contains(err.Error(), "local edits") {
		t.Fatalf("expected conflict, got %v", err)
	}
	if data, err := os.ReadFile(custom); err != nil || string(data) != "local edit" {
		t.Fatalf("local edit overwritten: %q, %v", data, err)
	}
}
