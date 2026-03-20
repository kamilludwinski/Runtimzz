package meta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppDir_Override(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(AppDirOverride, dir)
	defer os.Unsetenv(AppDirOverride)

	got := AppDir()
	if got != dir {
		t.Errorf("AppDir() with RTZ_APP_DIR = %q got %q", dir, got)
	}
}

func TestInstallationsDir(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(AppDirOverride, dir)
	defer os.Unsetenv(AppDirOverride)

	got := InstallationsDir()
	want := filepath.Join(dir, "installations")
	if got != want {
		t.Errorf("InstallationsDir() = %q, want %q", got, want)
	}
}

func TestShimsDir(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(AppDirOverride, dir)
	defer os.Unsetenv(AppDirOverride)

	got := ShimsDir()
	want := filepath.Join(dir, "shims")
	if got != want {
		t.Errorf("ShimsDir() = %q, want %q", got, want)
	}
}

func TestLogDir(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(AppDirOverride, dir)
	defer os.Unsetenv(AppDirOverride)

	got := LogDir()
	want := filepath.Join(dir, "logs")
	if got != want {
		t.Errorf("LogDir() = %q, want %q", got, want)
	}
}
