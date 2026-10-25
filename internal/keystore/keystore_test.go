package keystore

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/99designs/keyring"
)

type fakeKeyring struct {
	mu        sync.Mutex
	items     map[string]keyring.Item
	keysErr   error
	keysCalls int
	getErr    error
	setErr    error
	removeErr error
	sets      []keyring.Item
}

func newFakeKeyring() *fakeKeyring {
	return &fakeKeyring{items: make(map[string]keyring.Item)}
}

func (f *fakeKeyring) Get(key string) (keyring.Item, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return keyring.Item{}, f.getErr
	}
	item, found := f.items[key]
	if !found {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	item.Data = append([]byte(nil), item.Data...)
	return item, nil
}

func (f *fakeKeyring) GetMetadata(string) (keyring.Metadata, error) {
	return keyring.Metadata{}, keyring.ErrMetadataNotSupported
}

func (f *fakeKeyring) Set(item keyring.Item) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sets = append(f.sets, item)
	if f.setErr != nil {
		return f.setErr
	}
	item.Data = append([]byte(nil), item.Data...)
	f.items[item.Key] = item
	return nil
}

func (f *fakeKeyring) Remove(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.removeErr != nil {
		return f.removeErr
	}
	if _, found := f.items[key]; !found {
		return keyring.ErrKeyNotFound
	}
	delete(f.items, key)
	return nil
}

func (f *fakeKeyring) Keys() ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keysCalls++
	if f.keysErr != nil {
		return nil, f.keysErr
	}
	keys := make([]string, 0, len(f.items))
	for key := range f.items {
		keys = append(keys, key)
	}
	return keys, nil
}

func TestOSStoreUsesKeyringItem(t *testing.T) {
	ring := newFakeKeyring()
	store := &osStore{ring: ring, service: "dev.example.zt"}
	ctx := context.Background()

	value := []byte("small value")
	if err := store.Set(ctx, "github.abc", value); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(ring.sets) != 1 {
		t.Fatalf("keyring sets = %d", len(ring.sets))
	}
	set := ring.sets[0]
	if set.Key != "dev.example.zt:github.abc" || string(set.Data) != string(value) || !set.KeychainNotSynchronizable {
		t.Fatalf("keyring item = %+v", set)
	}
	if strings.Contains(set.Label, string(value)) || strings.Contains(set.Description, string(value)) {
		t.Fatal("keyring metadata contains the secret value")
	}

	got, found, err := store.Get(ctx, "github.abc")
	if err != nil || !found || string(got) != string(value) {
		t.Fatalf("Get = %q, %v, %v", got, found, err)
	}
	if err := store.Delete(ctx, "github.abc"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, found, err := store.Get(ctx, "github.abc"); err != nil || found {
		t.Fatalf("Get after Delete = %v, %v; want absent", found, err)
	}
}

func TestOSStoreNamespacesKeysByService(t *testing.T) {
	ring := newFakeKeyring()
	one := &osStore{ring: ring, service: "dev.example.one"}
	two := &osStore{ring: ring, service: "dev.example.two"}
	ctx := context.Background()
	if err := one.Set(ctx, "github.abc", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := two.Set(ctx, "github.abc", []byte("two")); err != nil {
		t.Fatal(err)
	}
	if len(ring.items) != 2 {
		t.Fatalf("keyring items = %d, want 2", len(ring.items))
	}
	gotOne, found, err := one.Get(ctx, "github.abc")
	if err != nil || !found || string(gotOne) != "one" {
		t.Fatalf("one.Get = %q, %v, %v", gotOne, found, err)
	}
	gotTwo, found, err := two.Get(ctx, "github.abc")
	if err != nil || !found || string(gotTwo) != "two" {
		t.Fatalf("two.Get = %q, %v, %v", gotTwo, found, err)
	}
}

func TestOSStoreSizeLimitAndKeyValidation(t *testing.T) {
	ring := newFakeKeyring()
	store := &osStore{ring: ring, service: "dev.example.zt"}
	ctx := context.Background()

	if err := store.Set(ctx, "github.max", make([]byte, MaxSecretSize)); err != nil {
		t.Fatalf("exact-limit Set error = %v", err)
	}
	if err := store.Set(ctx, "github.abc", make([]byte, MaxSecretSize+1)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversized Set error = %v, want ErrTooLarge", err)
	}
	if len(ring.sets) != 1 {
		t.Fatalf("keyring Set calls = %d, want only the exact-limit write", len(ring.sets))
	}
	ring.items["dev.example.zt:github.abc"] = keyring.Item{Key: "dev.example.zt:github.abc", Data: make([]byte, MaxSecretSize+1)}
	if _, _, err := store.Get(ctx, "github.abc"); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversized stored item error = %v, want ErrTooLarge", err)
	}

	for _, key := range []string{"", "has space", "slash/name", "line\nbreak", strings.Repeat("a", 257)} {
		if err := store.Set(ctx, key, []byte("x")); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Set(%q) error = %v, want ErrInvalidKey", key, err)
		}
	}
}

func TestOSStoreClassifiesKeyringErrors(t *testing.T) {
	ctx := context.Background()
	ring := newFakeKeyring()
	store := &osStore{ring: ring, service: "dev.example.zt"}

	ring.keysErr = keyring.ErrNoAvailImpl
	if err := store.Available(ctx); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Available error = %v, want ErrUnavailable", err)
	}
	ring.keysErr = nil
	ring.getErr = keyring.ErrKeyNotFound
	if _, found, err := store.Get(ctx, "github.abc"); err != nil || found {
		t.Fatalf("Get missing = found %v err %v", found, err)
	}
	ring.getErr = errors.New("keychain locked")
	if _, _, err := store.Get(ctx, "github.abc"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unknown error = %v, want ErrUnavailable", err)
	}
	ring.removeErr = keyring.ErrKeyNotFound
	if err := store.Delete(ctx, "github.abc"); err != nil {
		t.Fatalf("Delete missing error = %v, want nil", err)
	}
}

func TestOSStoreNeverEchoesOpaqueBackendErrors(t *testing.T) {
	secret := "ghu_SYNTHETIC_BACKEND_SECRET"
	ctx := context.Background()
	for _, operation := range []string{"available", "get", "set", "delete"} {
		t.Run(operation, func(t *testing.T) {
			ring := newFakeKeyring()
			store := &osStore{ring: ring, service: "dev.example.zt"}
			opaque := errors.New(secret)
			var err error
			switch operation {
			case "available":
				ring.keysErr = opaque
				err = store.Available(ctx)
			case "get":
				ring.getErr = opaque
				_, _, err = store.Get(ctx, "github.abc")
			case "set":
				ring.setErr = opaque
				err = store.Set(ctx, "github.abc", []byte(secret))
			case "delete":
				ring.removeErr = opaque
				err = store.Delete(ctx, "github.abc")
			}
			if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), secret) {
				t.Fatalf("backend error was not safely classified: %v", err)
			}
		})
	}
}

func TestOpenValidatesServiceAndIsLazy(t *testing.T) {
	if _, err := Open(Options{}); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Open(empty) error = %v, want ErrInvalidKey", err)
	}
	if _, err := Open(Options{Service: "has space"}); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Open(invalid) error = %v, want ErrInvalidKey", err)
	}
	if err := supportError(); err != nil {
		t.Skipf("native backend unavailable in this build: %v", err)
	}
	store, err := Open(Options{Service: "dev.example.zt"})
	if err != nil {
		t.Fatalf("Open(valid): %v", err)
	}
	if store == nil {
		t.Fatal("Open returned nil store")
	}
}
