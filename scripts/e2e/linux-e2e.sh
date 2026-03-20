#!/usr/bin/env bash
# E2E tests for rtz on Linux.
# Requires: RTZ_APP_DIR set to a temp dir (optional), rtz binary in current directory or PATH.
# Run from repo root after building: go build -o rtz .

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Prefer rtz in current directory (e.g. when run from repo root in CI)
rtz="./rtz"
if [[ ! -x "$rtz" ]]; then
  rtz="$REPO_ROOT/rtz"
fi
if [[ ! -x "$rtz" ]]; then
  if command -v rtz &>/dev/null; then
    rtz="rtz"
  else
    echo "rtz not found. Build with: go build -o rtz ." >&2
    exit 1
  fi
fi

app_dir="${RTZ_APP_DIR:-/tmp/rtz-e2e}"
mkdir -p "$app_dir"
export RTZ_APP_DIR="$app_dir"

assert_exit_zero() {
  local caption="$1"
  shift
  if ! "$@"; then
    echo "$caption failed with exit code $?" >&2
    exit 1
  fi
}

echo "E2E: rtz version"
out=$("$rtz" version 2>&1) || { echo "rtz version failed"; exit 1; }
if ! echo "$out" | grep -qE '[0-9]+\.[0-9]+\.[0-9]+'; then
  echo "version output should contain x.y.z: $out" >&2
  exit 1
fi

echo "E2E: rtz (help)"
out=$("$rtz" 2>&1) || { echo "rtz (no args) failed"; exit 1; }
echo "$out" | grep -q "version" || { echo "help should mention version: $out"; exit 1; }
echo "$out" | grep -q "purge" || { echo "help should mention purge: $out"; exit 1; }
echo "$out" | grep -q "update" || { echo "help should mention update: $out"; exit 1; }

echo "E2E: rtz purge (no app dir)"
out=$("$rtz" purge 2>&1) || { echo "rtz purge failed"; exit 1; }
if ! echo "$out" | grep -qE "Nothing to purge|purged|logs"; then
  echo "purge output unexpected: $out" >&2
  exit 1
fi

echo "E2E: rtz go purge (no versions)"
out=$("$rtz" go purge 2>&1) || { echo "rtz go purge failed"; exit 1; }
if ! echo "$out" | grep -qiE "nothing to purge|No.*versions"; then
  echo "go purge output unexpected: $out" >&2
  exit 1
fi

echo "E2E: rtz go ls (smoke)"
out=$("$rtz" go ls 2>&1) || { echo "rtz go ls failed: $out"; exit 1; }
if ! echo "$out" | grep -qE "Go|available|installed"; then
  echo "go ls should mention Go/versions: $out" >&2
  exit 1
fi

echo "E2E: all passed"
