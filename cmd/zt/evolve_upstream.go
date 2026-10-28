package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zeithrold/tools/internal/evolve"
	"github.com/zeithrold/tools/internal/githubauth"
)

func newEvolveUpstreamCommand(a *app) *cobra.Command {
	group := &cobra.Command{
		Use: "upstream", Short: "Fetch or inspect a managed complete pinned upstream baseline",
		Args: cobra.ArbitraryArgs, RunE: unknownSubcommand,
	}
	group.AddCommand(newUpstreamFetchCommand(a), newUpstreamStatusCommand(a))
	return group
}

func newUpstreamFetchCommand(a *app) *cobra.Command {
	options := evolve.FetchUpstreamOptions{}
	var asJSON bool
	var clientID, profile string
	cmd := &cobra.Command{
		Use: "fetch", Short: "Fetch a complete pinned baseline through the stored GitHub App credential",
		Long: "fetch requires --allow-network and an explicit policy. It obtains the exact pinned commit " +
			"and complete baseline objects into a managed bare cache under --root. It never checks out " +
			"user files, runs checks, approves publication or writes to GitHub. A complete existing cache " +
			"is reused without reading credentials; a damaged cache is refused, not overwritten.",
		Args: noPositionalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireEvolveFlags(cmd, "policy"); err != nil {
				return err
			}
			options.OpenGit = func(ctx context.Context, target evolve.Target) (githubauth.GitEndpoint, error) {
				if a.openGit == nil {
					return githubauth.GitEndpoint{}, fmt.Errorf("GitHub App Git authentication is unavailable")
				}
				id := a.resolveClientID(clientID)
				if err := validateClientIDFor("evolve upstream fetch", id); err != nil {
					return githubauth.GitEndpoint{}, err
				}
				return a.openGit(ctx, target.Host, id, normalizeProfile(profile), target.Repository)
			}
			result, err := evolve.FetchUpstream(cmd.Context(), options)
			if err != nil {
				return err
			}
			return renderUpstream(a, asJSON, result)
		},
	}
	cmd.Flags().StringVar(&options.Policy, "policy", "", "trusted operator policy fixing host, repository and full commit")
	cmd.Flags().StringVar(&options.Root, "root", ".", "existing workspace that owns the upstream cache")
	cmd.Flags().BoolVar(&options.AllowNetwork, "allow-network", false, "allow the App-authenticated Git download (never implies publication approval)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "public GitHub App client id")
	cmd.Flags().StringVar(&profile, "profile", defaultAuthProfile, "stored credential profile")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable result")
	return cmd
}

func newUpstreamStatusCommand(a *app) *cobra.Command {
	var policy, root string
	var asJSON bool
	cmd := &cobra.Command{
		Use: "status", Short: "Inspect only local baseline readiness without credentials, writes or network",
		Args: noPositionalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireEvolveFlags(cmd, "policy"); err != nil {
				return err
			}
			result, err := evolve.UpstreamStatus(cmd.Context(), policy, root)
			if err != nil {
				return err
			}
			return renderUpstream(a, asJSON, result)
		},
	}
	cmd.Flags().StringVar(&policy, "policy", "", "trusted operator policy JSON file")
	cmd.Flags().StringVar(&root, "root", ".", "existing workspace that owns the upstream cache")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable readiness report")
	return cmd
}

func renderUpstream(a *app, asJSON bool, result evolve.UpstreamResult) error {
	if asJSON {
		return writeJSON(a.stdout, result)
	}
	_, err := fmt.Fprintf(a.stdout, "upstream: %s\nrepository: %s/%s\nbase: %s\ncache: %s\nauthentication: %s\nchecks: %s\n",
		result.Status, result.Target.Host, result.Target.Repository, result.Target.BaseCommit,
		result.Upstream, result.Authentication, result.Checks)
	if err != nil {
		return err
	}
	if result.Problem != "" {
		if _, err := fmt.Fprintln(a.stdout, "problem:", result.Problem); err != nil {
			return err
		}
	}
	if len(result.NextCommand) > 0 {
		// Use POSIX single-argument quoting, including literal $, quotes and
		// newlines in paths, rather than Go strings interpreted by a shell.
		if _, err := fmt.Fprint(a.stdout, "next:"); err != nil {
			return err
		}
		for _, arg := range result.NextCommand {
			if _, err := fmt.Fprintf(a.stdout, " '%s'", strings.ReplaceAll(arg, "'", "'\\''")); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintln(a.stdout)
	}
	return err
}
