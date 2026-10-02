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
