package version

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/mod/semver"
)

// The releases list includes the fork's prereleases, which Gitea excludes from /latest.
func (s *VersionService) fetchLatestRelease(ctx context.Context) (latestRelease, error) {
	const pageSize = 50
	var latest latestRelease
	for page := 1; ; page++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			fmt.Sprintf("https://gitea.home.prettybaked.com/api/v1/repos/constructorfleet/arcane/releases?limit=%d&page=%d", pageSize, page), http.NoBody)
		if err != nil {
			return latestRelease{}, fmt.Errorf("create release request: %w", err)
		}
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return latestRelease{}, fmt.Errorf("get releases: %w", err)
		}
		var payload []struct {
			TagName     string `json:"tag_name"`
			Body        string `json:"body"`
			PublishedAt string `json:"published_at"`
			Draft       bool   `json:"draft"`
		}
		decodeErr := json.UnmarshalRead(io.LimitReader(resp.Body, 8<<20), &payload)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return latestRelease{}, fmt.Errorf("release API returned status %d", resp.StatusCode)
		}
		if decodeErr != nil {
			return latestRelease{}, fmt.Errorf("decode releases: %w", decodeErr)
		}
		for _, release := range payload {
			version := "v" + strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
			if release.Draft || release.TagName == "" || !semver.IsValid(version) {
				continue
			}
			if latest.TagName == "" || s.IsNewer(version, latest.TagName) {
				latest = latestRelease{TagName: release.TagName, Body: release.Body, PublishedAt: release.PublishedAt}
			}
		}
		if len(payload) < pageSize {
			break
		}
	}
	if latest.TagName == "" {
		return latestRelease{}, errors.New("release API returned no published semantic versions")
	}
	return latest, nil
}
