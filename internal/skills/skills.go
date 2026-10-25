// Package skills distributes complete, versioned Agent Skill directories.
//
// Every mutation happens through *os.Root handles opened on the managed tree,
// so no path used here can be swapped for a symlink between a check and a
// write: content writes, directory creation, renames and removal are all
// resolved relative to the opened directories. This needs the Go 1.25 os.Root
// methods (Rename, RemoveAll, MkdirAll, ReadFile, WriteFile).
package skills

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tools "github.com/zeithrold/tools"
)

const (
	lockName      = "zt.lock.json"
	agentsDirName = ".agents"
	// guardName is the exclusive transaction guard inside the managed tree.
	guardName = ".zt-sync.lock"
	// stagePrefix, backupPrefix, rollbackPrefix and lockTempPrefix name the
	// transient paths zt creates beside the managed Skill directories.
	stagePrefix    = ".zt-stage-"
	backupPrefix   = ".zt-backup-"
	rollbackPrefix = ".zt-rollback-"
	lockTempPrefix = ".zt-lock-"
)

// testHook is a test seam. It runs at named points inside Apply so races with a
// writer that does not take the tree guard can be reproduced deterministically:
//
//	"transaction"    while the guard is held, before the authoritative plan
//	"staged"         after a Skill directory is staged, before the recheck
//	"before-exchange" after the recheck, immediately before the exchange
var testHook func(point, skill, stage string)

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

// Target is one Skill distribution tree. zt writes only complete Skill
// directories below SkillsDir plus the digest lock at LockPath; Root bounds
// both and is the base of the relative paths reported in plans. The lock lives
// beside SkillsDir so it shares the scope directory zt guards.
type Target struct {
	Root      string
	SkillsDir string
	LockPath  string
}

// GlobalTarget returns the user-global tree for home: complete Skill
// directories in <home>/.agents/skills and the lock in
// <home>/.agents/zt.lock.json. The caller supplies home, so this package never
// resolves or depends on a real home directory.
func GlobalTarget(home string) Target {
	agents := filepath.Join(home, agentsDirName)
	return Target{
		Root:      home,
		SkillsDir: filepath.Join(agents, "skills"),
		LockPath:  filepath.Join(agents, lockName),
	}
}

func (t Target) validate() error {
	switch {
	case !filepath.IsAbs(t.Root):
		return fmt.Errorf("Skill target root must be an absolute path: %q", t.Root)
	case !filepath.IsAbs(t.SkillsDir):
		return fmt.Errorf("Skill target directory must be an absolute path: %q", t.SkillsDir)
	case !filepath.IsAbs(t.LockPath):
		return fmt.Errorf("Skill lock path must be an absolute path: %q", t.LockPath)
	}
	agents := filepath.Dir(t.SkillsDir)
	if filepath.Dir(t.LockPath) != agents {
		return fmt.Errorf("Skill lock must live beside the Skill directory, got %q", t.LockPath)
	}
	if base := filepath.Base(t.SkillsDir); base == "." || base == string(filepath.Separator) {
		return fmt.Errorf("invalid Skill target directory: %q", t.SkillsDir)
	}
	rel, err := filepath.Rel(t.Root, agents)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("Skill target escapes its root: %s", t.SkillsDir)
	}
	return nil
}

// tree holds the stable directory handles for one managed tree.
type tree struct {
	agents *os.Root
	skills *os.Root
}

// openTree opens the managed tree, creating the scope directory and the Skill
// directory when create is set. Symlinked directory components are refused.
func openTree(target Target, create bool) (*tree, error) {
	agentsDir := filepath.Dir(target.SkillsDir)
	info, err := os.Lstat(agentsDir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if !create {
			return &tree{}, nil
		}
		if err := os.MkdirAll(agentsDir, 0755); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	case info.Mode()&os.ModeSymlink != 0:
		return nil, fmt.Errorf("Skill target contains symlink: %s", agentsDir)
	case !info.IsDir():
		return nil, fmt.Errorf("Skill tree is not a directory: %s", agentsDir)
	}
	t := &tree{}
	t.agents, err = os.OpenRoot(agentsDir)
	if err != nil {
		return nil, err
	}
	if err := t.openSkills(filepath.Base(target.SkillsDir), create); err != nil {
		t.Close()
		return nil, err
	}
	return t, nil
}

func (t *tree) openSkills(name string, create bool) error {
	info, err := t.agents.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		if !create {
			return nil
		}
		if err := t.agents.Mkdir(name, 0755); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		if info, err = t.agents.Lstat(name); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Skill target contains symlink: %s", filepath.Join(t.agents.Name(), name))
	}
	if !info.IsDir() {
		return fmt.Errorf("Skill tree is not a directory: %s", filepath.Join(t.agents.Name(), name))
	}
	t.skills, err = t.agents.OpenRoot(name)
	return err
}

// Close releases the directory handles. A failed close leaves no state behind
// that a caller could act on.
func (t *tree) Close() {
	if t.skills != nil {
		_ = t.skills.Close()
	}
	if t.agents != nil {
		_ = t.agents.Close()
	}
}

// files reads one Skill directory through the tree handles and reports
// os.ErrNotExist when the directory does not exist.
func (t *tree) files(skill string) (map[string][]byte, error) {
	if t.skills == nil {
		return nil, os.ErrNotExist
	}
	return directoryFiles(t.skills, skill)
}

// Plan reports what Apply would change for the selected Skills. It only reads
// the lock and existing directories: it creates no directory, cache, guard, or
// lock, and it changes nothing outside target.
func Plan(target Target, selection []string) (PlanResult, error) {
	if err := target.validate(); err != nil {
		return PlanResult{}, err
	}
	t, err := openTree(target, false)
	if err != nil {
		return PlanResult{}, err
	}
	defer t.Close()
	lock, err := readLock(t.agents, filepath.Base(target.LockPath))
	if err != nil {
		return PlanResult{}, err
	}
	return planTree(t, target, selection, lock)
}

// planTree computes the actions for selection from the opened tree handles and
// the given lock. Every action, including unchanged and adopt, is decided from
// content read here, so Apply can call it again while holding the guard instead
// of trusting a plan that was made before the tree was locked.
func planTree(t *tree, target Target, selection []string, lock Lock) (PlanResult, error) {
	if len(selection) == 0 {
		return PlanResult{}, fmt.Errorf("no Skills selected")
	}
	result := PlanResult{SchemaVersion: 1, Actions: []Action{}}
	for _, name := range selection {
		files, err := bundledFiles(name)
		if err != nil {
			return PlanResult{}, err
		}
		desired := digest(files)
		relative, err := filepath.Rel(target.Root, filepath.Join(target.SkillsDir, name))
		if err != nil {
			return PlanResult{}, err
		}
		action := Action{Skill: name, Target: filepath.ToSlash(relative), Desired: desired}
		currentFiles, err := t.files(name)
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
				action.Action, action.Reason = "conflict", "existing Skill directory is not managed by zt"
			case action.Current != previous.SHA256:
				action.Action, action.Reason = "conflict", "managed Skill has local edits"
			default:
				action.Action = "update"
			}
		}
		result.Actions = append(result.Actions, action)
	}
	selected := make(map[string]bool, len(selection))
	for _, name := range selection {
		selected[name] = true
	}
	dropped := make([]string, 0, len(lock.Skills))
	for name := range lock.Skills {
		if !selected[name] {
			dropped = append(dropped, name)
		}
	}
	sort.Strings(dropped)
	for _, name := range dropped {
		result.Warnings = append(result.Warnings, fmt.Sprintf("previously managed Skill %s is no longer selected; zt does not remove it automatically", name))
	}
	return result, nil
}

func conflictError(plan PlanResult) error {
	for _, action := range plan.Actions {
		if action.Action == "conflict" {
			return fmt.Errorf("Skill %s: %s", action.Skill, action.Reason)
		}
	}
	return nil
}

// exchange records one replaced Skill directory.
type exchange struct {
	skill  string
	backup string // preserved previous directory, empty for an install
}

// Apply installs, updates or restores the selected Skill directories and then
// records them in the lock.
//
// The plan made before the guard is only a fast failure check: once the guard
// is held, Apply plans again from the opened tree handles and the lock it just
// read, so no action is applied or recorded from content observed earlier. Each
// replaced directory is rechecked after staging and the moved-aside copy is
// verified after the exchange before zt treats it as its own. A failure at any
// point restores the previous directories and keeps, rather than deletes,
// content zt cannot prove it wrote.
func Apply(target Target, selection []string) (PlanResult, error) {
	plan, err := Plan(target, selection)
	if err != nil {
		return plan, err
	}
	if err := conflictError(plan); err != nil {
		return plan, err
	}
	t, err := openTree(target, true)
	if err != nil {
		return plan, err
	}
	defer t.Close()
	release, err := acquireGuard(t.agents)
	if err != nil {
		return plan, err
	}
	defer release()
	lock, err := readLock(t.agents, filepath.Base(target.LockPath))
	if err != nil {
		return plan, err
	}
	if testHook != nil {
		testHook("transaction", "", "")
	}
	// Authoritative plan: the tree is locked and every action is decided from
	// the content the handles see now.
	plan, err = planTree(t, target, selection, lock)
	if err != nil {
		return plan, err
	}
	if err := conflictError(plan); err != nil {
		return plan, err
	}
	var done []exchange
	rollback := func(cause error) error {
		preserved := make([]string, 0, len(done))
		for i := len(done) - 1; i >= 0; i-- {
			entry := done[i]
			aside, err := randomName(rollbackPrefix)
			if err != nil {
				preserved = append(preserved, fmt.Sprintf("%s (%v)", entry.skill, err))
				continue
			}
			// Keep whatever the installed directory holds, including edits made
			// after zt wrote it; never delete it.
			if err := t.skills.Rename(entry.skill, aside); err != nil {
				preserved = append(preserved, fmt.Sprintf("%s (%v)", entry.skill, err))
				continue
			}
			preserved = append(preserved, filepath.Join(t.skills.Name(), aside))
			if entry.backup != "" {
				if err := t.skills.Rename(entry.backup, entry.skill); err != nil {
					preserved = append(preserved, fmt.Sprintf("%s (%v)", filepath.Join(t.skills.Name(), entry.backup), err))
				}
			}
		}
		if len(preserved) == 0 {
			return cause
		}
		return fmt.Errorf("%w; preserved content: %s", cause, strings.Join(preserved, ", "))
	}

	for _, action := range plan.Actions {
		if action.Action == "unchanged" || action.Action == "adopt" {
			// No content is written, so the lock entry is recorded only after
			// the identity pass below; nothing else is needed here.
			continue
		}
		files, err := bundledFiles(action.Skill)
		if err != nil {
			return plan, rollback(err)
		}
		stage, err := t.stage(files)
		if err != nil {
			return plan, rollback(err)
		}
		if testHook != nil {
			testHook("staged", action.Skill, stage)
		}
		// Recheck the live directory after staging, through the handles, so a
		// concurrent edit is refused instead of overwritten.
		current, err := t.files(action.Skill)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			_ = t.skills.RemoveAll(stage)
			return plan, rollback(err)
		}
		currentDigest := ""
		if err == nil {
			currentDigest = digest(current)
		}
		if currentDigest != action.Current {
			_ = t.skills.RemoveAll(stage)
			return plan, rollback(fmt.Errorf("Skill %s changed after planning", action.Skill))
		}
		if testHook != nil {
			testHook("before-exchange", action.Skill, stage)
		}
		backup, err := t.exchange(action.Skill, stage, action.Current != "")
		if err != nil {
			_ = t.skills.RemoveAll(stage)
			return plan, rollback(err)
		}
		done = append(done, exchange{skill: action.Skill, backup: backup})
		if backup != "" {
			// The moved-aside directory is only zt's own previous version when it
			// still holds the content the recheck saw. Anything else means an edit
			// landed during the exchange, and it is restored instead of deleted.
			previous, err := t.files(backup)
			if err != nil || digest(previous) != action.Current {
				return plan, rollback(fmt.Errorf("Skill %s changed during the exchange", action.Skill))
			}
		}
	}
	// unchanged and adopt write no content, so verify their identity once more
	// before the lock records them; installs and updates are their own proof.
	for _, action := range plan.Actions {
		if action.Action != "unchanged" && action.Action != "adopt" {
			continue
		}
		current, err := t.files(action.Skill)
		if err != nil || digest(current) != action.Current {
			return plan, rollback(fmt.Errorf("Skill %s changed while syncing", action.Skill))
		}
	}
	for _, action := range plan.Actions {
		lock.Skills[action.Skill] = LockedSkill{Version: tools.Version(), SHA256: action.Desired}
	}
	if err := writeLock(t.agents, filepath.Base(target.LockPath), lock); err != nil {
		return plan, rollback(err)
	}
	// The previous versions were verified above as zt's own copies; the installed
	// directories are the new state.
	for _, entry := range done {
		if entry.backup == "" {
			continue
		}
		if err := t.skills.RemoveAll(entry.backup); err != nil {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("could not remove previous version %s: %v", entry.backup, err))
		}
	}
	return plan, nil
}

// stage writes the complete Skill directory into a fresh directory inside the
// managed tree and returns its name.
func (t *tree) stage(files map[string][]byte) (string, error) {
	name, err := randomName(stagePrefix)
	if err != nil {
		return "", err
	}
	if err := t.skills.Mkdir(name, 0700); err != nil {
		return "", err
	}
	dir, err := t.skills.OpenRoot(name)
	if err != nil {
		_ = t.skills.RemoveAll(name)
		return "", err
	}
	err = writeFiles(dir, files)
	if closeErr := dir.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = t.skills.RemoveAll(name)
		return "", err
	}
	return name, nil
}

// exchange replaces the live Skill directory with the staged one and returns
// the name of the preserved previous directory, if any. Both renames are
// relative to the Skills handle, so a swapped path cannot redirect them.
func (t *tree) exchange(skill, stage string, replace bool) (string, error) {
	backup := ""
	if replace {
		name, err := randomName(backupPrefix)
		if err != nil {
			return "", err
		}
		if err := t.skills.Rename(skill, name); err != nil {
			return "", err
		}
		backup = name
	}
	if err := t.skills.Rename(stage, skill); err != nil {
		if backup != "" {
			_ = t.skills.Rename(backup, skill)
		}
		return "", err
	}
	return backup, nil
}

// acquireGuard takes the exclusive tree guard for one Apply. It fails while
// another run holds it, so two syncs cannot interleave their lock
// read/modify/write, and it is released before Apply returns.
func acquireGuard(root *os.Root) (func(), error) {
	guardPath := filepath.Join(root.Name(), guardName)
	f, err := root.OpenFile(guardName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, fs.ErrExist) {
		return nil, fmt.Errorf("another zt sync is running (guard %s); wait for it to finish, or remove the file when no other run is active", guardPath)
	}
	if err != nil {
		return nil, err
	}
	_, writeErr := fmt.Fprintf(f, "pid %d %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	if closeErr := f.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = root.Remove(guardName)
		return nil, writeErr
	}
	return func() { _ = root.Remove(guardName) }, nil
}

func randomName(prefix string) (string, error) {
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(suffix), nil
}

func bundledFiles(name string) (map[string][]byte, error) {
	base := "skills/" + name
	if _, err := fs.Stat(tools.SkillFiles, base+"/SKILL.md"); err != nil {
		return nil, fmt.Errorf("unknown bundled Skill %q", name)
	}
	files := map[string][]byte{}
	err := fs.WalkDir(tools.SkillFiles, base, func(entryPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(tools.SkillFiles, entryPath)
		if err != nil {
			return err
		}
		files[strings.TrimPrefix(entryPath, base+"/")] = data
		return nil
	})
	return files, err
}

// directoryFiles reads one Skill directory through a stable handle.
func directoryFiles(root *os.Root, skill string) (map[string][]byte, error) {
	info, err := root.Lstat(skill)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("Skill target contains symlink: %s", skill)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("Skill target is not a directory: %s", skill)
	}
	dir, err := root.OpenRoot(skill)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	files := map[string][]byte{}
	err = fs.WalkDir(dir.FS(), ".", func(entryPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("Skill contains non-regular file: %s/%s", skill, entryPath)
		}
		data, err := dir.ReadFile(entryPath)
		if err != nil {
			return err
		}
		files[entryPath] = data
		return nil
	})
	return files, err
}

// writeFiles writes a complete bundled directory through a stable handle.
func writeFiles(root *os.Root, files map[string][]byte) error {
	for name, data := range files {
		if dir := path.Dir(name); dir != "." {
			if err := root.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		if err := root.WriteFile(name, data, 0644); err != nil {
			return err
		}
	}
	return nil
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

func readLock(root *os.Root, name string) (Lock, error) {
	lock := Lock{SchemaVersion: 1, Skills: map[string]LockedSkill{}}
	if root == nil {
		return lock, nil
	}
	data, err := root.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return lock, nil
	}
	if err != nil {
		return Lock{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&lock); err != nil {
		return Lock{}, fmt.Errorf("%s: %w", name, err)
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Lock{}, fmt.Errorf("%s: trailing JSON data", name)
	}
	if lock.SchemaVersion != 1 || lock.Skills == nil {
		return Lock{}, fmt.Errorf("%s: unsupported or malformed lock", name)
	}
	return lock, nil
}

// writeLock replaces the lock through the tree handle: it writes a temporary
// file and renames it beside the lock, so the lock is never partially visible.
func writeLock(root *os.Root, name string, lock Lock) error {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := randomName(lockTempPrefix)
	if err != nil {
		return err
	}
	if err := root.WriteFile(temp, data, 0644); err != nil {
		return err
	}
	if err := root.Rename(temp, name); err != nil {
		_ = root.Remove(temp)
		return err
	}
	return nil
}
