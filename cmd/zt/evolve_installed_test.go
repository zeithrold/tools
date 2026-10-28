package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zeithrold/tools/internal/evolve"
	"github.com/zeithrold/tools/internal/skills"
)

func TestEvolveFromInstalledCLI(t *testing.T) {
	f := newEvolveFixture(t)
	home := t.TempDir()
	target := skills.GlobalTarget(home)
	writeFile(t, filepath.Join(target.SkillsDir, "fixture-skill", "SKILL.md"), "# Updated\n")
	writeFile(t, filepath.Join(target.SkillsDir, "fixture-skill", "references", "notes.md"), "# Notes\n")
	// Independent lock framing, in sorted path order.
	framed := fmt.Sprintf("%d:%s%d:%s%d:%s%d:%s",
		len("SKILL.md"), "SKILL.md", len(evolveSkillMarkdown), evolveSkillMarkdown,
		len("references/notes.md"), "references/notes.md", len("# Notes\n"), "# Notes\n")
	lock := fmt.Sprintf(`{"schemaVersion":1,"skills":{"fixture-skill":{"version":"test","sha256":"%x"}}}`, sha256.Sum256([]byte(framed)))
	writeFile(t, target.LockPath, lock)
	args := []string{"evolve", "from-installed", "--skill", "fixture-skill", "--policy", f.policy,
		"--upstream", f.upstream, "--root", f.root, "--source-repository", "example/upstream",
		"--source-commit", f.commit, "--license", "Apache-2.0", "--reason", "Clarify the gate.", "--json"}
	before := snapshotFiles(t, home)
	upstreamBefore := snapshotFiles(t, f.upstream)
	stdout := &bytes.Buffer{}
	a := newApp(stdout, &bytes.Buffer{})
	a.homeDir = func() (string, error) { return home, nil }
	a.newAuth = func(string) (authService, error) {
		t.Fatal("from-installed requested authentication")
		return nil, nil
	}
	if err := executeApp(a, context.Background(), args); err != nil {
		t.Fatal(err)
	}
	var result evolve.PrepareResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Created || len(result.Files) != 1 || result.Files[0].Path != "skills/fixture-skill/SKILL.md" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !reflect.DeepEqual(before, snapshotFiles(t, home)) || !reflect.DeepEqual(upstreamBefore, snapshotFiles(t, f.upstream)) {
		t.Fatal("from-installed changed installation or upstream")
	}
	candidate := filepath.Join(f.root, filepath.FromSlash(result.CandidatePath))
	if _, err := runEvolveCLI(t, f.planArgs(candidate)...); err != nil {
		t.Fatal(err)
	}
}

func TestEvolveFromInstalledMissingFlagsBeforeHome(t *testing.T) {
	_, err := runEvolveCLI(t, "evolve", "from-installed", "--skill", "fixture-skill")
	if err == nil || !strings.Contains(err.Error(), "requires --policy") {
		t.Fatalf("unexpected missing flag error: %v", err)
	}
}
