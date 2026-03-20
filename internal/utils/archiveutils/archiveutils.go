package archiveutils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
)

// Extract extracts the archive to a temp dir and returns the path
func Extract(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	logger.Debug("Extract", "path", path)
	switch {
	case strings.HasSuffix(path, ".tar.gz"), strings.HasSuffix(path, ".tgz"):
		return extractTarGz(path)
	case filepath.Ext(path) == ".zip":
		return extractZip(path)
	default:
		return "", fmt.Errorf("unsupported archive extension: %s", path)
	}
}

func extractTarGz(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open tar.gz: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	destDir, err := os.MkdirTemp("", "extract-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("tar read: %w", err)
		}
		destPath := filepath.Join(destDir, hdr.Name)
		if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return "", fmt.Errorf("invalid file path in tar: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return "", err
			}
			out, err := os.Create(destPath)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", err
			}
			out.Close()
		}
	}
	logger.Debug("extractTarGz done", "destDir", destDir)
	return destDir, nil
}

func extractZip(path string) (string, error) {
	logger.Debug("extractZip", "path", path)
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file: %w", err)
	}
	defer zr.Close()

	destDir, err := os.MkdirTemp("", "extract-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	for _, f := range zr.File {
		destPath := filepath.Join(destDir, f.Name)

		// Zip Slip protection => https://security.snyk.io/research/zip-slip-vulnerability
		rel, err := filepath.Rel(destDir, destPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("invalid file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return "", err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return "", err
		}

		dstFile, err := os.Create(destPath)
		if err != nil {
			return "", err
		}

		srcFile, err := f.Open()
		if err != nil {
			dstFile.Close()
			return "", err
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return "", err
		}

		srcFile.Close()
		dstFile.Close()
	}

	logger.Debug("extractZip done", "destDir", destDir)
	return destDir, nil
}
