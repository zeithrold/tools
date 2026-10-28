package skills

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeManagedFixture writes one installed Skill directory plus the version 1
// lock that records it, both by hand. Fixtures are synthetic: no test here
// touches a real home directory, a credential or a bundled Skill.
func writeManagedFixture(t *testing.T, target Target, skill string, files map[string][]byte, version string) string {
	t.Helper()
	dir := filepath.Join(target.SkillsDir, skill)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, data := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	recorded := ContentDigest(files)
	writeLockEntries(t, target, map[string]string{skill: recorded}, version)
	return recorded
}

// writeLockEntries writes a version 1 lock recording one digest per Skill.
func writeLockEntries(t *testing.T, target Target, digests map[string]string, version string) {
	t.Helper()
	skills := map[string]any{}
	for name, recorded := range digests {
		skills[name] = map[string]string{"version": version, "sha256": recorded}
	}
	data, err := json.MarshalIndent(map[string]any{"schemaVersion": 1, "skills": skills}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeRawLock(t, target, string(data)+"\n")
}

// writeRawLock writes the lock bytes verbatim so a test can present a shape the
// package must refuse.
func writeRawLock(t *testing.T, target Target, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target.LockPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target.LockPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// treeState records paths, modes, modification times and content below root so
// a test can prove a call left the fixture untouched.
func treeState(t *testing.T, root string) map[string]string {
	t.Helper()
	state := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			state[relative] = fmt.Sprintf("dir %s %d", entry.Type(), info.ModTime().UnixNano())
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		state[relative] = fmt.Sprintf("file %s %d %s", entry.Type(), info.ModTime().UnixNano(), ContentDigest(map[string][]byte{"content": data}))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

// readError runs ReadInstalled expecting a failure whose message contains want.
func readError(t *testing.T, target Target, skill, want string) {
	t.Helper()
	snapshot, err := ReadInstalled(target, skill)
	if err == nil {
		t.Fatalf("%s was read: %+v", skill, snapshot)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("%s error %q does not mention %q", skill, err, want)
	}
	if snapshot.Files != nil || snapshot.BaselineSHA256 != "" {
		t.Fatalf("%s returned a partial snapshot with an error: %+v", skill, snapshot)
	}
}

func TestReadInstalledCapturesBytesAndRecordedBaseline(t *testing.T) {
	target, _ := newTarget(t)
	files := map[string][]byte{
		"SKILL.md":                []byte("---\nname: ui-design\n---\nfixture body\n"),
		"references/checklist.md": []byte("reference content\n"),
		"LICENSE":                 []byte("synthetic license\n"),
	}
	recorded := writeManagedFixture(t, target, "ui-design", files, "0.4.1")

	snapshot, err := ReadInstalled(target, "ui-design")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.BaselineSHA256 != recorded {
		t.Fatalf("baseline %q, want the recorded digest %q", snapshot.BaselineSHA256, recorded)
	}
	if !reflect.DeepEqual(snapshot.Files, files) {
		t.Fatalf("files %v, want %v", snapshot.Files, files)
	}
	if got := ContentDigest(snapshot.Files); got != recorded {
		t.Fatalf("ContentDigest(Files) %q, want %q", got, recorded)
	}

	// The captured bytes belong to the caller: editing them must not reach the
	// tree, and a second read must report the installed content again.
	snapshot.Files["SKILL.md"][0] = 'X'
	snapshot.Files["LICENSE"] = []byte("rewritten")
	delete(snapshot.Files, "references/checklist.md")
	onsidek, err := os.ReadFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(onsidek, files["SKILL.md"]) {
		t.Fatalf("editing the snapshot changed the directory: %q", onsidek)
	}
	again, err := ReadInstalled(target, "ui-design")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again.Files, files) {
		t.Fatalf("second read %v, want %v", again.Files, files)
	}
}

func TestReadInstalledReportsRecordedBaselineNotCurrentDigest(t *testing.T) {
	target, _ := newTarget(t)
	files := map[string][]byte{"SKILL.md": []byte("installed now\n")}
	writeManagedFixture(t, target, "ui-design", files, "0.4.1")
	// A lock digest that is well formed but does not describe the directory: the
	// baseline must be reported as recorded, so the caller can detect drift.
	recordedElsewhere := strings.Repeat("a", 64)
	writeLockEntries(t, target, map[string]string{"ui-design": recordedElsewhere}, "0.4.1")

	snapshot, err := ReadInstalled(target, "ui-design")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.BaselineSHA256 != recordedElsewhere {
		t.Fatalf("baseline %q, want the recorded %q", snapshot.BaselineSHA256, recordedElsewhere)
	}
	if ContentDigest(snapshot.Files) == snapshot.BaselineSHA256 {
		t.Fatal("baseline was recomputed from the installed content")
	}
}

func TestReadInstalledWritesNothing(t *testing.T) {
	target, home := newTarget(t)
	files := map[string][]byte{
		"SKILL.md": []byte("fixture\n"),
		"LICENSE":  []byte("synthetic license\n"),
	}
	writeManagedFixture(t, target, "ui-design", files, "0.4.1")
	// An unrelated tree entry proves the read does not rewrite or tidy siblings.
	if err := os.WriteFile(filepath.Join(target.SkillsDir, "notes.txt"), []byte("sibling\n"), 0644); err != nil {
		t.Fatal(err)
	}
	before := treeState(t, home)
	if _, err := ReadInstalled(target, "ui-design"); err != nil {
		t.Fatal(err)
	}
	after := treeState(t, home)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("read changed the tree:\nbefore %v\nafter  %v", before, after)
	}
	for _, transient := range []string{guardName, "skills/" + stagePrefix, "skills/" + backupPrefix} {
		if _, err := os.Lstat(filepath.Join(home, ".agents", transient)); err == nil {
			t.Fatalf("read created %s", transient)
		}
	}
}

func TestReadInstalledIgnoresUnrelatedTreeEntries(t *testing.T) {
	target, _ := newTarget(t)
	files := map[string][]byte{"SKILL.md": []byte("managed\n")}
	recorded := writeManagedFixture(t, target, "ui-design", files, "0.4.1")
	// Unmanaged siblings and a sibling symlink are none of this read's business:
	// it selects one directory and never enumerates the tree globally.
	if err := os.MkdirAll(filepath.Join(target.SkillsDir, "project-ui"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(target.SkillsDir, "elsewhere")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	snapshot, err := ReadInstalled(target, "ui-design")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.BaselineSHA256 != recorded || !reflect.DeepEqual(snapshot.Files, files) {
		t.Fatalf("snapshot %+v, want %v", snapshot, files)
	}
}

func TestReadInstalledRejectsInvalidSkillNames(t *testing.T) {
	target, home := newTarget(t)
	for _, skill := range []string{
		"", "UI-design", "Ui-design", "1ui-design", "-ui-design", "ui design",
		"ui_design", "ui.design", "ui/design", "ui/../design", "..", ".", "a/b",
		"ui-design/", "ui\x00design",
	} {
		readError(t, target, skill, "lowercase letters, digits and hyphens")
	}
	if _, err := os.Stat(filepath.Join(home, agentsDirName)); !os.IsNotExist(err) {
		t.Fatalf("name validation touched the tree: %v", err)
	}
}

func TestReadInstalledRejectsUnmanagedAndMissing(t *testing.T) {
	t.Run("no lock at all", func(t *testing.T) {
		target, _ := newTarget(t)
		if err := os.MkdirAll(filepath.Join(target.SkillsDir, "ui-design"), 0755); err != nil {
			t.Fatal(err)
		}
		readError(t, target, "ui-design", "is not managed by zt")
	})

	t.Run("unmanaged directory", func(t *testing.T) {
		target, _ := newTarget(t)
		if err := os.MkdirAll(filepath.Join(target.SkillsDir, "ui-design"), 0755); err != nil {
			t.Fatal(err)
		}
		writeLockEntries(t, target, map[string]string{"verification": strings.Repeat("b", 64)}, "0.4.1")
		readError(t, target, "ui-design", "is not managed by zt")
	})

	t.Run("missing managed directory", func(t *testing.T) {
		target, _ := newTarget(t)
		writeLockEntries(t, target, map[string]string{"ui-design": strings.Repeat("c", 64)}, "0.4.1")
		readError(t, target, "ui-design", "is not installed")
	})

	t.Run("missing tree", func(t *testing.T) {
		target, _ := newTarget(t)
		readError(t, target, "ui-design", "is not installed")
	})

	t.Run("directory is a file", func(t *testing.T) {
		target, _ := newTarget(t)
		if err := os.MkdirAll(target.SkillsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target.SkillsDir, "ui-design"), []byte("not a directory"), 0644); err != nil {
			t.Fatal(err)
		}
		writeLockEntries(t, target, map[string]string{"ui-design": strings.Repeat("d", 64)}, "0.4.1")
		readError(t, target, "ui-design", "not a directory")
	})

	t.Run("unusable recorded digests", func(t *testing.T) {
		for _, recorded := range []string{
			"",
			strings.Repeat("A", 64),             // uppercase
			strings.Repeat("a", 63),             // truncated
			strings.Repeat("a", 65),             // too long
			strings.Repeat("g", 64),             // not hex
			"v0.4.1",                            // a version, not a digest
			strings.Repeat("a", 40),             // a SHA-1 Git object id
			strings.Repeat("a", 64) + "\n",      // trailing whitespace
			"  " + strings.Repeat("a", 64),      // leading whitespace
			strings.Repeat("a", 64) + "-dirty",  // annotated
			"sha256:" + strings.Repeat("a", 64), // labelled
		} {
			target, _ := newTarget(t)
			writeManagedFixture(t, target, "ui-design", map[string][]byte{"SKILL.md": []byte("x\n")}, "0.4.1")
			writeLockEntries(t, target, map[string]string{"ui-design": recorded}, "0.4.1")
			readError(t, target, "ui-design", "sha256")
		}
	})

	t.Run("empty recorded version", func(t *testing.T) {
		target, _ := newTarget(t)
		writeManagedFixture(t, target, "ui-design", map[string][]byte{"SKILL.md": []byte("x\n")}, "")
		readError(t, target, "ui-design", "records no version")
	})

	t.Run("relative target", func(t *testing.T) {
		readError(t, Target{Root: "relative", SkillsDir: "a/skills", LockPath: "a/zt.lock.json"}, "ui-design", "absolute")
	})
}

func TestReadInstalledRejectsSymlinks(t *testing.T) {
	t.Run("symlinked scope directory", func(t *testing.T) {
		target, home := newTarget(t)
		if err := os.Symlink(t.TempDir(), filepath.Join(home, agentsDirName)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		readError(t, target, "ui-design", "symlink")
	})

	t.Run("symlinked skill directory", func(t *testing.T) {
		target, _ := newTarget(t)
		outside := t.TempDir()
		if err := os.WriteFile(filepath.Join(outside, "SKILL.md"), []byte("outside\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(target.SkillsDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(target.SkillsDir, "ui-design")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		writeLockEntries(t, target, map[string]string{"ui-design": strings.Repeat("e", 64)}, "0.4.1")
		readError(t, target, "ui-design", "symlink")
	})

	t.Run("symlinked skills directory", func(t *testing.T) {
		target, _ := newTarget(t)
		if err := os.MkdirAll(filepath.Dir(target.SkillsDir), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(t.TempDir(), target.SkillsDir); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		writeLockEntries(t, target, map[string]string{"ui-design": strings.Repeat("f", 64)}, "0.4.1")
		readError(t, target, "ui-design", "symlink")
	})

	t.Run("symlinked lock", func(t *testing.T) {
		target, _ := newTarget(t)
		writeManagedFixture(t, target, "ui-design", map[string][]byte{"SKILL.md": []byte("x\n")}, "0.4.1")
		outside := filepath.Join(t.TempDir(), "lock.json")
		if err := os.WriteFile(outside, []byte(`{"schemaVersion":1,"skills":{}}`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(target.LockPath); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, target.LockPath); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		readError(t, target, "ui-design", "symlink")
	})

	t.Run("symlinked file in the skill", func(t *testing.T) {
		target, _ := newTarget(t)
		writeManagedFixture(t, target, "ui-design", map[string][]byte{"SKILL.md": []byte("x\n")}, "0.4.1")
		outside := filepath.Join(t.TempDir(), "secret")
		if err := os.WriteFile(outside, []byte("outside\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(target.SkillsDir, "ui-design", "leak.md")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		readError(t, target, "ui-design", "non-regular")
	})

	t.Run("symlinked directory in the skill", func(t *testing.T) {
		target, _ := newTarget(t)
		writeManagedFixture(t, target, "ui-design", map[string][]byte{"SKILL.md": []byte("x\n")}, "0.4.1")
		if err := os.Symlink(t.TempDir(), filepath.Join(target.SkillsDir, "ui-design", "references")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		readError(t, target, "ui-design", "non-regular")
	})
}

func TestInstalledEnumerationStopsAtBudget(t *testing.T) {
	for _, remaining := range []int{0, 3, 64, 70, maxInstalledEntries} {
		t.Run(fmt.Sprint(remaining), func(t *testing.T) {
			read := 0
			_, err := boundedInstalledEntries(func(n int) ([]fs.DirEntry, error) {
				if n <= 0 || n > 64 || read+n > remaining+1 {
					t.Fatalf("unbounded enumeration request: n=%d already=%d budget=%d", n, read, remaining)
				}
				read += n
				return make([]fs.DirEntry, n), nil
			}, remaining)
			if err == nil || !strings.Contains(err.Error(), "directory entries") || read != remaining+1 {
				t.Fatalf("did not stop at sentinel: read=%d err=%v", read, err)
			}
		})
	}
}

func TestInstalledEnumerationAcceptsExactBudget(t *testing.T) {
	read := 0
	entries, err := boundedInstalledEntries(func(n int) ([]fs.DirEntry, error) {
		if n <= 0 || n > 64 {
			t.Fatalf("unbounded request: %d", n)
		}
		count := min(n, 70-read)
		read += count
		if count == 0 {
			return nil, io.EOF
		}
		return make([]fs.DirEntry, count), nil
	}, 70)
	if err != nil || len(entries) != 70 {
		t.Fatalf("exact budget rejected: entries=%d err=%v", len(entries), err)
	}
}

func TestReadInstalledRejectsOversizedInput(t *testing.T) {
	t.Run("lock", func(t *testing.T) {
		target, _ := newTarget(t)
		writeManagedFixture(t, target, "ui-design", map[string][]byte{"SKILL.md": []byte("x\n")}, "0.4.1")
		writeRawLock(t, target, strings.Repeat(" ", maxInstalledLockBytes+1))
		readError(t, target, "ui-design", "larger than")
	})

	t.Run("single file", func(t *testing.T) {
		target, _ := newTarget(t)
		files := map[string][]byte{
			"SKILL.md": []byte("x\n"),
			"big.bin":  bytes.Repeat([]byte("b"), maxInstalledFileBytes+1),
		}
		writeManagedFixture(t, target, "ui-design", files, "0.4.1")
		readError(t, target, "ui-design", "larger than")
	})

	t.Run("entry count", func(t *testing.T) {
		target, _ := newTarget(t)
		files := map[string][]byte{"SKILL.md": []byte("x\n")}
		for i := 0; i < maxInstalledEntries; i++ {
			files[fmt.Sprintf("references/f%04d.md", i)] = []byte("x")
		}
		writeManagedFixture(t, target, "ui-design", files, "0.4.1")
		readError(t, target, "ui-design", "directory entries")
	})

	t.Run("total bytes", func(t *testing.T) {
		target, _ := newTarget(t)
		files := map[string][]byte{}
		chunk := bytes.Repeat([]byte("c"), maxInstalledFileBytes)
		for i := 0; i < maxInstalledTotalBytes/maxInstalledFileBytes+1; i++ {
			files[fmt.Sprintf("chunk-%02d.bin", i)] = chunk
		}
		writeManagedFixture(t, target, "ui-design", files, "0.4.1")
		readError(t, target, "ui-design", "larger than")
	})
}

func TestReadInstalledRejectsMalformedLock(t *testing.T) {
	valid := strings.Repeat("a", 64)
	entry := fmt.Sprintf(`{"version":"0.4.1","sha256":%q}`, valid)
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"empty file", "", "EOF"},
		{"not JSON", "not a lock", "invalid character"},
		{"not an object", "[]", "must be a JSON object"},
		{"schema version 2", `{"schemaVersion":2,"skills":{}}`, "schemaVersion"},
		{"schema version 1.0", `{"schemaVersion":1.0,"skills":{}}`, "schemaVersion"},
		{"schema version as string", `{"schemaVersion":"1","skills":{}}`, "schemaVersion"},
		{"schema version as bool", `{"schemaVersion":true,"skills":{}}`, "schemaVersion"},
		{"schema version as null", `{"schemaVersion":null,"skills":{}}`, "schemaVersion"},
		{"missing schema version", `{"skills":{}}`, "must record schemaVersion and skills"},
		{"missing skills", `{"schemaVersion":1}`, "must record schemaVersion and skills"},
		{"null skills", `{"schemaVersion":1,"skills":null}`, "must be a JSON object"},
		{"skills as array", `{"schemaVersion":1,"skills":[]}`, "must be a JSON object"},
		{"skills as string", `{"schemaVersion":1,"skills":"none"}`, "must be a JSON object"},
		{"unknown top-level field", `{"schemaVersion":1,"skills":{},"generatedAt":"2024-01-01"}`, "unknown field"},
		{"duplicate top-level field", `{"schemaVersion":1,"schemaVersion":1,"skills":{}}`, "appears twice"},
		{"trailing data", `{"schemaVersion":1,"skills":{}}{}`, "trailing JSON data"},
		{"duplicate skill", fmt.Sprintf(`{"schemaVersion":1,"skills":{"ui-design":%s,"ui-design":%s}}`, entry, entry), "twice"},
		{"entry not an object", `{"schemaVersion":1,"skills":{"ui-design":[]}}`, "must be a JSON object"},
		{"entry as string", `{"schemaVersion":1,"skills":{"ui-design":"0.4.1"}}`, "must be a JSON object"},
		{"null entry", `{"schemaVersion":1,"skills":{"ui-design":null}}`, "must be a JSON object"},
		{"null sha256", `{"schemaVersion":1,"skills":{"ui-design":{"version":"0.4.1","sha256":null}}}`, "must be a string"},
		{"numeric sha256", `{"schemaVersion":1,"skills":{"ui-design":{"version":"0.4.1","sha256":12345}}}`, "must be a string"},
		{"numeric version", fmt.Sprintf(`{"schemaVersion":1,"skills":{"ui-design":{"version":1,"sha256":%q}}}`, valid), "must be a string"},
		{"missing sha256", `{"schemaVersion":1,"skills":{"ui-design":{"version":"0.4.1"}}}`, "must record version and sha256"},
		{"missing version", fmt.Sprintf(`{"schemaVersion":1,"skills":{"ui-design":{"sha256":%q}}}`, valid), "must record version and sha256"},
		{"invented origin field", fmt.Sprintf(`{"schemaVersion":1,"skills":{"ui-design":{"version":"0.4.1","sha256":%q,"origin":"github.com/example"}}}`, valid), "unknown field"},
		{"duplicate entry field", fmt.Sprintf(`{"schemaVersion":1,"skills":{"ui-design":{"version":"0.4.1","sha256":%q,"sha256":%q}}}`, valid, valid), "appears twice"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			target, _ := newTarget(t)
			if err := os.MkdirAll(filepath.Join(target.SkillsDir, "ui-design"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(target.SkillsDir, "ui-design", "SKILL.md"), []byte("x\n"), 0644); err != nil {
				t.Fatal(err)
			}
			writeRawLock(t, target, test.content)
			readError(t, target, "ui-design", test.want)
		})
	}
}

func TestReadInstalledRejectsNonRegularLock(t *testing.T) {
	target, _ := newTarget(t)
	if err := os.MkdirAll(filepath.Join(target.SkillsDir, "ui-design"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target.LockPath, 0755); err != nil {
		t.Fatal(err)
	}
	readError(t, target, "ui-design", "not a regular file")
}
