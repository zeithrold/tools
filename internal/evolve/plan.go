package evolve

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PlanOptions selects a stored candidate, the trusted operator policy and the
// local upstream repository that holds the pinned base commit.
type PlanOptions struct {
	Candidate string
	Policy    string
	Upstream  string
	Root      string
}

// Plan re-reads one stored candidate and repeats every validation prepare
// performed: the canonical document bytes, the candidate identity, the operator
// policy digest, the pinned base commit and each preimage digest.
//
// It reads only. It creates no directory, cache, lock or artifact, writes no
// project file, runs no check, reads no credential and accesses no network. The
// result reports declared provenance and the policy's named checks as not_run, and
// reports publication as not_ready: planning is not publication.
func Plan(ctx context.Context, options PlanOptions) (PlanResult, error) {
	if err := ctx.Err(); err != nil {
		return PlanResult{}, err
	}
	policy, err := LoadPolicy(options.Policy)
	if err != nil {
		return PlanResult{}, err
	}
	data, id, err := readCandidate(options.Candidate)
	if err != nil {
		return PlanResult{}, err
	}
	var candidate Candidate
	if err := decodeDocument(data, &candidate, "candidate"); err != nil {
		return PlanResult{}, err
	}
	canonical, err := candidate.canonicalBytes()
	if err != nil {
		return PlanResult{}, fmt.Errorf("encode candidate: %w", err)
	}
	if !bytes.Equal(canonical, data) {
		return PlanResult{}, fmt.Errorf("candidate: bytes are not canonical, so the file was modified after preparation")
	}
	repo, err := resolveUpstream(ctx, options.Upstream, options.Root, policy)
	if err != nil {
		return PlanResult{}, err
	}
	if err := validateCandidate(ctx, repo, policy, candidate); err != nil {
		return PlanResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return PlanResult{}, err
	}
	return PlanResult{
		SchemaVersion:   schemaVersion,
		CandidateID:     id,
		CandidatePath:   options.Candidate,
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
	}, nil
}

// readCandidate reads one stored candidate and derives its identity. The owned
// layout is part of that identity: the containing directory is named by the
// content digest, so a candidate edited in place, or one moved to a path that
// does not name its own digest, is refused rather than planned under an
// unverified name. The path above that directory is not inspected: the bytes are
// self-verifying, so the same candidate may be read outside the workspace it was
// prepared in.
func readCandidate(path string) ([]byte, string, error) {
	directory := filepath.Dir(path)
	name := filepath.Base(directory)
	if filepath.Base(path) != candidateName || !isSHA256Hex(name) {
		return nil, "", fmt.Errorf("candidate must be stored as <content-id>/%s, where the directory names the SHA-256 of its bytes as prepare writes it; got %s", candidateName, safeQuote(path))
	}
	parent, err := os.Lstat(directory)
	if err != nil {
		return nil, "", fmt.Errorf("candidate: %w", err)
	}
	if parent.Mode()&os.ModeSymlink != 0 {
		return nil, "", fmt.Errorf("refusing symlinked candidate directory %s", safeQuote(directory))
	}
	if !parent.IsDir() {
		return nil, "", fmt.Errorf("candidate directory is not a directory: %s", safeQuote(directory))
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, "", fmt.Errorf("candidate: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, "", fmt.Errorf("refusing symlinked candidate file %s", safeQuote(path))
	}
	if !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("candidate is not a regular file: %s", safeQuote(path))
	}
	if info.Size() > maxCandidateBytes {
		return nil, "", fmt.Errorf("candidate exceeds %d bytes", maxCandidateBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("candidate: %w", err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, "", fmt.Errorf("candidate: %w", err)
	}
	if !os.SameFile(info, opened) {
		return nil, "", fmt.Errorf("candidate file changed while it was opened: %s", safeQuote(path))
	}
	data, err := io.ReadAll(io.LimitReader(file, maxCandidateBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("candidate: %w", err)
	}
	if int64(len(data)) > maxCandidateBytes {
		return nil, "", fmt.Errorf("candidate exceeds %d bytes", maxCandidateBytes)
	}
	id := candidateID(data)
	if id != name {
		return nil, "", fmt.Errorf("candidate content id %s does not match its directory name %s; the file was modified after preparation", id, name)
	}
	return data, id, nil
}
