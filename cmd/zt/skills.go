package main

import (
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/spf13/cobra"

	tools "github.com/zeithrold/tools"
)

func newSkillsCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Inspect and distribute bundled Skills in the user-global tree",
		Args:  cobra.ArbitraryArgs,
		RunE:  unknownSubcommand,
	}
	cmd.AddCommand(newSkillsListCommand(a), newSkillsSyncCommand(a))
	return cmd
}

func newSkillsSyncCommand(a *app) *cobra.Command {
	o := &syncOptions{}
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Distribute selected bundled Skills to ~/.agents/skills",
		Long:  "sync copies complete, versioned Skill directories into ~/.agents/skills and records managed digests in ~/.agents/zt.lock.json. It writes only the user-global tree, never project .agents/skills, and never removes an unselected Skill automatically.",
		Args:  noPositionalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(a, cmd, o)
		},
	}
	addSyncFlags(cmd, o)
	return cmd
}

type skillsListReport struct {
	SchemaVersion int      `json:"schemaVersion"`
	Skills        []string `json:"skills"`
}

func newSkillsListCommand(a *app) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Skills embedded in this zt build",
		Long:  "list reports the Skill ids shipped inside the zt binary. It reads no configuration, home directory, or network.",
		Args:  noPositionalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := bundledSkillIDs()
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(a.stdout, skillsListReport{SchemaVersion: 1, Skills: names})
			}
			for _, name := range names {
				if _, err := fmt.Fprintln(a.stdout, name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable list")
	return cmd
}

// bundledSkillIDs enumerates complete Skill directories in the embedded tree.
// A directory without SKILL.md is not a Skill and is skipped.
func bundledSkillIDs() ([]string, error) {
	entries, err := fs.ReadDir(tools.SkillFiles, "skills")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := fs.Stat(tools.SkillFiles, path.Join("skills", entry.Name(), "SKILL.md")); err != nil {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}
