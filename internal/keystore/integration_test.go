package keystore_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/zeithrold/tools/internal/keystore"
)

// TestOSStoreIntegration exercises the real native backend on the current OS.
// It is gated because it writes to an OS credential store. Native runners set
// ZT_KEYSTORE_INTEGRATION=1 in an isolated environment:
//   - Linux: isolated D-Bus session + GNOME Keyring/KWallet with an unlocked
//     default/login collection (the adapter uses the standard "login"
//     Secret Service collection and namespaces item keys by service)
//   - Windows: disposable user profile / runner credential vault
//   - macOS: a temporary Keychain via ZT_KEYSTORE_KEYCHAIN (required here so the
//     test never writes the user's login Keychain)
//
// The test uses a random service namespace, a random key, and a fake token. It
// never talks to GitHub and never reads an existing credential. A skipped run is
// not native-backend evidence.
func TestOSStoreIntegration(t *testing.T) {
	if os.Getenv("ZT_KEYSTORE_INTEGRATION") != "1" {
		t.Skip("set ZT_KEYSTORE_INTEGRATION=1 and provide an isolated native credential store")
	}
	keychainName := os.Getenv("ZT_KEYSTORE_KEYCHAIN")
	if runtime.GOOS == "darwin" && keychainName == "" {
		t.Fatal("ZT_KEYSTORE_KEYCHAIN is required on macOS so the test cannot touch the user's login Keychain")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suffix := randomHex(t)
	service := "zt-integration-" + suffix
	key := "probe-" + suffix
	value := []byte("fake-token-" + suffix)

	store, err := keystore.Open(keystore.Options{Service: service, KeychainName: keychainName})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Available(ctx); err != nil {
		t.Fatalf("Available: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_ = store.Delete(cleanupCtx, key)
	})

	if _, found, err := store.Get(ctx, key); err != nil || found {
		t.Fatalf("initial Get = found %v, err %v; want absent", found, err)
	}
	if err := store.Set(ctx, key, value); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, found, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || !bytes.Equal(got, value) {
		t.Fatalf("Get = found %v value %q; want %q", found, got, value)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, found, err := store.Get(ctx, key); err != nil || found {
		t.Fatalf("final Get = found %v, err %v; want absent", found, err)
	}
}

func randomHex(t *testing.T) string {
	t.Helper()
	var buffer [12]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		t.Fatalf("random namespace: %v", err)
	}
	return hex.EncodeToString(buffer[:])
}
