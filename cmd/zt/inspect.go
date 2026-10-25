package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/zeithrold/tools/internal/inspect"
)

func newInspectCommand(a *app) *cobra.Command {
	var rootDir string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Report configured modules and detected capabilities without running anything",
		Long:  "inspect reads zt.json and native manifests, then reports discovered capabilities. It never runs commands, installs dependencies, reads credentials, or accesses the network.",
		Args:  noPositionalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := inspect.Run(rootDir)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(a.stdout, report)
			}
			return renderInspect(a.stdout, report)
		},
	}
	cmd.Flags().StringVar(&rootDir, "root", ".", "project directory")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable report")
	return cmd
}

func renderInspect(output io.Writer, report inspect.Report) error {
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
}

// writeJSON writes one JSON document followed by a newline. JSON is the only
// thing on stdout in this mode, so machine consumers never parse progress text.
func writeJSON(output io.Writer, value any) error {
	return json.NewEncoder(output).Encode(value)
}
