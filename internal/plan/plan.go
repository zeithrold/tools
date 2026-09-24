package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zeithrold/tools/internal/inspect"
)

type Step struct {
	Argv   []string `json:"argv"`
	Dir    string   `json:"dir"`
	Reason string   `json:"reason"`
}

type Result struct {
	SchemaVersion int      `json:"schemaVersion"`
	Module        string   `json:"module"`
	Capability    string   `json:"capability"`
	Detection     string   `json:"detection"`
	Steps         []Step   `json:"steps"`
	Warnings      []string `json:"warnings,omitempty"`
}

// Build prepares native commands without executing them or probing the host.
func Build(root, moduleID, capability string) (Result, error) {
	report, err := inspect.Run(root)
	if err != nil {
		return Result{}, err
	}
	var module *inspect.Module
	for i := range report.Modules {
		if report.Modules[i].ID == moduleID {
			module = &report.Modules[i]
			break
		}
	}
	if module == nil {
		return Result{}, fmt.Errorf("unknown module %q", moduleID)
	}
	var found *inspect.Capability
	for i := range module.Capabilities {
		if module.Capabilities[i].Name == capability {
			found = &module.Capabilities[i]
			break
		}
	}
	if found == nil {
		return Result{}, fmt.Errorf("unknown capability %q", capability)
	}
	result := Result{SchemaVersion: 1, Module: moduleID, Capability: capability, Detection: found.Detection, Steps: []Step{}}
	if found.Detection != "discovered" {
		result.Warnings = append(result.Warnings, "no confirmed entry point; inspect project configuration before execution")
		return result, nil
	}
	moduleRoot := filepath.Join(report.Root, filepath.FromSlash(module.Path))
	switch module.Stack {
	case "go":
		result.planGo(moduleRoot, capability, *found)
	case "js-ts":
		result.planJSTS(moduleRoot, capability)
	default:
		result.Warnings = append(result.Warnings, "native command planning is not implemented for "+module.Stack+" yet")
	}
	if len(result.Steps) == 0 && len(result.Warnings) == 0 {
		result.Warnings = append(result.Warnings, "entry point found, but no reliable command could be planned")
	}
	return result, nil
}

func (r *Result) planGo(root, capability string, found inspect.Capability) {
	addJust := func(recipe string) bool {
		if !hasRecipe(filepath.Join(root, "justfile"), recipe) {
			return false
		}
		r.Steps = append(r.Steps, Step{Argv: []string{"just", recipe}, Dir: root, Reason: "existing project recipe"})
		return true
	}
	switch capability {
	case "unit":
		if !addJust("test-unit") && !addJust("test") {
			if info, err := os.Stat(filepath.Join(root, "tests", "integration")); err == nil && info.IsDir() {
				r.Warnings = append(r.Warnings, "integration tests exist and no unit-only recipe was identified; choose package scope manually")
			} else {
				r.Steps = append(r.Steps, Step{Argv: []string{"go", "test", "./..."}, Dir: root, Reason: "standard Go test command; inspect environment requirements"})
			}
		}
	case "integration":
		if !addJust("test-integration") {
			r.Warnings = append(r.Warnings, "integration tests found, but no project recipe was identified; inspect tags, services, and setup")
		}
	case "fuzz":
		if addJust("fuzz") {
			r.Warnings = append(r.Warnings, "inspect the existing fuzz recipe's time budget before execution")
			return
		}
		for _, evidence := range found.Evidence {
			file, name, ok := strings.Cut(evidence, ":")
			if !ok || !strings.HasPrefix(name, "Fuzz") {
				continue
			}
			directory := path.Dir(file)
			packagePath := "."
			if directory != "." {
				packagePath = "./" + directory
			}
			r.Steps = append(r.Steps, Step{
				Argv: []string{"go", "test", packagePath, "-fuzz=^" + regexp.QuoteMeta(name) + "$", "-fuzztime=30s"},
				Dir:  root, Reason: "bounded Go fuzz search; confirm target preconditions",
			})
		}
	case "mutation":
		for _, recipe := range []string{"mutation-accounting", "mutation", "mutants"} {
			if addJust(recipe) {
				r.Warnings = append(r.Warnings, "mutation testing may be expensive; inspect the existing recipe's scope and time budget")
				return
			}
		}
		r.Warnings = append(r.Warnings, "mutation command declared, but no safe project recipe was identified")
	case "lint":
		if !addJust("lint") {
			r.Warnings = append(r.Warnings, "lint configuration found; inspect the project's pinned lint runner")
		}
	}
}

func (r *Result) planJSTS(root, capability string) {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		r.Warnings = append(r.Warnings, "cannot read package.json: "+err.Error())
		return
	}
	var pkg struct {
		PackageManager string            `json:"packageManager"`
		Scripts        map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		r.Warnings = append(r.Warnings, "cannot parse package.json: "+err.Error())
		return
	}
	manager := "npm"
	if name, _, ok := strings.Cut(pkg.PackageManager, "@"); ok && (name == "pnpm" || name == "yarn" || name == "npm") {
		manager = name
	}
	var candidates []string
	switch capability {
	case "unit":
		candidates = []string{"test:unit", "test-unit", "test"}
	case "lint":
		candidates = []string{"lint", "lint:check"}
	case "typecheck":
		candidates = []string{"typecheck", "check:types", "check-types"}
	case "e2e":
		candidates = []string{"test:e2e", "e2e:test", "e2e"}
	default:
		r.Warnings = append(r.Warnings, "inspect the declared package script and its runtime semantics")
		return
	}
	for _, name := range candidates {
		if _, ok := pkg.Scripts[name]; !ok {
			continue
		}
		argv := []string{manager, name}
		if manager == "npm" {
			argv = []string{"npm", "run", name}
		}
		r.Steps = append(r.Steps, Step{Argv: argv, Dir: root, Reason: "declared package script: " + pkg.Scripts[name] + "; inspect service and browser preconditions"})
		return
	}
}

func hasRecipe(filename, recipe string) bool {
	data, err := os.ReadFile(filename)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line != strings.TrimLeft(line, " \t") {
			continue
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, recipe+":") || strings.HasPrefix(line, recipe+" ") {
			return true
		}
	}
	return false
}
