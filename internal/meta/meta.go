// Package meta provides app metadata
package meta

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	AppName      = "Runtimz"
	AppShortName = "rtz"
	AppVersion   = "1.0.0"

	// DevAppName overrides the app name used for directories (e.g. ~/.runtimz)
	DevAppName = "DEV_APP_NAME"
	// AppDirOverride when set (e.g. in tests) makes AppDir() return this path directly.
	AppDirOverride = "RTZ_APP_DIR"
)

var appNameForDir string

func init() {
	if name := os.Getenv(DevAppName); name != "" {
		appNameForDir = name
	} else {
		appNameForDir = AppName
	}
}

func AppDir() string {
	if dir := os.Getenv(AppDirOverride); dir != "" {
		return dir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, fmt.Sprintf(".%s", strings.ToLower(appNameForDir)))
}

func InstallationsDir() string {
	return filepath.Join(AppDir(), "installations")
}

func ShimsDir() string {
	return filepath.Join(AppDir(), "shims")
}

func LogDir() string {
	return filepath.Join(AppDir(), "logs")
}
