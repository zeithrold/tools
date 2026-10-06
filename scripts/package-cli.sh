#!/usr/bin/env bash
set -euo pipefail

if [[ $# != 2 ]]; then
  echo 'Usage: package-cli.sh <binary-directory> <archive.tar.gz|archive.zip>' >&2
  exit 1
fi

repository=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
directory=$(cd "$1" && pwd)
mkdir -p "$(dirname "$2")"
archive="$(cd "$(dirname "$2")" && pwd)/$(basename "$2")"
cp "$repository/LICENSE" "$repository/THIRD_PARTY_NOTICES.md" "$directory/"
mkdir -p "$directory/third-party"
cp "$repository/third-party/GO-BSD.txt" "$directory/third-party/"
binary=zt
if [[ -f "$directory/zt.exe" ]]; then binary=zt.exe; fi
[[ -f "$directory/$binary" ]] || { echo 'CLI binary is missing' >&2; exit 1; }
files=("$binary" LICENSE THIRD_PARTY_NOTICES.md third-party/GO-BSD.txt)

case "$archive" in
  *.zip) (cd "$directory" && zip -q "$archive" "${files[@]}") ;;
  *.tar.gz) tar -C "$directory" -czf "$archive" "${files[@]}" ;;
  *) echo 'Expected a .tar.gz or .zip archive' >&2; exit 1 ;;
esac
