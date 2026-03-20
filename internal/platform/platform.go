// Package platform provides platform-specific utilities
package platform

type Platform interface {
	GoArchiveExt() string

	// EnsurePath makes sure that the path given is on the PATH env var
	EnsurePath(path string) error

	// CreateGoShims creates shims for the given go version (installations/<version>/bin)
	// Assumes that the all other shims have been removed
	CreateGoShims(version string) error

	// CreateShims creates shims for tools in binDir (e.g. node, npm, npx in installations/node-<version>/bin)
	CreateShims(binDir string, tools []string) error
}
