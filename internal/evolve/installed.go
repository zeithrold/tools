package evolve

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/zeithrold/tools/internal/skills"
)

// FromInstalledOptions selects one managed installation and an explicit local
// target base. Provenance and evidence remain operator declarations about the
// proposed material, not facts inferred from an installation's version string.
type FromInstalledOptions struct {
	Skill      string
	Installed  skills.Target
	Policy     string
	Upstream   string
	Root       string
	Provenance Provenance
	Evidence   []EvidenceRef
}

// FromInstalled derives complete replacements without modifying the installation
// or its lock. The entire pinned Skill must match the recorded installation
// digest before local edits can be attributed to that base. Older installations
// therefore work only with an explicitly supplied, content-matching base; a
// version label alone never establishes origin. No rebase or merge is attempted.
func FromInstalled(ctx context.Context, options FromInstalledOptions) (PrepareResult, error) {
	if err := ctx.Err(); err != nil {
		return PrepareResult{}, err
	}
	if !skillID.MatchString(options.Skill) {
		return PrepareResult{}, fmt.Errorf("from-installed: select one valid Skill id")
	}
	policy, err := LoadPolicy(options.Policy)
	if err != nil {
		return PrepareResult{}, err
	}
	if !policy.allows(options.Skill) {
		return PrepareResult{}, fmt.Errorf("from-installed: skill %s is not allowed by the operator policy", safeQuote(options.Skill))
	}
	provenance, err := normalizeProvenance(options.Provenance)
	if err != nil {
		return PrepareResult{}, err
	}
	evidence, err := normalizeEvidence(options.Evidence)
	if err != nil {
		return PrepareResult{}, err
	}
	snapshot, err := skills.ReadInstalled(options.Installed, options.Skill)
	if err != nil {
		return PrepareResult{}, fmt.Errorf("from-installed: %w", err)
	}
	repo, err := resolveUpstream(ctx, options.Upstream, options.Root, policy)
	if err != nil {
		return PrepareResult{}, err
	}
	if err := repo.VerifyBaseCommit(ctx, policy.BaseCommit); err != nil {
		return PrepareResult{}, err
	}
	prefix := "skills/" + options.Skill
	entry, exists, err := repo.Entry(ctx, policy.BaseCommit, prefix)
	if err != nil {
		return PrepareResult{}, err
	}
	if !exists || entry.Mode != treeMode {
		return PrepareResult{}, fmt.Errorf("from-installed: pinned base must contain the selected Skill directory")
	}
	entries, err := repo.TreeEntries(ctx, policy.BaseCommit, prefix)
	if err != nil {
		return PrepareResult{}, err
	}
	base := make(map[string][]byte, len(entries))
	total := 0
	for name, entry := range entries {
		if entry.Mode != "100644" {
			return PrepareResult{}, fmt.Errorf("from-installed: installation baseline must contain only non-executable regular files: %s", safeQuote(name))
		}
		data, err := repo.ReadEntry(ctx, entry, maxWorkspaceFileBytes, "installed baseline bound")
		if err != nil {
			return PrepareResult{}, err
		}
		total += len(data)
		if total > maxWorkspaceBytes {
			return PrepareResult{}, fmt.Errorf("from-installed: baseline exceeds %d bytes", maxWorkspaceBytes)
		}
		base[strings.TrimPrefix(name, prefix+"/")] = data
	}
	if _, exists := base["SKILL.md"]; !exists {
		return PrepareResult{}, fmt.Errorf("from-installed: pinned Skill has no regular SKILL.md")
	}
	if skills.ContentDigest(base) != snapshot.BaselineSHA256 {
		return PrepareResult{}, fmt.Errorf("from-installed: pinned Skill does not match the recorded installation baseline; supply a content-matching pinned base, or use prepare with explicitly reviewed replacements; no merge was attempted")
	}
	for name := range base {
		if _, exists := snapshot.Files[name]; !exists {
			return PrepareResult{}, fmt.Errorf("from-installed: deletion of %s is unsupported; no candidate was written", safeQuote(name))
		}
	}
	names := make([]string, 0, len(snapshot.Files))
	for name := range snapshot.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	value := proposal{SchemaVersion: schemaVersion, Skill: options.Skill, Provenance: provenance, Evidence: evidence}
	for _, name := range names {
		data := snapshot.Files[name]
		before, exists := base[name]
		if exists && bytes.Equal(before, data) {
			continue
		}
		file := replacement{Path: prefix + "/" + name, Content: string(data)}
		if exists {
			file.PreimageSHA256 = sha256Hex(before)
		}
		value.Files = append(value.Files, file)
	}
	if len(value.Files) == 0 {
		return PrepareResult{}, fmt.Errorf("from-installed: selected Skill has no content changes")
	}
	if err := value.normalize(); err != nil {
		return PrepareResult{}, err
	}
	return prepareProposal(ctx, repo, policy, value, options.Root)
}
