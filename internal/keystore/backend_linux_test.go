//go:build linux

package keystore

import (
	"errors"
	"testing"

	"github.com/godbus/dbus"
)

func TestLinuxSecretServiceErrorClassification(t *testing.T) {
	tests := []struct {
		name string
		want error
	}{
		{"org.freedesktop.Secret.Error.IsLocked", ErrLocked},
		{"org.freedesktop.DBus.Error.InteractiveAuthorizationRequired", ErrLocked},
		{"org.freedesktop.DBus.Error.AccessDenied", ErrDenied},
		{"org.freedesktop.DBus.Error.ServiceUnknown", ErrUnavailable},
		{"org.freedesktop.DBus.Error.NoReply", ErrUnavailable},
	}
	for _, test := range tests {
		mapped := classifyPlatformError(dbus.Error{Name: test.name})
		if !errors.Is(mapped, test.want) {
			t.Errorf("classify(%s) = %v, want %v", test.name, mapped, test.want)
		}
	}
}
