//go:build darwin && cgo

package keystore

import (
	"errors"
	"fmt"

	gokeychain "github.com/99designs/go-keychain"
	"github.com/99designs/keyring"
)

func supportError() error { return nil }

func openKeyring(service, keychainName string) (keyring.Keyring, error) {
	// Trust the calling zt binary so a normal login Keychain item does not
	// prompt on every access; other applications still receive the Keychain
	// access prompt. Accessibility is restricted to an unlocked Keychain, and
	// iCloud synchronization is explicitly disabled.
	ring, err := keyring.Open(keyring.Config{
		ServiceName:                    service,
		KeychainName:                   keychainName,
		AllowedBackends:                []keyring.BackendType{keyring.KeychainBackend},
		KeychainSynchronizable:         false,
		KeychainTrustApplication:       true,
		KeychainAccessibleWhenUnlocked: true,
	})
	if err != nil {
		return nil, classifyKeyringError("open macOS Keychain", err)
	}
	return ring, nil
}

func availabilityCheck(ring keyring.Keyring, keychainName string) error {
	if keychainName != "" {
		keychain := gokeychain.NewWithPath(keychainName + ".keychain")
		return keychain.Status()
	}
	_, err := ring.Keys()
	return err
}

// keyring v1.2.2's macOS Get has a narrow bug: when QueryItem returns both
// an error and zero results, it reports ErrKeyNotFound. Re-check metadata with
// the same keyring instance before treating a missing data result as absent.
func confirmMissing(ring keyring.Keyring, itemKey string) error {
	keys, err := ring.Keys()
	if err != nil {
		return classifyKeyringError("verify missing macOS Keychain item", err)
	}
	for _, key := range keys {
		if key == itemKey {
			return fmt.Errorf("%w: macOS Keychain metadata for the item exists but its data could not be read; absence cannot be confirmed. Unlock the Keychain or allow access and retry", ErrDenied)
		}
	}
	return nil
}

func classifyPlatformError(err error) error {
	var keychainError gokeychain.Error
	if !errors.As(err, &keychainError) {
		return nil
	}
	switch keychainError {
	case gokeychain.ErrorInteractionNotAllowed:
		return fmt.Errorf("%w: the login Keychain is locked or needs interactive approval; unlock it in Keychain Access and retry (no plaintext fallback)", ErrLocked)
	case gokeychain.ErrorAuthFailed, gokeychain.ErrorNoAccessForItem:
		return fmt.Errorf("%w: macOS Keychain refused access; allow access in Keychain Access and retry (no plaintext fallback)", ErrDenied)
	case gokeychain.ErrorNotAvailable, gokeychain.ErrorNoSuchKeychain:
		return fmt.Errorf("%w: no usable default macOS Keychain", ErrUnavailable)
	case gokeychain.ErrorItemNotFound:
		return nil
	default:
		return fmt.Errorf("%w: macOS Keychain error %d", ErrUnavailable, int(keychainError))
	}
}
