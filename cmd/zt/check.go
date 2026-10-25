package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/zeithrold/tools/internal/check"
)

func newCheckCommand(a *app) *cobra.Command {
	var rootDir string
	var module string
	var profile string
	var artifacts string
	var timeout time.Duration
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run an explicit native verification profile and record evidence",
		Long:  "check runs the native commands of one explicit module/profile and retains logs and a report under the artifact directory. It never installs tools or expands the profile. Ctrl+C cancels running commands.",
		Args:  noPositionalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := check.Run(cmd.Context(), check.Options{Root: rootDir, Module: module, Profile: profile, Output: artifacts, Timeout: timeout})
			if err != nil {
				return err
			}
			if asJSON {
				if err := writeJSON(a.stdout, report); err != nil {
					return err
				}
			} else {
				for _, execution := range report.Checks {
					if _, err := fmt.Fprintf(a.stdout, "%s: %s (%s)\n", execution.Capability, execution.Status, execution.Expectation); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintf(a.stdout, "artifacts: %s\n", report.ArtifactDir); err != nil {
					return err
				}
				for _, warning := range report.Warnings {
					if _, err := fmt.Fprintln(a.stdout, "warning:", warning); err != nil {
						return err
					}
				}
			}
			if report.Status != "passed" {
				return fmt.Errorf("check failed; see %s", filepath.Join(report.ArtifactDir, "report.json"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&rootDir, "root", ".", "project directory")
	cmd.Flags().StringVar(&module, "module", "root", "module id")
	cmd.Flags().StringVar(&profile, "profile", "frontend", "explicit zt.json profile")
	cmd.Flags().StringVar(&artifacts, "artifacts", "", "artifact parent directory; defaults to .zt/artifacts")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "per-command timeout")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable execution report")
	return cmd
}
