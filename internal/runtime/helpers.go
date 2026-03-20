package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
	"github.com/kamilludwinski/runtimzzz/internal/msgs"
	"github.com/kamilludwinski/runtimzzz/internal/output"
	"github.com/kamilludwinski/runtimzzz/internal/state"
	"github.com/kamilludwinski/runtimzzz/internal/utils/versionutils"
)

// ListInstalledVersions reads installations/<rtName>/ and returns sorted version directory names.
func ListInstalledVersions(rtName string) ([]string, error) {
	dir := RuntimeRoot(rtName)
	logger.Debug("reading installations dir", "path", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read installations directory: %w", err)
	}

	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}
	slices.Sort(versions)
	return versions, nil
}

// PrintVersions prints the "Found X versions" header and bullet list with (installed)/(active) markers.
// available is the map from availableVersions; installed and active are from the runtime.
// limit is applied (use 0 for defaultLsLimit).
func PrintVersions(runtimeDisplayName string, available map[string]string, installed []string, active string, limit int) {
	if limit == 0 {
		limit = defaultLsLimit
	}

	output.Info(msgs.LsHeader(runtimeDisplayName, len(available), len(installed)))

	versions := make([]string, 0, len(available))
	for v := range available {
		versions = append(versions, v)
	}
	versionutils.SortVersions(versions, true)
	if limit > 0 && limit < len(versions) {
		versions = versions[:limit]
	}

	for _, version := range versions {
		var installedPart, activePart string
		if slices.Contains(installed, version) {
			installedPart = "\t(installed)"
		}
		if active == version {
			activePart = "\t(active)"
		}
		line := fmt.Sprintf("%s%s%s", version, installedPart, activePart)
		output.Line("\t• " + line)
	}
}

// EnsureRuntimeVersionDir creates the parent runtime directory and removes any existing version dir.
// Returns the version root path.
func EnsureRuntimeVersionDir(rt, version string) (string, error) {
	root := RuntimeRoot(rt)
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", fmt.Errorf("failed to create installations directory: %w", err)
	}
	versionRoot := filepath.Join(root, version)
	if err := os.RemoveAll(versionRoot); err != nil {
		return "", fmt.Errorf("failed to remove existing installation dir: %w", err)
	}
	return versionRoot, nil
}

// RunUse implements the common Use flow: check already active, check installed, run createShims, ensure PATH, set state.
func RunUse(rtName, displayName, version string, st *state.State, installed []string, createShims func() error) error {
	logger.Debug("Use", "version", version)
	output.Info(msgs.Activating(displayName, version))

	if st.IsActive(rtName, version) {
		output.Warn(msgs.AlreadyActive(displayName, version))
		return nil
	}

	if !slices.Contains(installed, version) {
		return fmt.Errorf("%s version %s not installed", rtName, version)
	}

	if err := createShims(); err != nil {
		return err
	}
	if err := st.SetActive(rtName, version); err != nil {
		return fmt.Errorf("failed to set active version: %w", err)
	}
	output.Success(msgs.ActiveSet(displayName, version))
	return nil
}
