package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, root, name, contents string) {
	t.Helper()
	filename := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGoFuzzPlanUsesBoundedTargetWhenNoRecipe(t *testing.T) {
	root := t.TempDir()
	put(t, root, "go.mod", "module example.test/demo\n\ngo 1.24\n")
	put(t, root, "pkg/x_test.go", "package pkg\nimport \"testing\"\nfunc FuzzValue(f *testing.F) {}\n")
	result, err := Build(root, "root", "fuzz")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Steps) != 1 || strings.Join(result.Steps[0].Argv, " ") != "go test ./pkg -fuzz=^FuzzValue$ -fuzztime=30s" {
		t.Fatalf("unexpected fuzz plan: %+v", result)
	}
	put(t, root, "justfile", "fuzz:\n    go test ./pkg -fuzz=FuzzValue -fuzztime=10s\n")
	result, err = Build(root, "root", "fuzz")
	if err != nil || len(result.Steps) != 1 || strings.Join(result.Steps[0].Argv, " ") != "just fuzz" {
		t.Fatalf("did not preserve project recipe: %+v, %v", result, err)
	}
}

func TestJSTSPlanUsesDeclaredReadOnlyScript(t *testing.T) {
	root := t.TempDir()
	put(t, root, "package.json", `{"packageManager":"pnpm@10.33.0","scripts":{"lint":"eslint .","lint:fix":"eslint . --fix","test":"vitest run"}}`)
	result, err := Build(root, "root", "lint")
	if err != nil || len(result.Steps) != 1 || strings.Join(result.Steps[0].Argv, " ") != "pnpm --config.verify-deps-before-run=error lint" {
		t.Fatalf("unexpected lint plan: %+v, %v", result, err)
	}
}

func TestJSTSUsesLockfileAndRejectsAmbiguityWithoutExecution(t *testing.T) {
	root := t.TempDir()
	put(t, root, "package.json", `{"scripts":{"lint:css":"touch must-not-exist","build":"native-build","test:a11y":"native-browser"}}`)
	put(t, root, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")
	for _, capability := range []string{"css", "build", "a11y"} {
		result, err := Build(root, "root", capability)
		if err != nil || len(result.Steps) != 1 || result.Steps[0].Argv[0] != "pnpm" {
			t.Fatalf("%s: %+v %v", capability, result, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-exist")); !os.IsNotExist(err) {
		t.Fatal("planning executed a script")
	}
	put(t, root, "package-lock.json", "{}")
	result, err := Build(root, "root", "css")
	if err != nil || len(result.Steps) != 0 || len(result.Warnings) == 0 {
		t.Fatalf("ambiguous manager was guessed: %+v %v", result, err)
	}
	put(t, root, "package.json", `{"packageManager":"pnpm@11.22.0","scripts":{"lint:css":"native-css"}}`)
	result, err = Build(root, "root", "css")
	if err != nil || len(result.Steps) != 1 {
		t.Fatalf("explicit manager not respected: %+v %v", result, err)
	}
}

func TestExplicitCommandsRemainArgv(t *testing.T) {
	root := t.TempDir()
	put(t, root, "zt.json", `{"schemaVersion":1,"modules":[{"id":"web","path":".","stack":"js-ts","commands":{"a11y":["runner","literal;argument","$TOKEN"]}}]}`)
	result, err := Build(root, "web", "a11y")
	if err != nil || len(result.Steps) != 1 || strings.Join(result.Steps[0].Argv, "|") != "runner|literal;argument|$TOKEN" {
		t.Fatalf("argv changed: %+v %v", result, err)
	}
}

func TestGoUnitPlanDoesNotRunDiscoveredIntegrationTests(t *testing.T) {
	for _, directory := range []string{"tests/integration", "internal/integration", "integration"} {
		t.Run(directory, func(t *testing.T) {
			root := t.TempDir()
			put(t, root, "go.mod", "module example.test/demo\n\ngo 1.24\n")
			put(t, root, "pkg/unit_test.go", "package pkg\nimport \"testing\"\nfunc TestUnit(t *testing.T) {}\n")
			put(t, root, directory+"/db_test.go", "package integration\nimport \"testing\"\nfunc TestDatabase(t *testing.T) {}\n")
			put(t, root, "justfile", "test:\n    go test ./...\n")
			result, err := Build(root, "root", "unit")
			if err != nil || len(result.Steps) != 0 || len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "commands.unit") {
				t.Fatalf("integration tests included in unit plan: %+v %v", result, err)
			}
			put(t, root, "justfile", "test-unit:\n    go test ./pkg\n")
			result, err = Build(root, "root", "unit")
			if err != nil || len(result.Steps) != 1 || strings.Join(result.Steps[0].Argv, " ") != "just test-unit" {
				t.Fatalf("reviewed unit recipe lost: %+v %v", result, err)
			}
			put(t, root, "zt.json", `{"schemaVersion":1,"modules":[{"id":"api","path":".","stack":"go","commands":{"unit":["go","test","./pkg"]}}]}`)
			result, err = Build(root, "api", "unit")
			if err != nil || len(result.Steps) != 1 || strings.Join(result.Steps[0].Argv, " ") != "go test ./pkg" {
				t.Fatalf("explicit unit scope lost: %+v %v", result, err)
			}
		})
	}
}

func TestGoConsumerSpecificMutationRequiresExplicitCommand(t *testing.T) {
	root := t.TempDir()
	put(t, root, "go.mod", "module example.test/demo\n\ngo 1.24\n")
	put(t, root, "consumer-policy.json", `{"commands":{"mutation-special-scope":{}}}`)
	put(t, root, "justfile", "mutation-special-scope:\n    touch must-not-exist\n")
	result, err := Build(root, "root", "mutation")
	if err != nil || len(result.Steps) != 0 || len(result.Warnings) == 0 {
		t.Fatalf("consumer command was inferred: %+v %v", result, err)
	}
	put(t, root, "zt.json", `{"schemaVersion":1,"modules":[{"id":"api","path":".","stack":"go","commands":{"mutation":["just","mutation-special-scope"]}}]}`)
	result, err = Build(root, "api", "mutation")
	if err != nil || len(result.Steps) != 1 || strings.Join(result.Steps[0].Argv, " ") != "just mutation-special-scope" {
		t.Fatalf("explicit consumer command lost: %+v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-exist")); !os.IsNotExist(err) {
		t.Fatal("planning executed the consumer recipe")
	}
}
