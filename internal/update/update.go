package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
	"github.com/kamilludwinski/runtimzzz/internal/meta"
	"github.com/kamilludwinski/runtimzzz/internal/utils/versionutils"
)

const defaultRepo = "kamilludwinski/runtimzzz"
const apiTimeout = 10 * time.Second

// GitHub release API response (subset we need)
type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// LatestReleaseVersion fetches the latest release tag from GitHub and returns the version (e.g. "1.2.3").
func LatestReleaseVersion(repo string) (string, error) {
	if repo == "" {
		repo = defaultRepo
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	client := &http.Client{Timeout: apiTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("releases API returned %s", resp.Status)
	}
	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", fmt.Errorf("failed to decode release: %w", err)
	}
	version := strings.TrimPrefix(r.TagName, "v")
	return version, nil
}

// IsOutdated returns true if latest is newer than current.
func IsOutdated(current, latest string) (bool, error) {
	cmp := versionutils.CompareVersions(current, latest)
	return cmp < 0, nil
}

var (
	checkOnce   sync.Once
	newerResult string // non-empty if a newer version is available
)

// CheckForNewerVersion fetches the latest release once per process and returns the latest version string if it is newer than current; otherwise "".
func CheckForNewerVersion() string {
	checkOnce.Do(func() {
		current := meta.AppVersion
		repo := defaultRepo
		latest, err := LatestReleaseVersion(repo)
		if err != nil {
			logger.Debug("version check failed", "err", err)
			return
		}
		outdated, err := IsOutdated(current, latest)
		if err != nil || !outdated {
			return
		}
		newerResult = latest
	})
	return newerResult
}

// FetchRelease returns the latest release metadata (for asset selection).
func FetchRelease(repo string) (*release, error) {
	if repo == "" {
		repo = defaultRepo
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	client := &http.Client{Timeout: apiTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("releases API returned %s", resp.Status)
	}
	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}
	return &r, nil
}
