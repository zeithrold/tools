package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const ConfigName = "zt.json"

var Capabilities = []string{"unit", "integration", "property", "fuzz", "mutation", "lint", "typecheck", "e2e", "build", "css", "a11y"}

func ValidCapability(name string) bool {
	for _, candidate := range Capabilities {
		if name == candidate {
			return true
		}
	}
	return false
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type Config struct {
	SchemaVersion int               `json:"schemaVersion"`
	Modules       []Module          `json:"modules"`
	Skills        []string          `json:"skills,omitempty"`
	Lint          map[string]string `json:"lint,omitempty"`
}

type Module struct {
	ID        string              `json:"id"`
	Path      string              `json:"path"`
	Stack     string              `json:"stack"`
	Expect    map[string]string   `json:"expect,omitempty"`
	Profiles  map[string][]string `json:"profiles,omitempty"`
	Commands  map[string][]string `json:"commands,omitempty"`
	Artifacts []string            `json:"artifacts,omitempty"`
}

// Load returns nil when a project has not adopted zt yet. Inspect can still
// discover native manifests in that case.
func Load(root string) (*Config, error) {
	var config Config
	found, err := decode(root, &config)
	if err != nil || !found {
		return nil, err
	}
	if err := config.Validate(root); err != nil {
		return nil, err
	}
	return &config, nil
}

// globalConfig is the exact schema of the user-global selector: schemaVersion
// and skills, nothing else. Any other key, including an empty "modules" or
// "lint", is rejected by the strict decode below.
type globalConfig struct {
	SchemaVersion int      `json:"schemaVersion"`
	Skills        []string `json:"skills"`
}

// LoadSelection returns the Skill selection of the user-global config at
// dir/zt.json. found is false when that file does not exist. An existing config
// must select at least one Skill: zt never expands a selection by itself.
func LoadSelection(dir string) (skills []string, found bool, err error) {
	var config globalConfig
	found, err = decode(dir, &config)
	if err != nil || !found {
		return nil, found, err
	}
	if config.SchemaVersion != 1 {
		return nil, true, fmt.Errorf("%s: unsupported schemaVersion %d", ConfigName, config.SchemaVersion)
	}
	if err := ValidateSkills(config.Skills); err != nil {
		return nil, true, fmt.Errorf("%s: %w", ConfigName, err)
	}
	if len(config.Skills) == 0 {
		return nil, true, fmt.Errorf("%s: \"skills\" selects no Skill", ConfigName)
	}
	return config.Skills, true, nil
}

// decode reads dir/zt.json strictly into target and reports whether it exists.
func decode(dir string, target any) (bool, error) {
	f, err := os.Open(filepath.Join(dir, ConfigName))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return true, fmt.Errorf("%s: %w", ConfigName, err)
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return true, fmt.Errorf("%s: trailing JSON data", ConfigName)
	}
	return true, nil
}

func (c Config) Validate(root string) error {
	if c.SchemaVersion != 1 {
		return fmt.Errorf("%s: unsupported schemaVersion %d", ConfigName, c.SchemaVersion)
	}
	if err := ValidateSkills(c.Skills); err != nil {
		return fmt.Errorf("%s: %w", ConfigName, err)
	}
	seenModules := make(map[string]bool)
	for _, module := range c.Modules {
		if !identifier.MatchString(module.ID) || seenModules[module.ID] {
			return fmt.Errorf("%s: invalid or duplicate module id %q", ConfigName, module.ID)
		}
		seenModules[module.ID] = true
		if !ValidStack(module.Stack) {
			return fmt.Errorf("%s: unsupported stack %q", ConfigName, module.Stack)
		}
		if _, err := ModulePath(root, module.Path); err != nil {
			return fmt.Errorf("%s: module %s: %w", ConfigName, module.ID, err)
		}
		for capability, level := range module.Expect {
			if !ValidCapability(capability) || (level != "required" && level != "warn" && level != "off") {
				return fmt.Errorf("%s: module %s: invalid expectation %q=%q", ConfigName, module.ID, capability, level)
			}
		}
		for profile, capabilities := range module.Profiles {
			if !identifier.MatchString(profile) {
				return fmt.Errorf("%s: module %s: invalid profile %q", ConfigName, module.ID, profile)
			}
			if len(capabilities) == 0 {
				return fmt.Errorf("empty profile %q", profile)
			}
			seen := map[string]bool{}
			for _, capability := range capabilities {
				if !ValidCapability(capability) || seen[capability] {
					return fmt.Errorf("%s: module %s: invalid capability %q", ConfigName, module.ID, capability)
				}
				seen[capability] = true
			}
		}
		for capability, argv := range module.Commands {
			if !ValidCapability(capability) || len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
				return fmt.Errorf("invalid command for %q", capability)
			}
			for _, arg := range argv {
				if strings.ContainsRune(arg, 0) {
					return fmt.Errorf("command contains NUL")
				}
				key, value, hasValue := strings.Cut(arg, "=")
				key = strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(key))
				if strings.HasPrefix(arg, "--") && key == "config.verifydepsbeforerun" && (!hasValue || value != "error") {
					return fmt.Errorf("native checks must not override pnpm dependency readiness with %q; use --config.verify-deps-before-run=error", arg)
				}
			}
		}
		for _, artifact := range module.Artifacts {
			clean := filepath.Clean(artifact)
			if artifact == "" || clean == "." || filepath.IsAbs(artifact) || strings.Contains(artifact, "\\") || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
				return fmt.Errorf("invalid artifact path %q", artifact)
			}
		}
	}
	return nil
}

// ValidateSkills rejects unknown or duplicated Skill ids.
func ValidateSkills(skills []string) error {
	seen := make(map[string]bool, len(skills))
	for _, skill := range skills {
		if !identifier.MatchString(skill) || seen[skill] {
			return fmt.Errorf("invalid or duplicate Skill id %q", skill)
		}
		seen[skill] = true
	}
	return nil
}

func ValidStack(stack string) bool {
	switch stack {
	case "go", "jvm", "js-ts", "flutter", "rust", "python":
		return true
	default:
		return false
	}
}

// ModulePath validates a project-relative path and returns its absolute path.
func ModulePath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, "\\") {
		return "", fmt.Errorf("invalid relative path %q", relative)
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root: %q", relative)
	}
	path := filepath.Join(root, clean)
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("module path escapes project root: %q", relative)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("module path is not a directory: %q", relative)
	}
	return path, nil
}
