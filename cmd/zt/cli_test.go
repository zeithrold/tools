package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootHelpListsEveryCommandWithoutErrors(t *testing.T) {
	stdout, stderr, err := runCLI(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Fatalf("help wrote to stderr: %q", stderr)
	}
	for _, name := range []string{"inspect", "plan", "check", "skills", "sync", "auth", "version", "license", "completion"} {
		if !strings.Contains(stdout, name) {
			t.Fatalf("help omits %q:\n%s", name, stdout)
		}
	}
	if strings.Contains(stdout, "evolve") {
		t.Fatalf("help advertises an unimplemented command:\n%s", stdout)
	}
}

func TestNoArgsAndUnknownCommandFailWithOneError(t *testing.T) {
	_, stderr, err := runCLI(t)
	if err == nil || !strings.Contains(err.Error(), "no command") {
		t.Fatalf("empty invocation did not fail: %v", err)
	}
	if stderr != "" {
		t.Fatalf("Cobra printed a second error to stderr: %q", stderr)
	}

	_, stderr, err = runCLI(t, "bogus")
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unknown command was accepted: %v", err)
	}
	if stderr != "" {
		t.Fatalf("Cobra printed a second error to stderr: %q", stderr)
	}
}

func TestGlobalFlagsAreAcceptedEverywhere(t *testing.T) {
	if _, _, err := runCLI(t, "skills", "list", "--no-input", "--no-color"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runCLI(t, "--no-input", "version"); err != nil {
		t.Fatal(err)
	}
}

func TestCompletionIsAvailable(t *testing.T) {
	stdout, _, err := runCLI(t, "completion", "bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "zt") || !strings.Contains(stdout, "completion") {
		t.Fatalf("unexpected completion script:\n%s", stdout)
	}
}

func TestSkillsListComesFromTheEmbeddedTree(t *testing.T) {
	stdout, stderr, err := runCLI(t, "skills", "list")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Fatalf("skills list wrote to stderr: %q", stderr)
	}
	for _, id := range []string{"verification", "ui-design", "frontend-engineering"} {
		if !strings.Contains(stdout, id) {
			t.Fatalf("skills list omits %q:\n%s", id, stdout)
		}
	}

	stdout, _, err = runCLI(t, "skills", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		SchemaVersion int      `json:"schemaVersion"`
		Skills        []string `json:"skills"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid skills list json %q: %v", stdout, err)
	}
	if report.SchemaVersion != 1 || len(report.Skills) == 0 {
		t.Fatalf("unexpected skills list: %+v", report)
	}
}

func TestPlanIsReadOnlyAndDoesNotResolveHome(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/fixture\n\ngo 1.24\n")
	writeFile(t, filepath.Join(root, "fixture_test.go"), "package fixture\n")
	writeFile(t, filepath.Join(root, "zt.json"), `{"schemaVersion":1,"modules":[{"id":"root","path":".","stack":"go"}]}`)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	a := newApp(stdout, stderr)
	a.homeDir = func() (string, error) {
		t.Fatal("plan resolved the home directory")
		return "", nil
	}
	a.newAuth = func(string) (authService, error) {
		t.Fatal("plan constructed the auth service")
		return nil, nil
	}
	if err := executeApp(a, context.Background(), []string{"plan", "unit", "--root", root}); err != nil {
		t.Fatalf("plan failed: %v\n%s", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".zt")); !os.IsNotExist(err) {
		t.Fatalf("plan created .zt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("plan created .agents: %v", err)
	}
}

func TestInspectJSONIsTheOnlyStdoutOutput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "zt.json"), `{"schemaVersion":1,"modules":[{"id":"root","path":".","stack":"go"}]}`)
	stdout, stderr, err := runCLI(t, "inspect", "--root", root, "--json")
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Fatalf("inspect --json wrote to stderr: %q", stderr)
	}
	var report map[string]any
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("stdout is not pure JSON %q: %v", stdout, err)
	}
}

func TestAuthAndSkillsRejectUnknownSubcommandsButHelpSucceeds(t *testing.T) {
	cases := []struct{ group, typo, subcommand string }{
		{"auth", "logni", "login"},
		{"skills", "sycn", "list"},
	}
	for _, c := range cases {
		stdout, stderr, err := runCLI(t, c.group, c.typo)
		if err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Fatalf("%s %s did not fail as an unknown subcommand: %v", c.group, c.typo, err)
		}
		if stdout != "" || stderr != "" {
			t.Fatalf("%s %s printed output before failing: stdout=%q stderr=%q", c.group, c.typo, stdout, stderr)
		}

		stdout, _, err = runCLI(t, c.group, "--help")
		if err != nil {
			t.Fatalf("%s --help failed: %v", c.group, err)
		}
		if !strings.Contains(stdout, c.subcommand) {
			t.Fatalf("%s --help omits %q:\n%s", c.group, c.subcommand, stdout)
		}

		stdout, _, err = runCLI(t, c.group)
		if err != nil {
			t.Fatalf("bare %s should print help and succeed: %v", c.group, err)
		}
		if !strings.Contains(stdout, c.subcommand) {
			t.Fatalf("bare %s help omits %q:\n%s", c.group, c.subcommand, stdout)
		}
	}
}
