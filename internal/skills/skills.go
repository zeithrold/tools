package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tools "github.com/zeithrold/tools"
	"github.com/zeithrold/tools/internal/project"
)

const lockName = "zt.lock.json"

type Lock struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Skills        map[string]LockedSkill `json:"skills"`
}

type LockedSkill struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type Action struct {
	Skill   string `json:"skill"`
	Action  string `json:"action"` // install, update, restore, adopt, unchanged, conflict
	Target  string `json:"target"`
	Current string `json:"current,omitempty"`
	Desired string `json:"desired"`
	Reason  string `json:"reason,omitempty"`
}

type PlanResult struct {
	SchemaVersion int      `json:"schemaVersion"`
	Actions       []Action `json:"actions"`
	Warnings      []string `json:"warnings,omitempty"`
}

func Plan(root string, config *project.Config) (PlanResult, error) {
	if config == nil {
		return PlanResult{}, fmt.Errorf("zt.json is required for skill sync")
	}
	lock, err := readLock(root)
	if err != nil {
		return PlanResult{}, err
	}
	result := PlanResult{SchemaVersion: 1, Actions: []Action{}}
	for _, name := range config.Skills {
		files, err := bundledFiles(name)
		if err != nil {
			return PlanResult{}, err
		}
		desired := digest(files)
		target := filepath.Join(root, ".agents", "skills", name)
		if err := noSymlinkAncestors(root, target); err != nil {
			return PlanResult{}, err
		}
		action := Action{Skill: name, Target: filepath.ToSlash(filepath.Join(".agents", "skills", name)), Desired: desired}
		currentFiles, err := directoryFiles(target)
		if errors.Is(err, os.ErrNotExist) {
			if _, managed := lock.Skills[name]; managed {
				action.Action = "restore"
			} else {
				action.Action = "install"
			}
		} else if err != nil {
			return PlanResult{}, err
		} else {
			action.Current = digest(currentFiles)
			previous, managed := lock.Skills[name]
			switch {
			case action.Current == desired && managed && previous.SHA256 == desired:
				action.Action = "unchanged"
			case action.Current == desired:
				action.Action = "adopt"
			case !managed:
				action.Action, action.Reason = "conflict", "existing project skill is not managed by zt"
			case action.Current != previous.SHA256:
				action.Action, action.Reason = "conflict", "managed skill has local edits"
			default:
				action.Action = "update"
			}
		}
		result.Actions = append(result.Actions, action)
	}
	for name := range lock.Skills {
		found := false
		for _, selected := range config.Skills {
			if name == selected {
				found = true
				break
			}
		}
		if !found {
			result.Warnings = append(result.Warnings, fmt.Sprintf("previously managed skill %s is no longer selected; zt does not remove it automatically", name))
		}
	}
	sort.Strings(result.Warnings)
	return result, nil
}

// Apply stages every selected directory before changing the project. It refuses
// collisions and rolls back moved directories if a later operation fails.
func Apply(root string, config *project.Config) (PlanResult, error) {
	plan, err := Plan(root, config)
	if err != nil {
		return plan, err
	}
	for _, action := range plan.Actions {
		if action.Action == "conflict" {
			return plan, fmt.Errorf("skill %s: %s", action.Skill, action.Reason)
		}
	}
	lock, err := readLock(root)
	if err != nil {
		return plan, err
	}
	parent := filepath.Join(root, ".agents", "skills")
	if err := noSymlinkAncestors(root, parent); err != nil {
		return plan, err
	}
	if err := os.MkdirAll(parent, 0755); err != nil {
		return plan, err
	}
	type move struct{ target, backup string }
	var moved []move
	rollback := func() {
		for i := len(moved) - 1; i >= 0; i-- {
			_ = os.RemoveAll(moved[i].target)
			if moved[i].backup != "" {
				_ = os.Rename(moved[i].backup, moved[i].target)
			}
		}
	}
	for _, action := range plan.Actions {
		lock.Skills[action.Skill] = LockedSkill{Version: tools.BundleVersion, SHA256: action.Desired}
		if action.Action == "unchanged" || action.Action == "adopt" {
			continue
		}
		target := filepath.Join(parent, action.Skill)
		// Recheck immediately before mutation; do not overwrite a concurrent edit.
		current, err := directoryFiles(target)
		if err == nil && digest(current) != action.Current {
			rollback()
			return plan, fmt.Errorf("skill %s changed after planning", action.Skill)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			rollback()
			return plan, err
		}
		if errors.Is(err, os.ErrNotExist) && action.Current != "" {
			rollback()
			return plan, fmt.Errorf("skill %s disappeared after planning", action.Skill)
		}
		staged, err := os.MkdirTemp(parent, ".zt-stage-")
		if err != nil {
			rollback()
			return plan, err
		}
		files, err := bundledFiles(action.Skill)
		if err == nil {
			for name, data := range files {
				path := filepath.Join(staged, filepath.FromSlash(name))
				if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					break
				}
				if err = os.WriteFile(path, data, 0644); err != nil {
					break
				}
			}
		}
		if err != nil {
			_ = os.RemoveAll(staged)
			rollback()
			return plan, err
		}
		backup := ""
		if action.Current != "" {
			backup, err = os.MkdirTemp(parent, ".zt-backup-")
			if err != nil {
				_ = os.RemoveAll(staged)
				rollback()
				return plan, err
			}
			_ = os.Remove(backup)
			if err = os.Rename(target, backup); err != nil {
				_ = os.RemoveAll(staged)
				rollback()
				return plan, err
			}
		}
		if err = os.Rename(staged, target); err != nil {
			_ = os.RemoveAll(staged)
			if backup != "" {
				_ = os.Rename(backup, target)
			}
			rollback()
			return plan, err
		}
		moved = append(moved, move{target: target, backup: backup})
	}
	if err := writeLock(root, lock); err != nil {
		rollback()
		return plan, err
	}
	for _, move := range moved {
		if move.backup != "" {
			_ = os.RemoveAll(move.backup)
		}
	}
	return plan, nil
}

func bundledFiles(name string) (map[string][]byte, error) {
	base := "skills/" + name
	if _, err := fs.Stat(tools.SkillFiles, base+"/SKILL.md"); err != nil {
		return nil, fmt.Errorf("unknown bundled skill %q", name)
	}
	files := map[string][]byte{}
	err := fs.WalkDir(tools.SkillFiles, base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(tools.SkillFiles, path)
		if err != nil {
			return err
		}
		files[strings.TrimPrefix(path, base+"/")] = data
		return nil
	})
	return files, err
}

func directoryFiles(dir string) (map[string][]byte, error) {
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skill target is not a directory: %s", dir)
	}
	files := map[string][]byte{}
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("skill contains non-regular file: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	return files, err
}

func digest(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		_, _ = io.WriteString(h, fmt.Sprintf("%d:%s%d:", len(name), name, len(files[name])))
		_, _ = h.Write(files[name])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func readLock(root string) (Lock, error) {
	lock := Lock{SchemaVersion: 1, Skills: map[string]LockedSkill{}}
	f, err := os.Open(filepath.Join(root, lockName))
	if errors.Is(err, os.ErrNotExist) {
		return lock, nil
	}
	if err != nil {
		return lock, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&lock); err != nil {
		return Lock{}, fmt.Errorf("%s: %w", lockName, err)
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Lock{}, fmt.Errorf("%s: trailing JSON data", lockName)
	}
	if lock.SchemaVersion != 1 || lock.Skills == nil {
		return Lock{}, fmt.Errorf("%s: unsupported or malformed lock", lockName)
	}
	return lock, nil
}

func writeLock(root string, lock Lock) error {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(root, ".zt-lock-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(root, lockName))
}

func noSymlinkAncestors(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("target escapes project root: %s", target)
	}
	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("sync target contains symlink: %s", current)
		}
	}
	return nil
}
