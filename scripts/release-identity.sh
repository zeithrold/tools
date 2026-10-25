#!/usr/bin/env bash
# Resolve the public GitHub App client ID embedded in release binaries.
#
# Prints the client ID on stdout and nothing else. A rejected value is never
# printed, and a numeric app ID is never accepted as a client identity.
set -euo pipefail

fail() {
  printf 'release-identity: %s\n' "$1" >&2
  exit 1
}

# GitHub App client IDs are public identifiers, not secrets. Accept the classic
# Iv1.<hex> form and the modern Iv23 prefix; Ov23 is an OAuth App, not a
# GitHub App, and numeric values are app IDs that cannot drive device flow.
is_client_id() {
  [[ "$1" =~ ^Iv1\.[0-9a-fA-F]{16}$ ]] || [[ "$1" =~ ^Iv23[0-9A-Za-z]{8,64}$ ]]
}

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

value="${ZEITHROLD_GITHUB_CLIENT_ID:-}"
[[ -n "$value" ]] || fail 'no GitHub App client ID configured; set the ZEITHROLD_GITHUB_CLIENT_ID secret before releasing'
value="$(trim "$value")"
is_client_id "$value" || fail 'ZEITHROLD_GITHUB_CLIENT_ID does not hold a GitHub App client ID (Iv1.<hex> or Iv23…); no value was embedded'
printf '%s\n' "$value"
