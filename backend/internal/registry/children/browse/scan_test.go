package browse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/types/v3"
)

func TestScanReportUsesSavedCredentialAndPlatformManifest(t *testing.T) {
	index := digest.FromString("index").String()
	child := digest.FromString("linux/amd64").String()
	other := digest.FromString("linux/arm64").String()
	path := "/api/v2.0/projects/project/repositories/team%252Fapp/artifacts/"
	scanRequests := 0
	scanStatus := http.StatusAccepted
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		user, token, ok := r.BasicAuth()
		if !ok || user != "test-robot" || token != "fixture-token" || r.Header.Get("X-Accept-Vulnerabilities") != harborVulnerabilityMediaType {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.EscapedPath() {
		case path + index:
			_, _ = fmt.Fprintf(w, `{"digest":%q,"references":[
 {"child_digest":%q,"platform":{"os":"linux","architecture":"amd64"}},
 {"child_digest":%q,"platform":{"os":"linux","architecture":"arm64"}}]}`, index, child, other)
		case path + child:
			_, _ = fmt.Fprintf(w, `{"digest":%q,"extra_attrs":{"os":"linux","architecture":"amd64"},
 "scan_overview":{%q:{"scan_status":"Success","severity":"Medium","summary":{"total":1,"summary":{"Medium":1}}}}}`, child, harborVulnerabilityMediaType)
		case path + child + "/scan":
			assert.Equal(t, http.MethodPost, r.Method)
			scanRequests++
			w.WriteHeader(scanStatus)
		case path + child + "/additions/vulnerabilities":
			_, _ = fmt.Fprintf(w, `{%q:{"vulnerabilities":[{"id":"CVE-fixture","package":"test","version":"1","fix_version":"2","severity":"Medium"}]}}`, harborVulnerabilityMediaType)
		case path + other:
			_, _ = fmt.Fprintf(w, `{"digest":%q,"extra_attrs":{"os":"linux","architecture":"arm64"}}`, other)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	profiles := testRegistries{}
	id := profiles.add(server.URL+"/project", "test-robot", "fixture-token")
	service := newTestService(profiles)
	service.transport = server.Client().Transport

	report, err := service.ScanArtifact(t.Context(), types.RegistryArtifactScanOptions{RegistryID: id, Repository: "project/team/app", Digest: index, Platform: "linux/amd64"})
	require.NoError(t, err)
	require.Equal(t, child, report.Digest)
	require.Equal(t, "Success", report.Status)
	require.Equal(t, 1, report.Total)
	require.Len(t, report.Findings, 1)
	require.Equal(t, "2", report.Findings[0].FixedVersion)

	report, err = service.ScanArtifact(t.Context(), types.RegistryArtifactScanOptions{RegistryID: id, Repository: "project/team/app", Digest: index, Platform: "linux/arm64"})
	require.NoError(t, err)
	require.Equal(t, "Not scanned", report.Status)
	require.Empty(t, report.Findings)

	report, err = service.ScanArtifact(t.Context(), types.RegistryArtifactScanOptions{RegistryID: id, Repository: "project/team/app", Digest: other, Platform: "linux/amd64"})
	require.NoError(t, err)
	require.Equal(t, "Platform unavailable", report.Status)
	require.Zero(t, scanRequests)
	options := types.RegistryArtifactScanOptions{RegistryID: id, Repository: "project/team/app", Digest: index, Platform: "linux/amd64", Trigger: true}
	report, err = service.ScanArtifact(t.Context(), options)
	require.NoError(t, err)
	require.Equal(t, child, report.Digest)
	require.Equal(t, "Running", report.Status)
	require.Equal(t, 1, scanRequests)
	scanStatus = http.StatusForbidden
	_, err = service.ScanArtifact(t.Context(), options)
	require.ErrorIs(t, err, common.ErrForbidden)
	scanStatus = http.StatusPreconditionFailed
	report, err = service.ScanArtifact(t.Context(), options)
	require.NoError(t, err)
	require.Equal(t, "Unsupported", report.Status)
}
