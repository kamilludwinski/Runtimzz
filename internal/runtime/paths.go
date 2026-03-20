package runtime

import (
	"path/filepath"

	"github.com/kamilludwinski/runtimzzz/internal/meta"
)

// RuntimeVersionRootWithBase returns the installation directory for a runtime/version
// given an app base directory (e.g. ~/.runtimz). Used by the shim launcher.
func RuntimeVersionRootWithBase(appDir, rt, version string) string {
	return filepath.Join(appDir, "installations", rt, version)
}

// RuntimeRoot returns the installations directory for a runtime, e.g. ~/.runtimz/installations/go.
func RuntimeRoot(rt string) string {
	return filepath.Join(meta.InstallationsDir(), rt)
}

// RuntimeVersionRoot returns the installation directory for a specific version,
// e.g. ~/.runtimz/installations/go/1.22.0.
func RuntimeVersionRoot(rt, version string) string {
	return filepath.Join(meta.InstallationsDir(), rt, version)
}

// RuntimeBinDir returns the bin directory inside a version root, e.g. .../node/20.10.0/bin.
func RuntimeBinDir(rt, version string) string {
	return filepath.Join(RuntimeVersionRoot(rt, version), "bin")
}

// RuntimeScriptsDir returns the Scripts directory inside a version root (e.g. Python).
func RuntimeScriptsDir(rt, version string) string {
	return filepath.Join(RuntimeVersionRoot(rt, version), "Scripts")
}
