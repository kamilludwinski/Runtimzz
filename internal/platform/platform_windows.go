//go:build windows

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
	"github.com/kamilludwinski/runtimzzz/internal/meta"
	"github.com/kamilludwinski/runtimzzz/internal/shimembed"
)

type PlatformImpl struct{}

var _ Platform = (*PlatformImpl)(nil)

func (p PlatformImpl) GoArchiveExt() string {
	return "zip"
}

func (p PlatformImpl) EnsurePath(path string) error {
	logger.Debug("EnsurePath", "path", path)
	dir := filepath.Clean(path)
	escaped := strings.ReplaceAll(dir, `"`, `""`)

	psCommand := fmt.Sprintf(`
$target = "%s"
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not $currentPath) { $currentPath = "" }

$paths = $currentPath.Split(";", [System.StringSplitOptions]::RemoveEmptyEntries)
$exists = $false
foreach ($p in $paths) {
	if ($p.Trim().ToLower() -eq $target.Trim().ToLower()) {
		$exists = $true
		break
	}
}

if (-not $exists) {
	# Prefer putting the shims directory *before* WindowsApps so we override
	# Store aliases like python.exe.
	$windowsApps = $paths | Where-Object {
		$clean = $_.TrimEnd('\')
		$clean.ToLower().EndsWith("\microsoft\windowsapps")
	} | Select-Object -First 1

	if ($windowsApps) {
		$newPaths = @()
		foreach ($p in $paths) {
			if ($p -eq $windowsApps) {
				$newPaths += $target
			}
			$newPaths += $p
		}
		$newPath = [string]::Join(";", $newPaths)
	} else {
		$newPath = $currentPath
		if ($newPath -and -not $newPath.EndsWith(";")) { $newPath += ";" }
		$newPath += $target
	}
	[Environment]::SetEnvironmentVariable("Path", $newPath, "User")
	exit 0
} else {
	exit 1
}
`, escaped)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCommand)
	err := cmd.Run()
	if err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			logger.Debug("path already in user PATH")
			return nil
		}
		logger.Error("EnsurePath powershell failed", "err", err)
		return fmt.Errorf("failed to update path persistently: %v", err)
	}

	logger.Debug("path added to user PATH")
	return nil
}

// shimTarget describes a single shim: base name (e.g. "go") and path to the real executable.
type shimTarget struct {
	baseName string
	exePath  string
}

// writeShimsForTargets writes .cmd, bash, and optionally .exe shims for each target.
func (p PlatformImpl) writeShimsForTargets(shimsDir string, targets []shimTarget, embedExeShims bool) error {
	for _, t := range targets {
		if _, err := os.Stat(t.exePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to stat %s: %w", t.exePath, err)
		}

		absExe, err := filepath.Abs(t.exePath)
		if err != nil {
			return fmt.Errorf("failed to resolve path for %s: %w", t.baseName, err)
		}

		cmdPath := filepath.Join(shimsDir, t.baseName+".cmd")
		cmdContent := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", absExe)
		if err := os.WriteFile(cmdPath, []byte(cmdContent), 0755); err != nil {
			return fmt.Errorf("failed to write shim %s.cmd: %w", t.baseName, err)
		}

		bashPath := filepath.Join(shimsDir, t.baseName)
		unixPath := windowsPathToGitBash(absExe)
		bashContent := fmt.Sprintf("#!/usr/bin/env bash\nexec \"%s\" \"$@\"\n", unixPath)
		if err := os.WriteFile(bashPath, []byte(bashContent), 0755); err != nil {
			return fmt.Errorf("failed to write shim %s: %w", t.baseName, err)
		}
		logger.Debug("shim written", "baseName", t.baseName, "cmd", cmdPath, "bash", bashPath)

		if embedExeShims && len(shimembed.ShimExe) > 0 {
			exeShimPath := filepath.Join(shimsDir, t.baseName+".exe")
			if err := os.WriteFile(exeShimPath, shimembed.ShimExe, 0755); err != nil {
				return fmt.Errorf("failed to write shim %s.exe: %w", t.baseName, err)
			}
			logger.Debug("exe shim written", "path", exeShimPath)
		}
	}
	return nil
}

func (p PlatformImpl) CreateGoShims(version string) error {
	logger.Debug("CreateGoShims", "version", version)
	if version == "" {
		return fmt.Errorf("version is required")
	}

	appDir := meta.AppDir()
	shimsDir := meta.ShimsDir()
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		return fmt.Errorf("failed to create shims directory: %w", err)
	}

	binDir := filepath.Join(appDir, "installations", "go", version, "bin")
	targets := []shimTarget{
		{"go", filepath.Join(binDir, "go.exe")},
		{"gofmt", filepath.Join(binDir, "gofmt.exe")},
	}
	if err := p.writeShimsForTargets(shimsDir, targets, true); err != nil {
		return err
	}
	logger.Debug("CreateGoShims done")
	return nil
}

// CreateShims creates shims for each tool in binDir (e.g. node, npm, npx).
// Each tool is resolved as binDir/<tool>.exe or binDir/<tool>.cmd.
func (p PlatformImpl) CreateShims(binDir string, tools []string) error {
	logger.Debug("CreateShims", "binDir", binDir, "tools", tools)
	shimsDir := meta.ShimsDir()
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		return fmt.Errorf("failed to create shims directory: %w", err)
	}

	var targets []shimTarget
	for _, baseName := range tools {
		exePath := filepath.Join(binDir, baseName+".exe")
		if _, err := os.Stat(exePath); err != nil {
			if os.IsNotExist(err) {
				exePath = filepath.Join(binDir, baseName+".cmd")
			}
			if _, err := os.Stat(exePath); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("failed to stat %s: %w", exePath, err)
			}
		}
		targets = append(targets, shimTarget{baseName, exePath})
	}

	if err := p.writeShimsForTargets(shimsDir, targets, true); err != nil {
		return err
	}
	logger.Debug("CreateShims done")
	return nil
}

// windowsPathToGitBash converts a Windows path (e.g. C:\Users\foo\go.exe) to Git Bash form (/c/Users/foo/go.exe).
func windowsPathToGitBash(winPath string) string {
	slash := filepath.ToSlash(winPath)
	if len(slash) >= 2 && slash[1] == ':' {
		drive := strings.ToLower(slash[:1])
		return "/" + drive + slash[2:]
	}

	return slash
}
