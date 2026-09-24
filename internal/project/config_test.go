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
