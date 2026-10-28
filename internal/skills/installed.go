package skills

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

const (
	// maxInstalledLockBytes bounds the lock parsed here. A version 1 lock lists
	// one short entry per managed Skill, so 1 MiB is far beyond a real lock.
	maxInstalledLockBytes = 1 << 20
	// maxInstalledEntries bounds the directory entries one Skill may hold.
	maxInstalledEntries = 4096
	// maxInstalledFileBytes bounds one file inside a Skill.
	maxInstalledFileBytes = 4 << 20
	// maxInstalledTotalBytes bounds the whole captured snapshot.
	maxInstalledTotalBytes = 32 << 20
)

var (
	// skillName mirrors the shared Skill id convention used by evolve: lowercase
	// letters, digits and hyphens, starting with a letter.
	skillName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	// sha256Hex is the only recorded digest shape this API accepts: a complete
	// lowercase SHA-256, never a Git object id and never a version string.
	sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// InstalledSnapshot is one installed Skill directory plus the digest the lock
// recorded for it.
type InstalledSnapshot struct {
	// BaselineSHA256 is the digest recorded in the lock when the Skill was
	// written, not a digest recomputed from Files. A caller comparing the two
	// can tell a recorded baseline from the content that is installed now.
	BaselineSHA256 string
	// Files holds the Skill content keyed by slash-separated path relative to
	// the Skill directory, using the same keys the lock digests use. The bytes
	// are detached copies owned by the caller.
	Files map[string][]byte
}

// ContentDigest returns the directory digest of files: the same digest the lock
// records for an installed Skill.
func ContentDigest(files map[string][]byte) string {
	return digest(files)
}

// ReadInstalled reads the selected Skill directory and its lock entry from
// target.
//
// It reads that directory and that lock and nothing else: no bundled Skill is
// consulted, no target is resolved globally, nothing is written and no network
// or credential is touched. Reads go through the same stable os.Root handles
// Plan uses, so a path swapped for a symlink is refused rather than followed
// out of the managed tree, and every read is bounded so a tree read across a
// machine or process boundary cannot make the caller allocate without limit.
//
// The Skill must be a conservative lowercase id and must be managed: the lock
// must already record an entry for it whose sha256 is a complete lowercase
// digest, and the directory must exist and hold only regular files. Anything
// else, including an unmanaged directory, a missing lock entry, a malformed
// lock and a symlink at the lock, the Skill directory or a file in it, is an
// error instead of a partial snapshot.
func ReadInstalled(target Target, skill string) (InstalledSnapshot, error) {
	if err := target.validate(); err != nil {
		return InstalledSnapshot{}, err
	}
	if !skillName.MatchString(skill) {
		return InstalledSnapshot{}, fmt.Errorf("Skill name must be lowercase letters, digits and hyphens, starting with a letter: %q", skill)
	}
	t, err := openTree(target, false)
	if err != nil {
		return InstalledSnapshot{}, err
	}
	defer t.Close()
	if t.agents == nil {
		return InstalledSnapshot{}, fmt.Errorf("Skill %q is not installed: %s does not exist", skill, filepath.Dir(target.SkillsDir))
	}
	lock, err := readInstalledLock(t.agents, filepath.Base(target.LockPath))
	if err != nil {
		return InstalledSnapshot{}, err
	}
	entry, managed := lock[skill]
	if !managed {
		return InstalledSnapshot{}, fmt.Errorf("Skill %q is not managed by zt: %s records no entry for it", skill, target.LockPath)
	}
	if !sha256Hex.MatchString(entry.SHA256) {
		return InstalledSnapshot{}, fmt.Errorf("%s: Skill %q lock entry sha256 is not a complete lowercase SHA-256", target.LockPath, skill)
	}
	if entry.Version == "" {
		return InstalledSnapshot{}, fmt.Errorf("%s: Skill %q lock entry records no version", target.LockPath, skill)
	}
	if t.skills == nil {
		return InstalledSnapshot{}, fmt.Errorf("Skill %q is not installed: %s does not exist", skill, target.SkillsDir)
	}
	files, err := readInstalledFiles(t.skills, skill)
	if err != nil {
		return InstalledSnapshot{}, err
	}
	return InstalledSnapshot{BaselineSHA256: entry.SHA256, Files: files}, nil
}

// readInstalledFiles captures one Skill directory through the stable handle,
// bounded in entries and bytes. Symlinks are listed as ordinary entries,
// never followed, and are refused together with every other non-regular
// file.
func readInstalledFiles(skills *os.Root, skill string) (map[string][]byte, error) {
	info, err := skills.Lstat(skill)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("Skill target contains symlink: %s", skill)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("Skill target is not a directory: %s", skill)
	}
	dir, err := skills.OpenRoot(skill)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	opened, err := dir.Stat(".")
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, opened) {
		return nil, fmt.Errorf("Skill directory changed while opening: %s", skill)
	}
	files := map[string][]byte{}
	entries, total := 0, 0
	pending := []string{"."}
	for len(pending) > 0 {
		name := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		children, err := readInstalledDirectory(dir, name, maxInstalledEntries-entries)
		if err != nil {
			return nil, err
		}
		entries += len(children)
		for _, entry := range children {
			entryPath := filepath.Join(name, entry.Name())
			if entry.IsDir() {
				pending = append(pending, entryPath)
				continue
			}
			if !entry.Type().IsRegular() {
				return nil, fmt.Errorf("Skill %s contains non-regular file: %s", skill, entryPath)
			}
			data, err := readInstalledFile(dir, entryPath, maxInstalledFileBytes)
			if err != nil {
				return nil, err
			}
			total += len(data)
			if total > maxInstalledTotalBytes {
				return nil, fmt.Errorf("Skill %s is larger than %d bytes", skill, maxInstalledTotalBytes)
			}
			files[filepath.ToSlash(entryPath)] = data
		}
	}
	return files, nil
}

// readInstalledDirectory opens one real directory and closes it before descent.
// Unlike WalkDir, positive-size ReadDir batches never allocate for all siblings
// before the shared entry budget can reject them.
func readInstalledDirectory(root *os.Root, name string, remaining int) ([]fs.DirEntry, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("Skill directory is not a real directory: %s", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, opened) {
		return nil, fmt.Errorf("Skill directory changed while opening: %s", name)
	}
	return boundedInstalledEntries(file.ReadDir, remaining)
}

func boundedInstalledEntries(readDir func(int) ([]fs.DirEntry, error), remaining int) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	for {
		// Read at most one sentinel beyond the remaining budget. Passing zero
		// would mean "read everything", so even an exhausted budget needs one.
		batch, err := readDir(min(64, remaining-len(entries)+1))
		entries = append(entries, batch...)
		if len(entries) > remaining {
			return nil, fmt.Errorf("Skill holds more than %d directory entries", maxInstalledEntries)
		}
		if errors.Is(err, io.EOF) {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

// readInstalledFile reads one file through the stable handle. The entry is
// checked before and after opening, so a path replaced by a symlink or a
// special file between enumeration and the read is refused, and no read goes past
// limit.
func readInstalledFile(root *os.Root, name string, limit int) ([]byte, error) {
	path := filepath.Join(root.Name(), name)
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("Skill target contains symlink: %s", path)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Skill target is not a regular file: %s", path)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() {
		return nil, fmt.Errorf("Skill target is not a regular file: %s", path)
	}
	if !os.SameFile(info, opened) {
		return nil, fmt.Errorf("Skill file changed while opening: %s", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, limit)
	}
	return data, nil
}

// readInstalledLock reads and parses the lock entry map through the stable
// handle. The lock is a single bounded read; an absent lock is reported as an
// empty map, and every other problem is an error.
func readInstalledLock(agents *os.Root, name string) (map[string]LockedSkill, error) {
	path := filepath.Join(agents.Name(), name)
	info, err := agents.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]LockedSkill{}, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("Skill lock is a symlink: %s", path)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Skill lock is not a regular file: %s", path)
	}
	file, err := agents.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("Skill lock changed while opening: %s", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxInstalledLockBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxInstalledLockBytes {
		return nil, fmt.Errorf("Skill lock is larger than %d bytes: %s", maxInstalledLockBytes, path)
	}
	lock, err := decodeInstalledLock(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return lock, nil
}

// decodeInstalledLock parses the narrow version 1 view of the lock this file
// needs: {"schemaVersion":1,"skills":{<id>:{"version","sha256"}}}.
//
// It walks JSON tokens itself instead of decoding the shared Lock type so that
// duplicate object keys, null fields, unknown fields, wrong types and trailing
// data are refused rather than silently collapsed, ignored or defaulted. Only
// the shape is enforced here; the entry the caller selects is validated by
// ReadInstalled.
func decodeInstalledLock(data []byte) (map[string]LockedSkill, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	opening, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := opening.(json.Delim); !ok || delim != '{' {
		return nil, errors.New("lock must be a JSON object")
	}
	seen := map[string]bool{}
	var skills map[string]LockedSkill
	for dec.More() {
		field, err := decodeInstalledField(dec, seen)
		if err != nil {
			return nil, err
		}
		switch field {
		case "schemaVersion":
			value, err := dec.Token()
			if err != nil {
				return nil, err
			}
			if number, ok := value.(json.Number); !ok || number.String() != "1" {
				return nil, fmt.Errorf("lock schemaVersion must be 1, got %v", value)
			}
		case "skills":
			if skills, err = decodeInstalledSkills(dec); err != nil {
				return nil, err
			}
		}
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, errors.New("lock has trailing JSON data")
	}
	if !seen["schemaVersion"] || !seen["skills"] {
		return nil, errors.New("lock must record schemaVersion and skills")
	}
	return skills, nil
}

// decodeInstalledField reads one object key and reports whether it may appear
// once. Unknown keys are refused because the version 1 lock has no others.
func decodeInstalledField(dec *json.Decoder, seen map[string]bool) (string, error) {
	token, err := dec.Token()
	if err != nil {
		return "", err
	}
	field, ok := token.(string)
	if !ok {
		return "", errors.New("lock keys must be strings")
	}
	if seen[field] {
		return "", fmt.Errorf("lock field %q appears twice", field)
	}
	seen[field] = true
	if field != "schemaVersion" && field != "skills" {
		return "", fmt.Errorf("lock has unknown field %q", field)
	}
	return field, nil
}

// decodeInstalledSkills reads the skills object, refusing duplicate Skill keys
// so one entry can never shadow another.
func decodeInstalledSkills(dec *json.Decoder) (map[string]LockedSkill, error) {
	opening, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := opening.(json.Delim); !ok || delim != '{' {
		return nil, errors.New("lock skills must be a JSON object")
	}
	entries := map[string]LockedSkill{}
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		name, ok := token.(string)
		if !ok {
			return nil, errors.New("lock skills must be a JSON object")
		}
		if _, duplicate := entries[name]; duplicate {
			return nil, fmt.Errorf("lock records Skill %q twice", name)
		}
		entry, err := decodeInstalledEntry(dec)
		if err != nil {
			return nil, fmt.Errorf("lock Skill %q: %w", name, err)
		}
		entries[name] = entry
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return entries, nil
}

// decodeInstalledEntry reads one Skill entry. Version and sha256 must both be
// recorded as strings; neither a null nor a missing value is defaulted.
func decodeInstalledEntry(dec *json.Decoder) (LockedSkill, error) {
	opening, err := dec.Token()
	if err != nil {
		return LockedSkill{}, err
	}
	if delim, ok := opening.(json.Delim); !ok || delim != '{' {
		return LockedSkill{}, errors.New("entry must be a JSON object")
	}
	var entry LockedSkill
	seen := map[string]bool{}
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return LockedSkill{}, err
		}
		field, ok := token.(string)
		if !ok {
			return LockedSkill{}, errors.New("entry keys must be strings")
		}
		if seen[field] {
			return LockedSkill{}, fmt.Errorf("field %q appears twice", field)
		}
		seen[field] = true
		if field != "version" && field != "sha256" {
			return LockedSkill{}, fmt.Errorf("unknown field %q", field)
		}
		value, err := dec.Token()
		if err != nil {
			return LockedSkill{}, err
		}
		text, ok := value.(string)
		if !ok {
			return LockedSkill{}, fmt.Errorf("field %q must be a string", field)
		}
		if field == "version" {
			entry.Version = text
		} else {
			entry.SHA256 = text
		}
	}
	if _, err := dec.Token(); err != nil {
		return LockedSkill{}, err
	}
	if !seen["version"] || !seen["sha256"] {
		return LockedSkill{}, errors.New("entry must record version and sha256")
	}
	return entry, nil
}
