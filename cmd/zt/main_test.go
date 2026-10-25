package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tools "github.com/zeithrold/tools"
)

// runCLI drives one invocation through the injected command tree. It never
// reads os.Args, so tests cannot leak state between cases.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	err = executeApp(newApp(out, errOut), context.Background(), args)
	return out.String(), errOut.String(), err
}

func TestLicenseCommandRetainsEmbeddedNotices(t *testing.T) {
	output, _, err := runCLI(t, "license")
	if err != nil {
		t.Fatal(err)
	}
	names, err := licenseFileNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no embedded notices")
	}
	last := -1
	for _, name := range names {
		data, err := tools.LicenseFiles.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains([]byte(output), data) {
			t.Fatalf("license command omitted %s", name)
		}
		at := strings.Index(output, name+"\n")
		if at <= last {
			t.Fatalf("license output is not in stable path order at %s", name)
		}
		last = at
	}
	if _, _, err := runCLI(t, "license", "extra"); err == nil {
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
	output, _, err := runCLI(t, "check", "--root", root, "--module", "web", "--json")
	if err == nil {
		t.Fatal("required failure returned success")
	}
	var report struct{ Status, ArtifactDir string }
	if err := json.Unmarshal([]byte(output), &report); err != nil || report.Status != "failed" {
		t.Fatalf("invalid report %s: %v", output, err)
	}
	if _, err := os.Stat(filepath.Join(report.ArtifactDir, "report.json")); err != nil {
		t.Fatal(err)
	}
}
