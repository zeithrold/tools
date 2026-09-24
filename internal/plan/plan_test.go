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
	if err != nil || len(result.Steps) != 1 || strings.Join(result.Steps[0].Argv, " ") != "pnpm lint" {
		t.Fatalf("unexpected lint plan: %+v, %v", result, err)
	}
}
