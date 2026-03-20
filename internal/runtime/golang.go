package runtime

import (
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
	goVersionsUrl = "https://go.dev/dl/?mode=json&include=all"
	// args: version, os, arch, ext
	goDownloadUrlFormat = "https://go.dev/dl/go%s.%s-%s.%s"
)

func Init(state *state.State) Runtime {
	logger.Debug("GoRuntime Init")
	return &GoRuntime{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		platform: platform.PlatformImpl{},
		state:    state,
	}
}

type GoRuntime struct {
	client   *http.Client
	platform platform.Platform
	state    *state.State
}

var _ Runtime = (*GoRuntime)(nil)

func (g GoRuntime) Name() string {
	return "go"
}

func (g GoRuntime) Ls(limit int) error {
	logger.Debug("Ls", "limit", limit)

	var available map[string]string
	err := spinner.Run("Fetching Go versions...", func() error {
		var e error
		available, e = g.availableVersions(true)
		return e
	})
	if err != nil {
		logger.Error("availableVersions failed", "err", err)
		return fmt.Errorf("failed to get available go versions: %w", err)
	}

	installed, err := g.installedVersions()
	if err != nil {
		logger.Error("installedVersions failed", "err", err)
		return fmt.Errorf("failed to get installed go versions: %w", err)
	}

	active, err := g.Active()
	if err != nil {
		return fmt.Errorf("failed to get active go version: %w", err)
	}

	PrintVersions("Go", available, installed, active, limit)
	return nil
}

func (g GoRuntime) Install(version string) error {
	logger.Debug("Install", "version", version)
	output.Info(msgs.Installing("Go", version))

	installed, err := g.installedVersions()
	if err != nil {
		return fmt.Errorf("failed to get installed go versions: %w", err)
	}

	if slices.Contains(installed, version) {
		return fmt.Errorf("go version %s already installed", version)
	}

	logger.Debug("version is not installed", "version", version)

	available, err := g.availableVersions(true)
	if err != nil {
		return fmt.Errorf("failed to get available go versions: %w", err)
	}

	if _, ok := available[version]; !ok {
		return fmt.Errorf("go version %s not available", version)
	}

	logger.Debug("version is available", "version", version)

	downloadUrl := fmt.Sprintf(goDownloadUrlFormat, version, runtime.GOOS, runtime.GOARCH, g.platform.GoArchiveExt())
	logger.Debug("downloading", "url", downloadUrl)

	var downloadPath string
	err = spinner.Run("Downloading Go...", func() error {
		var e error
		downloadPath, e = httputils.DownloadFile(g.client, downloadUrl)
		return e
	})
	if err != nil {
		logger.Error("download failed", "url", downloadUrl, "err", err)
		return fmt.Errorf("failed to download go: %w", err)
	}
	logger.Debug("downloaded", "path", downloadPath)

	checksum, err := cryptoutils.SHA256(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to calculate go checksum: %w", err)
	}

	expectedChecksum, ok := available[version]
	if !ok {
		return fmt.Errorf("could not find checksum for go version %s", version)
	}

	if checksum != expectedChecksum {
		return fmt.Errorf("go checksum mismatch: got %s want %s", checksum, expectedChecksum)
	}
	logger.Debug("checksum matched")

	var extractDir string
	err = spinner.Run("Extracting archive...", func() error {
		var e error
		extractDir, e = archiveutils.Extract(downloadPath)
		return e
	})
	if err != nil {
		logger.Error("extract failed", "path", downloadPath, "err", err)
		return fmt.Errorf("failed to extract go: %w", err)
	}
	logger.Debug("extracted", "dir", extractDir)

	runtimeDir := RuntimeRoot(g.Name())
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return fmt.Errorf("failed to create installations directory: %w", err)
	}

	if err := os.Rename(filepath.Join(extractDir, g.Name()), RuntimeVersionRoot(g.Name(), version)); err != nil {
		logger.Error("rename failed", "err", err)
		return fmt.Errorf("failed to move go to installations directory: %w", err)
	}
	logger.Debug("install complete", "version", version)

	output.Success(msgs.Installed("Go", version))
	output.Line(msgs.UseHint(g.Name(), version))
	return nil
}

func (g GoRuntime) Uninstall(version string) error {
	logger.Debug("Uninstall", "version", version)
	output.Info(msgs.Uninstalling("Go", version))

	installed, err := g.installedVersions()
	if err != nil {
		return fmt.Errorf("failed to get installed go versions: %w", err)
	}

	if !slices.Contains(installed, version) {
		return fmt.Errorf("go version %s not installed", version)
	}

	logger.Debug("version is installed", "version", version)

	err = spinner.Run("Uninstalling Go...", func() error {
		if g.state.IsActive(g.Name(), version) {
			logger.Debug("removing shims (version was active)")
			if e := g.removeShims(); e != nil {
				return fmt.Errorf("failed to remove shims: %w", e)
			}
			if e := g.state.SetActive(g.Name(), ""); e != nil {
				return fmt.Errorf("failed to set active go version: %w", e)
			}
			logger.Debug("cleared active version")
		}

		versionDir := RuntimeVersionRoot(g.Name(), version)
		if e := os.RemoveAll(versionDir); e != nil {
			logger.Error("failed to remove installation directory", "path", versionDir, "err", e)
			return fmt.Errorf("failed to remove installation directory: %w", e)
		}
		logger.Debug("removed installation directory", "path", versionDir)
		return nil
	})
	if err != nil {
		return err
	}

	output.Success(msgs.Uninstalled("Go", version))
	return nil
}

func (g GoRuntime) Active() (string, error) {
	return g.state.Active(g.Name()), nil
}

func (g GoRuntime) Use(version string) error {
	installed, err := g.installedVersions()
	if err != nil {
		return fmt.Errorf("failed to get installed go versions: %w", err)
	}
	createShims := func() error {
		return spinner.Run("Creating shims...", func() error {
			if e := g.platform.CreateGoShims(version); e != nil {
				return e
			}
			return g.platform.EnsurePath(meta.ShimsDir())
		})
	}
	return RunUse(g.Name(), "Go", version, g.state, installed, createShims)
}

type goRelease struct {
	Version string   `json:"version"`
	Stable  bool     `json:"stable"`
	Files   []goFile `json:"files"`
}

type goFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	Kind     string `json:"kind"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
}

// availableVersions returns a map <version> : <checksum>
func (g GoRuntime) availableVersions(stableOnly bool) (map[string]string, error) {
	logger.Debug("fetching available versions", "stableOnly", stableOnly)
	res, err := g.client.Get(goVersionsUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch go releases: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read go releases: %w", err)
	}

	var releases []goRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to decode go releases: %w", err)
	}

	versions := make(map[string]string)

	for _, r := range releases {
		// just in case
		if !strings.HasPrefix(r.Version, g.Name()) {
			continue
		}

		if stableOnly && !r.Stable {
			continue
		}

		version := strings.TrimPrefix(r.Version, g.Name())

		for _, f := range r.Files {
			if f.OS != runtime.GOOS {
				continue
			}

			if f.Arch != runtime.GOARCH {
				continue
			}

			if f.Kind != "archive" {
				continue
			}

			versions[version] = f.SHA256
			break
		}
	}

	logger.Debug("available versions parsed", "count", len(versions))
	return versions, nil
}

func (g GoRuntime) installedVersions() ([]string, error) {
	return ListInstalledVersions(g.Name())
}

func (g GoRuntime) AvailableVersions(stableOnly bool) (map[string]string, error) {
	return g.availableVersions(stableOnly)
}

func (g GoRuntime) InstalledVersions() ([]string, error) {
	return g.installedVersions()
}

func (g GoRuntime) RemoveShims() error {
	return g.removeShims()
}

func (g GoRuntime) removeShims() error {
	shimsDir := meta.ShimsDir()
	logger.Debug("removeShims", "dir", shimsDir)
	entries, err := os.ReadDir(shimsDir)
	if err != nil {
		return fmt.Errorf("failed to read shims directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), g.Name()) {
			p := filepath.Join(shimsDir, entry.Name())
			logger.Debug("removing shim", "path", p)
			if err := os.Remove(p); err != nil {
				return fmt.Errorf("failed to remove shim: %w", err)
			}
		}
	}

	logger.Debug("removeShims done")
	return nil
}
