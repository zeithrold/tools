#!/usr/bin/env bash
set -eu

if [ "${GITHUB_REF-}" != "refs/heads/main" ]; then
  printf '%s\n' 'Staging requires a push to refs/heads/main.' >&2
  exit 1
fi
: "${GITHUB_SHA:?Staging requires an immutable workflow SHA.}"
test "$(git rev-parse HEAD)" = "$GITHUB_SHA"
git fetch --no-tags --depth=1 origin refs/heads/main:refs/remotes/origin/main
test "$(git rev-parse refs/remotes/origin/main)" = "$GITHUB_SHA"
git switch --create main --track origin/main
test "$(git rev-parse HEAD)" = "$GITHUB_SHA"
