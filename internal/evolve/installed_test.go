package evolve

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/zeithrold/tools/internal/skills"
)

func installedOptions(t *testing.T, h *harness) FromInstalledOptions {
	t.Helper()
	target := skills.GlobalTarget(t.TempDir())
	files := map[string]string{}
	prefix := "skills/" + fixtureSkill + "/"
	for name, content := range baseFileSet() {
		if relative, ok := strings.CutPrefix(name, prefix); ok {
			files[relative] = content
			writeFile(t, filepath.Join(target.SkillsDir, fixtureSkill, relative), content)
		}
	}
	// Compute the documented lock framing independently of skills.ContentDigest.
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var framed strings.Builder
	for _, name := range names {
		fmt.Fprintf(&framed, "%d:%s%d:%s", len(name), name, len(files[name]), files[name])
	}
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(framed.String())))
	writeJSONFile(t, target.LockPath, skills.Lock{SchemaVersion: 1, Skills: map[string]skills.LockedSkill{
		fixtureSkill: {Version: "not-a-git-commit", SHA256: digest},
	}})
	h.writePolicy(policyDoc(h.commit))
	return FromInstalledOptions{
		Skill: fixtureSkill, Installed: target, Policy: h.policy, Upstream: h.upstream, Root: h.workspace,
		Provenance: Provenance{Repository: "example/upstream", Commit: h.commit, License: "Apache-2.0", Reason: "Clarify the first gate for reviewers."},
	}
}

func TestFromInstalledMatchesExplicitPrepare(t *testing.T) {
	h := newHarness(t)
	options := installedOptions(t, h)
	options.Evidence = []EvidenceRef{{Label: "review-thread", Digest: strings.Repeat("ab", 32)}}
	newContent := "# Go testing\n\nRun the unit gate, then the focused package.\n"
	writeFile(t, filepath.Join(options.Installed.SkillsDir, fixtureSkill, "SKILL.md"), newContent)
	writeFile(t, filepath.Join(options.Installed.SkillsDir, fixtureSkill, "references/guide.md"), "# Guide\n\nA new reference.\n")
	// Uncommitted upstream edits are not the installation baseline.
	writeFile(t, filepath.Join(h.upstream, "skills", fixtureSkill, "SKILL.md"), "dirty upstream worktree\n")
	before := snapshotTree(t, options.Installed.Root)
	upstreamBefore := snapshotTree(t, h.upstream)
	result, err := FromInstalled(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	h.writeProposal(proposalDoc(h.commit, defaultProposalFiles()...))
	explicit := h.mustPrepare()
	if result.CandidateID != explicit.CandidateID {
		t.Fatalf("derived candidate %s differs from explicit %s", result.CandidateID, explicit.CandidateID)
	}
	again, err := FromInstalled(context.Background(), options)
	if err != nil || again.Created || again.CandidateID != result.CandidateID {
		t.Fatalf("not idempotent: %+v %v", again, err)
	}
	if !reflect.DeepEqual(before, snapshotTree(t, options.Installed.Root)) ||
		!reflect.DeepEqual(upstreamBefore, snapshotTree(t, h.upstream)) {
		t.Fatal("modified installation or upstream")
	}
	plan, err := Plan(context.Background(), PlanOptions{Candidate: h.candidatePath(result), Policy: h.policy, Upstream: h.upstream})
	if err != nil || plan.CandidateID != result.CandidateID || plan.Publication != publicationNotReady {
		t.Fatalf("plan: %+v %v", plan, err)
	}
}

func TestFromInstalledRejectsWithoutWriting(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*testing.T, *harness, *FromInstalledOptions)
		want   string
	}{
		{"unchanged", func(t *testing.T, h *harness, o *FromInstalledOptions) {}, "no content changes"},
		{"deleted", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			if err := os.Remove(filepath.Join(o.Installed.SkillsDir, fixtureSkill, "LICENSE")); err != nil {
				t.Fatal(err)
			}
		}, "deletion"},
		{"out-of-scope", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			writeFile(t, filepath.Join(o.Installed.SkillsDir, fixtureSkill, "script.sh"), "echo bad\n")
		}, "must be"},
		{"binary", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			writeFile(t, filepath.Join(o.Installed.SkillsDir, fixtureSkill, "SKILL.md"), "\x00")
		}, "control character"},
		{"too-large", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			writeFile(t, filepath.Join(o.Installed.SkillsDir, fixtureSkill, "SKILL.md"), strings.Repeat("a", maxFileBytes+1))
		}, "content exceeds"},
		{"too-many", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			for i := 0; i <= maxFiles; i++ {
				writeFile(t, filepath.Join(o.Installed.SkillsDir, fixtureSkill, "references", fmt.Sprintf("new-%d.md", i)), "# New\n")
			}
		}, "at most"},
		{"case-collision", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			writeFile(t, filepath.Join(o.Installed.SkillsDir, fixtureSkill, "references", "NOTES.md"), "# New\n")
			lower, err := os.Stat(filepath.Join(o.Installed.SkillsDir, fixtureSkill, "references", "notes.md"))
			if err != nil {
				t.Fatal(err)
			}
			upper, err := os.Stat(filepath.Join(o.Installed.SkillsDir, fixtureSkill, "references", "NOTES.md"))
			if err != nil {
				t.Fatal(err)
			}
			if os.SameFile(lower, upper) {
				t.Skip("host cannot represent case-colliding files")
			}
		}, "case"},
		{"unmanaged", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			if err := os.Remove(o.Installed.LockPath); err != nil {
				t.Fatal(err)
			}
		}, "managed"},
		{"disallowed", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			doc := policyDoc(h.commit)
			doc["allowedSkills"] = []string{"another-skill"}
			h.writePolicy(doc)
		}, "not allowed"},
		{"base-drift", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			writeFile(t, filepath.Join(h.upstream, "skills", fixtureSkill, "references/notes.md"), "changed upstream\n")
			runGit(t, h.upstream, "add", "--all")
			runGit(t, h.upstream, "commit", "--quiet", "-m", "drift")
			h.writePolicy(policyDoc(runGit(t, h.upstream, "rev-parse", "HEAD")))
		}, "does not match"},
		{"invalid-source", func(t *testing.T, h *harness, o *FromInstalledOptions) {
			o.Provenance.Commit = "main"
		}, "complete lowercase"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			options := installedOptions(t, h)
			tc.mutate(t, h, &options)
			homeBefore := snapshotTree(t, options.Installed.Root)
			rootBefore := snapshotTree(t, h.workspace)
			upstreamBefore := snapshotTree(t, h.upstream)
			_, err := FromInstalled(context.Background(), options)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if !reflect.DeepEqual(homeBefore, snapshotTree(t, options.Installed.Root)) ||
				!reflect.DeepEqual(rootBefore, snapshotTree(t, h.workspace)) ||
				!reflect.DeepEqual(upstreamBefore, snapshotTree(t, h.upstream)) {
				t.Fatal("rejected input changed files")
			}
		})
	}
}

func TestFromInstalledCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := FromInstalled(ctx, FromInstalledOptions{}); err != context.Canceled {
		t.Fatalf("cancellation: %v", err)
	}
}
