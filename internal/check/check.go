// Package check executes reviewed native plans and records execution evidence.
package check

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tools "github.com/zeithrold/tools"
	"github.com/zeithrold/tools/internal/plan"
	"github.com/zeithrold/tools/internal/project"
)

type Options struct {
	Root, Module, Profile, Output string
	Timeout                       time.Duration // Per native command; includes child processes on Unix.
}

type Execution struct {
	Capability  string `json:"capability"`
	Expectation string `json:"expectation"`
	Status      string `json:"status"` // passed, failed, blocked, not_run, off
	Steps       []Step `json:"steps"`
	Detail      string `json:"detail,omitempty"`
}

type Step struct {
	Argv       []string `json:"argv"`
	Dir        string   `json:"dir"`
	Status     string   `json:"status"`
	ExitCode   int      `json:"exitCode"`
	DurationMS int64    `json:"durationMs"`
	Log        string   `json:"log"`
	Detail     string   `json:"detail,omitempty"`
}

type Report struct {
	SchemaVersion int         `json:"schemaVersion"`
	ToolsVersion  string      `json:"toolsVersion"`
	Module        string      `json:"module"`
	Profile       string      `json:"profile"`
	StartedAt     string      `json:"startedAt"`
	Status        string      `json:"status"`
	ArtifactDir   string      `json:"artifactDir"`
	Checks        []Execution `json:"checks"`
	Artifacts     []string    `json:"artifacts"`
	Warnings      []string    `json:"warnings,omitempty"`
}

// Run requires an explicit module/profile. It never installs tools, changes
// policy, invokes a shell for argv, or interprets script names as test coverage.
func Run(ctx context.Context, options Options) (Report, error) {
	root, err := filepath.Abs(options.Root)
	if err != nil {
		return Report{}, err
	}
	config, err := project.Load(root)
	if err != nil {
		return Report{}, err
	}
	if config == nil {
		return Report{}, fmt.Errorf("check requires zt.json with an explicit profile")
	}
	var module *project.Module
	for i := range config.Modules {
		if config.Modules[i].ID == options.Module {
			module = &config.Modules[i]
			break
		}
	}
	if module == nil {
		return Report{}, fmt.Errorf("unknown module %q", options.Module)
	}
	capabilities, ok := module.Profiles[options.Profile]
	if !ok {
		return Report{}, fmt.Errorf("unknown profile %q for module %q", options.Profile, options.Module)
	}
	if options.Timeout <= 0 {
		return Report{}, fmt.Errorf("timeout must be positive")
	}
	moduleRoot, err := project.ModulePath(root, module.Path)
	if err != nil {
		return Report{}, err
	}
	// Plan everything before executing; configuration errors cannot cause a partial run.
	plans := make([]plan.Result, len(capabilities))
	for i, capability := range capabilities {
		plans[i], err = plan.Build(root, module.ID, capability)
		if err != nil {
			return Report{}, err
		}
	}
	output := options.Output
	if output == "" {
		output = filepath.Join(root, ".zt", "artifacts")
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return Report{}, err
	}
	if err := noSymlinks(output); err != nil {
		return Report{}, err
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		return Report{}, err
	}
	directory, err := os.MkdirTemp(output, "check-")
	if err != nil {
		return Report{}, err
	}
	if err := os.Mkdir(filepath.Join(directory, "logs"), 0755); err != nil {
		return Report{}, err
	}
	report := Report{SchemaVersion: 1, ToolsVersion: tools.Version(), Module: module.ID, Profile: options.Profile, StartedAt: time.Now().UTC().Format(time.RFC3339Nano), Status: "passed", ArtifactDir: directory, Checks: []Execution{}, Artifacts: []string{}}
	stopped := false
	for i, capability := range capabilities {
		level := module.Expect[capability]
		if level == "" {
			level = "required"
		}
		execution := Execution{Capability: capability, Expectation: level, Status: "not_run", Steps: []Step{}}
		switch {
		case level == "off":
			execution.Status = "off"
		case stopped:
			execution.Detail = "earlier required check failed or was blocked"
		case len(plans[i].Steps) == 0:
			execution.Status = "blocked"
			execution.Detail = strings.Join(plans[i].Warnings, "; ")
		default:
			execution.Status = "passed"
			for j, planned := range plans[i].Steps {
				log := filepath.Join("logs", fmt.Sprintf("%02d-%s-%02d.log", i, capability, j))
				step := execute(ctx, planned, options.Timeout, directory, log)
				execution.Steps = append(execution.Steps, step)
				report.Artifacts = append(report.Artifacts, filepath.ToSlash(log))
				if step.Status != "passed" {
					execution.Status = step.Status
					break
				}
			}
		}
		if execution.Status == "failed" || execution.Status == "blocked" {
			if level == "required" {
				report.Status = "failed"
				stopped = true
			} else {
				report.Warnings = append(report.Warnings, capability+": "+execution.Status)
			}
		}
		report.Checks = append(report.Checks, execution)
	}
	for _, artifact := range module.Artifacts {
		files, err := collect(moduleRoot, directory, artifact)
		if err != nil {
			report.Status = "failed"
			report.Warnings = append(report.Warnings, err.Error())
		}
		report.Artifacts = append(report.Artifacts, files...)
	}
	// Include native files written through ZT_ARTIFACTS_DIR, not only copies/logs.
	report.Artifacts = []string{}
	err = filepath.WalkDir(directory, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("native artifact is a symlink: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("native artifact is not a regular file: %s", path)
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		report.Artifacts = append(report.Artifacts, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		report.Status = "failed"
		report.Warnings = append(report.Warnings, err.Error())
	}
	report.Artifacts = append(report.Artifacts, "report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return report, err
	}
	if err := os.WriteFile(filepath.Join(directory, "report.json"), append(data, '\n'), 0644); err != nil {
		return report, err
	}
	return report, nil
}

func execute(ctx context.Context, planned plan.Step, timeout time.Duration, directory, relativeLog string) Step {
	step := Step{Argv: planned.Argv, Dir: planned.Dir, Status: "blocked", ExitCode: -1, Log: filepath.ToSlash(relativeLog)}
	log, err := os.OpenFile(filepath.Join(directory, relativeLog), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		step.Detail = err.Error()
		return step
	}
	defer log.Close()
	childContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	command := exec.CommandContext(childContext, planned.Argv[0], planned.Argv[1:]...)
	command.Dir = planned.Dir
	command.Env = append(os.Environ(), "ZT_ARTIFACTS_DIR="+directory, "pnpm_config_verify_deps_before_run=error")
	command.Stdout, command.Stderr = log, log
	command.WaitDelay = 2 * time.Second
	configureProcess(command)
	start := time.Now()
	err = command.Run()
	step.DurationMS = time.Since(start).Milliseconds()
	if command.ProcessState != nil {
		step.ExitCode = command.ProcessState.ExitCode()
	}
	if err == nil {
		step.Status = "passed"
	} else {
		if command.ProcessState != nil {
			step.Status = "failed"
		}
		step.Detail = err.Error()
		if childContext.Err() != nil {
			step.Status = "failed"
			step.Detail = childContext.Err().Error()
		}
	}
	return step
}

func noSymlinks(path string) error {
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink artifact path %s", current)
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if filepath.Dir(current) == current {
			return nil
		}
	}
}

func collect(root, output, relative string) ([]string, error) {
	source := filepath.Join(root, filepath.FromSlash(relative))
	if err := noSymlinks(source); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(source); os.IsNotExist(err) {
		return nil, fmt.Errorf("declared artifact missing: %s", relative)
	} else if err != nil {
		return nil, err
	}
	// Do not recursively collect this run or older check outputs.
	rel, err := filepath.Rel(source, output)
	if err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return nil, fmt.Errorf("artifact source contains check output: %s", relative)
	}
	files := []string{}
	var bytes int64
	err = filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink artifact %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("artifact is not a regular file: %s", path)
		}
		bytes += info.Size()
		if bytes > 100<<20 || len(files) >= 10000 {
			return fmt.Errorf("artifact collection exceeds 100 MiB or 10000 files")
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		target := filepath.Join(output, "collected", relativePath)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		destination, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(destination, input)
		closeErr := destination.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		files = append(files, filepath.ToSlash(filepath.Join("collected", relativePath)))
		return nil
	})
	return files, err
}
