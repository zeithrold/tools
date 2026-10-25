//go:build darwin && cgo

package keystore

import (
	"context"
	"errors"
	"testing"

	gokeychain "github.com/99designs/go-keychain"
	"github.com/99designs/keyring"
)

func TestDarwinGetMissingGuard(t *testing.T) {
	ctx := context.Background()
	ring := newFakeKeyring()
	store := &osStore{ring: ring, service: "dev.example.zt"}

	ring.getErr = keyring.ErrKeyNotFound
	ring.items["dev.example.zt:github.abc"] = keyring.Item{Key: "dev.example.zt:github.abc", Data: []byte("x")}
	if _, found, err := store.Get(ctx, "github.abc"); found || !errors.Is(err, ErrDenied) {
		t.Fatalf("existing item with swallowed Get error = found %v, err %v; want ErrDenied", found, err)
	}

	ring.items = make(map[string]keyring.Item)
	ring.keysErr = errors.New("locked")
	if _, found, err := store.Get(ctx, "github.abc"); found || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("metadata check error = found %v, err %v; want ErrUnavailable", found, err)
	}

	ring.getErr = nil
	ring.keysErr = nil
	if _, found, err := store.Get(ctx, "github.abc"); found || err != nil {
		t.Fatalf("truly missing item = found %v, err %v; want absent", found, err)
	}

	// A successful Get must return data without consulting Keys metadata.
	ring.items["dev.example.zt:github.abc"] = keyring.Item{Key: "dev.example.zt:github.abc", Data: []byte("present")}
	ring.keysErr = errors.New("Keys must not be called for a successful Get")
	ring.keysCalls = 0
	data, found, err := store.Get(ctx, "github.abc")
	if err != nil || !found || string(data) != "present" {
		t.Fatalf("successful Get = %q, %v, %v", data, found, err)
	}
	if ring.keysCalls != 0 {
		t.Fatalf("successful Get called Keys %d times", ring.keysCalls)
	}
}

func TestDarwinKeychainErrorClassification(t *testing.T) {
	tests := []struct {
		err  error
		want error
	}{
		{gokeychain.ErrorInteractionNotAllowed, ErrLocked},
		{gokeychain.ErrorAuthFailed, ErrDenied},
		{gokeychain.ErrorNoAccessForItem, ErrDenied},
		{gokeychain.ErrorNotAvailable, ErrUnavailable},
		{gokeychain.ErrorNoSuchKeychain, ErrUnavailable},
	}
	for _, test := range tests {
		mapped := classifyPlatformError(test.err)
		if !errors.Is(mapped, test.want) {
			t.Errorf("classify(%v) = %v, want %v", test.err, mapped, test.want)
		}
	}
}
