package inspect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func capability(t *testing.T, report Report, name string) Capability {
	t.Helper()
	for _, item := range report.Modules[0].Capabilities {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("capability %s missing", name)
	return Capability{}
}

func TestGoInspectKeepsTestKindsAndExecutionSeparate(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module example.test/demo\n\ngo 1.24\n")
	write(t, root, "internal/x/x_test.go", "package x\nimport \"testing\"\nfunc TestUnit(t *testing.T) {}\nfunc FuzzInput(f *testing.F) {}\n")
	write(t, root, "tests/integration/db_test.go", "package integration\nimport \"testing\"\nfunc TestDatabase(t *testing.T) {}\n")
	write(t, root, "governance.json", `{"commands":{"mutation-accounting":{}}}`)
	report, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"unit", "integration", "fuzz", "mutation"} {
		item := capability(t, report, name)
		if item.Detection != "discovered" || item.LastRun != "not_run" || len(item.Evidence) != 1 {
			t.Fatalf("%s: unexpected report %+v", name, item)
		}
	}
}

func TestJSTSInspectDoesNotTreatFixOrBuildAsChecks(t *testing.T) {
	root := t.TempDir()
	write(t, root, "package.json", `{"scripts":{"test":"vitest run","lint:fix":"eslint --fix .","e2e:build":"build","test:e2e":"playwright test"}}`)
	report, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if item := capability(t, report, "lint"); item.Detection != "not_detected" {
		t.Fatalf("lint:fix is not a read-only check: %+v", item)
	}
	if item := capability(t, report, "e2e"); len(item.Evidence) != 1 || !strings.Contains(item.Evidence[0], "test:e2e") {
		t.Fatalf("e2e build was misclassified: %+v", item)
	}
}

func TestJVMRootDoesNotClaimChildTestsAreAbsent(t *testing.T) {
	root := t.TempDir()
	write(t, root, "settings.gradle.kts", `include(":service")`)
	write(t, root, "build.gradle.kts", `plugins { kotlin("jvm") version "2.0" apply false }`)
	write(t, root, "service/src/test/kotlin/ServiceTest.kt", "class ServiceTest")
	report, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if item := capability(t, report, "unit"); item.Detection != "unknown" {
		t.Fatalf("root-only JVM inspection must remain inconclusive: %+v", item)
	}
}
