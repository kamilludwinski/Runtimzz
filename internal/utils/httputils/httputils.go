package httputils

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
)

// DownloadFile downloads a file from the url to temp file and returns the path
// if client is nil, a default client is used
func DownloadFile(client *http.Client, rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("url is required")
	}

	logger.Debug("DownloadFile", "url", rawURL)

	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	resp, err := client.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	ext := extensionFromURL(rawURL)

	// fallback
	if ext == "" {
		ext = extensionFromContentType(resp.Header.Get("Content-Type"))
	}

	tmpFile, err := os.CreateTemp("", "download-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	logger.Debug("DownloadFile done", "path", tmpFile.Name())
	return tmpFile.Name(), nil
}

// knownMultiSuffixes are URL path suffixes that should be used as-is for the temp file
// so that archiveutils.Extract() recognizes them (e.g. .tar.gz, not just .gz).
var knownMultiSuffixes = []string{".tar.gz", ".tar.xz", ".tgz"}

func extensionFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	path := u.Path
	for _, suffix := range knownMultiSuffixes {
		if len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix {
			return suffix
		}
	}
	return filepath.Ext(path)
}

func extensionFromContentType(ct string) string {
	if ct == "" {
		return ""
	}

	exts, err := mime.ExtensionsByType(ct)
	if err != nil || len(exts) == 0 {
		return ""
	}

	return exts[0]
}
