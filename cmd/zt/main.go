package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tools "github.com/zeithrold/tools"
	"github.com/zeithrold/tools/internal/inspect"
	"github.com/zeithrold/tools/internal/plan"
	"github.com/zeithrold/tools/internal/project"
	"github.com/zeithrold/tools/internal/skills"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "zt:", err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: zt inspect [--root DIR] [--json] | zt plan CAPABILITY [--module ID] [--root DIR] [--json] | zt sync [--root DIR] [--plan] [--json] | zt version")
	}
	switch args[0] {
	case "version":
		if len(args) != 1 {
			return fmt.Errorf("version accepts no arguments")
		}
		_, err := fmt.Fprintln(output, tools.BundleVersion)
		return err
	case "inspect":
		flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		root := flags.String("root", ".", "project directory")
		asJSON := flags.Bool("json", false, "machine-readable report")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("inspect: unexpected arguments %v", flags.Args())
		}
		report, err := inspect.Run(*root)
		if err != nil {
			return err
		}
		if *asJSON {
			return json.NewEncoder(output).Encode(report)
		}
		for _, module := range report.Modules {
			if _, err := fmt.Fprintf(output, "%s (%s, %s)\n", module.ID, module.Stack, module.Path); err != nil {
				return err
			}
			for _, capability := range module.Capabilities {
				if capability.Detection == "not_detected" && capability.Expectation == "" {
					continue
				}
				note := ""
				if capability.Expectation != "" {
					note = ", expect=" + capability.Expectation
				}
				if _, err := fmt.Fprintf(output, "  %-12s %s%s; run=%s\n", capability.Name, capability.Detection, note, capability.LastRun); err != nil {
					return err
				}
				for _, evidence := range capability.Evidence {
					if _, err := fmt.Fprintf(output, "    - %s\n", evidence); err != nil {
						return err
					}
				}
				if len(capability.Next) > 0 {
					encoded, err := json.Marshal(capability.Next)
					if err != nil {
						return err
					}
					if _, err := fmt.Fprintf(output, "    next argv: %s\n", encoded); err != nil {
						return err
					}
				}
			}
		}
		if len(report.Modules) == 0 {
			if _, err := fmt.Fprintln(output, "No supported stack manifest found at the project root. Add zt.json to declare modules."); err != nil {
				return err
			}
		}
		for _, warning := range report.Warnings {
			if _, err := fmt.Fprintln(output, "warning:", warning); err != nil {
				return err
			}
		}
		return nil
	case "plan":
		if len(args) < 2 || strings.HasPrefix(args[1], "-") {
			return fmt.Errorf("plan requires a capability name")
		}
		capability := args[1]
		flags := flag.NewFlagSet("plan", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		root := flags.String("root", ".", "project directory")
		module := flags.String("module", "root", "module id")
		asJSON := flags.Bool("json", false, "machine-readable plan")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("plan: unexpected arguments %v", flags.Args())
		}
		result, err := plan.Build(*root, *module, capability)
		if err != nil {
			return err
		}
		if *asJSON {
			return json.NewEncoder(output).Encode(result)
		}
		for _, step := range result.Steps {
			if _, err := fmt.Fprintf(output, "in %s: %s\n  %s\n", step.Dir, strings.Join(step.Argv, " "), step.Reason); err != nil {
				return err
			}
		}
		for _, warning := range result.Warnings {
			if _, err := fmt.Fprintln(output, "warning:", warning); err != nil {
				return err
			}
		}
		return nil
	case "sync":
		flags := flag.NewFlagSet("sync", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		root := flags.String("root", ".", "project directory")
		planOnly := flags.Bool("plan", false, "show changes without writing")
		asJSON := flags.Bool("json", false, "machine-readable plan")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("sync: unexpected arguments %v", flags.Args())
		}
		config, err := project.Load(*root)
		if err != nil {
			return err
		}
		var result skills.PlanResult
		if *planOnly {
			result, err = skills.Plan(*root, config)
		} else {
			result, err = skills.Apply(*root, config)
		}
		if err != nil {
			return err
		}
		if *asJSON {
			return json.NewEncoder(output).Encode(result)
		}
		for _, action := range result.Actions {
			line := fmt.Sprintf("%s: %s", action.Action, action.Target)
			if action.Reason != "" {
				line += " (" + action.Reason + ")"
			}
			if _, err := fmt.Fprintln(output, line); err != nil {
				return err
			}
		}
		for _, warning := range result.Warnings {
			if _, err := fmt.Fprintln(output, "warning:", warning); err != nil {
				return err
			}
		}
		if len(result.Actions) == 0 && len(result.Warnings) == 0 {
			_, err := fmt.Fprintln(output, "No shared skills selected.")
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown command %q; use inspect, plan, sync, or version", strings.TrimSpace(args[0]))
	}
}
