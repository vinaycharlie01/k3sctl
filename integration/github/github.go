package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	k3sReleasesURL    = "https://api.github.com/repos/k3s-io/k3s/releases"
	k3sctlReleasesURL = "https://api.github.com/repos/vinaycharlie01/k3sctl/releases"
)

type release struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func LatestK3sVersion(ctx context.Context, allowPre bool) (string, error) {
	releases, err := fetchReleases(ctx, k3sReleasesURL)
	if err != nil {
		return "", fmt.Errorf("fetch k3s releases: %w", err)
	}

	for _, r := range releases {
		if r.Draft {
			continue
		}
		if r.Prerelease && !allowPre {
			continue
		}
		if r.TagName == "" {
			continue
		}
		return r.TagName, nil
	}

	return "", fmt.Errorf("no suitable k3s release found")
}

func LatestK3sBinaryURL(ctx context.Context, version, arch string) (string, error) {
	binaryName := "k3s"
	switch arch {
	case "arm64":
		binaryName = "k3s-arm64"
	case "arm":
		binaryName = "k3s-armhf"
	}

	return fmt.Sprintf("https://github.com/k3s-io/k3s/releases/download/%s/%s", version, binaryName), nil
}

func LatestK3sctlVersion(ctx context.Context) (string, error) {
	releases, err := fetchReleases(ctx, k3sctlReleasesURL)
	if err != nil {
		log.Debugf("failed to fetch k3sctl releases: %v", err)
		return "", nil
	}

	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		return r.TagName, nil
	}
	return "", nil
}

func IsValidK3sVersion(ctx context.Context, version string) (bool, error) {
	if !strings.HasPrefix(version, "v") {
		return false, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/k3s-io/k3s/releases/tags/%s", version)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK, nil
}

func fetchReleases(ctx context.Context, url string) ([]release, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url+"?per_page=20", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return releases, nil
}
