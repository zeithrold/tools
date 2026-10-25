package keystore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Unlocker releases a cross-process lock.
type Unlocker interface {
	Unlock() error
}

// Locker serializes credential mutations across zt processes. It is a
// coordination lock only; it never stores a token or other secret.
type Locker interface {
	Lock(ctx context.Context, key string) (Unlocker, error)
}

// LockerOptions configures a file-backed process lock. Dir defaults to
// <user-cache-dir>/zt/locks; Service scopes lock names to one keyring service.
// The lock directory is created lazily on the first Lock, so constructing a
// Locker does not touch the filesystem. A lock is never age-stolen: if a zt
// process crashes, acquisition fails closed and the error names the lock path
// for manual confirmation and cleanup.
type LockerOptions struct {
	Service string
	Dir     string
}

// NewLocker returns a cross-process lock suitable for Client.Locker. Lock files
// contain no credentials; they only prevent concurrent refresh/login/logout
// mutations from racing the native keyring's read-to-write sequence.
func NewLocker(opts LockerOptions) (Locker, error) {
	if err := validateService(opts.Service); err != nil {
		return nil, err
	}
	if opts.Dir == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("%w: no user cache directory for credential locks: %v", ErrUnavailable, err)
		}
		opts.Dir = filepath.Join(cache, "zt", "locks")
	}
	return &fileLocker{dir: opts.Dir, service: opts.Service}, nil
}

type fileLocker struct {
	dir     string
	service string
}

func (l *fileLocker) Lock(ctx context.Context, key string) (Unlocker, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(l.dir, 0o700); err != nil {
		return nil, fmt.Errorf("%w: cannot create credential lock directory: %v", ErrUnavailable, err)
	}
	sum := sha256.Sum256([]byte(l.service + "\x00" + key))
	path := filepath.Join(l.dir, hex.EncodeToString(sum[:])+".lock")
	for {
		if err := os.Mkdir(path, 0o700); err == nil {
			token, err := newLockToken()
			if err != nil {
				_ = os.RemoveAll(path)
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(path, "owner"), []byte(token), 0o600); err != nil {
				_ = os.RemoveAll(path)
				return nil, fmt.Errorf("%w: cannot record credential lock owner: %v", ErrUnavailable, err)
			}
			return &fileUnlocker{path: path, token: token}, nil
		} else if !os.IsExist(err) {
			return nil, fmt.Errorf("%w: cannot create credential lock: %v", ErrUnavailable, err)
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("%w: another zt process is using this credential (%s); retry after it exits. If no zt process is running, confirm the lock is stale and remove that path manually: %w", ErrLocked, path, ctx.Err())
		case <-timer.C:
		}
	}
}

func newLockToken() (string, error) {
	var buffer [16]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return "", fmt.Errorf("%w: cannot generate credential lock token: %v", ErrUnavailable, err)
	}
	return hex.EncodeToString(buffer[:]), nil
}

type fileUnlocker struct {
	path  string
	token string
}

func (u *fileUnlocker) Unlock() error {
	if u == nil || u.path == "" {
		return nil
	}
	owner, err := os.ReadFile(filepath.Join(u.path, "owner"))
	if err != nil || string(owner) != u.token {
		// The lock was manually removed as stale and replaced by another process.
		// Never remove a lock this unlocker does not own.
		return nil
	}
	if err := os.RemoveAll(u.path); err != nil {
		return fmt.Errorf("%w: cannot release credential lock: %v", ErrUnavailable, err)
	}
	return nil
}
