package keystore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileLockerExcludesConcurrentHolders(t *testing.T) {
	locker, err := NewLocker(LockerOptions{Service: "dev.example.zt", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewLocker: %v", err)
	}
	ctx := context.Background()
	first, err := locker.Lock(ctx, "github.abc")
	if err != nil {
		t.Fatalf("first Lock: %v", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if _, err := locker.Lock(waitCtx, "github.abc"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second Lock error = %v, want deadline exceeded", err)
	}
	if err := first.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	second, err := locker.Lock(ctx, "github.abc")
	if err != nil {
		t.Fatalf("Lock after Unlock: %v", err)
	}
	if err := second.Unlock(); err != nil {
		t.Fatalf("second Unlock: %v", err)
	}
	if err := second.Unlock(); err != nil {
		t.Fatalf("idempotent Unlock: %v", err)
	}
}

func TestFileLockerSeparatesKeysAndServices(t *testing.T) {
	dir := t.TempDir()
	one, err := NewLocker(LockerOptions{Service: "dev.example.one", Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	two, err := NewLocker(LockerOptions{Service: "dev.example.two", Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first, err := one.Lock(ctx, "github.abc")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Unlock()
	second, err := two.Lock(ctx, "github.abc")
	if err != nil {
		t.Fatalf("different service Lock: %v", err)
	}
	defer second.Unlock()
	third, err := one.Lock(ctx, "github.other")
	if err != nil {
		t.Fatalf("different key Lock: %v", err)
	}
	defer third.Unlock()
}

func TestFileLockerDoesNotStealOldLock(t *testing.T) {
	dir := t.TempDir()
	locker, err := NewLocker(LockerOptions{Service: "dev.example.zt", Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first, err := locker.Lock(ctx, "github.abc")
	if err != nil {
		t.Fatal(err)
	}
	verification, err := filepath.Glob(filepath.Join(dir, "*.lock"))
	if err != nil || len(verification) != 1 {
		t.Fatalf("lock files = %v, %v", verification, err)
	}
	if err := os.Chtimes(verification[0], time.Now().Add(-24*time.Hour), time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatalf("age lock: %v", err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if _, err := locker.Lock(waitCtx, "github.abc"); !errors.Is(err, ErrLocked) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("old lock was stolen or misclassified: %v", err)
	}
	if err := first.Unlock(); err != nil {
		t.Fatal(err)
	}
	second, err := locker.Lock(ctx, "github.abc")
	if err != nil {
		t.Fatalf("Lock after manual unlock: %v", err)
	}
	if err := second.Unlock(); err != nil {
		t.Fatal(err)
	}
}

func TestFileLockerOwnerTokenDoesNotRemoveReplacement(t *testing.T) {
	dir := t.TempDir()
	locker, err := NewLocker(LockerOptions{Service: "dev.example.zt", Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first, err := locker.Lock(ctx, "github.abc")
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.lock"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("lock files = %v, %v", paths, err)
	}
	// Simulate the user manually confirming and removing a stale lock.
	if err := os.RemoveAll(paths[0]); err != nil {
		t.Fatal(err)
	}
	second, err := locker.Lock(ctx, "github.abc")
	if err != nil {
		t.Fatalf("replacement Lock: %v", err)
	}
	if err := first.Unlock(); err != nil {
		t.Fatalf("stale owner Unlock: %v", err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if _, err := locker.Lock(waitCtx, "github.abc"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("replacement lock was removed by stale owner: %v", err)
	}
	if err := second.Unlock(); err != nil {
		t.Fatal(err)
	}
}

func TestNewLockerValidatesOptions(t *testing.T) {
	if _, err := NewLocker(LockerOptions{}); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("empty service error = %v, want ErrInvalidKey", err)
	}
}
