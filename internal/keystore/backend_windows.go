//go:build windows

package keystore

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/99designs/keyring"
	"github.com/danieljoos/wincred"
)

func supportError() error { return nil }

func openKeyring(service, _ string) (keyring.Keyring, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:     service,
		AllowedBackends: []keyring.BackendType{keyring.WinCredBackend},
		WinCredPrefix:   "zt",
	})
	if err != nil {
		return nil, classifyKeyringError("open Windows Credential Manager", err)
	}
	return ring, nil
}

func availabilityCheck(keyring.Keyring, string) error {
	_, err := wincred.List()
	return err
}

func confirmMissing(keyring.Keyring, string) error { return nil }

func classifyPlatformError(err error) error {
	switch {
	case errors.Is(err, wincred.ErrElementNotFound):
		return nil
	case errors.Is(err, syscall.ERROR_ACCESS_DENIED):
		return fmt.Errorf("%w: Windows Credential Manager refused access for the current user (no plaintext fallback)", ErrDenied)
	case errors.Is(err, syscall.Errno(1312)):
		return fmt.Errorf("%w: Windows has no credential store for this logon session", ErrUnavailable)
	case errors.Is(err, wincred.ErrInvalidParameter):
		return fmt.Errorf("%w: Windows Credential Manager rejected the credential item", ErrUnavailable)
	}
	return nil
}
