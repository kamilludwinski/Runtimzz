package shimpaths

import "path/filepath"

// RuntimeVersionRootWithBase returns the installation directory for a runtime/version
// given an app base directory (e.g. ~/.runtimz). Used by the shim launcher.
func RuntimeVersionRootWithBase(appDir, rt, version string) string {
	return filepath.Join(appDir, "installations", rt, version)
}

