package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zeithrold/tools/internal/evolve"
	"github.com/zeithrold/tools/internal/githubauth"
)

// evolveCandidateArg accepts exactly one candidate path. It refuses a missing value
// or a flag-looking value without echoing the argument, so a mistyped command cannot
// print a pasted secret.
func evolveCandidateArg(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("evolve %s requires the path to a prepared candidate file", cmd.Name())
	}
	if len(args) > 1 {
		return fmt.Errorf("evolve %s: unexpected arguments", cmd.Name())
	}
	return nil
}

// evolveOpenAPI adapts the app's credential seam to the evolve package. It is only
// built for a run that will reach GitHub.
func (a *app) evolveOpenAPI(command, clientID, profile string) func(ctx context.Context, target evolve.Target) (githubauth.API, error) {
	return func(ctx context.Context, target evolve.Target) (githubauth.API, error) {
		if a.openAPI == nil {
			return githubauth.API{}, fmt.Errorf("GitHub authentication is unavailable in this build: the githubauth/keystore adapter is not linked")
		}
		id := a.resolveClientID(clientID)
		if err := validateClientIDFor(command, id); err != nil {
			return githubauth.API{}, fmt.Errorf("%w; this command uses the same stored GitHub App credential as `zt auth`", err)
		}
		return a.openAPI(ctx, target.Host, id, normalizeProfile(profile), target.Repository)
	}
}

func newEvolveCheckCommand(a *app) *cobra.Command {
	options := evolve.CheckOptions{}
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "check CANDIDATE_FILE",
		Short: "Run the trusted validation contract against a materialised pinned base",
		Long: `check materialises the complete pinned base commit with the candidate replacements applied, then
runs the operator's validation commands with the existing native argv, timeout and log handling.

Execution only happens with --trusted-local-execution. That flag is the operator's reviewed decision
to run these commands as native processes on this host. The run is NOT an OS sandbox and NOT
credential isolation: the commands keep the operator's own access to files, the network, the OS
keychain and this machine. Bounds, a minimal environment and per-command timeouts reduce accidents;
they do not contain a malicious candidate. Results are operator-local evidence that the same user can
alter, and they are never uploaded.`,
		Args: evolveCandidateArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireEvolveFlags(cmd, "policy", "upstream", "validation"); err != nil {
				return err
			}
			options.Candidate = args[0]
			result, err := evolve.Verify(cmd.Context(), options)
			if err != nil {
				return err
			}
			if asJSON {
				if err := writeJSON(a.stdout, result); err != nil {
					return err
				}
			} else if err := renderEvolveCheck(a.stdout, result); err != nil {
				return err
			}
			// The result is reported either way; a run that did not pass is still a
			// failure of the command, exactly like `zt check`.
			if result.Status != "passed" {
				return fmt.Errorf("evolve check did not pass (status %s); see %s", result.Status, result.ResultPath)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&options.Policy, "policy", "", "trusted operator policy JSON file")
	cmd.Flags().StringVar(&options.Upstream, "upstream", "", "local Git repository holding the pinned base commit")
	cmd.Flags().StringVar(&options.Validation, "validation", "", "trusted validation contract JSON file")
	cmd.Flags().StringVar(&options.Root, "root", ".", "existing local workspace that owns .zt/artifacts/evolve")
	cmd.Flags().BoolVar(&options.TrustedLocalExecution, "trusted-local-execution", false, "run the reviewed native commands on this host (NOT an OS sandbox, NOT credential isolation)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable verification record")
	return cmd
}

func newEvolveSubmitCommand(a *app) *cobra.Command {
	options := evolve.SubmitOptions{}
	var asJSON bool
	var clientID, profile string
	cmd := &cobra.Command{
		Use:   "submit CANDIDATE_FILE",
		Short: "Open a draft pull request for one approved candidate",
		Long: `submit requires a passed local verification, an operator approval naming the exact candidate,
target, base commit, validation contract and public payload, and stored GitHub App credentials.

It creates one deterministic branch and one draft pull request, or returns an exactly matching one
without writing; a branch that already holds exactly the approved commit but no pull request is
completed by creating only the pull request. It never merges, publishes, activates, force-updates a
ref or reopens a closed pull request. A new write against a moved base is refused.

--show-payload prints the exact document that would be uploaded, and its digest, without reading any
credential, contacting the network, running anything or writing anything. Approve by writing an
approval file naming that digest.`,
		Args: evolveCandidateArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireEvolveFlags(cmd, "policy", "upstream", "validation"); err != nil {
				return err
			}
			if !options.ShowPayload {
				if err := requireEvolveFlags(cmd, "approval"); err != nil {
					return err
				}
			}
			options.Candidate = args[0]
			options.OpenAPI = a.evolveOpenAPI("evolve submit", clientID, profile)
			result, err := evolve.Submit(cmd.Context(), options)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(a.stdout, result)
			}
			return renderEvolveSubmit(a.stdout, result)
		},
	}
	cmd.Flags().StringVar(&options.Policy, "policy", "", "trusted operator policy JSON file")
	cmd.Flags().StringVar(&options.Upstream, "upstream", "", "local Git repository holding the pinned base commit")
	cmd.Flags().StringVar(&options.Validation, "validation", "", "trusted validation contract JSON file")
	cmd.Flags().StringVar(&options.Approval, "approval", "", "operator approval JSON file binding this exact candidate and payload")
	cmd.Flags().StringVar(&options.Root, "root", ".", "existing local workspace that owns .zt/artifacts/evolve")
	cmd.Flags().StringVar(&clientID, "client-id", "", "public GitHub App client id that keys the stored credential")
	cmd.Flags().StringVar(&profile, "profile", defaultAuthProfile, "stored credential profile (account)")
	cmd.Flags().BoolVar(&options.ShowPayload, "show-payload", false, "print the exact public payload and its digest, then stop without network, credential or writes")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable result")
	return cmd
}

func newEvolveStatusCommand(a *app) *cobra.Command {
	options := evolve.StatusOptions{}
	var asJSON bool
	var clientID, profile string
	cmd := &cobra.Command{
		Use:   "status CANDIDATE_FILE",
		Short: "Report local candidate state, or reconcile the pull request with --remote",
		Long: `status revalidates one stored candidate and reports the local verification record, the
submission journal and the publication state. It reads no credential, opens no connection and writes
nothing.

--remote explicitly reads GitHub through the stored GitHub App credential: pull request state, draft
flag, exact head and GitHub-observed check-run counts. Differences are reported as drift; they are
never rewritten and never reported as an exact match.`,
		Args: evolveCandidateArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireEvolveFlags(cmd, "policy", "upstream"); err != nil {
				return err
			}
			options.Candidate = args[0]
			if options.Remote {
				options.OpenAPI = a.evolveOpenAPI("evolve status", clientID, profile)
			}
			result, err := evolve.Status(cmd.Context(), options)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(a.stdout, result)
			}
			return renderEvolveStatus(a.stdout, result)
		},
	}
	cmd.Flags().StringVar(&options.Policy, "policy", "", "trusted operator policy JSON file")
	cmd.Flags().StringVar(&options.Upstream, "upstream", "", "local Git repository holding the pinned base commit")
	cmd.Flags().StringVar(&options.Validation, "validation", "", "trusted validation contract JSON file")
	cmd.Flags().StringVar(&options.Root, "root", ".", "existing local workspace that owns .zt/artifacts/evolve")
	cmd.Flags().BoolVar(&options.Remote, "remote", false, "also reconcile the pull request and checks on GitHub (network and stored credential)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "public GitHub App client id that keys the stored credential")
	cmd.Flags().StringVar(&profile, "profile", defaultAuthProfile, "stored credential profile (account)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable status")
	return cmd
}

func renderEvolveCheck(output io.Writer, result evolve.CheckResult) error {
	state := "verified"
	switch result.Status {
	case "passed":
	case "failed":
		state = "failed"
	default:
		state = result.Status
	}
	if result.Reused {
		state += " (reused an earlier identical run)"
	}
	if _, err := fmt.Fprintf(output, "candidate %s %s\n", result.CandidateID, state); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "validation: %s\n", result.ValidationDigest); err != nil {
		return err
	}
	if result.ResultPath != "" {
		if _, err := fmt.Fprintf(output, "result: %s\n", result.ResultPath); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(output, "workspace: base %s, inputs %s\n", result.BaseTreeDigest, unchangedLabel(result.WorkspaceChanged)); err != nil {
		return err
	}
	for _, step := range result.Checks {
		detail := ""
		if step.Detail != "" {
			detail = " (" + step.Detail + ")"
		}
		if _, err := fmt.Fprintf(output, "  %s %s exit %d in %dms log %s %d bytes%s\n", step.Name, step.Status, step.ExitCode, step.DurationMS, step.Log, step.LogBytes, detail); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(output, "execution: trusted local native command — not an OS sandbox and not credential isolation"); err != nil {
		return err
	}
	for _, note := range result.Notes {
		if _, err := fmt.Fprintln(output, "note: "+note); err != nil {
			return err
		}
	}
	return nil
}

func unchangedLabel(changed bool) string {
	if changed {
		return "changed during the run (result not reusable)"
	}
	return "unchanged"
}

func renderEvolveSubmit(output io.Writer, result evolve.SubmitResult) error {
	if result.Payload != nil {
		data, err := result.Payload.CanonicalBytes()
		if err != nil {
			return err
		}
		if _, err := output.Write(data); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(output, "public payload digest: %s\n", result.PublicPayloadDigest); err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "\nNothing was read from the OS keystore, executed, or written; no network connection was made.\nReview the document above (it is the exact publication, including file content). To approve it,\nwrite an approval file naming candidateId %s, target %s/%s@%s, validationDigest %s,\npublicPayloadDigest %s, your reviewer identifier, an RFC 3339 UTC reviewedAt, and\nattestations humanReview/privacy/license set to \"approved\".\n",
			result.CandidateID, result.Target.Host, result.Target.Repository, result.Target.BaseCommit, result.ValidationDigest, result.PublicPayloadDigest)
		return err
	}
	if _, err := fmt.Fprintf(output, "submission %s (%s)\n", result.Action, result.State); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "candidate %s\n", result.CandidateID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "branch %s -> %s\n", result.Branch, result.BaseBranch); err != nil {
		return err
	}
	if result.Commit != "" {
		if _, err := fmt.Fprintf(output, "commit %s\n", result.Commit); err != nil {
			return err
		}
	}
	if result.PullRequest != nil {
		if _, err := fmt.Fprintf(output, "pull request #%d %s (state %s, draft %t)\n", result.PullRequest.Number, result.PullRequest.URL, result.PullRequest.State, result.PullRequest.Draft); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(output, "writes: %d\n", result.Writes); err != nil {
		return err
	}
	if result.Attestations != nil {
		if _, err := fmt.Fprintf(output, "attestations: humanReview=%s privacy=%s license=%s (attested by %s at %s, not verified by zt)\n",
			result.Attestations.HumanReview, result.Attestations.Privacy, result.Attestations.License, result.Attestations.Reviewer, result.Attestations.ReviewedAt); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(output, "publication: %s (a draft pull request is not a merge, publication or activation)\n", result.Publication); err != nil {
		return err
	}
	if result.JournalPath != "" {
		if _, err := fmt.Fprintf(output, "journal: %s\n", result.JournalPath); err != nil {
			return err
		}
	}
	for _, note := range result.Notes {
		if _, err := fmt.Fprintln(output, "note: "+note); err != nil {
			return err
		}
	}
	return nil
}

func renderEvolveStatus(output io.Writer, result evolve.StatusResult) error {
	if _, err := fmt.Fprintf(output, "candidate %s\n", result.CandidateID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "skill %s on %s/%s@%s\n", result.Skill, result.Target.Host, result.Target.Repository, result.Target.BaseCommit); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "branch %s\n", result.Branch); err != nil {
		return err
	}
	if result.ValidationDigest != "" {
		if _, err := fmt.Fprintf(output, "validation %s\n", result.ValidationDigest); err != nil {
			return err
		}
	}
	if result.Check == nil {
		if _, err := fmt.Fprintln(output, "local verification: not run"); err != nil {
			return err
		}
	} else {
		line := "local verification: " + result.Check.Status
		if result.Check.WorkspaceChanged {
			line += " (inputs changed during the run; not reusable)"
		}
		if result.Check.StartedAt != "" {
			line += " at " + result.Check.StartedAt
		}
		if _, err := fmt.Fprintln(output, line); err != nil {
			return err
		}
	}
	if result.Journal == nil {
		if _, err := fmt.Fprintln(output, "submission journal: none"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(output, "submission journal: %s (updated %s)\n", result.Journal.State, result.Journal.UpdatedAt); err != nil {
			return err
		}
		if result.Journal.PullRequest != nil {
			if _, err := fmt.Fprintf(output, "  pull request #%d %s\n", result.Journal.PullRequest.Number, result.Journal.PullRequest.URL); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(output, "publication: %s\n", result.Publication); err != nil {
		return err
	}
	if result.Remote != nil {
		if _, err := fmt.Fprintln(output, "remote (explicit --remote read):"); err != nil {
			return err
		}
		if result.Remote.PullRequest == nil {
			if _, err := fmt.Fprintln(output, "  pull request: none"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(output, "  pull request #%d %s state %s draft %t\n", result.Remote.PullRequest.Number, result.Remote.PullRequest.URL, result.Remote.PullRequest.State, result.Remote.PullRequest.Draft); err != nil {
				return err
			}
		}
		if result.Remote.HeadSHA != "" {
			if _, err := fmt.Fprintf(output, "  head %s (base branch %s)\n", result.Remote.HeadSHA, result.Remote.DefaultBranch); err != nil {
				return err
			}
		}
		if result.Remote.CheckRuns != nil {
			if _, err := fmt.Fprintf(output, "  github check runs: %d total, %d success, %d failure, %d pending (GitHub observation, not zt execution)\n", result.Remote.CheckRuns.Total, result.Remote.CheckRuns.Success, result.Remote.CheckRuns.Failure, result.Remote.CheckRuns.Pending); err != nil {
				return err
			}
		}
		if len(result.Remote.Drift) > 0 {
			if _, err := fmt.Fprintf(output, "  drift: %s\n", strings.Join(result.Remote.Drift, ", ")); err != nil {
				return err
			}
		}
	}
	for _, warning := range result.Warnings {
		if _, err := fmt.Fprintln(output, "warning: "+warning); err != nil {
			return err
		}
	}
	return nil
}
