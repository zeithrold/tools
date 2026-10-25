package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zeithrold/tools/internal/plan"
)

func newPlanCommand(a *app) *cobra.Command {
	var rootDir string
	var module string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "plan CAPABILITY",
		Short: "Print the native commands for one capability without executing them",
		Long:  "plan resolves one capability for one module and prints the argv it would run. It executes nothing, creates no directory, reads no credential, and accesses no network.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || strings.HasPrefix(args[0], "-") {
				return fmt.Errorf("plan requires a capability name")
			}
			if len(args) > 1 {
				return fmt.Errorf("plan: unexpected arguments %v", args[1:])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := plan.Build(rootDir, module, args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(a.stdout, result)
			}
			return renderPlan(a.stdout, result)
		},
	}
	cmd.Flags().StringVar(&rootDir, "root", ".", "project directory")
	cmd.Flags().StringVar(&module, "module", "root", "module id")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable plan")
	return cmd
}

func renderPlan(output io.Writer, result plan.Result) error {
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
}
