package archiveutils_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/kamilludwinski/runtimzzz/internal/utils/archiveutils"
)

func createZip(t *testing.T, files map[string]string) string {
	t.Helper()

	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "test.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)

	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}

		_, err = fw.Write([]byte(content))
		if err != nil {
			t.Fatal(err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return zipPath
}

func TestExtract_ZipSuccess(t *testing.T) {
	files := map[string]string{
		"file1.txt": "hello",
		"file2.txt": "world",
	}

	zipPath := createZip(t, files)

	dest, err := archiveutils.Extract(zipPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for name, expected := range files {
		p := filepath.Join(dest, name)

		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("failed to read extracted file: %v", err)
		}

		if string(data) != expected {
			t.Fatalf("unexpected content: got %q want %q", data, expected)
		}
	}
}

func TestExtract_ZipNestedDirs(t *testing.T) {
	files := map[string]string{
		"dir1/file.txt": "nested",
	}

	zipPath := createZip(t, files)

	dest, err := archiveutils.Extract(zipPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := filepath.Join(dest, "dir1", "file.txt")

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(data) != "nested" {
		t.Fatalf("unexpected content")
	}
}

func TestExtract_EmptyPath(t *testing.T) {
	_, err := archiveutils.Extract("")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtract_ZipSlip(t *testing.T) {
	files := map[string]string{
		"../../evil.txt": "bad",
	}

	zipPath := createZip(t, files)

	_, err := archiveutils.Extract(zipPath)
	if err == nil {
		t.Fatal("expected zip slip error")
	}
}
