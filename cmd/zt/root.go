package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

// newRootCommand assembles the command tree. It performs no I/O and installs no
// PersistentPreRun hooks, so Cobra itself never reads credentials or logs in.
func newRootCommand(a *app) *cobra.Command {
	root := &cobra.Command{
		Use:           "zt",
		Short:         "Project-native verification and global Skill distribution",
		Long:          "zt inspects project configuration, plans native verification commands, runs explicit checks, and distributes bundled Skills to the user-global tree.\n\ninspect, plan and sync --plan are read-only: they do not run tests, install dependencies, read credentials, or access the network.",
		SilenceErrors: true,
		SilenceUsage:  true,
		// ArbitraryArgs lets RunE produce the unknown-command error while all
		// real subcommands still resolve normally.
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown command %q; run 'zt --help' for available commands", args[0])
			}
			return fmt.Errorf("no command given; run 'zt --help' for available commands")
		},
	}
	root.PersistentFlags().BoolVar(&a.noInput, "no-input", false, "never read interactive input; fail instead of prompting")
	root.PersistentFlags().BoolVar(&a.noColor, "no-color", false, "disable ANSI color in human-readable output")

	root.AddCommand(
		newInspectCommand(a),
		newPlanCommand(a),
		newCheckCommand(a),
		newSkillsCommand(a),
		newSyncCommand(a),
		newAuthCommand(a),
		newVersionCommand(a),
		newLicenseCommand(a),
	)
	return root
}

// executeApp runs one command invocation against explicit streams. Tests use
// it directly so no test depends on os.Args or the real user's home directory.
func executeApp(a *app, ctx context.Context, args []string) error {
	if err := rejectTokenLikeArguments(args); err != nil {
		return err
	}
	root := newRootCommand(a)
	root.SetIn(a.stdin)
	root.SetOut(a.stdout)
	root.SetErr(a.stderr)
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}

// tokenLikeArgument matches the public prefixes GitHub uses for credentials.
// zt never accepts a token as an argument; this guard stops an accidental one
// from reaching Cobra error paths that would echo it. It covers even short
// prefixes because a pasted token can be truncated or fake-shaped.
var tokenLikeArgument = regexp.MustCompile(`(?i)(gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+)`)

func rejectTokenLikeArguments(args []string) error {
	for _, arg := range args {
		if tokenLikeArgument.MatchString(arg) {
			return fmt.Errorf("an argument contains something that looks like a GitHub token; remove it (zt never accepts tokens as arguments)")
		}
	}
	return nil
}

// noArgs keeps the historical "accepts no arguments" error without Cobra's
// wrapping, so main prints exactly one error line.
func noArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("%s accepts no arguments", cmd.Name())
	}
	return nil
}

// noPositionalArgs preserves the historical "unexpected arguments" error for
// commands that never accepted positional arguments.
func noPositionalArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("%s: unexpected arguments %v", cmd.Name(), args)
	}
	return nil
}

// unknownSubcommand makes a command group reject unmatched subcommands instead
// of Cobra's non-runnable help fallback, which would exit successfully. An
// empty invocation still prints help and succeeds.
func unknownSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	return cmd.Help()
}

// noEchoArgs rejects positional arguments without echoing their values. Auth
// commands use it because a pasted token can arrive as a stray positional.
func noEchoArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("%s: unexpected arguments", cmd.Name())
	}
	return nil
}

// authUnknownSubcommand is the auth-group fallback. It never echoes the
// unmatched input, which may be a pasted token.
func authUnknownSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command for %q; run '%s --help' for available commands", cmd.CommandPath(), cmd.CommandPath())
	}
	return cmd.Help()
}

// sanitizeAuthFlagError hides unknown flag text for auth commands, where a
// pasted token can appear in the flag name. It is inherited by auth leaves.
func sanitizeAuthFlagError(cmd *cobra.Command, err error) error {
	return fmt.Errorf("invalid flag for %s; run '%s --help' for available flags", cmd.CommandPath(), cmd.CommandPath())
}
