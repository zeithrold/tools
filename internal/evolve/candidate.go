package evolve

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Target is the upstream repository and pinned base commit a candidate applies
// to. Both values come from the trusted policy, never from a proposal.
type Target struct {
	Host       string `json:"host"`
	Repository string `json:"repository"`
	BaseCommit string `json:"baseCommit"`
}

// File is one complete replacement in a stored candidate: the path, the digest
// of the bytes it replaces (empty for a new file), the digest and size of the new
// content, and the new content itself.
type File struct {
	Path           string `json:"path"`
	PreimageSHA256 string `json:"preimageSha256"`
	SHA256         string `json:"sha256"`
	Bytes          int    `json:"bytes"`
	Content        string `json:"content"`
}

// Candidate is the stored, canonical replacement manifest. Its bytes are the
// candidate identity: the directory that holds it is named by the SHA-256 of
// exactly these bytes. No field records a timestamp, check result or publication
// state, so nothing mutable is bound into that identity.
type Candidate struct {
	SchemaVersion int           `json:"schemaVersion"`
	Skill         string        `json:"skill"`
	Target        Target        `json:"target"`
	PolicyDigest  string        `json:"policyDigest"`
	Checks        []string      `json:"checks"`
	Provenance    Provenance    `json:"provenance"`
	Evidence      []EvidenceRef `json:"evidence,omitempty"`
	Files         []File        `json:"files"`
}

// canonicalBytes renders the exact bytes that are stored and hashed: two-space
// indented JSON followed by one newline. prepare writes these bytes and plan
// refuses a stored candidate that does not re-render to the same bytes.
func (c Candidate) canonicalBytes() ([]byte, error) {
	return canonicalJSON(c)
}

// candidateID is the content id of one stored candidate document.
func candidateID(data []byte) string {
	return sha256Hex(data)
}

// validateCandidate applies every rule to a candidate. prepare builds a
// candidate and plan re-reads one, and both call this function, so a prepared
// candidate and a re-read candidate are held to the same contract.
//
// The trusted policy decides which Skill may change and which upstream tree is
// pinned; the pinned local commit decides whether each preimage digest describes
// real bytes in a real base tree.
func validateCandidate(ctx context.Context, repo *Repo, policy Policy, c Candidate) error {
	if c.SchemaVersion != schemaVersion {
		return fmt.Errorf("candidate: unsupported schemaVersion %d", c.SchemaVersion)
	}
	if !skillID.MatchString(c.Skill) {
		return fmt.Errorf("candidate: skill is not a valid Skill id")
	}
	if !policy.allows(c.Skill) {
		return fmt.Errorf("candidate: skill %s is not allowed by the operator policy", safeQuote(c.Skill))
	}
	if c.Target != policy.target() {
		return fmt.Errorf("candidate: target does not match the operator policy; re-prepare against the pinned host, repository and base commit")
	}
	digest, err := policy.digest()
	if err != nil {
		return err
	}
	if c.PolicyDigest != digest {
		return fmt.Errorf("candidate: policy digest differs from the operator policy; the policy changed, so the candidate is stale")
	}
	if !equalLists(c.Checks, policy.RequiredChecks) {
		return fmt.Errorf("candidate: checks do not match the policy's requiredChecks")
	}
	provenance, err := normalizeProvenance(c.Provenance)
	if err != nil {
		return err
	}
	if provenance != c.Provenance {
		return fmt.Errorf("candidate: provenance is not in normalized form")
	}
	evidence, err := normalizeEvidence(c.Evidence)
	if err != nil {
		return err
	}
	if !equalEvidence(evidence, c.Evidence) {
		return fmt.Errorf("candidate: evidence is not in normalized form")
	}
	if err := validateFiles(c.Files, c.Skill); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return validateAgainstBase(ctx, repo, c)
}

// validateFiles checks the replacement set of a proposal or a stored candidate:
// path scope, content encoding, digest fields and canonical order. It runs before
// any Git read, so a malformed set never reaches the repository.
func validateFiles(files []File, skill string) error {
	if len(files) == 0 || len(files) > maxFiles {
		return fmt.Errorf("candidate: files must list 1-%d replacements", maxFiles)
	}
	folded := make(map[string]string, len(files))
	total := 0
	for i, file := range files {
		what := fmt.Sprintf("files[%d]", i)
		if err := validateReplacementPath(file.Path, skill, what); err != nil {
			return err
		}
		if i > 0 && files[i-1].Path >= file.Path {
			return fmt.Errorf("%s: replacements must be sorted by path and have no duplicate", what)
		}
		lower := strings.ToLower(file.Path)
		if other, ok := folded[lower]; ok {
			return fmt.Errorf("%s: path %s differs only in case from %s", what, safeQuote(file.Path), safeQuote(other))
		}
		folded[lower] = file.Path
		if len(file.Content) > maxFileBytes {
			return fmt.Errorf("%s: content exceeds %d bytes", what, maxFileBytes)
		}
		total += len(file.Content)
		if total > maxTotalBytes {
			return fmt.Errorf("%s: replacements exceed %d bytes in total", what, maxTotalBytes)
		}
		if err := validateText(file.Content, what+" content"); err != nil {
			return err
		}
		if file.Bytes != len(file.Content) {
			return fmt.Errorf("%s: byte count %d does not match the content", what, file.Bytes)
		}
		if file.SHA256 != sha256Hex([]byte(file.Content)) {
			return fmt.Errorf("%s: sha256 does not match the content", what)
		}
		if file.PreimageSHA256 != "" && !isSHA256Hex(file.PreimageSHA256) {
			return fmt.Errorf("%s: preimageSha256 must be empty or a complete lowercase SHA-256", what)
		}
	}
	// A path in the set cannot also be a directory of another path in the set,
	// and that holds case-insensitively: a case-insensitive checkout cannot hold
	// both "Guide.md" as a file and "guide.md/child.md" as a directory.
	fileset := make(map[string]string, len(files))
	for _, file := range files {
		fileset[strings.ToLower(file.Path)] = file.Path
	}
	for _, file := range files {
		for parent := parentPath(file.Path); parent != ""; parent = parentPath(parent) {
			if other, ok := fileset[strings.ToLower(parent)]; ok {
				return fmt.Errorf("files: path %s is both a file and a directory of %s", safeQuote(other), safeQuote(file.Path))
			}
		}
	}
	// Two replacements that reach the same directory with a different spelling
	// would also merge, or fail to merge, depending on the checkout's case rules.
	directories := make(map[string]string, len(files))
	for _, file := range files {
		for parent := parentPath(file.Path); parent != ""; parent = parentPath(parent) {
			folded := strings.ToLower(parent)
			if other, ok := directories[folded]; ok {
				if other != parent {
					return fmt.Errorf("files: directory %s differs only in case from %s", safeQuote(parent), safeQuote(other))
				}
				continue
			}
			directories[folded] = parent
		}
	}
	return nil
}

// validateAgainstBase checks every replacement against the pinned base tree: the
// whole directory chain must be real directories, an existing target must be a
// regular file whose bytes match the declared preimage digest, and a new target
// must not exist at all.
func validateAgainstBase(ctx context.Context, repo *Repo, c Candidate) error {
	commit := c.Target.BaseCommit
	if err := repo.VerifyBaseCommit(ctx, commit); err != nil {
		return err
	}
	if err := validateBaseDirectories(ctx, repo, commit, c.Files); err != nil {
		return err
	}
	entries, err := repo.TreeEntries(ctx, commit, "skills/"+c.Skill)
	if err != nil {
		return err
	}
	for i, file := range c.Files {
		what := fmt.Sprintf("files[%d]", i)
		if err := ctx.Err(); err != nil {
			return err
		}
		entry, exists := entries[file.Path]
		switch {
		case exists:
			if entry.Mode != "100644" && entry.Mode != "100755" {
				return fmt.Errorf("%s: base entry %s is not a regular file (mode %s)", what, safeQuote(file.Path), entry.Mode)
			}
			if file.PreimageSHA256 == "" {
				return fmt.Errorf("%s: preimageSha256 is empty but %s exists at the base commit", what, safeQuote(file.Path))
			}
			data, err := repo.Blob(ctx, entry)
			if err != nil {
				return err
			}
			if actual := sha256Hex(data); actual != file.PreimageSHA256 {
				return fmt.Errorf("%s: declared preimage sha256 %s does not match %s at the base commit (%s)", what, file.PreimageSHA256, safeQuote(file.Path), actual)
			}
		case file.PreimageSHA256 != "":
			return fmt.Errorf("%s: preimageSha256 is set but %s does not exist at the base commit", what, safeQuote(file.Path))
		}
		// The replacement must not become a file where base content already sits
		// below it: that would silently drop existing files. The comparison folds
		// case because "Guide.md/child.md" blocks "guide.md" on a case-insensitive
		// checkout.
		lowerPrefix := strings.ToLower(file.Path) + "/"
		for basePath := range entries {
			if strings.HasPrefix(strings.ToLower(basePath), lowerPrefix) {
				return fmt.Errorf("%s: base contains %s below the replacement path, so the path cannot become a file", what, safeQuote(basePath))
			}
		}
	}
	return nil
}

// validateBaseDirectories checks every directory component a replacement writes
// into. Each component must be a directory in the pinned tree: a regular file,
// symlink or submodule above a replacement would send it somewhere else. Each
// component must also keep its exact spelling, because a base name that differs
// only in case names the same directory on a case-insensitive checkout. The
// scoped recursive listing cannot answer either question: a file or symlink at
// "skills" itself is outside the listing, and directories never appear in a
// recursive listing at all, so each parent directory is inspected directly.
func validateBaseDirectories(ctx context.Context, repo *Repo, commit string, files []File) error {
	wanted := make(map[string][]string) // parent directory -> proposed child names
	for _, file := range files {
		for current := file.Path; current != ""; current = parentPath(current) {
			parent := parentPath(current)
			wanted[parent] = append(wanted[parent], lastSegment(current))
		}
	}
	parents := make([]string, 0, len(wanted))
	for parent := range wanted {
		parents = append(parents, parent)
	}
	sort.Strings(parents)
	for _, parent := range parents {
		if err := ctx.Err(); err != nil {
			return err
		}
		if parent != "" {
			entry, exists, err := repo.Entry(ctx, commit, parent)
			if err != nil {
				return err
			}
			if exists && entry.Mode != treeMode {
				return fmt.Errorf("base path %s is not a directory (mode %s), so a replacement cannot be placed below it", safeQuote(parent), entry.Mode)
			}
			if !exists {
				// Nothing exists at this level, so there is nothing to collide with.
				continue
			}
		}
		names, err := repo.DirectoryNames(ctx, commit, parent)
		if err != nil {
			return err
		}
		label := safeQuote(parent)
		if parent == "" {
			label = "the repository root"
		}
		for _, name := range names {
			for _, component := range wanted[parent] {
				if name != component && strings.EqualFold(name, component) {
					return fmt.Errorf("base directory %s contains %s, which differs only in case from %s", label, safeQuote(name), safeQuote(component))
				}
			}
		}
	}
	return nil
}

// parentPath returns the parent of a slash-separated repository path, or "" at
// the repository root.
func parentPath(value string) string {
	if index := strings.LastIndexByte(value, '/'); index >= 0 {
		return value[:index]
	}
	return ""
}

func equalLists(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func equalEvidence(left, right []EvidenceRef) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
