package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
	"github.com/kamilludwinski/runtimzzz/internal/meta"
	"github.com/kamilludwinski/runtimzzz/internal/msgs"
	"github.com/kamilludwinski/runtimzzz/internal/output"
	"github.com/kamilludwinski/runtimzzz/internal/platform"
	"github.com/kamilludwinski/runtimzzz/internal/spinner"
	"github.com/kamilludwinski/runtimzzz/internal/state"
	"github.com/kamilludwinski/runtimzzz/internal/utils/archiveutils"
	"github.com/kamilludwinski/runtimzzz/internal/utils/cryptoutils"
	"github.com/kamilludwinski/runtimzzz/internal/utils/httputils"
)

const (
	nodeIndexURL      = "https://nodejs.org/download/release/index.json"
	nodeDistURLFmt    = "https://nodejs.org/dist/v%s/%s"
	nodeShasumsURLFmt = "https://nodejs.org/dist/v%s/SHASUMS256.txt"
)

// nodeFileType maps (GOOS, GOARCH) to Node dist "files" entry.
// The value is used to match entries in index.json, not directly in filenames.
var nodeFileType = map[string]map[string]string{
	"windows": {"amd64": "win-x64-zip", "arm64": "win-arm64-zip"},
	"linux":   {"amd64": "linux-x64", "arm64": "linux-arm64"},
	"darwin":  {"amd64": "darwin-x64", "arm64": "darwin-arm64"},
}

// nodeArchiveExt returns the archive extension for a given files entry.
func nodeArchiveExt(fileType string) string {
	if strings.HasSuffix(fileType, "-zip") {
		return "zip"
	}
	// For now we only support .tar.gz-style archives for non-zip platforms.
	return "tar.gz"
}

// nodeArchiveSegment returns the OS/arch segment used in the dist filename
// (e.g. "win-x64-zip" -> "win-x64").
func nodeArchiveSegment(fileType string) string {
	switch fileType {
	case "win-x64-zip":
		return "win-x64"
	case "win-arm64-zip":
		return "win-arm64"
	case "linux-x64":
		return "linux-x64"
	case "linux-arm64":
		return "linux-arm64"
	case "darwin-x64":
		return "darwin-x64"
	case "darwin-arm64":
		return "darwin-arm64"
	default:
		return fileType
	}
}

type nodeRelease struct {
	Version string   `json:"version"`
	Files   []string `json:"files"`
	LTS     any      `json:"lts"`
}

func InitNode(state *state.State) Runtime {
	logger.Debug("NodeRuntime Init")
	return &NodeRuntime{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		platform: platform.PlatformImpl{},
		state:    state,
	}
}

type NodeRuntime struct {
	client   *http.Client
	platform platform.Platform
	state    *state.State
}

var _ Runtime = (*NodeRuntime)(nil)

func (n NodeRuntime) Name() string {
	return "node"
}

func (n NodeRuntime) Ls(limit int) error {
	logger.Debug("Ls", "limit", limit)

	var available map[string]string
	err := spinner.Run("Fetching Node versions...", func() error {
		var e error
		available, e = n.availableVersions(true)
		return e
	})
	if err != nil {
		logger.Error("availableVersions failed", "err", err)
		return fmt.Errorf("failed to get available node versions: %w", err)
	}

	installed, err := n.installedVersions()
	if err != nil {
		logger.Error("installedVersions failed", "err", err)
		return fmt.Errorf("failed to get installed node versions: %w", err)
	}

	active, _ := n.Active()
	PrintVersions("Node", available, installed, active, limit)
	return nil
}

func (n NodeRuntime) Install(version string) error {
	logger.Debug("Install", "version", version)
	output.Info(msgs.Installing("Node", version))

	installed, err := n.installedVersions()
	if err != nil {
		return fmt.Errorf("failed to get installed node versions: %w", err)
	}

	if slices.Contains(installed, version) {
		return fmt.Errorf("node version %s already installed", version)
	}

	available, err := n.availableVersions(true)
	if err != nil {
		return fmt.Errorf("failed to get available node versions: %w", err)
	}

	if _, ok := available[version]; !ok {
		return fmt.Errorf("node version %s not available", version)
	}

	fileType := n.fileTypeForCurrent()
	if fileType == "" {
		return fmt.Errorf("unsupported platform for Node: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	ext := nodeArchiveExt(fileType)
	versionWithV := "v" + version
	segment := nodeArchiveSegment(fileType)
	fileName := fmt.Sprintf("node-%s-%s.%s", versionWithV, segment, ext)
	downloadURL := fmt.Sprintf(nodeDistURLFmt, version, fileName)
	logger.Debug("downloading", "url", downloadURL)

	var downloadPath string
	err = spinner.Run("Downloading Node...", func() error {
		var e error
		downloadPath, e = httputils.DownloadFile(n.client, downloadURL)
		return e
	})
	if err != nil {
		return fmt.Errorf("failed to download node: %w", err)
	}

	shasums, err := n.fetchChecksumForVersion(version)
	if err != nil {
		return fmt.Errorf("failed to fetch SHASUMS256: %w", err)
	}

	expectedChecksum, ok := shasums[fileName]
	if !ok {
		return fmt.Errorf("checksum not found for %s", fileName)
	}

	checksum, err := cryptoutils.SHA256(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if !strings.EqualFold(checksum, expectedChecksum) {
		return fmt.Errorf("node checksum mismatch: got %s want %s", checksum, expectedChecksum)
	}

	output.Success("Node checksum matches")

	var extractDir string
	err = spinner.Run("Extracting archive...", func() error {
		var e error
		extractDir, e = archiveutils.Extract(downloadPath)
		return e
	})
	if err != nil {
		return fmt.Errorf("failed to extract node: %w", err)
	}

	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return fmt.Errorf("failed to read extract dir: %w", err)
	}
	var topDir string
	for _, e := range entries {
		if e.IsDir() {
			topDir = e.Name()
			break
		}
	}
	if topDir == "" {
		return fmt.Errorf("no top-level dir in node archive")
	}

	runtimeDir := RuntimeRoot(n.Name())
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return fmt.Errorf("failed to create installations directory: %w", err)
	}

	if err := os.Rename(filepath.Join(extractDir, topDir), RuntimeVersionRoot(n.Name(), version)); err != nil {
		return fmt.Errorf("failed to move node to installations: %w", err)
	}

	output.Success(msgs.Installed("Node", version))
	output.Line(msgs.UseHint(n.Name(), version))
	return nil
}

func (n NodeRuntime) Uninstall(version string) error {
	logger.Debug("Uninstall", "version", version)
	output.Info(msgs.Uninstalling("Node", version))

	installed, err := n.installedVersions()
	if err != nil {
		return fmt.Errorf("failed to get installed node versions: %w", err)
	}
	if !slices.Contains(installed, version) {
		return fmt.Errorf("node version %s not installed", version)
	}

	err = spinner.Run("Uninstalling Node...", func() error {
		if n.state.IsActive(n.Name(), version) {
			if e := n.removeShims(); e != nil {
				return fmt.Errorf("failed to remove shims: %w", e)
			}
			if e := n.state.SetActive(n.Name(), ""); e != nil {
				return fmt.Errorf("failed to set active node version: %w", e)
			}
		}
		versionDir := RuntimeVersionRoot(n.Name(), version)
		if e := os.RemoveAll(versionDir); e != nil {
			return fmt.Errorf("failed to remove installation directory: %w", e)
		}
		return nil
	})
	if err != nil {
		return err
	}
	output.Success(msgs.Uninstalled("Node", version))
	return nil
}

func (n NodeRuntime) Active() (string, error) {
	return n.state.Active(n.Name()), nil
}

func (n NodeRuntime) Use(version string) error {
	installed, err := n.installedVersions()
	if err != nil {
		return fmt.Errorf("failed to get installed node versions: %w", err)
	}
	baseDir := RuntimeVersionRoot(n.Name(), version)
	binDir := RuntimeBinDir(n.Name(), version)
	if info, err := os.Stat(binDir); err != nil || !info.IsDir() {
		binDir = baseDir
	}
	createShims := func() error {
		return spinner.Run("Creating shims...", func() error {
			if e := n.platform.CreateShims(binDir, []string{"node", "npm", "npx"}); e != nil {
				return e
			}
			return n.platform.EnsurePath(meta.ShimsDir())
		})
	}
	return RunUse(n.Name(), "Node", version, n.state, installed, createShims)
}

func (n NodeRuntime) availableVersions(stableOnly bool) (map[string]string, error) {
	res, err := n.client.Get(nodeIndexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch node index: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var releases []nodeRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to decode node index: %w", err)
	}

	fileType := n.fileTypeForCurrent()
	if fileType == "" {
		return nil, fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	versions := make(map[string]string)
	for _, r := range releases {
		version := strings.TrimPrefix(r.Version, "v")
		// skip any pre-release tags (e.g. -rc, -nightly).
		if stableOnly && strings.Contains(version, "-") {
			continue
		}
		for _, f := range r.Files {
			if f != fileType {
				continue
			}
			// We'll fill checksum when we fetch SHASUMS256 for this version
			versions[version] = ""
			break
		}
	}

	logger.Debug("available node versions", "count", len(versions))

	return versions, nil
}

func (n NodeRuntime) fetchChecksumForVersion(version string) (map[string]string, error) {
	u := fmt.Sprintf(nodeShasumsURLFmt, version)
	res, err := n.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	scanner := bufio.NewScanner(res.Body)
	out := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			out[strings.Join(parts[1:], " ")] = parts[0]
		}
	}
	return out, scanner.Err()
}

func (n NodeRuntime) fileTypeForCurrent() string {
	m, ok := nodeFileType[runtime.GOOS]
	if !ok {
		return ""
	}
	arch := runtime.GOARCH
	if arch == "386" {
		arch = "x86"
	}
	ft, ok := m[arch]
	if !ok {
		ft, _ = m["amd64"]
	}
	return ft
}

func (n NodeRuntime) installedVersions() ([]string, error) {
	return ListInstalledVersions(n.Name())
}

func (n NodeRuntime) AvailableVersions(stableOnly bool) (map[string]string, error) {
	return n.availableVersions(stableOnly)
}

func (n NodeRuntime) InstalledVersions() ([]string, error) {
	return n.installedVersions()
}

func (n NodeRuntime) RemoveShims() error {
	return n.removeShims()
}

func (n NodeRuntime) removeShims() error {
	shimsDir := meta.ShimsDir()
	entries, err := os.ReadDir(shimsDir)
	if err != nil {
		return err
	}
	toRemove := map[string]bool{"node": true, "npm": true, "npx": true}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := e.Name()
		base = strings.TrimSuffix(base, ".exe")
		base = strings.TrimSuffix(base, ".cmd")
		if toRemove[base] {
			if err := os.Remove(filepath.Join(shimsDir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
