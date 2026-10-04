package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsUnknownFieldsAndEscapingPaths(t *testing.T) {
	root := t.TempDir()
	put := func(data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, ConfigName), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	put(`{"schemaVersion":1,"unexpected":true}`)
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
	put(`{"schemaVersion":1,"modules":[{"id":"api","path":"../elsewhere","stack":"go"}]}`)
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected path escape error, got %v", err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err == nil {
		put(`{"schemaVersion":1,"modules":[{"id":"api","path":"linked","stack":"go"}]}`)
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "escapes") {
			t.Fatalf("expected symlink escape error, got %v", err)
		}
	}
}

func TestProfilesCommandsAndArtifactsRejectInvalidConfiguration(t *testing.T) {
	root := t.TempDir()
	for _, fields := range []string{
		`"profiles":{"frontend":[]}`,
		`"profiles":{"frontend":["css","css"]}`,
		`"profiles":{"frontend":["csss"]}`,
		`"commands":{"css":[]}`,
		`"commands":{"css":["pnpm","--config.verify-deps-before-run=install","lint"]}`,
		`"commands":{"css":["pnpm","--config.verifyDepsBeforeRun=install","lint"]}`,
		`"commands":{"css":["pnpm","--config.verifyDepsBeforeRun=false","lint"]}`,
		`"commands":{"css":["pnpm","--config.verifyDepsBeforeRun","install","lint"]}`,
		`"commands":{"css":["pnpm","--config.verify-deps-before-run","error","lint"]}`,
		`"commands":{"css":["runner","\u0000"]}`,
		`"artifacts":["../outside"]`,
		`"artifacts":["."]`,
		`"expect":{"a11yy":"required"}`,
	} {
		data := `{"schemaVersion":1,"modules":[{"id":"web","path":".","stack":"js-ts",` + fields + `}]}`
		if err := os.WriteFile(filepath.Join(root, ConfigName), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(root); err == nil {
			t.Fatalf("accepted %s", fields)
		}
	}
}

func TestDependencyReadinessErrorAliasesAreAllowed(t *testing.T) {
	root := t.TempDir()
	for _, arg := range []string{"--config.verify-deps-before-run=error", "--config.verifyDepsBeforeRun=error"} {
		config := Config{SchemaVersion: 1, Modules: []Module{{ID: "web", Path: ".", Stack: "js-ts", Commands: map[string][]string{"lint": {"pnpm", arg, "lint"}}}}}
		if err := config.Validate(root); err != nil {
			t.Fatalf("rejected safe argument %q: %v", arg, err)
		}
	}
}
