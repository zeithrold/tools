//go:build !darwin && !windows && !linux

package keystore

import (
	"fmt"

	"github.com/99designs/keyring"
)

func supportError() error {
	return fmt.Errorf("%w: this platform has no supported native keyring backend; zt does not fall back to files, environment variables, or arguments", ErrUnavailable)
}

func openKeyring(string, string) (keyring.Keyring, error) { return nil, supportError() }

func availabilityCheck(keyring.Keyring, string) error { return supportError() }

func confirmMissing(keyring.Keyring, string) error { return nil }

func classifyPlatformError(error) error { return nil }
