#!/bin/sh
set -eu

repository=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
cd "$repository"

command -v docker >/dev/null
command -v go >/dev/null
arch=$(docker version --format '{{.Server.Arch}}')
case "$arch" in
    amd64|arm64) ;;
    *) printf '%s\n' "Unsupported Docker architecture: $arch" >&2; exit 1 ;;
esac

directory=$(mktemp -d)
trap 'rm -rf "$directory"' EXIT HUP INT TERM
chmod 755 "$directory"

# Compile only; this does not run a native credential test on the host.
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" \
    go test -c -o "$directory/keystore.test" ./internal/keystore

docker build --tag zt-keystore-test:local scripts/keystore
docker run --rm --network none --read-only \
    --cap-drop ALL --security-opt no-new-privileges \
    --tmpfs /tmp:rw,nosuid,nodev,mode=1777 \
    --tmpfs /home/zt-test:rw,nosuid,nodev,uid=10001,gid=10001,mode=0700 \
    --mount "type=bind,source=$directory,target=/tests,readonly" \
    zt-keystore-test:local
