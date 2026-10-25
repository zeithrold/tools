// Package keystore is a thin adapter over github.com/99designs/keyring for the
// current user's native credential store.
//
// macOS uses the login Keychain (the keyring "keychain" backend, which requires
// cgo), Windows uses Credential Manager ("wincred"), and Linux uses the
// standard "login" Secret Service collection ("secret-service"). Item keys are
// namespaced by Service. Only the current platform's secure backend is in
// AllowedBackends; keyring's file, pass, keyctl, and kwallet backends are never
// allowed. There is no file, environment variable, command argument, or
// plaintext fallback: missing, locked, or denied stores fail closed.
//
// The package stores one opaque byte string per key. Callers are responsible
// for choosing a stable, non-secret key and for never logging the value they
// store. GitHub credentials belong in internal/githubauth, which owns their
// format and lifecycle.
//
// keyring's Linux package initializes a D-Bus session-bus connection in its
// package init. Importing this package therefore may perform IPC at process
// start; it does not read a credential, mint a token, or write project files.
// keystore.Open itself is lazy and does not open a keyring. CLI commands that
// must stay local (inspect/plan/check) should not call Store methods.
package keystore

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/99designs/keyring"
)

// MaxSecretSize is the largest value Set accepts. Windows Credential Manager
// limits a generic credential blob to 5*512 bytes; enforcing that limit on
// every platform keeps behavior portable and rejects oversized writes before
// any native call can partially replace an existing item.
const MaxSecretSize = 5 * 512

var (
	// ErrUnavailable means the native credential store cannot be used. The
	// message explains what is missing; callers must not substitute a
	// plaintext store.
	ErrUnavailable = errors.New("secure credential store unavailable")
	// ErrLocked means the store exists but is locked or needs interactive
	// approval. The user must unlock it; callers must not bypass the prompt.
	ErrLocked = errors.New("secure credential store locked")
	// ErrDenied means the user or the operating system refused access.
	ErrDenied = errors.New("secure credential store access denied")
	// ErrTooLarge means a value exceeds MaxSecretSize.
	ErrTooLarge = errors.New("secret exceeds secure credential store size limit")
	// ErrInvalidKey means the service name or key is empty or malformed.
	ErrInvalidKey = errors.New("invalid credential key")
)

// Store is a native credential store. Get reports false when key is absent.
// Set replaces the complete value for key atomically; implementations never
// write a partially formed value. Delete is idempotent.
//
// Implementations must not include values in returned errors.
type Store interface {
	Available(ctx context.Context) error
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
}

// Options identifies the credential service. Service is a stable, non-secret
// namespace, for example "dev.zeithrold.zt".
//
// KeychainName is optional and used only on macOS, where it selects a named or
// temporary Keychain for native CI. Production callers leave it empty to use
// the user's default login Keychain. It is a non-secret path/name, never a
// credential, and other platforms ignore it.
type Options struct {
	Service      string
	KeychainName string
}

// Open returns the native store for the current platform. It validates the
// service and platform support but does not open a keyring or read/write a
// credential; the first Store method performs the keyring open. Call
// Store.Available before an authorization that must fail before any network
// exchange.
func Open(opts Options) (Store, error) {
	if err := validateService(opts.Service); err != nil {
		return nil, err
	}
	if err := supportError(); err != nil {
		return nil, err
	}
	return &osStore{service: opts.Service, keychainName: opts.KeychainName}, nil
}

type osStore struct {
	service      string
	keychainName string

	mu   sync.Mutex
	ring keyring.Keyring
}

func (s *osStore) keyring() (keyring.Keyring, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ring != nil {
		return s.ring, nil
	}
	ring, err := openKeyring(s.service, s.keychainName)
	if err != nil {
		return nil, err
	}
	s.ring = ring
	return ring, nil
}

func (s *osStore) Available(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ring, err := s.keyring()
	if err != nil {
		return err
	}
	if err := availabilityCheck(ring, s.keychainName); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return classifyKeyringError("check secure credential store", err)
	}
	return nil
}

func (s *osStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := validateKey(key); err != nil {
		return nil, false, err
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	ring, err := s.keyring()
	if err != nil {
		return nil, false, err
	}
	itemKey := s.itemKey(key)
	item, err := ring.Get(itemKey)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		if err := confirmMissing(ring, itemKey); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, classifyKeyringError("read secure credential", err)
	}
	if len(item.Data) > MaxSecretSize {
		return nil, false, fmt.Errorf("%w: stored item is %d bytes; remove it with the operating system credential tool and log in again", ErrTooLarge, len(item.Data))
	}
	return item.Data, true, nil
}

func (s *osStore) Set(ctx context.Context, key string, value []byte) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if len(value) > MaxSecretSize {
		return fmt.Errorf("%w: %d bytes exceeds the %d byte limit", ErrTooLarge, len(value), MaxSecretSize)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ring, err := s.keyring()
	if err != nil {
		return err
	}
	item := keyring.Item{
		Key:                         s.itemKey(key),
		Data:                        value,
		Label:                       "zt secure credential",
		Description:                 "zt credential for " + s.service,
		KeychainNotSynchronizable:   true,
		KeychainNotTrustApplication: false,
	}
	if err := ring.Set(item); err != nil {
		return classifyKeyringError("write secure credential", err)
	}
	return nil
}

func (s *osStore) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ring, err := s.keyring()
	if err != nil {
		return err
	}
	if err := ring.Remove(s.itemKey(key)); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return classifyKeyringError("delete secure credential", err)
	}
	return nil
}

func (s *osStore) itemKey(key string) string {
	return s.service + ":" + key
}

func classifyKeyringError(action string, err error) error {
	switch {
	case errors.Is(err, keyring.ErrKeyNotFound):
		return fmt.Errorf("%s: credential item not found", action)
	case errors.Is(err, keyring.ErrNoAvailImpl):
		return fmt.Errorf("%w: %s: no allowed native keyring backend is available; check the platform store, cgo build, or Secret Service session (no plaintext fallback)", ErrUnavailable, action)
	}
	if mapped := classifyPlatformError(err); mapped != nil {
		return fmt.Errorf("%s: %w", action, mapped)
	}
	// Unknown backend errors may include request data. Known native error codes
	// above are actionable; never expose opaque backend text with credentials.
	return fmt.Errorf("%w: %s failed; check the operating system credential service (no plaintext fallback)", ErrUnavailable, action)
}

var (
	servicePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	keyPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,255}$`)
)

func validateService(service string) error {
	if !servicePattern.MatchString(service) {
		return fmt.Errorf("%w: service must match %s", ErrInvalidKey, servicePattern)
	}
	return nil
}

func validateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("%w: key must match %s", ErrInvalidKey, keyPattern)
	}
	return nil
}
