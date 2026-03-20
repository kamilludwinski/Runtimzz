package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
	"github.com/kamilludwinski/runtimzzz/internal/meta"
	"github.com/kamilludwinski/runtimzzz/internal/output"
	"github.com/kamilludwinski/runtimzzz/internal/state"
	"github.com/kamilludwinski/runtimzzz/internal/update"
	"github.com/kamilludwinski/runtimzzz/internal/utils/archiveutils"
	"github.com/kamilludwinski/runtimzzz/internal/utils/httputils"
)

// runState is set by Run() so runtime purge can clear state.
var runState *state.State

const (
	globalVersionAlias = "v"
	runtimesList       = "go, node, python"
)

func printHelp() {
	output.Line("Global usage:")
	output.Line("\trtz version - print app version")
	output.Line("\trtz update - update rtz to the latest version")
	output.Line("\trtz purge - remove all runtimz data except logs")
	output.Line("")
	output.Line("Runtime usage:")
	output.Line("\trtz <runtime> ls - print available and installed versions")
	output.Line("\trtz <runtime> install <version> - install a version")
	output.Line("\trtz <runtime> i <version> - install a version")
	output.Line("\trtz <runtime> uninstall <version> - uninstall a version")
	output.Line("\trtz <runtime> u <version> - uninstall a version")
	output.Line("\trtz <runtime> use <version> - set active version")
	output.Line("\trtz <runtime> purge - uninstall all versions for a runtime")
	output.Line("\t  runtimes: " + runtimesList)
	output.Line("")
}

func runGlobalPurge() {
	appDir := meta.AppDir()
	if appDir == "" {
		output.Error("Failed to resolve app directory")
		return
	}

	entries, err := os.ReadDir(appDir)
	if err != nil {
		// If the directory does not exist yet, there's nothing to purge.
		if os.IsNotExist(err) {
			output.Warn("Nothing to purge: no runtimz data directory found")
			return
		}
		output.Error("Failed to read app directory: " + err.Error())
		return
	}

	output.Info("Purging all runtimz data (keeping logs)...")

	for _, entry := range entries {
		name := entry.Name()
		if name == "logs" {
			continue
		}
		path := filepath.Join(appDir, name)
		if err := os.RemoveAll(path); err != nil {
			output.Error("Failed to remove " + path + ": " + err.Error())
			return
		}
	}

	output.Success("All runtimz data purged (logs preserved)")
}

func runUpdate() {
	output.Info("Checking for updates...")
	rel, err := update.FetchRelease("")
	if err != nil {
		output.Error("Failed to fetch release: " + err.Error())
		return
	}
	version := rel.TagName
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}
	url := update.AssetURLByPlatform(rel, runtime.GOOS, runtime.GOARCH)
	if url == "" {
		output.Error(fmt.Sprintf("No release asset for %s/%s", runtime.GOOS, runtime.GOARCH))
		return
	}
	path, err := httputils.DownloadFile(nil, url)
	if err != nil {
		output.Error("Download failed: " + err.Error())
		return
	}
	defer os.Remove(path)

	var newBinary string
	if filepath.Ext(path) == ".exe" {
		newBinary = path
	} else {
		extDir, err := archiveutils.Extract(path)
		if err != nil {
			output.Error("Extract failed: " + err.Error())
			return
		}
		defer os.RemoveAll(extDir)
		// Find rtz or rtz.exe in extracted dir
		if runtime.GOOS == "windows" {
			newBinary = filepath.Join(extDir, "rtz.exe")
		} else {
			newBinary = filepath.Join(extDir, "rtz")
		}
		if _, err := os.Stat(newBinary); err != nil {
			// Some zips have a single file at root
			entries, _ := os.ReadDir(extDir)
			for _, e := range entries {
				if !e.IsDir() && (e.Name() == "rtz" || e.Name() == "rtz.exe") {
					newBinary = filepath.Join(extDir, e.Name())
					break
				}
			}
		}
		if _, err := os.Stat(newBinary); err != nil {
			output.Error("rtz binary not found in archive")
			return
		}
	}
	currentExe, err := os.Executable()
	if err != nil {
		output.Error("Failed to get executable path: " + err.Error())
		return
	}
	dir := filepath.Dir(currentExe)
	newPath := filepath.Join(dir, "rtz.new.exe")
	if runtime.GOOS != "windows" {
		newPath = filepath.Join(dir, "rtz.new")
	}
	data, err := os.ReadFile(newBinary)
	if err != nil {
		output.Error("Failed to read new binary: " + err.Error())
		return
	}
	if err := os.WriteFile(newPath, data, 0755); err != nil {
		output.Error("Failed to write new binary: " + err.Error())
		return
	}
	if runtime.GOOS == "windows" {
		// Spawn updater that waits for us to exit then renames
		cmd := exec.Command(currentExe)
		cmd.Env = append(os.Environ(),
			update.EnvUpdatePID+"="+fmt.Sprint(os.Getpid()),
			update.EnvUpdateNew+"="+newPath,
			update.EnvUpdateTarget+"="+currentExe,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			output.Error("Failed to start updater: " + err.Error())
			return
		}
		output.Success("Update scheduled. Restart rtz to use version " + version)
		return
	}
	// Unix: replace in place (often works when not running from same path)
	if err := os.Rename(newPath, currentExe); err != nil {
		output.Error("Failed to replace binary: " + err.Error())
		return
	}
	output.Success("Updated to version " + version)
}

func Run(args []string, st *state.State) {
	runState = st
	logger.Debug("cmd.Run", "args", args)

	// Version check: warn if a newer release is available (once per run)
	if newer := update.CheckForNewerVersion(); newer != "" {
		output.Warn(fmt.Sprintf("A newer Runtimz version %s is available (you are on %s). Run 'rtz update' to upgrade.", newer, meta.AppVersion))
	}

	if len(args) < 2 {
		printHelp()
		return
	}

	cmd := args[1]

	switch cmd {
	case "version", globalVersionAlias:
		output.Line(meta.AppVersion)
	case "update":
		runUpdate()
	case "purge":
		runGlobalPurge()
	default:
		HandleRuntime(args[1:])
	}
}
