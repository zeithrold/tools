package inspect

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zeithrold/tools/internal/project"
)

type Report struct {
	SchemaVersion int      `json:"schemaVersion"`
	Root          string   `json:"root"`
	Configured    bool     `json:"configured"`
	Modules       []Module `json:"modules"`
	Warnings      []string `json:"warnings,omitempty"`
}

type Module struct {
	ID           string       `json:"id"`
	Path         string       `json:"path"`
	Stack        string       `json:"stack"`
	Capabilities []Capability `json:"capabilities"`
}

type Capability struct {
	Name        string   `json:"name"`
	Detection   string   `json:"detection"` // discovered, not_detected, or unknown
	Evidence    []string `json:"evidence,omitempty"`
	Expectation string   `json:"expectation,omitempty"`
	Environment string   `json:"environment"` // inspect never runs tool probes
	LastRun     string   `json:"lastRun"`     // inspect never claims a check passed
	Next        []string `json:"next,omitempty"`
}

type collector struct {
	root       string
	evidence   map[string][]string
	warnings   []string
	incomplete bool
}

func Run(root string) (Report, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	config, err := project.Load(abs)
	if err != nil {
		return Report{}, err
	}
	report := Report{SchemaVersion: 1, Root: abs, Configured: config != nil, Modules: []Module{}}
	var modules []project.Module
	if config == nil || len(config.Modules) == 0 {
		stack := detectStack(abs)
		if stack == "" {
			return report, nil
		}
		modules = []project.Module{{ID: "root", Path: ".", Stack: stack}}
	} else {
		modules = config.Modules
	}
	for _, declared := range modules {
		moduleRoot, err := project.ModulePath(abs, declared.Path)
		if err != nil {
			return Report{}, err
		}
		c := collector{root: moduleRoot, evidence: make(map[string][]string)}
		switch declared.Stack {
		case "go":
			c.inspectGo()
		case "js-ts":
			c.inspectJSTS()
		case "flutter":
			c.inspectFlutter()
		case "rust":
			c.inspectRust()
		case "python":
			c.inspectPython()
		case "jvm":
			c.inspectJVM()
		}
		module := Module{ID: declared.ID, Path: declared.Path, Stack: declared.Stack, Capabilities: []Capability{}}
		for _, name := range []string{"unit", "integration", "property", "fuzz", "mutation", "lint", "typecheck", "e2e"} {
			evidence := c.evidence[name]
			status := "not_detected"
			if len(evidence) > 0 {
				status = "discovered"
			} else if c.incomplete {
				status = "unknown"
			} else if declared.Stack == "jvm" && (name == "fuzz" || name == "property") {
				status = "unknown" // a task name alone cannot distinguish the algorithms
			}
			var next []string
			if status == "discovered" && (declared.Stack == "go" || declared.Stack == "js-ts") {
				next = []string{"zt", "plan", name, "--module", declared.ID, "--root", abs}
			}
			module.Capabilities = append(module.Capabilities, Capability{
				Name: name, Detection: status, Evidence: evidence,
				Expectation: declared.Expect[name], Environment: "not_checked", LastRun: "not_run", Next: next,
			})
		}
		report.Modules = append(report.Modules, module)
		for _, warning := range c.warnings {
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s: %s", declared.ID, warning))
		}
		for _, capability := range module.Capabilities {
			if capability.Expectation == "required" && capability.Detection != "discovered" {
				report.Warnings = append(report.Warnings, fmt.Sprintf("%s: required %s capability has no confirmed entry point", declared.ID, capability.Name))
			}
			if capability.Expectation == "warn" && capability.Detection == "not_detected" {
				report.Warnings = append(report.Warnings, fmt.Sprintf("%s: no %s entry point detected", declared.ID, capability.Name))
			}
		}
	}
	return report, nil
}

func detectStack(root string) string {
	for _, item := range []struct{ file, stack string }{
		{"go.mod", "go"}, {"pubspec.yaml", "flutter"}, {"Cargo.toml", "rust"},
		{"package.json", "js-ts"}, {"pyproject.toml", "python"},
		{"build.gradle.kts", "jvm"}, {"build.gradle", "jvm"}, {"pom.xml", "jvm"},
	} {
		if regular(filepath.Join(root, item.file)) {
			return item.stack
		}
	}
	return ""
}

func (c *collector) add(name, evidence string) {
	c.evidence[name] = append(c.evidence[name], evidence)
}

func (c *collector) inspectGo() {
	c.walk(func(path, relative string) {
		if !strings.HasSuffix(path, "_test.go") {
			return
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			c.warnings = append(c.warnings, fmt.Sprintf("cannot parse %s: %v", relative, err))
			c.incomplete = true
			return
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
				continue
			}
			field := fn.Type.Params.List[0]
			pointer, ok := field.Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			selector, ok := pointer.X.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok || pkg.Name != "testing" {
				continue
			}
			name := fn.Name.Name
			if strings.HasPrefix(name, "Test") && selector.Sel.Name == "T" {
				if strings.HasPrefix(relative, "tests/integration/") || strings.Contains(relative, "/integration/") {
					c.add("integration", relative+":"+name)
				} else {
					c.add("unit", relative+":"+name)
				}
			} else if strings.HasPrefix(name, "Fuzz") && selector.Sel.Name == "F" {
				c.add("fuzz", relative+":"+name)
			}
		}
	})
	c.inspectDeclaredCommands("governance.json")
	if regular(filepath.Join(c.root, ".golangci.yml")) || regular(filepath.Join(c.root, ".golangci.yaml")) {
		c.add("lint", ".golangci.yml/.yaml")
	}
}

func (c *collector) inspectJSTS() {
	data, err := os.ReadFile(filepath.Join(c.root, "package.json"))
	if err != nil {
		c.warnings = append(c.warnings, "cannot read package.json: "+err.Error())
		return
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		c.warnings = append(c.warnings, "cannot parse package.json: "+err.Error())
		return
	}
	for name := range pkg.Scripts {
		lower := strings.ToLower(name)
		evidence := "package.json:scripts." + name
		switch {
		case lower == "lint" || lower == "lint:check":
			c.add("lint", evidence)
		case lower == "typecheck" || lower == "check:types" || lower == "check-types":
			c.add("typecheck", evidence)
		case lower == "test:e2e" || lower == "e2e" || lower == "e2e:test":
			c.add("e2e", evidence)
		case lower == "test" || lower == "test:unit" || lower == "test-unit":
			c.add("unit", evidence)
		case strings.Contains(lower, "mutation") || strings.Contains(lower, "mutant"):
			c.add("mutation", evidence)
		case strings.Contains(lower, "fuzz"):
			c.add("fuzz", evidence+" (task name only; verify semantics)")
		}
	}
}

func (c *collector) inspectFlutter() {
	c.walk(func(path, relative string) {
		if !strings.HasSuffix(path, "_test.dart") {
			return
		}
		if strings.HasPrefix(relative, "integration_test/") {
			c.add("integration", relative)
		} else if strings.HasPrefix(relative, "test/") {
			c.add("unit", relative+" (unit or widget test)")
		}
	})
	if regular(filepath.Join(c.root, "analysis_options.yaml")) {
		c.add("lint", "analysis_options.yaml")
	}
}

func (c *collector) inspectRust() {
	c.walk(func(path, relative string) {
		if !strings.HasSuffix(relative, ".rs") {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			c.warnings = append(c.warnings, "cannot read "+relative+": "+err.Error())
			c.incomplete = true
			return
		}
		if strings.Contains(string(data), "#[test]") || strings.Contains(string(data), "#[tokio::test]") {
			c.add("unit", relative+": test annotation candidate")
		}
	})
	if regular(filepath.Join(c.root, "fuzz/Cargo.toml")) {
		c.add("fuzz", "fuzz/Cargo.toml")
	}
	if regular(filepath.Join(c.root, "justfile")) {
		c.inspectTextRecipes("justfile")
	}
}

func (c *collector) inspectPython() {
	c.walk(func(path, relative string) {
		base := filepath.Base(path)
		if strings.HasSuffix(base, ".py") && (strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py")) {
			c.add("unit", relative)
		}
	})
	data, err := os.ReadFile(filepath.Join(c.root, "pyproject.toml"))
	if err == nil && strings.Contains(string(data), "[tool.ruff") {
		c.add("lint", "pyproject.toml:[tool.ruff]")
	}
}

func (c *collector) inspectJVM() {
	// A root Gradle build may contain all tests in child modules. Until those
	// modules are declared in zt.json, absence at the root is inconclusive.
	if regular(filepath.Join(c.root, "settings.gradle.kts")) || regular(filepath.Join(c.root, "settings.gradle")) {
		c.incomplete = true
		c.warnings = append(c.warnings, "multi-module JVM build: declare module paths in zt.json for a complete inventory")
	}
	if dir(filepath.Join(c.root, "src/test")) {
		c.add("unit", "src/test/")
	}
	if dir(filepath.Join(c.root, "src/integrationTest")) {
		c.add("integration", "src/integrationTest/")
	}
	for _, name := range []string{"build.gradle.kts", "build.gradle", "pom.xml"} {
		data, err := os.ReadFile(filepath.Join(c.root, name))
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(data))
		if strings.Contains(lower, "detekt") || strings.Contains(lower, "spotless") || strings.Contains(lower, "checkstyle") {
			c.add("lint", name+": quality plugin declaration")
		}
		if strings.Contains(lower, "pitest") {
			c.add("mutation", name+": pitest declaration")
		}
	}
}

func (c *collector) inspectDeclaredCommands(name string) {
	data, err := os.ReadFile(filepath.Join(c.root, name))
	if err != nil {
		return
	}
	var config struct {
		Commands map[string]json.RawMessage `json:"commands"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		c.warnings = append(c.warnings, "cannot parse "+name+": "+err.Error())
		return
	}
	for command := range config.Commands {
		lower := strings.ToLower(command)
		if strings.Contains(lower, "mutation") || strings.Contains(lower, "mutant") {
			c.add("mutation", name+":commands."+command)
		}
	}
}

func (c *collector) inspectTextRecipes(name string) {
	data, err := os.ReadFile(filepath.Join(c.root, name))
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "mutants:") || strings.HasPrefix(line, "mutants-gate:") {
			c.add("mutation", name+":"+line)
		}
		if strings.HasPrefix(line, "clippy:") {
			c.add("lint", name+":"+line)
		}
	}
}

func (c *collector) walk(visit func(path, relative string)) {
	const maxFiles = 100000
	count := 0
	err := filepath.WalkDir(c.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != c.root && excludedDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		count++
		if count > maxFiles {
			return fmt.Errorf("file inventory exceeded %d files", maxFiles)
		}
		relative, err := filepath.Rel(c.root, path)
		if err != nil {
			return err
		}
		visit(path, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		c.warnings = append(c.warnings, "incomplete file inventory: "+err.Error())
		c.incomplete = true
	}
	for name := range c.evidence {
		sort.Strings(c.evidence[name])
	}
}

func excludedDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".venv", "venv", "target", "build", "dist", ".dart_tool", "coverage", ".gradle", ".next", ".svelte-kit", "vendor":
		return true
	default:
		return false
	}
}

func regular(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func dir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
