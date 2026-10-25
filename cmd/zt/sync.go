package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zeithrold/tools/internal/project"
	"github.com/zeithrold/tools/internal/skills"
)

// syncOptions holds the flags shared by `zt skills sync` and its legacy `zt
// sync` alias. Both write only the user-global tree; neither reads the project
// zt.json.
type syncOptions struct {
	root   string
	global bool
	skills string
	plan   bool
	json   bool
}

func addSyncFlags(cmd *cobra.Command, o *syncOptions) {
	cmd.Flags().StringVar(&o.root, "root", "", "unsupported project directory")
	// --global restates the default so the destination is explicit on the
	// command line. There is no project scope to select.
	cmd.Flags().BoolVar(&o.global, "global", false, "synchronize the user-global Skill tree (default)")
	cmd.Flags().StringVar(&o.skills, "skills", "", "comma-separated Skill ids for this run")
	cmd.Flags().BoolVar(&o.plan, "plan", false, "show changes without writing")
	cmd.Flags().BoolVar(&o.json, "json", false, "machine-readable plan")
}

// newSyncCommand keeps the historical `zt sync` spelling as an alias. The flag
// set is identical to `zt skills sync` so both paths validate the same way.
func newSyncCommand(a *app) *cobra.Command {
	o := &syncOptions{}
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Alias for 'zt skills sync' (user-global distribution)",
		Long:  "sync is kept for existing automation and behaves exactly like 'zt skills sync'.",
		Args:  noPositionalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(a, cmd, o)
		},
	}
	addSyncFlags(cmd, o)
	return cmd
}

func runSync(a *app, cmd *cobra.Command, o *syncOptions) error {
	// Both command spellings reject --root by presence, including --root=,
	// so old project-scoped automation cannot silently become a global write.
	if cmd.Flags().Changed("root") {
		return fmt.Errorf("sync --root is no longer supported: project Skill sync was removed, so zt never writes project .agents/skills or a project zt.lock.json; run zt skills sync (config ~/.agents/zt.json) or zt skills sync --skills ID[,ID] for user-global distribution, and migrate project Skills by hand")
	}
	home, err := a.homeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	target := skills.GlobalTarget(home)
	// The global config sits next to the lock and the skills it selects.
	selected, err := syncSelection(cmd, o, filepath.Dir(target.LockPath))
	if err != nil {
		return err
	}
	var result skills.PlanResult
	if o.plan {
		result, err = skills.Plan(target, selected)
	} else {
		result, err = skills.Apply(target, selected)
	}
	if err != nil {
		return err
	}
	if o.json {
		return writeJSON(a.stdout, result)
	}
	for _, action := range result.Actions {
		line := fmt.Sprintf("%s: %s", action.Action, action.Target)
		if action.Reason != "" {
			line += " (" + action.Reason + ")"
		}
		if _, err := fmt.Fprintln(a.stdout, line); err != nil {
			return err
		}
	}
	for _, warning := range result.Warnings {
		if _, err := fmt.Fprintln(a.stdout, "warning:", warning); err != nil {
			return err
		}
	}
	return nil
}

// syncSelection resolves the Skills to synchronize. An explicit --skills list
// wins for this run and leaves the global config untouched. An explicitly
// empty --skills is an error rather than a silent fallback to the config, so a
// caller cannot accidentally distribute a different selection. Without either
// source, zt selects nothing and explains how to configure a selection.
func syncSelection(cmd *cobra.Command, o *syncOptions, agentsDir string) ([]string, error) {
	configPath := filepath.Join(agentsDir, project.ConfigName)
	if cmd.Flags().Changed("skills") {
		if strings.TrimSpace(o.skills) == "" {
			return nil, fmt.Errorf("sync --skills requires at least one Skill id; it does not fall back to %s", configPath)
		}
		names := strings.Split(o.skills, ",")
		for i, name := range names {
			names[i] = strings.TrimSpace(name)
		}
		if err := project.ValidateSkills(names); err != nil {
			return nil, fmt.Errorf("sync --skills: %w", err)
		}
		return names, nil
	}
	names, found, err := project.LoadSelection(agentsDir)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("sync: no Skills selected: %s does not exist; create it with {\"schemaVersion\": 1, \"skills\": [\"ui-foundation\"]} or pass --skills ID[,ID]", configPath)
	}
	return names, nil
}
