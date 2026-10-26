package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zeithrold/tools/internal/evolve"
)

// evolveLong states the contract of this slice: a structured replacement
// proposal, an offline re-check, and no publication step. It deliberately does
// not describe a --patch flag, because zt never accepted a patch stream.
const evolveLong = `evolve prepares, verifies and submits immutable local candidates for evolving one Skill.

A candidate is a structured replacement proposal, not a unified diff. It names one
Skill, declares the material's source, license and reason, and lists explicit file
replacements. Each replacement carries a repository path, the SHA-256 of the bytes
it replaces (empty for a new file) and the complete new content. zt never applies a
diff, so there is no --patch flag and no patch stream is read.

prepare validates a proposal against a trusted operator policy and a pinned local
upstream commit, then writes only .zt/artifacts/evolve/<content-id>/candidate.json
below --root. plan re-reads such a candidate and repeats every one of those checks;
both are offline and read-only beyond prepare's own artifact directory.

check runs the trusted validation contract against a complete materialised pinned
base with the candidate applied. It only executes with --trusted-local-execution:
those commands are native local processes, which is neither an OS sandbox nor
credential isolation.

submit requires a passed local verification and an operator approval naming the
exact candidate, target, base commit, validation contract and public payload. It
creates one deterministic branch and one draft pull request, or returns an exactly
matching existing one without writing. --show-payload prints the exact publication
and its digest without credentials, network or writes.

status reports local state read-only; --remote explicitly reconciles the pull
request and GitHub's checks. Nothing here merges, publishes, activates, force-updates
a ref or reopens a closed pull request.`

func newEvolveCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evolve",
		Short: "Prepare, verify and submit Skill evolution candidates",
		Long:  evolveLong,
		Args:  cobra.ArbitraryArgs,
		RunE:  unknownSubcommand,
	}
	cmd.AddCommand(
		newEvolvePrepareCommand(a),
		newEvolvePlanCommand(a),
		newEvolveCheckCommand(a),
		newEvolveSubmitCommand(a),
		newEvolveStatusCommand(a),
	)
	return cmd
}

func newEvolvePrepareCommand(a *app) *cobra.Command {
	options := evolve.PrepareOptions{}
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Validate a replacement proposal and write an immutable candidate",
		Long:  "prepare validates a proposal against the operator policy and the pinned local base commit, then writes one new directory below .zt/artifacts/evolve. It writes nothing until the whole candidate is valid, and repeating it for the same inputs writes nothing.",
		Args:  noPositionalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireEvolveFlags(cmd, "proposal", "policy", "upstream"); err != nil {
				return err
			}
			result, err := evolve.Prepare(cmd.Context(), options)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(a.stdout, result)
			}
			return renderPrepare(a.stdout, result)
		},
	}
	cmd.Flags().StringVar(&options.Proposal, "proposal", "", "replacement proposal JSON file")
	cmd.Flags().StringVar(&options.Policy, "policy", "", "trusted operator policy JSON file")
	cmd.Flags().StringVar(&options.Upstream, "upstream", "", "local Git repository holding the pinned base commit")
	cmd.Flags().StringVar(&options.Root, "root", ".", "existing local workspace that owns .zt/artifacts/evolve")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable result")
	return cmd
}

func newEvolvePlanCommand(a *app) *cobra.Command {
	var policy, upstream string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "plan CANDIDATE_FILE",
		Short: "Re-check a stored candidate without writing, running or publishing anything",
		Long:  "plan re-reads a candidate written by prepare and revalidates its canonical bytes, content id, policy digest, pinned base commit and preimage digests. It writes nothing and reports required checks as not_run and publication as not_ready.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || strings.HasPrefix(args[0], "-") {
				return fmt.Errorf("evolve plan requires the path to a prepared candidate file")
			}
			if len(args) > 1 {
				return fmt.Errorf("evolve plan: unexpected arguments %v", args[1:])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireEvolveFlags(cmd, "policy", "upstream"); err != nil {
				return err
			}
			result, err := evolve.Plan(cmd.Context(), evolve.PlanOptions{Candidate: args[0], Policy: policy, Upstream: upstream})
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(a.stdout, result)
			}
			return renderEvolvePlan(a.stdout, result)
		},
	}
	cmd.Flags().StringVar(&policy, "policy", "", "trusted operator policy JSON file")
	cmd.Flags().StringVar(&upstream, "upstream", "", "local Git repository holding the pinned base commit")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable result")
	return cmd
}

// requireEvolveFlags refuses a missing required input before any file is read.
func requireEvolveFlags(cmd *cobra.Command, names ...string) error {
	for _, name := range names {
		value, err := cmd.Flags().GetString(name)
		if err != nil {
			return err
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s requires --%s", cmd.CommandPath(), name)
		}
	}
	return nil
}

func renderPrepare(output io.Writer, result evolve.PrepareResult) error {
	state := "already present"
	if result.Created {
		state = "created"
	}
	if _, err := fmt.Fprintf(output, "candidate %s (%s)\n", result.CandidateID, state); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "path: %s\n", result.CandidatePath); err != nil {
		return err
	}
	return renderEvolveBody(output, result.Skill, result.Target, result.Files, result.Checks, result.Provenance, result.Evidence, result.Publication, result.MissingEvidence)
}

func renderEvolvePlan(output io.Writer, result evolve.PlanResult) error {
	if _, err := fmt.Fprintf(output, "candidate %s\n", result.CandidateID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "path: %s\n", result.CandidatePath); err != nil {
		return err
	}
	return renderEvolveBody(output, result.Skill, result.Target, result.Files, result.Checks, result.Provenance, result.Evidence, result.Publication, result.MissingEvidence)
}

// renderEvolveBody prints what prepare and plan both report. Every declaration is
// labelled as declared and every check as not run.
func renderEvolveBody(output io.Writer, skill string, target evolve.Target, files []evolve.FileSummary, checks []evolve.Check, provenance evolve.Provenance, evidence []evolve.EvidenceRef, publication string, missing []string) error {
	if _, err := fmt.Fprintf(output, "skill: %s\n", skill); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "target: %s/%s@%s\n", target.Host, target.Repository, target.BaseCommit); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(output, "replacements:"); err != nil {
		return err
	}
	for _, file := range files {
		replaces := "new file"
		if file.PreimageSHA256 != "" {
			replaces = "replaces sha256 " + file.PreimageSHA256
		}
		if _, err := fmt.Fprintf(output, "  %s\n    %s; becomes sha256 %s (%d bytes)\n", file.Path, replaces, file.SHA256, file.Bytes); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(output, "checks (not run):"); err != nil {
		return err
	}
	for _, check := range checks {
		if _, err := fmt.Fprintf(output, "  %s %s\n", check.Name, check.Status); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(output, "provenance (declared, not verified): source %s@%s, license %s\n", provenance.Repository, provenance.Commit, provenance.License); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "reason: %s\n", provenance.Reason); err != nil {
		return err
	}
	for _, ref := range evidence {
		if _, err := fmt.Fprintf(output, "evidence (declared, not verified): %s sha256 %s\n", ref.Label, ref.Digest); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(output, "publication: %s\n", publication); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "missing independent evidence: %s\n", strings.Join(missing, ", ")); err != nil {
		return err
	}
	return nil
}
