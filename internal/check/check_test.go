package check

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeithrold/tools/internal/plan"
	"github.com/zeithrold/tools/internal/project"
)

// The current test binary is a real, portable subprocess (no mocked runner).
func TestNativeHelper(t *testing.T) {
	if os.Getenv("ZT_CHECK_TEST_CHILD") != "1" {
		return
	}
	args := os.Args
	mode := args[len(args)-1]
	switch mode {
	case "fail":
		os.Stdout.WriteString("native failure evidence\n")
		os.Exit(7)
	case "wait":
		time.Sleep(30 * time.Second)
	case "artifact":
		os.MkdirAll("browser-results", 0755)
		os.WriteFile("browser-results/evidence.json", []byte(`{"browser":"chromium"}`), 0644)
	}
	os.Stdout.WriteString("native success evidence\n")
	os.Exit(0)
}

func fixture(t *testing.T, modes map[string]string, levels map[string]string, artifacts []string) (string, Options) {
	t.Helper()
	t.Setenv("ZT_CHECK_TEST_CHILD", "1")
	root := t.TempDir()
	commands := map[string][]string{}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for capability, mode := range modes {
		commands[capability] = []string{binary, "-test.run=^TestNativeHelper$", "--", mode}
	}
	config := project.Config{SchemaVersion: 1, Modules: []project.Module{{ID: "web", Path: ".", Stack: "js-ts", Expect: levels, Commands: commands, Profiles: map[string][]string{"frontend": {"lint", "css", "a11y"}}, Artifacts: artifacts}}}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "zt.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return root, Options{Root: root, Module: "web", Profile: "frontend", Timeout: 2 * time.Second}
}

func TestSuccessCollectsActualLogsAndArtifacts(t *testing.T) {
	_, options := fixture(t, map[string]string{"lint": "pass", "css": "pass", "a11y": "artifact"}, nil, []string{"browser-results"})
	report, err := Run(context.Background(), options)
	if err != nil || report.Status != "passed" {
		t.Fatalf("%+v %v", report, err)
	}
	if len(report.Checks) != 3 || report.Checks[2].Steps[0].ExitCode != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	for _, name := range []string{"report.json", "logs/00-lint-00.log", "collected/browser-results/evidence.json"} {
		data, err := os.ReadFile(filepath.Join(report.ArtifactDir, name))
		if err != nil || len(data) == 0 {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestFailureStopsRequiredChecksAndKeepsEvidence(t *testing.T) {
	_, options := fixture(t, map[string]string{"lint": "fail", "css": "pass", "a11y": "pass"}, nil, nil)
	report, err := Run(context.Background(), options)
	if err != nil || report.Status != "failed" || report.Checks[0].Status != "failed" || report.Checks[1].Status != "not_run" {
		t.Fatalf("%+v %v", report, err)
	}
	if report.Checks[0].Steps[0].ExitCode != 7 {
		t.Fatal("lost native exit code")
	}
	data, err := os.ReadFile(filepath.Join(report.ArtifactDir, report.Checks[0].Steps[0].Log))
	if err != nil || !strings.Contains(string(data), "native failure evidence") {
		t.Fatalf("lost failure log: %s %v", data, err)
	}
}

func TestWarnFailureContinuesAndRequiredMissingBlocks(t *testing.T) {
	_, options := fixture(t, map[string]string{"lint": "fail", "css": "pass"}, map[string]string{"lint": "warn"}, nil)
	report, err := Run(context.Background(), options)
	if err != nil || report.Status != "failed" || report.Checks[1].Status != "passed" || report.Checks[2].Status != "blocked" || len(report.Warnings) != 1 {
		t.Fatalf("%+v %v", report, err)
	}
}

func TestTimeoutAndMissingExecutableAreNotPasses(t *testing.T) {
	_, options := fixture(t, map[string]string{"lint": "wait"}, nil, nil)
	options.Timeout = 50 * time.Millisecond
	report, err := Run(context.Background(), options)
	if err != nil || report.Status != "failed" || !strings.Contains(report.Checks[0].Steps[0].Detail, "deadline") {
		t.Fatalf("%+v %v", report, err)
	}
	step := execute(context.Background(), plan.Step{Dir: options.Root, Argv: []string{"zt-nonexistent-test-runner"}}, time.Second, report.ArtifactDir, "logs/missing.log")
	if step.Status != "blocked" || step.ExitCode != -1 {
		t.Fatalf("%+v", step)
	}
}

func TestArtifactSymlinksAndMissingSourcesFail(t *testing.T) {
	root, options := fixture(t, map[string]string{"lint": "pass", "css": "pass", "a11y": "pass"}, nil, []string{"absent"})
	report, err := Run(context.Background(), options)
	if err != nil || report.Status != "failed" {
		t.Fatalf("%+v %v", report, err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "linked")); err != nil {
		t.Skip(err)
	}
	if _, err := collect(root, report.ArtifactDir, "linked"); err == nil {
		t.Fatal("followed external symlink")
	}
	options.Output = filepath.Join(root, "linked", "output")
	if _, err := Run(context.Background(), options); err == nil {
		t.Fatal("wrote through symlink")
	}
}

func TestSelectedBaseAliasesPreserveArtifactCollection(t *testing.T) {
	root, options := fixture(t, map[string]string{"lint": "pass", "css": "pass", "a11y": "artifact"}, nil, []string{"browser-results"})
	rootAlias := filepath.Join(t.TempDir(), "selected-project")
	if err := os.Symlink(root, rootAlias); err != nil {
		t.Skip(err)
	}
	options.Root = rootAlias
	for _, externalOutput := range []bool{false, true} {
		t.Run(fmt.Sprint(externalOutput), func(t *testing.T) {
			if externalOutput {
				outputAlias := filepath.Join(t.TempDir(), "selected-output")
				if err := os.Symlink(t.TempDir(), outputAlias); err != nil {
					t.Fatal(err)
				}
				options.Output = filepath.Join(outputAlias, "new", "checks")
			}
			report, err := Run(context.Background(), options)
			if err != nil || report.Status != "passed" {
				t.Fatalf("selected base alias rejected: %+v %v", report, err)
			}
			if _, err := os.ReadFile(filepath.Join(report.ArtifactDir, "collected/browser-results/evidence.json")); err != nil {
				t.Fatalf("artifact collection through root alias failed: %v", err)
			}
		})
	}
}

func TestDependencyReadinessAliasesAreRejectedBeforeExecution(t *testing.T) {
	for _, arg := range []string{"--config.verify-deps-before-run=install", "--config.verifyDepsBeforeRun=install", "--config.verifyDepsBeforeRun=false"} {
		t.Run(arg, func(t *testing.T) {
			root := t.TempDir()
			config := project.Config{SchemaVersion: 1, Modules: []project.Module{{ID: "web", Path: ".", Stack: "js-ts", Commands: map[string][]string{"lint": {"pnpm", arg, "run", "lint"}}, Profiles: map[string][]string{"native": {"lint"}}}}}
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "zt.json"), data, 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := Run(context.Background(), Options{Root: root, Module: "web", Profile: "native", Timeout: time.Second}); err == nil || !strings.Contains(err.Error(), "dependency readiness") {
				t.Fatalf("accepted readiness override: %v", err)
			}
			for _, name := range []string{".zt", "node_modules", "pnpm-lock.yaml"} {
				if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
					t.Fatalf("check wrote %s before rejecting configuration: %v", name, err)
				}
			}
		})
	}
}

func TestNativePackageScriptsUsePNPMAndPropagateFailure(t *testing.T) {
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skip("pnpm not installed")
	}
	root := t.TempDir()
	// No packageManager: the lock is sufficient, and no installation is needed.
	os.WriteFile(filepath.Join(root, "pnpm-lock.yaml"), []byte("lockfileVersion: '9.0'\n"), 0644)
	os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"lint":"node -e \"console.log('native pnpm');process.exit(9)\""}}`), 0644)
	os.WriteFile(filepath.Join(root, "zt.json"), []byte(`{"schemaVersion":1,"modules":[{"id":"web","path":".","stack":"js-ts","profiles":{"frontend":["lint"]}}]}`), 0644)
	// Provision this dependency-free fixture explicitly. check must never do so.
	t.Setenv("PNPM_HOME", filepath.Join(root, "pnpm-home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	install := exec.Command("pnpm", "install", "--ignore-scripts")
	install.Dir = root
	if data, err := install.CombinedOutput(); err != nil {
		t.Fatalf("fixture install: %s %v", data, err)
	}
	report, err := Run(context.Background(), Options{Root: root, Module: "web", Profile: "frontend", Timeout: 10 * time.Second})
	if err != nil || report.Status != "failed" || report.Checks[0].Steps[0].ExitCode == 0 || report.Checks[0].Steps[0].Argv[0] != "pnpm" {
		t.Fatalf("%+v %v", report, err)
	}

	data, readErr := os.ReadFile(filepath.Join(report.ArtifactDir, report.Checks[0].Steps[0].Log))
	if readErr != nil || !strings.Contains(string(data), "native pnpm") {
		t.Fatalf("native script was not reached: %s %v", data, readErr)
	}
}

func TestPNPMMissingDependenciesCannotInstallDuringCheck(t *testing.T) {
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skip("pnpm not installed")
	}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"lint":"node -e \"console.log('should not run')\""}}`), 0644)
	os.WriteFile(filepath.Join(root, "pnpm-lock.yaml"), []byte("lockfileVersion: '9.0'\n"), 0644)
	os.WriteFile(filepath.Join(root, "zt.json"), []byte(`{"schemaVersion":1,"modules":[{"id":"web","path":".","stack":"js-ts","profiles":{"frontend":["lint"]}}]}`), 0644)
	report, err := Run(context.Background(), Options{Root: root, Module: "web", Profile: "frontend", Timeout: 10 * time.Second})
	if err != nil || report.Status != "failed" {
		t.Fatalf("%+v %v", report, err)
	}
	if _, err := os.Stat(filepath.Join(root, "node_modules")); !os.IsNotExist(err) {
		t.Fatal("check installed dependencies")
	}
	data, err := os.ReadFile(filepath.Join(report.ArtifactDir, report.Checks[0].Steps[0].Log))
	if err != nil || !strings.Contains(string(data), "VERIFY_DEPS_BEFORE_RUN") {
		t.Fatalf("readiness blocker not surfaced: %s %v", data, err)
	}
}

func TestInvalidProfileRunsNothing(t *testing.T) {
	_, options := fixture(t, map[string]string{"lint": "pass"}, nil, nil)
	options.Profile = "missing"
	if _, err := Run(context.Background(), options); err == nil {
		t.Fatal("accepted missing profile")
	}
	if _, err := os.Stat(filepath.Join(options.Root, ".zt")); !os.IsNotExist(err) {
		t.Fatal("created artifacts before validating profile")
	}
}
