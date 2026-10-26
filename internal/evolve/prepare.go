package evolve

import (
	"context"
	"fmt"
)

// PrepareOptions selects the three inputs of a prepare run: the untrusted
// proposal, the trusted operator policy, and the local upstream repository whose
// pinned commit is the base. Root is the existing workspace that owns the
// artifact directory.
type PrepareOptions struct {
	Proposal string
	Policy   string
	Upstream string
	Root     string
}

// Prepare validates a replacement proposal against the operator policy and the
// pinned local base commit, then installs the derived candidate.
//
// It writes exactly one new directory, .zt/artifacts/evolve/<content-id>/, below
// the selected workspace, and only after the whole candidate is valid. It reads
// no credential, opens no network connection, runs no check and changes no
// project file, branch or Git index. Repeating it for the same inputs rewrites
// nothing. Installation of that directory is the commit point: a canceled run
// that has not reached it leaves no candidate behind.
func Prepare(ctx context.Context, options PrepareOptions) (PrepareResult, error) {
	if err := ctx.Err(); err != nil {
		return PrepareResult{}, err
	}
	policy, err := LoadPolicy(options.Policy)
	if err != nil {
		return PrepareResult{}, err
	}
	value, err := loadProposal(options.Proposal)
	if err != nil {
		return PrepareResult{}, err
	}
	// The policy decides which Skill may change, so refuse an unlisted one before
	// reading any repository object.
	if !policy.allows(value.Skill) {
		return PrepareResult{}, fmt.Errorf("proposal: skill %s is not allowed by the operator policy", safeQuote(value.Skill))
	}
	repo, err := OpenRepo(ctx, options.Upstream)
	if err != nil {
		return PrepareResult{}, err
	}
	candidate, err := value.toCandidate(policy)
	if err != nil {
		return PrepareResult{}, err
	}
	if err := validateCandidate(ctx, repo, policy, candidate); err != nil {
		return PrepareResult{}, err
	}
	data, err := candidate.canonicalBytes()
	if err != nil {
		return PrepareResult{}, fmt.Errorf("encode candidate: %w", err)
	}
	// The producer may never write a document the reader refuses. Canonical
	// encoding keeps the document close to its content size, and this bound makes
	// the producer and reader limits one rule rather than two.
	if len(data) > maxCandidateBytes {
		return PrepareResult{}, fmt.Errorf("candidate document would be %d bytes, above the %d byte limit plan enforces; reduce or split the replacement content", len(data), maxCandidateBytes)
	}
	id := candidateID(data)
	if err := ctx.Err(); err != nil {
		return PrepareResult{}, err
	}
	store, err := openStore(options.Root)
	if err != nil {
		return PrepareResult{}, err
	}
	defer store.Close()
	created, err := store.install(ctx, id, data)
	if err != nil {
		return PrepareResult{}, err
	}
	return PrepareResult{
		SchemaVersion:   schemaVersion,
		CandidateID:     id,
		CandidatePath:   candidateRelative(id),
		Created:         created,
		Skill:           candidate.Skill,
		Target:          candidate.Target,
		PolicyDigest:    candidate.PolicyDigest,
		Files:           summarizeFiles(candidate.Files),
		Checks:          describeChecks(candidate.Checks),
		Provenance:      candidate.Provenance,
		ProvenanceState: declaredUnverified,
		Evidence:        candidate.Evidence,
		EvidenceState:   declaredUnverified,
		Publication:     publicationNotReady,
		MissingEvidence: append([]string(nil), missingEvidence...),
	}, nil
}
