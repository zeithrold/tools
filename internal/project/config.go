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

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type Config struct {
	SchemaVersion int               `json:"schemaVersion"`
	Modules       []Module          `json:"modules"`
	Skills        []string          `json:"skills,omitempty"`
	Lint          map[string]string `json:"lint,omitempty"`
}

type Module struct {
	ID       string              `json:"id"`
	Path     string              `json:"path"`
	Stack    string              `json:"stack"`
	Expect   map[string]string   `json:"expect,omitempty"`
	Profiles map[string][]string `json:"profiles,omitempty"`
}

// Load returns nil when a project has not adopted zt yet. Inspect can still
// discover native manifests in that case; sync requires an explicit config.
func Load(root string) (*Config, error) {
	f, err := os.Open(filepath.Join(root, ConfigName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	var config Config
	if err := dec.Decode(&config); err != nil {
		return nil, fmt.Errorf("%s: %w", ConfigName, err)
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: trailing JSON data", ConfigName)
	}
	if err := config.Validate(root); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c Config) Validate(root string) error {
	if c.SchemaVersion != 1 {
		return fmt.Errorf("%s: unsupported schemaVersion %d", ConfigName, c.SchemaVersion)
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
			if !identifier.MatchString(capability) || (level != "required" && level != "warn" && level != "off") {
				return fmt.Errorf("%s: module %s: invalid expectation %q=%q", ConfigName, module.ID, capability, level)
			}
		}
		for profile, capabilities := range module.Profiles {
			if !identifier.MatchString(profile) {
				return fmt.Errorf("%s: module %s: invalid profile %q", ConfigName, module.ID, profile)
			}
			for _, capability := range capabilities {
				if !identifier.MatchString(capability) {
					return fmt.Errorf("%s: module %s: invalid capability %q", ConfigName, module.ID, capability)
				}
			}
		}
	}
	seenSkills := make(map[string]bool)
	for _, skill := range c.Skills {
		if !identifier.MatchString(skill) || seenSkills[skill] {
			return fmt.Errorf("%s: invalid or duplicate skill %q", ConfigName, skill)
		}
		seenSkills[skill] = true
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
