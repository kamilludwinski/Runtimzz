package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kamilludwinski/runtimzzz/internal/meta"
)

func TestListInstalledVersions(t *testing.T) {
	dir := t.TempDir()
	os.Setenv(meta.AppDirOverride, dir)
	defer os.Unsetenv(meta.AppDirOverride)

	// Empty dir -> empty list
	versions, err := ListInstalledVersions("go")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 0 {
		t.Errorf("expected no versions, got %v", versions)
	}

	// Create installations/go/1.22.0 and 1.21.0
	goRoot := filepath.Join(dir, "installations", "go")
	for _, v := range []string{"1.22.0", "1.21.0"} {
		if err := os.MkdirAll(filepath.Join(goRoot, v), 0755); err != nil {
			t.Fatal(err)
		}
	}
	versions, err = ListInstalledVersions("go")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %v", versions)
	}
	// Should be sorted
	if versions[0] != "1.21.0" || versions[1] != "1.22.0" {
		t.Errorf("expected sorted [1.21.0 1.22.0], got %v", versions)
	}
}
