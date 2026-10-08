package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	tools "github.com/zeithrold/tools"
)

func TestLicenseCommandRetainsEmbeddedNotices(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"license"}, &output); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"LICENSE", "third-party/GO-BSD.txt"} {
		data, err := tools.LicenseFiles.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(output.Bytes(), data) {
			t.Fatalf("license command omitted %s", name)
		}
	}
	if err := run([]string{"license", "extra"}, &output); err == nil {
		t.Fatal("license accepted unexpected arguments")
	}
}

func TestCheckCLIReportsRequiredFailureAndRetainsJSON(t *testing.T) {
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skip("pnpm not installed")
	}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"packageManager":"pnpm@11.22.0","scripts":{"lint":"node -e \"process.exit(2)\""}}`), 0644)
	os.WriteFile(filepath.Join(root, "zt.json"), []byte(`{"schemaVersion":1,"modules":[{"id":"web","path":".","stack":"js-ts","profiles":{"frontend":["lint"]}}]}`), 0644)
	t.Setenv("PNPM_HOME", filepath.Join(root, "pnpm-home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	install := exec.Command("pnpm", "install", "--ignore-scripts")
	install.Dir = root
	if data, err := install.CombinedOutput(); err != nil {
		t.Fatalf("fixture install: %s %v", data, err)
	}
	var output bytes.Buffer
	if err := run([]string{"check", "--root", root, "--module", "web", "--json"}, &output); err == nil {
		t.Fatal("required failure returned success")
	}
	var report struct{ Status, ArtifactDir string }
	if err := json.Unmarshal(output.Bytes(), &report); err != nil || report.Status != "failed" {
		t.Fatalf("invalid report %s: %v", output.String(), err)
	}
	if _, err := os.Stat(filepath.Join(report.ArtifactDir, "report.json")); err != nil {
		t.Fatal(err)
	}
}
