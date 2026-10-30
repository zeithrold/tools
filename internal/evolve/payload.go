package evolve

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Bounds on the publication. They are enforced before the digest is printed, so
// `--show-payload` can only ever describe a payload that fits the reviewed shape.
const (
	maxPayloadBytes       = 1 << 20
	maxCommitMessageBytes = 4096
	maxPullTitleBytes     = 120
	maxPullBodyBytes      = 8192

	// branchPrefix and the fixed author keep every submission attributable to zt
	// itself. zt never impersonates the reviewing user, and the deterministic
	// branch name is derived from content, never from a timestamp or a run id.
	branchPrefix      = "evolve/"
	evolveAuthorName  = "zt-evolve"
	evolveAuthorEmail = "zt-evolve@users.noreply.github.com"
)

var branchPattern = regexp.MustCompile(`^evolve/[a-z][a-z0-9-]*-[0-9a-f]{12}$`)

// PayloadFile is one file exactly as it is published. Content is authoritative:
// a reviewer has to be able to read the bytes that will be uploaded, not only a
// digest of them.
type PayloadFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// PayloadCommit is the commit metadata a submission creates. CommittedAt comes
// from the pinned base commit, so the payload (and therefore its digest) is
// reproducible on every replay.
type PayloadCommit struct {
	Message     string `json:"message"`
	AuthorName  string `json:"authorName"`
	AuthorEmail string `json:"authorEmail"`
	CommittedAt string `json:"committedAt"`
}

// PayloadPullRequest is the draft pull request a submission opens.
type PayloadPullRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Draft bool   `json:"draft"`
}

// PublicPayload is the complete, reviewable publication: the exact bytes that
// leave the operator's machine. Its canonical rendering is what
// publicPayloadDigest covers, so the document a reviewer reads and the content
// that is uploaded cannot diverge without changing the digest.
type PublicPayload struct {
	SchemaVersion   int                `json:"schemaVersion"`
	Host            string             `json:"host"`
	Repository      string             `json:"repository"`
	BaseCommit      string             `json:"baseCommit"`
	BaseCommittedAt string             `json:"baseCommittedAt"`
	Branch          string             `json:"branch"`
	Files           []PayloadFile      `json:"files"`
	Commit          PayloadCommit      `json:"commit"`
	PullRequest     PayloadPullRequest `json:"pullRequest"`
}

// branchFor derives the one dedicated branch a candidate may own. Two different
// candidates cannot share it, and the same candidate cannot produce a second
// branch by re-running.
func branchFor(skill, candidateID string) string {
	return branchPrefix + skill + "-" + candidateID[:12]
}

// buildPayload derives the publication from values zt already validated: the
// stored candidate bytes and the pinned base commit. It reads nothing else, so
// `--show-payload` prints the same document without a credential, a network
// connection or a write.
func buildPayload(candidate Candidate, candidateID string, info CommitInfo) (PublicPayload, error) {
	branch := branchFor(candidate.Skill, candidateID)
	if !branchPattern.MatchString(branch) {
		return PublicPayload{}, fmt.Errorf("payload: derived branch %s is not a valid evolve branch", safeQuote(branch))
	}
	committedAt := info.CommittedAt.UTC().Format(time.RFC3339)
	files := make([]PayloadFile, len(candidate.Files))
	for i, file := range candidate.Files {
		files[i] = PayloadFile{Path: file.Path, Content: file.Content}
	}
	payload := PublicPayload{
		SchemaVersion:   schemaVersion,
		Host:            candidate.Target.Host,
		Repository:      candidate.Target.Repository,
		BaseCommit:      candidate.Target.BaseCommit,
		BaseCommittedAt: committedAt,
		Branch:          branch,
		Files:           files,
		Commit: PayloadCommit{
			Message:     commitMessage(candidate, candidateID),
			AuthorName:  evolveAuthorName,
			AuthorEmail: evolveAuthorEmail,
			CommittedAt: committedAt,
		},
		PullRequest: PayloadPullRequest{
			Title: pullTitle(candidate, candidateID),
			Body:  pullBody(candidate, candidateID),
			Draft: true,
		},
	}
	if err := payload.validate(candidate.Skill); err != nil {
		return PublicPayload{}, err
	}
	return payload, nil
}

// commitMessage records the identity of the candidate in the commit itself, so the
// commit remains self-describing after the local artifacts are gone. It cannot
// contain the public payload digest: that digest covers this message.
func commitMessage(candidate Candidate, candidateID string) string {
	return fmt.Sprintf("evolve(%s): candidate %s\n\nCandidate-Id: %s\nBase-Commit: %s\n\nDeclared reason (not verified by zt):\n%s\n",
		candidate.Skill, candidateID[:12], candidateID, candidate.Target.BaseCommit, strings.TrimSpace(candidate.Provenance.Reason))
}

func pullTitle(candidate Candidate, candidateID string) string {
	return fmt.Sprintf("evolve(%s): candidate %s", candidate.Skill, candidateID[:12])
}

// pullBody is the review surface inside the pull request: the facts of what is being
// proposed, and an explicit label on everything zt did not verify. No log, local path,
// transcript or unrelated source is published.
func pullBody(candidate Candidate, candidateID string) string {
	var body strings.Builder
	fmt.Fprintf(&body, "## Skill evolution candidate `%s`\n\n", candidateID[:12])
	fmt.Fprintf(&body, "| Field | Value |\n| --- | --- |\n| Skill | `%s` |\n| Target | `%s/%s` |\n| Base commit | `%s` |\n| Candidate | `%s` |\n\n",
		candidate.Skill, candidate.Target.Host, candidate.Target.Repository, candidate.Target.BaseCommit, candidateID)
	body.WriteString("### Published files\n")
	for _, file := range candidate.Files {
		fmt.Fprintf(&body, "- `%s` (%d bytes)\n", file.Path, file.Bytes)
	}
	body.WriteString("\n### Declared provenance (not verified by zt)\n")
	fmt.Fprintf(&body, "- source repository: `%s`\n- source commit: `%s`\n- license: `%s`\n",
		candidate.Provenance.Repository, candidate.Provenance.Commit, candidate.Provenance.License)
	for _, ref := range candidate.Evidence {
		fmt.Fprintf(&body, "- evidence: `%s` sha256 `%s`\n", ref.Label, ref.Digest)
	}
	return body.String()
}

// validate enforces the publication bounds on the document that will be hashed and
// uploaded, so a payload zt printed for review cannot be oversized.
func (p PublicPayload) validate(skill string) error {
	if p.SchemaVersion != schemaVersion {
		return fmt.Errorf("payload: unsupported schemaVersion %d", p.SchemaVersion)
	}
	if !hostname.MatchString(p.Host) {
		return fmt.Errorf("payload: host is not a lowercase hostname")
	}
	if !repositoryName.MatchString(p.Repository) {
		return fmt.Errorf("payload: repository must be owner/name")
	}
	if _, ok := fullCommitID(p.BaseCommit); !ok {
		return fmt.Errorf("payload: baseCommit must be a complete lowercase Git object id")
	}
	if _, err := time.Parse(time.RFC3339, p.BaseCommittedAt); err != nil {
		return fmt.Errorf("payload: baseCommittedAt is not an RFC 3339 instant")
	}
	if !branchPattern.MatchString(p.Branch) {
		return fmt.Errorf("payload: branch is not a valid evolve branch")
	}
	if len(p.Files) == 0 {
		return fmt.Errorf("payload: no files would be published")
	}
	total := 0
	for i, file := range p.Files {
		if err := validateReplacementPath(file.Path, skill, fmt.Sprintf("payload files[%d]", i)); err != nil {
			return err
		}
		total += len(file.Content)
		if total > maxTotalBytes {
			return fmt.Errorf("payload: replacements exceed %d bytes in total", maxTotalBytes)
		}
	}
	if p.Commit.AuthorName != evolveAuthorName || p.Commit.AuthorEmail != evolveAuthorEmail {
		return fmt.Errorf("payload: author identity must be the fixed zt-evolve identity")
	}
	if p.Commit.CommittedAt != p.BaseCommittedAt {
		return fmt.Errorf("payload: commit date must be the pinned base committer instant")
	}
	if len(p.Commit.Message) == 0 || len(p.Commit.Message) > maxCommitMessageBytes {
		return fmt.Errorf("payload: commit message must be 1-%d bytes", maxCommitMessageBytes)
	}
	if len(p.PullRequest.Title) == 0 || len(p.PullRequest.Title) > maxPullTitleBytes {
		return fmt.Errorf("payload: pull request title must be 1-%d bytes", maxPullTitleBytes)
	}
	if len(p.PullRequest.Body) == 0 || len(p.PullRequest.Body) > maxPullBodyBytes {
		return fmt.Errorf("payload: pull request body must be 1-%d bytes", maxPullBodyBytes)
	}
	if !p.PullRequest.Draft {
		return fmt.Errorf("payload: a submission must request a draft pull request")
	}
	data, err := p.CanonicalBytes()
	if err != nil {
		return err
	}
	if len(data) > maxPayloadBytes {
		return fmt.Errorf("payload: canonical document would be %d bytes, above the %d byte limit", len(data), maxPayloadBytes)
	}
	return nil
}

// CanonicalBytes renders the payload exactly as it is hashed: two-space indented
// JSON, no HTML escaping and one trailing newline. A reviewer can therefore hash the
// printed document and get the digest submit reports.
func (p PublicPayload) CanonicalBytes() ([]byte, error) {
	return canonicalJSON(p)
}

// digest is the value `--show-payload` prints and submit re-derives.
func (p PublicPayload) digest() (string, error) {
	data, err := p.CanonicalBytes()
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}
