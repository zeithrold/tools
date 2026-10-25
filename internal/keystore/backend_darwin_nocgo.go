//go:build darwin && !cgo

package keystore

import (
	"fmt"

	"github.com/99designs/keyring"
)

func supportError() error {
	return fmt.Errorf("%w: macOS Keychain support requires cgo; rebuild with CGO_ENABLED=1 (no plaintext fallback)", ErrUnavailable)
}

func openKeyring(string, string) (keyring.Keyring, error) { return nil, supportError() }

func availabilityCheck(keyring.Keyring, string) error { return supportError() }

func confirmMissing(keyring.Keyring, string) error { return nil }

func classifyPlatformError(error) error { return nil }
