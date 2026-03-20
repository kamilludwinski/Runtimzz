#!/usr/bin/env bash

set -e

isWindows() {
	[ "${GOOS:-}" = "windows" ] || { [ -z "${GOOS:-}" ] && [[ "$(uname -s)" == MINGW* || "$(uname -s)" == MSYS* || "$(uname -s)" == CYGWIN* ]]; }
}

cd "$(dirname "$0")/.."
mkdir -p dist

# Windows needs embedding shim launcher first
if isWindows; then
	GOOS=windows go build -o internal/shimembed/shim.exe ./cmd/shim
fi

# Windows needs .exe
if isWindows; then
	BINARY=rtz.exe
else
	BINARY=rtz
fi

go build -o "$BINARY" .
