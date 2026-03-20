package cryptoutils_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/kamilludwinski/runtimzzz/internal/utils/cryptoutils"
)

func TestSHA256(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "empty file",
			content: "",
		},
		{
			name:    "small content",
			content: "hello world",
		},
		{
			name:    "larger content",
			content: "hello worldhello worldhello worldhello worldhello worldhello worldhello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "testfile")

			if err := os.WriteFile(file, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			got, err := cryptoutils.SHA256(file)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			sum := sha256.Sum256([]byte(tt.content))
			want := hex.EncodeToString(sum[:])

			if got != want {
				t.Fatalf("checksum mismatch: got %s want %s", got, want)
			}
		})
	}
}

func TestSHA256_FileNotExist(t *testing.T) {
	_, err := cryptoutils.SHA256("non-existent-file")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}
