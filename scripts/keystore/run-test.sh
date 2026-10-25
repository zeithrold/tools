#!/bin/sh
set -eu

mkdir -p "$XDG_RUNTIME_DIR"
chmod 700 "$XDG_RUNTIME_DIR"

# This container has no host HOME or credentials. The password and bootstrap
# item are synthetic and exist only for this disposable D-Bus session.
exec dbus-run-session -- sh -eu -c '
    eval "$(printf "%s\n" "zt-disposable-keyring" | gnome-keyring-daemon --unlock --components=secrets)"
    printf "%s" "synthetic-bootstrap" |
        timeout 15s secret-tool store --label=zt-test-bootstrap service zt-test-init key bootstrap
    ZT_KEYSTORE_INTEGRATION=1 /tests/keystore.test \
        -test.run "^TestOSStoreIntegration$" -test.v -test.timeout 45s
'
