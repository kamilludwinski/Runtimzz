package update

import (
	"os"
	"strconv"
	"strings"
)

// Env vars used when re-execing as the updater process
const (
	EnvUpdatePID    = "RTZ_UPDATE_PID"
	EnvUpdateNew    = "RTZ_UPDATE_NEW"
	EnvUpdateTarget = "RTZ_UPDATE_TARGET"
)

// RunUpdaterIfRequested checks for updater env vars; if set, waits for the parent process to exit then renames new binary over target and exits. Returns true if updater ran (caller should exit).
func RunUpdaterIfRequested() bool {
	pidStr := os.Getenv(EnvUpdatePID)
	newPath := os.Getenv(EnvUpdateNew)
	targetPath := os.Getenv(EnvUpdateTarget)
	if pidStr == "" || newPath == "" || targetPath == "" {
		return false
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false
	}
	if err := waitForProcessExit(pid); err != nil {
		return false
	}
	if err := os.Rename(newPath, targetPath); err != nil {
		return false
	}
	os.Exit(0)
	return true
}

// AssetURLByPlatform returns the download URL for the current OS/ARCH from the release, or "" if none.
// Prefers .exe on Windows, then .zip; .tar.gz on Unix.
func AssetURLByPlatform(r *release, goos, goarch string) string {
	var match string
	for _, a := range r.Assets {
		if !assetMatches(a.Name, goos, goarch) {
			continue
		}
		// Prefer .exe on Windows
		if goos == "windows" && strings.HasSuffix(a.Name, ".exe") {
			return a.BrowserDownloadURL
		}
		match = a.BrowserDownloadURL
	}
	return match
}

func assetMatches(name, goos, goarch string) bool {
	switch goos {
	case "windows":
		if goarch == "amd64" {
			return strings.Contains(name, "windows_amd64") && (strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".zip"))
		}
		if goarch == "arm64" {
			return strings.Contains(name, "windows_arm64") && (strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".zip"))
		}
	case "linux":
		return strings.Contains(name, "linux") && strings.Contains(name, goarch) && strings.HasSuffix(name, ".tar.gz")
	case "darwin":
		return strings.Contains(name, "darwin") && strings.Contains(name, goarch) && strings.HasSuffix(name, ".tar.gz")
	}
	return false
}
