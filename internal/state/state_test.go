package state

import (
	"os"
	"testing"

	"github.com/kamilludwinski/runtimzzz/internal/meta"
)

func TestState_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(meta.AppDirOverride, dir)
	defer os.Unsetenv(meta.AppDirOverride)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	st := NewState()
	if err := st.SetActive("go", "1.22.0"); err != nil {
		t.Fatal(err)
	}
	st2 := NewState()
	if err := st2.Load(); err != nil {
		t.Fatal(err)
	}
	if st2.Active("go") != "1.22.0" {
		t.Errorf("after Load: Active(go) = %q, want 1.22.0", st2.Active("go"))
	}
}

func TestState_ClearRuntime(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(meta.AppDirOverride, dir)
	defer os.Unsetenv(meta.AppDirOverride)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	st := NewState()
	if err := st.SetActive("go", "1.22.0"); err != nil {
		t.Fatal(err)
	}
	if err := st.SetActive("node", "20.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := st.ClearRuntime("go"); err != nil {
		t.Fatal(err)
	}
	if st.Active("go") != "" {
		t.Errorf("ClearRuntime(go): Active(go) = %q", st.Active("go"))
	}
	if st.Active("node") != "20.0.0" {
		t.Errorf("ClearRuntime(go): Active(node) = %q", st.Active("node"))
	}
	// Reload and verify persisted
	st2 := NewState()
	if err := st2.Load(); err != nil {
		t.Fatal(err)
	}
	if st2.Active("go") != "" {
		t.Errorf("after reload: Active(go) = %q", st2.Active("go"))
	}
}

func TestState_LoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(meta.AppDirOverride, dir)
	defer os.Unsetenv(meta.AppDirOverride)
	// No state.json
	st := NewState()
	if err := st.Load(); err != nil {
		t.Fatal(err)
	}
	if st.Active("go") != "" {
		t.Errorf("Load with no file: Active(go) = %q", st.Active("go"))
	}
}
