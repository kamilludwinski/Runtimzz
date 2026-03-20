// Shim launcher for Windows: runs the active runtimz binary for a tool (go, gofmt, node, npm, npx, python, pip).
// Reads state.json, resolves (tool -> runtime -> version), and tries to run:
//   - installations/<runtime>/<version>/bin/<tool>.exe (or .cmd)
//   - falling back to installations/<runtime>/<version>/<tool>.exe (or .cmd)
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kamilludwinski/runtimzzz/internal/shimpaths"
)

// toolToRuntime maps shim tool name to runtime name (state.Active key).
var toolToRuntime = map[string]string{
	"go": "go", "gofmt": "go",
	"node": "node", "npm": "node", "npx": "node",
	"python": "python",
	"pip":    "python",
}

func main() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "runtimz shim: cannot get executable path:", err)
		os.Exit(1)
	}
	shimsDir := filepath.Dir(exe)
	appDir := filepath.Dir(shimsDir)
	statePath := filepath.Join(appDir, "state.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "runtimz shim: cannot read state:", err)
		os.Exit(1)
	}
	var state struct {
		Active map[string]string `json:"active"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		fmt.Fprintln(os.Stderr, "runtimz shim: invalid state:", err)
		os.Exit(1)
	}

	base := filepath.Base(exe)
	toolName := strings.TrimSuffix(base, ".exe")
	rtName, ok := toolToRuntime[toolName]
	if !ok {
		fmt.Fprintln(os.Stderr, "runtimz shim: unknown tool:", toolName)
		os.Exit(1)
	}

	version, ok := state.Active[rtName]
	if !ok || version == "" {
		fmt.Fprintf(os.Stderr, "runtimz shim: no active %s version set (run: rtz %s use <version>)\n", rtName, rtName)
		os.Exit(1)
	}

	installDir := shimpaths.RuntimeVersionRootWithBase(appDir, rtName, version)

	// Prefer bin/<tool>.exe or .cmd, but fall back to root/<tool>.exe or .cmd
	// to support layouts like the Node Windows .zip which puts node.exe at root.
	candidates := []string{
		filepath.Join(installDir, "bin", toolName+".exe"),
		filepath.Join(installDir, "bin", toolName+".cmd"),
		filepath.Join(installDir, toolName+".exe"),
		filepath.Join(installDir, toolName+".cmd"),
	}

	var toolPath string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			toolPath = c
			break
		}
	}
	if toolPath == "" {
		fmt.Fprintln(os.Stderr, "runtimz shim: active version binary not found in", installDir)
		os.Exit(1)
	}

	cmd := exec.Command(toolPath, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
