package evolve

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

const (
	maxApprovalBytes = 64 << 10
	maxReviewerBytes = 200
)

// attestationApproved is the only value an attestation may hold. Anything else,
// including a plausible-looking "pending" or "yes", is refused rather than
// interpreted.
const attestationApproved = "approved"

// ApprovalAttestations are the operator's own statements about the exact public
// payload named by this approval. zt never verifies any of them and never
// substitutes its own judgement: the CLI reports them as operator attestations, so
// a reader can tell an approval from an observation.
type ApprovalAttestations struct {
	HumanReview string `json:"humanReview"`
	Privacy     string `json:"privacy"`
	License     string `json:"license"`
}

// Approval is the human review decision for one exact candidate, target, base,
// validation contract and public payload.
//
// It is supplied by the operator out of band and can never be produced by a
// candidate: the candidate cannot grant itself an approval, and every field here
// is compared with what zt derived itself. approvedAt-shaped fields are
// declarations for audit; the commit date comes from the pinned base, so an
// approval is not an input to candidate or payload identity.
type Approval struct {
	SchemaVersion       int                  `json:"schemaVersion"`
	CandidateID         string               `json:"candidateId"`
	Target              Target               `json:"target"`
	ValidationDigest    string               `json:"validationDigest"`
	PublicPayloadDigest string               `json:"publicPayloadDigest"`
	Reviewer            string               `json:"reviewer"`
	ReviewedAt          string               `json:"reviewedAt"`
	Attestations        ApprovalAttestations `json:"attestations"`
}

// LoadApproval reads and validates one approval document.
func LoadApproval(path string) (Approval, error) {
	if strings.TrimSpace(path) == "" {
		return Approval{}, fmt.Errorf("--approval requires an operator-supplied approval file")
	}
	data, err := readDocument(path, "approval file", maxApprovalBytes)
	if err != nil {
		return Approval{}, err
	}
	var approval Approval
	if err := decodeDocument(data, &approval, "approval file"); err != nil {
		return Approval{}, err
	}
	return approval.normalized()
}

// normalized validates the document on its own terms and canonicalizes the review
// time, so the approval digest describes meaning rather than formatting.
func (a Approval) normalized() (Approval, error) {
	if a.SchemaVersion != schemaVersion {
		return Approval{}, fmt.Errorf("approval: unsupported schemaVersion %d", a.SchemaVersion)
	}
	if !isSHA256Hex(a.CandidateID) {
		return Approval{}, fmt.Errorf("approval: candidateId must be a complete lowercase SHA-256 of the candidate bytes")
	}
	target := Target{
		Host:       strings.ToLower(strings.TrimSpace(a.Target.Host)),
		Repository: strings.TrimSpace(a.Target.Repository),
		BaseCommit: strings.TrimSpace(a.Target.BaseCommit),
	}
	if !hostname.MatchString(target.Host) {
		return Approval{}, fmt.Errorf("approval: target host must be one lowercase hostname")
	}
	if !repositoryName.MatchString(target.Repository) {
		return Approval{}, fmt.Errorf("approval: target repository must be owner/name")
	}
	if _, ok := fullCommitID(target.BaseCommit); !ok {
		return Approval{}, fmt.Errorf("approval: target baseCommit must be a complete lowercase Git object id")
	}
	a.Target = target
	if !isSHA256Hex(a.ValidationDigest) {
		return Approval{}, fmt.Errorf("approval: validationDigest must be a complete lowercase SHA-256 of the validation contract")
	}
	if !isSHA256Hex(a.PublicPayloadDigest) {
		return Approval{}, fmt.Errorf("approval: publicPayloadDigest must be a complete lowercase SHA-256; run `zt evolve submit --show-payload` and review that exact document before approving")
	}
	reviewer := strings.TrimSpace(a.Reviewer)
	if reviewer == "" || len(reviewer) > maxReviewerBytes {
		return Approval{}, fmt.Errorf("approval: reviewer must be 1-%d bytes", maxReviewerBytes)
	}
	for _, r := range reviewer {
		if !unicode.IsPrint(r) {
			return Approval{}, fmt.Errorf("approval: reviewer contains a control character")
		}
	}
	a.Reviewer = reviewer
	reviewed, err := parseReviewTime(a.ReviewedAt)
	if err != nil {
		return Approval{}, err
	}
	a.ReviewedAt = formatReviewTime(reviewed)
	for _, attestation := range []struct {
		name  string
		value string
	}{
		{"humanReview", a.Attestations.HumanReview},
		{"privacy", a.Attestations.Privacy},
		{"license", a.Attestations.License},
	} {
		if attestation.value != attestationApproved {
			return Approval{}, fmt.Errorf("approval: attestations.%s must be %s; nothing else counts as an approval", attestation.name, safeQuote(attestationApproved))
		}
	}
	return a, nil
}

// parseReviewTime accepts one RFC 3339 instant and requires it to be stated in
// UTC, so the recorded review time cannot be reinterpreted by a local zone.
func parseReviewTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("approval: reviewedAt must be an RFC 3339 instant in UTC, for example 2026-02-01T09:30:00Z")
	}
	if !strings.HasSuffix(trimmed, "Z") || parsed.Format(time.RFC3339) != trimmed {
		return time.Time{}, fmt.Errorf("approval: reviewedAt must be a UTC instant written as YYYY-MM-DDTHH:MM:SSZ, so the reviewed instant cannot be reinterpreted in another zone")
	}
	return parsed.UTC(), nil
}

func formatReviewTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

// reviewedTime is the audited review instant. It is deliberately not used to date
// a commit: that comes from the pinned base commit.
func (a Approval) reviewedTime() (time.Time, error) {
	return parseReviewTime(a.ReviewedAt)
}

// canonicalBytes renders the approval exactly as it is hashed.
func (a Approval) canonicalBytes() ([]byte, error) {
	return canonicalJSON(a)
}

// digest identifies the complete approval document, so the reviewer, the review
// time and every attestation are bound into the same value submit checks.
func (a Approval) digest() (string, error) {
	data, err := a.canonicalBytes()
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}

// validateFor requires the approval to name exactly what zt derived locally.
// Nothing is taken from the approval's word alone.
func (a Approval) validateFor(candidateID string, target Target, validationDigest, payloadDigest string) error {
	if a.CandidateID != candidateID {
		return fmt.Errorf("approval: candidateId %s does not match this candidate (%s); approve the exact bytes being submitted", a.CandidateID, candidateID)
	}
	if a.Target != target {
		return fmt.Errorf("approval: target does not match the operator policy; approve the exact host, repository and base commit")
	}
	if a.ValidationDigest != validationDigest {
		return fmt.Errorf("approval: validationDigest does not match the validation contract supplied now; re-run `zt evolve submit --show-payload` and approve the current contract")
	}
	if a.PublicPayloadDigest != payloadDigest {
		return fmt.Errorf("approval: publicPayloadDigest does not match the payload derived from this candidate and contract; review `zt evolve submit --show-payload` and approve that exact document")
	}
	return nil
}
