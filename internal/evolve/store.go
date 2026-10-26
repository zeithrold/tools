package evolve

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	// artifactRoot is the only directory prepare owns, relative to the selected
	// workspace.
	artifactRoot  = ".zt/artifacts/evolve"
	candidateName = "candidate.json"
	// stagePrefix marks the transient directories prepare creates beside its
	// candidates. An interrupted run leaves one of these, never a half-written
	// candidate and never a deleted user file.
	stagePrefix = ".prepare-"
)

// Store is the owned candidate tree below one workspace. Every operation is
// relative to an os.Root handle, so a path component swapped for a symlink between
// a check and a write cannot redirect it. dir is the resolved absolute path of the
// same directory: a verification workspace must hand a real path to a native
// command, and the caller may not derive it from an unresolved argument.
type Store struct {
	evolve *os.Root
	dir    string
}

// openStore opens the owned artifact tree below workspace, creating it when it
// does not exist. The workspace itself must already exist: root selects a local
// workspace, it never creates one.
func openStore(workspace string) (*Store, error) {
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
	// Resolve aliases of the selected workspace once, so the artifact path below
	// is the real one and a later path comparison cannot be fooled by them.
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}
	handle, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	handles := []*os.Root{handle}
	closeAll := func() {
		for _, opened := range handles {
			_ = opened.Close()
		}
	}
	display := ""
	for _, name := range []string{".zt", "artifacts", "evolve"} {
		next, err := openChild(handles[len(handles)-1], name, path.Join(display, name))
		if err != nil {
			closeAll()
			return nil, err
		}
		handles = append(handles, next)
		display = path.Join(display, name)
	}
	// Only the deepest handle is needed from here on.
	for _, opened := range handles[:len(handles)-1] {
		_ = opened.Close()
	}
	return &Store{evolve: handles[len(handles)-1], dir: filepath.Join(root, filepath.FromSlash(artifactRoot))}, nil
}

// openChild opens name inside parent, creating the directory when it does not
// exist. A symlink or non-directory component is refused instead of followed.
func openChild(parent *os.Root, name, display string) (*os.Root, error) {
	info, err := parent.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		if err := parent.Mkdir(name, 0755); err != nil {
			return nil, fmt.Errorf("create %s: %w", display, err)
		}
		if info, err = parent.Lstat(name); err != nil {
			return nil, fmt.Errorf("create %s: %w", display, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("artifact path %s: %w", display, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing symlinked artifact path %s", display)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("artifact path %s is not a directory", display)
	}
	return parent.OpenRoot(name)
}

// Close releases the owned tree handle.
func (s *Store) Close() error {
	if s == nil || s.evolve == nil {
		return nil
	}
	return s.evolve.Close()
}

// candidateRelative is the workspace-relative path of one stored candidate.
func candidateRelative(id string) string {
	return path.Join(artifactRoot, id, candidateName)
}

// install writes data as the immutable candidate id and reports whether the
// directory was created. Repeating prepare for the same inputs finds the same
// candidate and writes nothing; a directory that already holds different bytes
// is never overwritten.
//
// Installation is the commit point: the context is honored until the staged
// directory is renamed into place, and a canceled run leaves no candidate.
func (s *Store) install(ctx context.Context, id string, data []byte) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	installed, err := s.matches(ctx, id, data)
	if err != nil {
		return false, err
	}
	if installed {
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	stage, err := s.stage(data)
	if err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		_ = s.evolve.RemoveAll(stage)
		return false, err
	}
	if err := s.evolve.Rename(stage, id); err != nil {
		// A concurrent prepare may have installed the same candidate first, which
		// is the only acceptable outcome for this name.
		installed, matchErr := s.matches(ctx, id, data)
		_ = s.evolve.RemoveAll(stage)
		if matchErr == nil && installed {
			return false, nil
		}
		return false, fmt.Errorf("install candidate %s: %w", id, err)
	}
	return true, nil
}

// matches reports whether the owned candidate id already holds exactly data. The
// existing candidate is read the way plan reads it: only a regular file of
// bounded size is opened. A symlink or FIFO at that path is refused instead of
// followed or waited on, so a replaced candidate cannot redirect the read or
// block a canceled run.
func (s *Store) matches(ctx context.Context, id string, data []byte) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	display := path.Join(artifactRoot, id)
	info, err := s.evolve.Lstat(id)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("candidate path %s: %w", display, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("refusing symlinked candidate path %s", display)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("candidate path %s exists and is not a directory", display)
	}
	directory, err := s.evolve.OpenRoot(id)
	if err != nil {
		return false, fmt.Errorf("candidate path %s: %w", display, err)
	}
	defer directory.Close()
	entry, err := directory.Lstat(candidateName)
	if errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("owned candidate directory %s holds no %s; refusing to replace it", display, candidateName)
	}
	if err != nil {
		return false, fmt.Errorf("candidate %s: %w", id, err)
	}
	if entry.Mode()&os.ModeSymlink != 0 || !entry.Mode().IsRegular() {
		return false, fmt.Errorf("owned candidate %s holds a non-regular %s; refusing to read it", display, candidateName)
	}
	if entry.Size() > maxCandidateBytes {
		return false, fmt.Errorf("owned candidate %s exceeds %d bytes", display, maxCandidateBytes)
	}
	file, err := directory.OpenFile(candidateName, os.O_RDONLY, 0)
	if err != nil {
		return false, fmt.Errorf("candidate %s: %w", id, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return false, fmt.Errorf("candidate %s: %w", id, err)
	}
	if !os.SameFile(entry, opened) {
		return false, fmt.Errorf("owned candidate %s changed while it was opened", display)
	}
	current, err := io.ReadAll(io.LimitReader(file, maxCandidateBytes+1))
	if err != nil {
		return false, fmt.Errorf("candidate %s: %w", id, err)
	}
	if int64(len(current)) > maxCandidateBytes {
		return false, fmt.Errorf("owned candidate %s exceeds %d bytes", display, maxCandidateBytes)
	}
	if !bytes.Equal(current, data) {
		return false, fmt.Errorf("candidate %s exists with different content; candidates are immutable, so inspect and remove that directory by hand", id)
	}
	return true, nil
}

// stage writes data into a fresh, identifiable directory below the owned tree and
// returns its name.
func (s *Store) stage(data []byte) (string, error) {
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return "", err
	}
	name := stagePrefix + hex.EncodeToString(suffix)
	if err := s.evolve.Mkdir(name, 0755); err != nil {
		return "", fmt.Errorf("create stage %s/%s: %w", artifactRoot, name, err)
	}
	directory, err := s.evolve.OpenRoot(name)
	if err != nil {
		// Only the directory this run just created under its own random name is
		// removed.
		_ = s.evolve.RemoveAll(name)
		return "", err
	}
	writeErr := directory.WriteFile(candidateName, data, 0644)
	closeErr := directory.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = s.evolve.RemoveAll(name)
		return "", writeErr
	}
	return name, nil
}
