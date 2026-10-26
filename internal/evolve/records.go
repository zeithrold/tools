package evolve

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Owned record paths below .zt/artifacts/evolve.
//
// They sit beside the candidate directories, never inside one: a candidate's
// bytes are its identity, so the mutable observations about it (verification
// results, the submission journal) must not be able to change it. Every path
// here is derived from content digests, so a replayed run finds the same record
// and a changed input looks for a different one.
const (
	checkRecordRoot  = "checks"
	submitRecordRoot = "submit"
)

// checkRunRelative is the owned directory of one verification of one candidate
// under one validation contract.
func checkRunRelative(candidateID, validationDigest string) string {
	return path.Join(checkRecordRoot, candidateID, validationDigest)
}

// submitJournalRelative is the owned journal of one candidate submitted under
// one public payload.
func submitJournalRelative(candidateID, payloadDigest string) string {
	return path.Join(submitRecordRoot, candidateID+"-"+payloadDigest+".json")
}

// readOwned reads one bounded regular file below the owned tree. A missing file
// is reported as (nil, false, nil); a symlink, non-regular file or oversize file
// is refused rather than followed or truncated.
func (s *Store) readOwned(relative string, limit int64) ([]byte, bool, error) {
	if s.evolve == nil {
		return nil, false, nil
	}
	info, err := s.evolve.Lstat(relative)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", relative, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, false, fmt.Errorf("refusing symlinked artifact %s", relative)
	}
	if !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("artifact %s is not a regular file", relative)
	}
	if info.Size() > limit {
		return nil, false, fmt.Errorf("artifact %s exceeds %d bytes", relative, limit)
	}
	file, err := s.evolve.Open(relative)
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", relative, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", relative, err)
	}
	if !os.SameFile(info, opened) {
		return nil, false, fmt.Errorf("artifact %s changed while it was opened", relative)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", relative, err)
	}
	if int64(len(data)) > limit {
		return nil, false, fmt.Errorf("artifact %s exceeds %d bytes", relative, limit)
	}
	return data, true, nil
}

// listOwned names the regular files directly inside one owned directory. A missing
// directory, or an unopened store, is an empty listing rather than an error: status
// reports what exists without creating anything.
func (s *Store) listOwned(relative string) ([]string, error) {
	if s.evolve == nil {
		return nil, nil
	}
	handle, err := s.evolve.Open(relative)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", relative, err)
	}
	defer handle.Close()
	info, err := handle.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("artifact %s is not a directory", relative)
	}
	entries, err := handle.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", relative, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if len(names) >= 512 {
			return nil, fmt.Errorf("artifact directory %s holds more than 512 records", relative)
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

// openExistingStore opens the owned artifact tree for reading only. A workspace
// without that tree is not an error and is not modified: status must be able to
// report an unprepared workspace without creating anything.
func openExistingStore(workspace string) (*Store, error) {
	if strings.TrimSpace(workspace) == "" {
		return nil, fmt.Errorf("--root requires an existing local workspace directory")
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace is not a directory: %s", safeQuote(abs))
	}
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}
	dir := filepath.Join(root, filepath.FromSlash(artifactRoot))
	entry, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return &Store{dir: dir}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("artifact path %s: %w", artifactRoot, err)
	}
	if entry.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing symlinked artifact path %s", artifactRoot)
	}
	if !entry.IsDir() {
		return nil, fmt.Errorf("artifact path %s is not a directory", artifactRoot)
	}
	handle, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &Store{evolve: handle, dir: dir}, nil
}

// writeOwned writes one owned record atomically: a temporary sibling inside the
// owned tree is renamed over the target, so a reader never sees a half-written
// journal or result and an interrupted write leaves only a `.tmp-*` name.
func (s *Store) writeOwned(ctx context.Context, relative string, data []byte) error {
	if s.evolve == nil {
		return fmt.Errorf("the owned artifact tree is not open for writing")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	parent := path.Dir(relative)
	if parent != "." && parent != "" {
		if err := s.evolve.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("create %s: %w", parent, err)
		}
	}
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return err
	}
	temporary := path.Join(parent, ".tmp-"+hex.EncodeToString(suffix))
	if parent == "." {
		temporary = ".tmp-" + hex.EncodeToString(suffix)
	}
	if err := s.evolve.WriteFile(temporary, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", relative, err)
	}
	if err := ctx.Err(); err != nil {
		_ = s.evolve.Remove(temporary)
		return err
	}
	if err := s.evolve.Rename(temporary, relative); err != nil {
		_ = s.evolve.Remove(temporary)
		return fmt.Errorf("install %s: %w", relative, err)
	}
	return nil
}
