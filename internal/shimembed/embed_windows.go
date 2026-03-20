//go:build windows

package shimembed

import _ "embed"

// ShimExe is the Windows go/gofmt launcher. Build it before building runtimz for Windows:
//
//	go build -o internal/shimembed/shim.exe ./cmd/shim
//
// Or from repo root: go generate ./internal/shimembed
//
//go:embed shim.exe
var ShimExe []byte
