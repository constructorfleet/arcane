package version

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchLatestRelease(t *testing.T) {
	tests := []struct {
		name, payload, want, wantErr string
		status                       int
	}{
		{name: "numeric fork prereleases", payload: `[{"tag_name":"v3.0.1-cf.9"},{"tag_name":"v3.0.1-cf.10","body":"fork notes","published_at":"2026-11-19T09:00:00Z"}]`, want: "v3.0.1-cf.10", status: 200},
		{
			name:    "drafts and invalid tags excluded",
			payload: `[{"tag_name":"v9.0.0","draft":true},{"tag_name":"latest"},{"tag_name":"v8.0.0-invalid!"},{"tag_name":"v3.0.1-cf.8"}]`, want: "v3.0.1-cf.8", status: 200,
		},
		{name: "no published releases", payload: `[]`, wantErr: "no published semantic versions", status: 200},
		{name: "invalid payload", payload: `<html>unavailable</html>`, wantErr: "decode releases", status: 200},
		{name: "API failure", payload: `{}`, wantErr: "status 503", status: 503},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewVersionService(VersionServiceParams{HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "gitea.home.prettybaked.com", req.URL.Host)
				require.Equal(t, "/api/v1/repos/constructorfleet/arcane/releases", req.URL.Path)
				return &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.payload)), Header: make(http.Header), Request: req}, nil
			})}})
			release, err := svc.fetchLatestRelease(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, release.TagName)
			if tt.name == "numeric fork prereleases" {
				require.Equal(t, "fork notes", release.Body)
				require.Equal(t, "2026-11-19T09:00:00Z", release.PublishedAt)
			}
			require.Equal(t, "https://gitea.home.prettybaked.com/constructorfleet/arcane/releases/tag/"+tt.want, svc.ReleaseURL(release.TagName))
		})
	}
}

func TestFetchLatestReleasePages(t *testing.T) {
	calls := 0
	svc := NewVersionService(VersionServiceParams{HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, strconv.Itoa(calls), req.URL.Query().Get("page"))
		require.Equal(t, "50", req.URL.Query().Get("limit"))
		entries := make([]map[string]string, 50)
		for i := range entries {
			entries[i] = map[string]string{"tag_name": "v3.0.1-cf.9"}
		}
		if calls == 2 {
			entries = []map[string]string{{"tag_name": "v3.0.1-cf.10"}}
		}
		require.LessOrEqual(t, calls, 2)
		payload, err := json.Marshal(entries)
		require.NoError(t, err)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(payload))), Header: make(http.Header), Request: req}, nil
	})}})
	release, err := svc.fetchLatestRelease(t.Context())
	require.NoError(t, err)
	require.Equal(t, "v3.0.1-cf.10", release.TagName)
	require.Equal(t, 2, calls)
}

func TestFetchLatestReleaseCancellation(t *testing.T) {
	svc := NewVersionService(VersionServiceParams{HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, req.Context().Err()
	})}})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := svc.fetchLatestRelease(ctx)
	require.ErrorIs(t, err, context.Canceled)
}
