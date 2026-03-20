package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	"github.com/kamilludwinski/runtimzzz/internal/utils/httputils"
)

const (
	pythonIndexURL        = "https://www.python.org/ftp/python/"
	pythonDownloadFmt     = "https://www.python.org/ftp/python/%s/%s"
	pythonReleaseCycleURL = "https://peps.python.org/api/release-cycle.json"
)

var pythonVersionDirRe = regexp.MustCompile(`href="([0-9]+\.[0-9]+\.[0-9]+)/"`)

func InitPython(state *state.State) Runtime {
	logger.Debug("PythonRuntime Init")
	return &PythonRuntime{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		platform: platform.PlatformImpl{},
		state:    state,
	}
}

type PythonRuntime struct {
	client   *http.Client
	platform platform.Platform
	state    *state.State
}

var _ Runtime = (*PythonRuntime)(nil)

func (p PythonRuntime) Name() string {
	return "python"
}

func (p PythonRuntime) Ls(limit int) error {
	logger.Debug("Ls", "limit", limit)

	var available map[string]string
	err := spinner.Run("Fetching Python versions...", func() error {
		var e error
		available, e = p.availableVersions(true)
		return e
	})
	if err != nil {
		logger.Error("availableVersions failed", "err", err)
		return fmt.Errorf("failed to get available python versions: %w", err)
	}

	installed, err := p.installedVersions()
	if err != nil {
		logger.Error("installedVersions failed", "err", err)
		return fmt.Errorf("failed to get installed python versions: %w", err)
	}

	active, _ := p.Active()
	PrintVersions("Python", available, installed, active, limit)
	return nil
}

func (p PythonRuntime) Install(version string) error {
	logger.Debug("Install", "version", version)
	output.Info(msgs.Installing("Python", version))

	if runtime.GOOS != "windows" {
		return fmt.Errorf("python runtime is currently supported on Windows only")
	}

	installed, err := p.installedVersions()
	if err != nil {
		return fmt.Errorf("failed to get installed python versions: %w", err)
	}
	if slices.Contains(installed, version) {
		return fmt.Errorf("python version %s already installed", version)
	}

	available, err := p.availableVersions(true)
	if err != nil {
		return fmt.Errorf("failed to get available python versions: %w", err)
	}
	if _, ok := available[version]; !ok {
		return fmt.Errorf("python version %s not available", version)
	}

	fileName, err := pythonArchiveName(version)
	if err != nil {
		return err
	}
	downloadURL := fmt.Sprintf(pythonDownloadFmt, version, fileName)
	logger.Debug("downloading", "url", downloadURL)

	var downloadPath string
	err = spinner.Run("Downloading Python...", func() error {
		var e error
		downloadPath, e = httputils.DownloadFile(p.client, downloadURL)
		return e
	})
	if err != nil {
		return fmt.Errorf("failed to download python: %w", err)
	}

	var extractDir string
	err = spinner.Run("Extracting archive...", func() error {
		var e error
		extractDir, e = archiveutils.Extract(downloadPath)
		return e
	})
	if err != nil {
		return fmt.Errorf("failed to extract python: %w", err)
	}

	installDir, err := EnsureRuntimeVersionDir(p.Name(), version)
	if err != nil {
		return err
	}
	if err := os.Rename(extractDir, installDir); err != nil {
		return fmt.Errorf("failed to move python to installations: %w", err)
	}

	// Install matching pip version into the embedded distribution at its final path.
	if err := spinner.Run("Bootstrapping pip...", func() error {
		return p.installPip(installDir, version)
	}); err != nil {
		return fmt.Errorf("failed to install pip for python %s: %w", version, err)
	}

	output.Success(msgs.Installed("Python", version))
	output.Line(msgs.UseHint(p.Name(), version))
	return nil
}

func (p PythonRuntime) Uninstall(version string) error {
	logger.Debug("Uninstall", "version", version)
	output.Info(msgs.Uninstalling("Python", version))

	installed, err := p.installedVersions()
	if err != nil {
		return fmt.Errorf("failed to get installed python versions: %w", err)
	}
	if !slices.Contains(installed, version) {
		return fmt.Errorf("python version %s not installed", version)
	}

	err = spinner.Run("Uninstalling Python...", func() error {
		if p.state.IsActive(p.Name(), version) {
			if e := p.removeShims(); e != nil {
				return fmt.Errorf("failed to remove shims: %w", e)
			}
			if e := p.state.SetActive(p.Name(), ""); e != nil {
				return fmt.Errorf("failed to set active python version: %w", e)
			}
		}
		versionDir := RuntimeVersionRoot(p.Name(), version)
		if e := os.RemoveAll(versionDir); e != nil {
			return fmt.Errorf("failed to remove installation directory: %w", e)
		}
		return nil
	})
	if err != nil {
		return err
	}
	output.Success(msgs.Uninstalled("Python", version))
	return nil
}

func (p PythonRuntime) Active() (string, error) {
	return p.state.Active(p.Name()), nil
}

func (p PythonRuntime) Use(version string) error {
	installed, err := p.installedVersions()
	if err != nil {
		return fmt.Errorf("failed to get installed python versions: %w", err)
	}
	createShims := func() error {
		return spinner.Run("Creating shims...", func() error {
			baseDir := RuntimeVersionRoot(p.Name(), version)
			if e := p.platform.CreateShims(baseDir, []string{"python"}); e != nil {
				return e
			}
			scriptsDir := RuntimeScriptsDir(p.Name(), version)
			if e := p.platform.CreateShims(scriptsDir, []string{"pip"}); e != nil {
				return e
			}
			return p.platform.EnsurePath(meta.ShimsDir())
		})
	}
	return RunUse(p.Name(), "Python", version, p.state, installed, createShims)
}

func (p PythonRuntime) availableVersions(stableOnly bool) (map[string]string, error) {
	res, err := p.client.Get(pythonIndexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch python index: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	matches := pythonVersionDirRe.FindAllStringSubmatch(string(body), -1)
	versions := make(map[string]string)
	var stableBranches map[string]bool
	if stableOnly {
		stableBranches, err = p.pythonStableBranches()
		if err != nil {
			logger.Error("pythonStableBranches failed, falling back to unfiltered list", "err", err)
		}
	}

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		version := m[1]
		if stableOnly && stableBranches != nil {
			branch := pythonBranch(version)
			if branch == "" {
				continue
			}
			if !stableBranches[branch] {
				// Skip versions that belong to pre-release-only branches (e.g. 3.15.x while 3.15 is still in pre-release).
				continue
			}
		}
		versions[version] = ""
	}
	logger.Debug("available python versions", "count", len(versions))
	return versions, nil
}

// pythonBranch returns the feature branch (major.minor) for a version like 3.14.2 -> 3.14.
func pythonBranch(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "." + parts[1]
}

// pythonStableBranches queries the release-cycle API and returns branches that
// are past the pre-release phase (bugfix, security, or end-of-life).
func (p PythonRuntime) pythonStableBranches() (map[string]bool, error) {
	type cycle struct {
		Status string `json:"status"`
	}

	res, err := p.client.Get(pythonReleaseCycleURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch python release cycles: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read python release cycles: %w", err)
	}

	var raw map[string]cycle
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode python release cycles: %w", err)
	}

	stable := make(map[string]bool)
	for branch, c := range raw {
		status := strings.ToLower(c.Status)
		// Only expose branches that are past the pre-release/feature phase.
		if status == "bugfix" || status == "security" || status == "end-of-life" {
			stable[branch] = true
		}
	}

	logger.Debug("pythonStableBranches parsed", "count", len(stable))
	return stable, nil
}

// installPip bootstraps pip into the extracted embedded Python distribution located at pythonRoot.
// It:
//   - Enables site-packages by uncommenting "import site" in pythonXY._pth (if present)
//   - Downloads get-pip.py
//   - Runs "python.exe get-pip.py" inside pythonRoot so that pip and Scripts/ are created there
func (p PythonRuntime) installPip(pythonRoot, version string) error {
	logger.Debug("installPip", "root", pythonRoot, "version", version)

	entries, err := os.ReadDir(pythonRoot)
	if err != nil {
		return fmt.Errorf("failed to read python root: %w", err)
	}

	// Enable site-packages by uncommenting "import site" in pythonXY._pth if present.
	var pthPath string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, "._pth") {
			pthPath = filepath.Join(pythonRoot, name)
			break
		}
	}
	if pthPath != "" {
		data, err := os.ReadFile(pthPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", pthPath, err)
		}
		lines := strings.Split(string(data), "\n")
		changed := false
		for i, line := range lines {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "#import site") {
				lines[i] = "import site"
				changed = true
				break
			}
		}
		if changed {
			if err := os.WriteFile(pthPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
				return fmt.Errorf("failed to update %s: %w", pthPath, err)
			}
		}
	}

	// Download get-pip.py.
	getPipURL := "https://bootstrap.pypa.io/get-pip.py"
	tmpPath, err := httputils.DownloadFile(p.client, getPipURL)
	if err != nil {
		return fmt.Errorf("failed to download get-pip.py: %w", err)
	}
	defer os.Remove(tmpPath)

	getPipPath := filepath.Join(pythonRoot, "get-pip.py")
	if err := os.Rename(tmpPath, getPipPath); err != nil {
		return fmt.Errorf("failed to place get-pip.py: %w", err)
	}

	pythonExe := filepath.Join(pythonRoot, "python.exe")
	if _, err := os.Stat(pythonExe); err != nil {
		return fmt.Errorf("python.exe not found in %s: %w", pythonRoot, err)
	}

	cmd := exec.Command(pythonExe, "get-pip.py", "--no-warn-script-location")
	cmd.Dir = pythonRoot
	cmd.Env = append(os.Environ(), "PYTHONHOME="+pythonRoot)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("get-pip failed", "output", string(outputBytes), "err", err)
		return fmt.Errorf("get-pip.py failed: %w", err)
	}

	logger.Debug("pip installed for python", "version", version)
	return nil
}

func (p PythonRuntime) installedVersions() ([]string, error) {
	return ListInstalledVersions(p.Name())
}

func (p PythonRuntime) AvailableVersions(stableOnly bool) (map[string]string, error) {
	return p.availableVersions(stableOnly)
}

func (p PythonRuntime) InstalledVersions() ([]string, error) {
	return p.installedVersions()
}

func (p PythonRuntime) RemoveShims() error {
	return p.removeShims()
}

func (p PythonRuntime) removeShims() error {
	shimsDir := meta.ShimsDir()
	entries, err := os.ReadDir(shimsDir)
	if err != nil {
		return err
	}
	toRemove := map[string]bool{"python": true, "python3": true, "pip": true, "pip3": true}
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

func pythonArchiveName(version string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("python runtime is currently supported on Windows only")
	}
	switch runtime.GOARCH {
	case "amd64":
		return fmt.Sprintf("python-%s-embed-amd64.zip", version), nil
	default:
		return "", fmt.Errorf("unsupported platform for Python: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

