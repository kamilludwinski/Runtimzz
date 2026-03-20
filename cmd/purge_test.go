package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kamilludwinski/runtimzzz/internal/meta"
	"github.com/kamilludwinski/runtimzzz/internal/state"

	"github.com/kamilludwinski/runtimzzz/internal/runtime"
)

func TestRunGlobalPurge(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(meta.AppDirOverride, dir)
	defer os.Unsetenv(meta.AppDirOverride)

	// Create app dir structure: logs + installations + shims + state.json
	logsDir := filepath.Join(dir, "logs")
	installDir := filepath.Join(dir, "installations", "go", "1.22.0")
	shimsDir := filepath.Join(dir, "shims")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		t.Fatal(err)
	}
	logFile := filepath.Join(logsDir, "test.log")
	if err := os.WriteFile(logFile, []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(dir, "state.json")
	stateData, _ := json.Marshal(map[string]interface{}{"active": map[string]string{"go": "1.22.0"}})
	if err := os.WriteFile(statePath, stateData, 0644); err != nil {
		t.Fatal(err)
	}

	runGlobalPurge()

	// logs/ and its contents must remain
	if _, err := os.Stat(logsDir); err != nil {
		t.Errorf("logs dir should remain: %v", err)
	}
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("log file should remain: %v", err)
	}
	// installations, shims, state.json must be gone
	if _, err := os.Stat(installDir); err == nil {
		t.Error("installations should be removed")
	}
	if _, err := os.Stat(shimsDir); err == nil {
		t.Error("shims dir should be removed")
	}
	if _, err := os.Stat(statePath); err == nil {
		t.Error("state.json should be removed")
	}
}

func TestRunGlobalPurge_NoAppDir(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(meta.AppDirOverride, dir)
	defer os.Unsetenv(meta.AppDirOverride)
	// Don't create anything - app dir doesn't exist
	os.RemoveAll(dir)

	runGlobalPurge()
	// Should not panic; nothing to purge
}

func TestState_ClearRuntime(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(meta.AppDirOverride, dir)
	defer os.Unsetenv(meta.AppDirOverride)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	st := state.NewState()
	st.SetActive("go", "1.22.0")
	st.SetActive("node", "20.0.0")

	if err := st.ClearRuntime("go"); err != nil {
		t.Fatal(err)
	}

	if st.Active("go") != "" {
		t.Errorf("go should be cleared, got %q", st.Active("go"))
	}
	if st.Active("node") != "20.0.0" {
		t.Errorf("node should remain, got %q", st.Active("node"))
	}

	// Reload from disk and verify
	st2 := state.NewState()
	if err := st2.Load(); err != nil {
		t.Fatal(err)
	}
	if st2.Active("go") != "" {
		t.Errorf("after reload: go should be cleared, got %q", st2.Active("go"))
	}
	if st2.Active("node") != "20.0.0" {
		t.Errorf("after reload: node should remain, got %q", st2.Active("node"))
	}
}

func TestHandleRuntime_Purge(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(meta.AppDirOverride, dir)
	defer os.Unsetenv(meta.AppDirOverride)

	// Create app dir: installations/go/1.22.0, shims with go shim, state with go active
	installDir := filepath.Join(dir, "installations", "go", "1.22.0")
	shimsDir := filepath.Join(dir, "shims")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Go removeShims removes files whose name starts with "go"
	shimFile := filepath.Join(shimsDir, "go.exe")
	if err := os.WriteFile(shimFile, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(dir, "state.json")
	stateData, _ := json.Marshal(map[string]interface{}{"active": map[string]string{"go": "1.22.0"}})
	if err := os.WriteFile(statePath, stateData, 0644); err != nil {
		t.Fatal(err)
	}

	st := state.NewState()
	if err := st.Load(); err != nil {
		t.Fatal(err)
	}
	runState = st
	runtime.Register(runtime.Init(st))

	HandleRuntime([]string{"go", "purge"})

	// installations/go/1.22.0 should be removed
	if _, err := os.Stat(installDir); err == nil {
		t.Error("go installation dir should be removed")
	}
	// go shim should be removed
	if _, err := os.Stat(shimFile); err == nil {
		t.Error("go shim should be removed")
	}
	// state should have no "go" key
	st2 := state.NewState()
	if err := st2.Load(); err != nil {
		t.Fatal(err)
	}
	if st2.Active("go") != "" {
		t.Errorf("state should have no go entry after purge, got %q", st2.Active("go"))
	}
}
