package evolve

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	// spdxID is a single SPDX license identifier. Expressions, spacing and the
	// SPDX "NOASSERTION"/"NONE" placeholders are refused: they do not declare a
	// redistribution right.
	spdxID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.+-]*$`)
	// pathSegment is one conservative file name: no leading dot or hyphen, no
	// spaces, no trailing dot.
	pathSegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
)

// Provenance records what a proposal declares about the origin of its
// replacement content. Every field is a declaration: prepare and plan store and
// report it and never treat it as verified permission to redistribute.
type Provenance struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	License    string `json:"license"`
	Reason     string `json:"reason"`
}

// EvidenceRef is an opaque pointer a proposal offers for later review. zt never
// dereferences it: it is a label and a digest, never a path or a URL.
type EvidenceRef struct {
	Label  string `json:"label"`
	Digest string `json:"digest"`
}

// proposal is the untrusted replacement proposal supplied with --proposal. It
// names one Skill, declares where the material came from and under which
// license, explains the change, and lists explicit file replacements.
//
// There is no field for deletions, modes, symlinks, scripts, commands or an
// arbitrary patch: an unknown field is a decode error, so a proposal cannot
// describe anything outside these replacements.
type proposal struct {
	SchemaVersion int           `json:"schemaVersion"`
	Skill         string        `json:"skill"`
	Provenance    Provenance    `json:"provenance"`
	Evidence      []EvidenceRef `json:"evidence,omitempty"`
	Files         []replacement `json:"files"`
}

// replacement is one complete file after the change. The preimage digest binds
// the bytes the proposal expects to replace; it is empty only for a new file.
type replacement struct {
	Path           string `json:"path"`
	PreimageSHA256 string `json:"preimageSha256"`
	Content        string `json:"content"`
}

// loadProposal reads, decodes and normalizes one proposal.
func loadProposal(path string) (proposal, error) {
	data, err := readDocument(path, "proposal", maxProposalBytes)
	if err != nil {
		return proposal{}, err
	}
	var value proposal
	if err := decodeDocument(data, &value, "proposal"); err != nil {
		return proposal{}, err
	}
	if err := value.normalize(); err != nil {
		return proposal{}, err
	}
	return value, nil
}

// normalize validates the proposal fields that do not depend on the trusted
// policy or the pinned base commit, and trims the declared text in place. The
// replacement set itself is validated together with a stored candidate, so
// prepare and plan apply exactly the same rules.
func (p *proposal) normalize() error {
	if p.SchemaVersion != schemaVersion {
		return fmt.Errorf("proposal: unsupported schemaVersion %d", p.SchemaVersion)
	}
	p.Skill = strings.TrimSpace(p.Skill)
	if !skillID.MatchString(p.Skill) {
		return fmt.Errorf("proposal: skill must be one Skill id of lowercase letters, digits and hyphens")
	}
	provenance, err := normalizeProvenance(p.Provenance)
	if err != nil {
		return err
	}
	p.Provenance = provenance
	evidence, err := normalizeEvidence(p.Evidence)
	if err != nil {
		return err
	}
	p.Evidence = evidence
	if len(p.Files) == 0 {
		return fmt.Errorf("proposal: files must list at least one replacement")
	}
	if len(p.Files) > maxFiles {
		return fmt.Errorf("proposal: files declares %d replacements; at most %d are supported", len(p.Files), maxFiles)
	}
	return nil
}

// toCandidate binds the validated proposal to the trusted policy and the pinned
// target. Only the policy contributes the target, its digest and the required
// check names; the proposal contributes the Skill, provenance, evidence and
// replacements.
func (p proposal) toCandidate(policy Policy) (Candidate, error) {
	digest, err := policy.digest()
	if err != nil {
		return Candidate{}, err
	}
	files := make([]File, len(p.Files))
	for i, entry := range p.Files {
		// The document bound means content is already bounded overall; the
		// per-file bound is enforced by validateFiles.
		files[i] = File{
			Path:           entry.Path,
			PreimageSHA256: entry.PreimageSHA256,
			SHA256:         sha256Hex([]byte(entry.Content)),
			Bytes:          len(entry.Content),
			Content:        entry.Content,
		}
	}
	// Sorting is a normalization: the stored manifest has one canonical order,
	// so the same replacements always produce the same candidate identity.
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return Candidate{
		SchemaVersion: schemaVersion,
		Skill:         p.Skill,
		Target:        policy.target(),
		PolicyDigest:  digest,
		Checks:        append([]string(nil), policy.RequiredChecks...),
		Provenance:    p.Provenance,
		Evidence:      append([]EvidenceRef(nil), p.Evidence...),
		Files:         files,
	}, nil
}

// normalizeProvenance trims and validates the declared origin. A declaration is
// required, but it is never verified: an SPDX identifier says which license the
// proposer claims, not that redistribution is permitted.
func normalizeProvenance(value Provenance) (Provenance, error) {
	value.Repository = strings.TrimSpace(value.Repository)
	value.Commit = strings.TrimSpace(value.Commit)
	value.License = strings.TrimSpace(value.License)
	value.Reason = strings.TrimSpace(value.Reason)
	if !repositoryName.MatchString(value.Repository) {
		return Provenance{}, fmt.Errorf("provenance: repository must name the material's source as owner/name, got %s", safeQuote(value.Repository))
	}
	if _, ok := fullCommitID(value.Commit); !ok {
		return Provenance{}, fmt.Errorf("provenance: commit must be the complete lowercase Git object id of the source revision")
	}
	if !spdxID.MatchString(value.License) || strings.EqualFold(value.License, "NOASSERTION") || strings.EqualFold(value.License, "NONE") {
		return Provenance{}, fmt.Errorf("provenance: license must declare one known redistribution license as an SPDX identifier, got %s", safeQuote(value.License))
	}
	if value.Reason == "" {
		return Provenance{}, fmt.Errorf("provenance: reason must explain why the Skill should change")
	}
	if len(value.Reason) > maxReasonBytes {
		return Provenance{}, fmt.Errorf("provenance: reason exceeds %d bytes", maxReasonBytes)
	}
	if err := validateText(value.Reason, "provenance reason"); err != nil {
		return Provenance{}, err
	}
	return value, nil
}

// normalizeEvidence validates the opaque evidence references. They are carried
// for a later review step and are never read from disk or fetched.
func normalizeEvidence(refs []EvidenceRef) ([]EvidenceRef, error) {
	if len(refs) > maxEvidenceRefs {
		return nil, fmt.Errorf("evidence: declares %d references; at most %d are supported", len(refs), maxEvidenceRefs)
	}
	seen := make(map[string]bool, len(refs))
	result := make([]EvidenceRef, 0, len(refs))
	for i, ref := range refs {
		ref.Label = strings.TrimSpace(ref.Label)
		ref.Digest = strings.TrimSpace(ref.Digest)
		if ref.Label == "" || len(ref.Label) > maxLabelBytes {
			return nil, fmt.Errorf("evidence[%d]: label must be 1-%d bytes", i, maxLabelBytes)
		}
		if err := validateText(ref.Label, fmt.Sprintf("evidence[%d] label", i)); err != nil {
			return nil, err
		}
		if !isSHA256Hex(ref.Digest) {
			return nil, fmt.Errorf("evidence[%d]: digest must be a complete lowercase SHA-256", i)
		}
		if seen[ref.Label] {
			return nil, fmt.Errorf("evidence[%d]: duplicate label %s", i, safeQuote(ref.Label))
		}
		seen[ref.Label] = true
		result = append(result, ref)
	}
	return result, nil
}

// validateText refuses anything zt will not handle as UTF-8 text: invalid UTF-8
// and control characters other than tab, newline and carriage return. Combined
// with the path allowlist this keeps scripts and binaries out of the payload.
func validateText(text, what string) error {
	if !utf8.ValidString(text) {
		return fmt.Errorf("%s: is not valid UTF-8", what)
	}
	for _, r := range text {
		if r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%s: contains a control character", what)
		}
	}
	return nil
}

// validateReplacementPath accepts only the initial replacement scope of one
// Skill: skills/<id>/SKILL.md, skills/<id>/LICENSE and reference documents below
// skills/<id>/references/. Deletions, modes, symlinks, absolute paths,
// backslashes and traversal are outside the schema rather than handled later.
func validateReplacementPath(value, skill, what string) error {
	if value == "" {
		return fmt.Errorf("%s: path is empty", what)
	}
	if len(value) > maxPathBytes {
		return fmt.Errorf("%s: path exceeds %d bytes", what, maxPathBytes)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s: path is not valid UTF-8", what)
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%s: path contains a control character", what)
		}
	}
	if strings.HasPrefix(value, "/") {
		return fmt.Errorf("%s: path must be relative to the repository root", what)
	}
	if strings.Contains(value, "\\") {
		return fmt.Errorf("%s: path must use forward slashes", what)
	}
	segments := strings.Split(value, "/")
	if len(segments) > maxPathSegments {
		return fmt.Errorf("%s: path is more than %d segments deep", what, maxPathSegments)
	}
	for _, segment := range segments {
		if !validPathSegment(segment) {
			return fmt.Errorf("%s: path segment %s is not a plain file name", what, safeQuote(segment))
		}
	}
	prefix := "skills/" + skill + "/"
	if !strings.HasPrefix(value, prefix) {
		return fmt.Errorf("%s: path %s is outside skills/%s/", what, safeQuote(value), skill)
	}
	relative := strings.TrimPrefix(value, prefix)
	switch {
	case relative == "SKILL.md", relative == "LICENSE":
		return nil
	case strings.HasPrefix(relative, "references/") && strings.HasSuffix(relative, ".md"):
		return nil
	}
	return fmt.Errorf("%s: path %s must be skills/%s/SKILL.md, skills/%s/LICENSE or a .md document below skills/%s/references/", what, safeQuote(value), skill, skill, skill)
}

// validPathSegment accepts one plain file name: no dots-only name, no leading
// dot or hyphen, no trailing dot and no character a case-insensitive or
// Windows-hosted checkout would reinterpret.
func validPathSegment(segment string) bool {
	return pathSegment.MatchString(segment) && !strings.HasSuffix(segment, ".")
}
