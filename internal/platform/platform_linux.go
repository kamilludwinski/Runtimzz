//go:build linux

package platform

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
	"github.com/kamilludwinski/runtimzzz/internal/meta"
	"github.com/kamilludwinski/runtimzzz/internal/output"
)

type PlatformImpl struct{}

var _ Platform = (*PlatformImpl)(nil)

func (p PlatformImpl) GoArchiveExt() string {
	return "tar.gz"
}

func (p PlatformImpl) EnsurePath(path string) error {
	logger.Debug("EnsurePath", "path", path)
	dir := filepath.Clean(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create shims directory: %w", err)
	}

	if pathOnEnv(dir, os.Getenv("PATH")) {
		logger.Debug("path already on PATH")
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to resolve home directory: %w", err)
	}

	shell := os.Getenv("SHELL")
	profiles := profileFilesForShell(shell, home)
	if len(profiles) == 0 {
		// Fallback to ~/.profile
		profiles = []string{filepath.Join(home, ".profile")}
	}

	line := fmt.Sprintf("export PATH=%q:$PATH", dir)
	updatedAny := false
	for _, pf := range profiles {
		ok, err := ensurePathInProfile(pf, dir, line)
		if err != nil {
			logger.Error("failed to update profile", "file", pf, "err", err)
			continue
		}
		if ok {
			updatedAny = true
		}
	}

	if !updatedAny {
		// As a last resort, just tell the user what to run.
		output.Info(fmt.Sprintf("Add the shims directory to your PATH. Run: %s", line))
		return nil
	}

	// Update this process' PATH so subsequent commands in the same process see it.
	os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	output.Info("Updated your shell profile to include the Runtimz shims directory on PATH. Restart your shell or open a new terminal for it to take effect.")
	return nil
}

// pathOnEnv reports whether dir already appears on the PATH string.
func pathOnEnv(dir, envPath string) bool {
	clean := filepath.Clean(dir)
	for _, entry := range filepath.SplitList(envPath) {
		if filepath.Clean(entry) == clean {
			return true
		}
	}
	return false
}

// profileFilesForShell returns candidate profile files for the given shell.
func profileFilesForShell(shell, home string) []string {
	shell = filepath.Base(shell)
	switch shell {
	case "zsh":
		return []string{filepath.Join(home, ".zshrc")}
	case "bash":
		// Prefer .bashrc, fall back to .bash_profile if present.
		return []string{filepath.Join(home, ".bashrc"), filepath.Join(home, ".bash_profile")}
	case "fish":
		return []string{filepath.Join(home, ".config", "fish", "config.fish")}
	default:
		// Unknown shell; common fallbacks.
		return []string{filepath.Join(home, ".bashrc"), filepath.Join(home, ".profile")}
	}
}

// ensurePathInProfile appends the export line to the profile file if the dir is not already present.
// It returns true if the file was created or modified.
func ensurePathInProfile(profilePath, dir, exportLine string) (bool, error) {
	// If the file exists and already mentions the directory, do nothing.
	if f, err := os.Open(profilePath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), dir) {
				return false, nil
			}
		}
		if err := scanner.Err(); err != nil {
			return false, err
		}
	}

	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return false, fmt.Errorf("failed to create profile dir %s: %w", filepath.Dir(profilePath), err)
	}

	f, err := os.OpenFile(profilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false, fmt.Errorf("failed to open profile %s: %w", profilePath, err)
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, ""); err != nil {
		return false, err
	}
	if _, err := fmt.Fprintln(f, "# Added by Runtimz: ensure shims directory is on PATH"); err != nil {
		return false, err
	}
	if _, err := fmt.Fprintln(f, exportLine); err != nil {
		return false, err
	}

	logger.Debug("profile updated", "file", profilePath)
	return true, nil
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
	targets := []struct {
		name string
		path string
	}{
		{"go", filepath.Join(binDir, "go")},
		{"gofmt", filepath.Join(binDir, "gofmt")},
	}

	for _, t := range targets {
		if _, err := os.Stat(t.path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to stat %s: %w", t.path, err)
		}

		absPath, err := filepath.Abs(t.path)
		if err != nil {
			return fmt.Errorf("failed to resolve path for %s: %w", t.name, err)
		}

		// Ensure the target binary itself is executable; some archives may not
		// preserve mode bits as expected.
		if err := os.Chmod(absPath, 0755); err != nil && !os.IsPermission(err) {
			return fmt.Errorf("failed to chmod target %s: %w", absPath, err)
		}

		shimPath := filepath.Join(shimsDir, t.name)
		content := fmt.Sprintf("#!/usr/bin/env bash\nexec %q \"$@\"\n", absPath)
		if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to write shim %s: %w", t.name, err)
		}
		logger.Debug("shim written", "name", t.name, "path", shimPath)
	}

	logger.Debug("CreateGoShims done")
	return nil
}

// CreateShims creates shims for each tool in binDir (e.g. node, npm, npx).
// On Linux each tool is resolved as binDir/<tool> (no extension).
func (p PlatformImpl) CreateShims(binDir string, tools []string) error {
	logger.Debug("CreateShims", "binDir", binDir, "tools", tools)
	shimsDir := meta.ShimsDir()
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		return fmt.Errorf("failed to create shims directory: %w", err)
	}

	for _, name := range tools {
		targetPath := filepath.Join(binDir, name)
		var content string
		if _, err := os.Stat(targetPath); err != nil {
			if os.IsNotExist(err) {
				// Node: npm/npx may live under lib/node_modules/npm/bin/ as .js scripts
				if name == "npm" || name == "npx" {
					baseDir := filepath.Dir(binDir)
					scriptName := name + "-cli.js"
					scriptPath := filepath.Join(baseDir, "lib", "node_modules", "npm", "bin", scriptName)
					if _, scriptErr := os.Stat(scriptPath); scriptErr == nil {
						nodePath := filepath.Join(binDir, "node")
						nodeAbs, errNode := filepath.Abs(nodePath)
						scriptAbs, errScript := filepath.Abs(scriptPath)
						if errNode == nil && errScript == nil {
							content = fmt.Sprintf("#!/usr/bin/env bash\nexec %q %q \"$@\"\n", nodeAbs, scriptAbs)
						}
					}
				}
				// Python: on Linux layout is .../bin/python3, .../bin/pip3 (or python, pip)
				if content == "" && (name == "python" || name == "pip") {
					baseDir := binDir
					if name == "pip" {
						// pip is looked up in Scripts on Windows; on Linux use version root's bin
						baseDir = filepath.Dir(binDir)
					}
					for _, candidate := range []string{name + "3", name} {
						tryPath := filepath.Join(baseDir, "bin", candidate)
						if _, statErr := os.Stat(tryPath); statErr == nil {
							absPath, absErr := filepath.Abs(tryPath)
							if absErr == nil {
								content = fmt.Sprintf("#!/usr/bin/env bash\nexec %q \"$@\"\n", absPath)
								break
							}
						}
					}
				}
				if content == "" {
					continue
				}
			} else {
				return fmt.Errorf("failed to stat %s: %w", targetPath, err)
			}
		} else {
			absPath, err := filepath.Abs(targetPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path for %s: %w", name, err)
			}

			if err := os.Chmod(absPath, 0755); err != nil && !os.IsPermission(err) {
				return fmt.Errorf("failed to chmod target %s: %w", absPath, err)
			}

			content = fmt.Sprintf("#!/usr/bin/env bash\nexec %q \"$@\"\n", absPath)
		}

		shimPath := filepath.Join(shimsDir, name)
		if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to write shim %s: %w", name, err)
		}
		logger.Debug("shim written", "name", name, "path", shimPath)
	}

	logger.Debug("CreateShims done")
	return nil
}
