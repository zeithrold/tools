package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckOutputHelper(t *testing.T) {
	if os.Getenv("ZT_CHECK_OUTPUT_HELPER") != "1" {
		return
	}
	_, _ = os.Stdout.WriteString("build stdout progress\n")
	_, _ = os.Stderr.WriteString("build stderr progress\n")
	os.Exit(0)
}

func TestCheckStreamsOutputAndPreservesJSON(t *testing.T) {
	for _, asJSON := range []bool{false, true} {
		t.Run(map[bool]string{false: "text", true: "json"}[asJSON], func(t *testing.T) {
			root := t.TempDir()
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("ZT_CHECK_OUTPUT_HELPER", "1")
			config := map[string]any{
				"schemaVersion": 1,
				"modules": []any{map[string]any{
					"id": "app", "path": ".", "stack": "js-ts",
					"commands": map[string]any{"build": []string{binary, "-test.run=^TestCheckOutputHelper$"}},
					"profiles": map[string]any{"ci": []string{"build"}},
				}},
			}
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "zt.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{"check", "--root", root, "--module", "app", "--profile", "ci"}
			if asJSON {
				args = append(args, "--json")
			}
			stdout, stderr, err := runCLI(t, args...)
			if err != nil {
				t.Fatal(err)
			}
			progress := stdout
			if asJSON {
				progress = stderr
				var report struct{ Status string }
				if err := json.Unmarshal([]byte(stdout), &report); err != nil || report.Status != "passed" {
					t.Fatalf("stdout is not a clean successful JSON report: %q, %v", stdout, err)
				}
			}
			for _, message := range []string{"build stdout progress", "build stderr progress"} {
				if !strings.Contains(progress, message) {
					t.Fatalf("live output missing %q: stdout=%q stderr=%q", message, stdout, stderr)
				}
			}
			logs, err := filepath.Glob(filepath.Join(root, ".zt", "artifacts", "check-*", "logs", "*.log"))
			if err != nil || len(logs) != 1 {
				t.Fatalf("missing retained log: %v, %v", logs, err)
			}
			log, err := os.ReadFile(logs[0])
			if err != nil || !strings.Contains(string(log), "build stdout progress") || !strings.Contains(string(log), "build stderr progress") {
				t.Fatalf("retained log missing command output: %q, %v", log, err)
			}
		})
	}
}
