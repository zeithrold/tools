#!/usr/bin/env bash
# Verify that a built zt binary ships the MIT license text.
#
# The full output is captured before matching. Piping `zt license` into
# `grep -q` would let grep exit early and SIGPIPE the binary, which `pipefail`
# turns into a false failure as the embedded notices grow.
set -euo pipefail

if [[ $# != 1 ]]; then
  echo 'Usage: check-cli-license.sh <zt-binary>' >&2
  exit 1
fi

output="$("$1" license)"
if [[ "$output" != *'MIT License'* ]]; then
  echo 'CLI license output is missing the MIT license text' >&2
  exit 1
fi
