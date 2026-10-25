//go:build linux

package keystore

import (
	"errors"
	"fmt"

	"github.com/99designs/keyring"
	"github.com/godbus/dbus"
)

func supportError() error { return nil }

func openKeyring(service, _ string) (keyring.Keyring, error) {
	// Use the standard GNOME login collection rather than creating a new
	// collection (new collections invoke the Secret Service prompt, which can
	// block headless/native test sessions). The service is part of the item key
	// in osStore, so different zt services do not collide in this shared
	// collection.
	ring, err := keyring.Open(keyring.Config{
		ServiceName:             service,
		LibSecretCollectionName: "login",
		AllowedBackends:         []keyring.BackendType{keyring.SecretServiceBackend},
	})
	if err != nil {
		return nil, classifyKeyringError("open Linux Secret Service", err)
	}
	return ring, nil
}

func availabilityCheck(ring keyring.Keyring, _ string) error {
	_, err := ring.Keys()
	return err
}

func confirmMissing(keyring.Keyring, string) error { return nil }

func classifyPlatformError(err error) error {
	var value dbus.Error
	name := ""
	if errors.As(err, &value) {
		name = value.Name
	} else {
		var pointer *dbus.Error
		if errors.As(err, &pointer) && pointer != nil {
			name = pointer.Name
		}
	}
	switch name {
	case "org.freedesktop.Secret.Error.IsLocked", "org.freedesktop.DBus.Error.InteractiveAuthorizationRequired":
		return fmt.Errorf("%w: the Secret Service keyring is locked; unlock it and retry (no plaintext fallback)", ErrLocked)
	case "org.freedesktop.DBus.Error.AccessDenied":
		return fmt.Errorf("%w: the session bus denied Secret Service access (no plaintext fallback)", ErrDenied)
	case "org.freedesktop.DBus.Error.ServiceUnknown", "org.freedesktop.DBus.Error.NameHasNoOwner",
		"org.freedesktop.DBus.Error.Disconnected", "org.freedesktop.DBus.Error.NoReply",
		"org.freedesktop.DBus.Error.TimedOut":
		return fmt.Errorf("%w: Secret Service is not running or not responding on the session bus; start GNOME Keyring or KWallet (no plaintext fallback)", ErrUnavailable)
	}
	return nil
}
