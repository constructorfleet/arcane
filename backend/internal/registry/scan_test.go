package registry

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/types/v3"
)

func TestScanImageSelectsSavedRegistryAndLocalFallback(t *testing.T) {
	artifact := digest.FromString("running").String()
	denied := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if denied {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2.0/projects/project/repositories/app/artifacts/" + artifact:
			_, _ = fmt.Fprintf(w, `{"digest":%q,"extra_attrs":{"os":"linux","architecture":"amd64"}}`, artifact)
		case "/api/v2.0/projects/project/repositories/app/artifacts/" + artifact + "/scan":
			w.WriteHeader(http.StatusPreconditionFailed)
		case "/api/v2.0/systeminfo":
			_, _ = w.Write([]byte(`{"harbor_version":"fixture"}`))
		case "/api/v2.0/projects":
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	db := setupContainerRegistryTestDB(t)
	createTestPullRegistry(t, db, server.URL, "robot", "fixture")
	service := newRegistryTestService(t, db, nil, server.Client())
	options := types.ImageRegistryScanOptions{
		References: []string{strings.TrimPrefix(server.URL, "http://") + "/project/app@" + artifact}, Platform: "linux/amd64",
	}
	report, err := service.ScanImage(t.Context(), options)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, "Not scanned", report.Status)
	report, err = service.ScanImage(t.Context(), types.ImageRegistryScanOptions{References: options.References, Platform: options.Platform, Trigger: true})
	require.NoError(t, err)
	require.Nil(t, report, "a Harbor project without a scanner uses local scanning")
	report, err = service.ScanImage(t.Context(), types.ImageRegistryScanOptions{
		References: []string{strings.TrimPrefix(server.URL, "http://") + "/project/app@" + digest.FromString("missing").String()},
		Platform:   options.Platform,
	})
	require.NoError(t, err)
	require.Nil(t, report, "an artifact unavailable in Harbor can use local scanning")
	report, err = service.ScanImage(t.Context(), types.ImageRegistryScanOptions{References: options.References, Platform: "linux/arm64"})
	require.NoError(t, err)
	require.Nil(t, report, "an unavailable platform tries the remaining Harbor paths before local fallback")
	denied = true
	_, err = service.ScanImage(t.Context(), options)
	require.ErrorIs(t, err, common.ErrForbidden)
	report, err = service.ScanImage(t.Context(), types.ImageRegistryScanOptions{})
	require.NoError(t, err)
	require.Nil(t, report, "a local build without a registry digest uses local scanning")
}
