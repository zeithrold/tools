//go:build windows

package keystore

import (
	"errors"
	"syscall"
	"testing"

	"github.com/danieljoos/wincred"
)

func TestWindowsCredentialErrorClassification(t *testing.T) {
	if err := classifyPlatformError(wincred.ErrElementNotFound); err != nil {
		t.Errorf("not-found classification = %v, want nil", err)
	}
	if err := classifyPlatformError(syscall.ERROR_ACCESS_DENIED); !errors.Is(err, ErrDenied) {
		t.Errorf("access denied classification = %v, want ErrDenied", err)
	}
	if err := classifyPlatformError(syscall.Errno(1312)); !errors.Is(err, ErrUnavailable) {
		t.Errorf("no logon session classification = %v, want ErrUnavailable", err)
	}
}
