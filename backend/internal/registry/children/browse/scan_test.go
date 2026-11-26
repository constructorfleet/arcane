package browse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func TestScanReportUsesSavedCredentialAndPlatformManifest(t *testing.T) {
	index := digest.FromString("index").String()
	child := digest.FromString("linux/amd64").String()
	other := digest.FromString("linux/arm64").String()
	path := "/api/v2.0/projects/project/repositories/team%252Fapp/artifacts/"
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

	report, err := service.ScanReport(t.Context(), id, "project/team/app", index, "linux/amd64")
	require.NoError(t, err)
	require.Equal(t, child, report.Digest)
	require.Equal(t, "Success", report.Status)
	require.Equal(t, 1, report.Total)
	require.Len(t, report.Findings, 1)
	require.Equal(t, "2", report.Findings[0].FixedVersion)

	report, err = service.ScanReport(t.Context(), id, "project/team/app", index, "linux/arm64")
	require.NoError(t, err)
	require.Equal(t, "Not scanned", report.Status)
	require.Empty(t, report.Findings)

	report, err = service.ScanReport(t.Context(), id, "project/team/app", other, "linux/amd64")
	require.NoError(t, err)
	require.Equal(t, "Platform unavailable", report.Status)
}
