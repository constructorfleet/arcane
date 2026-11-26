package browse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/types/v3"
)

func TestProxyScanResolvesUpstreamThroughHarbor(t *testing.T) {
	image := empty.Image
	digest, err := image.Digest()
	require.NoError(t, err)
	var pulls atomic.Int32
	var scans atomic.Int32
	var denied atomic.Bool
	var indexed atomic.Bool
	distribution := registry.New()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v2/") {
			if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/manifests/") {
				pulls.Add(1)
			}
			distribution.ServeHTTP(w, r)
			return
		}
		user, token, ok := r.BasicAuth()
		if !ok || user != "robot" || token != "fixture" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if denied.Load() {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2.0/systeminfo":
			_, _ = w.Write([]byte(`{"auth_mode":"oidc_auth"}`))
		case "/api/v2.0/projects":
			_, _ = w.Write([]byte(`[{"name":"dockerhub","registry_id":7}]`))
		case "/api/v2.0/projects/dockerhub/summary":
			_, _ = w.Write([]byte(`{"registry":{"url":"https://hub.docker.com"}}`))
		case "/api/v2.0/projects/dockerhub/repositories/library%2Ffixture/artifacts/" + digest.String():
			if !indexed.Load() {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = fmt.Fprintf(w, `{"digest":%q,"extra_attrs":{"os":"linux","architecture":"amd64"}}`, digest.String())
		case "/api/v2.0/projects/dockerhub/repositories/library%2Ffixture/artifacts/" + digest.String() + "/scan":
			assert.Equal(t, http.MethodPost, r.Method)
			scans.Add(1)
			w.WriteHeader(http.StatusAccepted)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	ref, err := name.NewTag(strings.TrimPrefix(server.URL, "https://") + "/dockerhub/library/fixture:live")
	require.NoError(t, err)
	index := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{Add: image, Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: "amd64"}}})
	indexDigest, err := index.Digest()
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(ref, index, remote.WithTransport(server.Client().Transport)))
	profiles := testRegistries{}
	id := profiles.add(server.URL, "robot", "fixture")
	service := newTestService(profiles)
	service.transport = server.Client().Transport
	options := types.ImageRegistryScanOptions{References: []string{"docker.io/library/fixture@" + indexDigest.String()}, Platform: "linux/amd64"}
	report, err := service.ProxyScan(t.Context(), id, options)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, digest.String(), report.Digest)
	require.Positive(t, pulls.Load())
	require.Zero(t, scans.Load())
	require.Equal(t, "Not scanned", report.Status)
	indexed.Store(true)
	report, err = service.ProxyScan(t.Context(), id, types.ImageRegistryScanOptions{References: options.References, Platform: options.Platform, Trigger: true})
	require.NoError(t, err)
	require.Equal(t, "Running", report.Status)
	require.EqualValues(t, 1, scans.Load())
	report, err = service.ProxyScan(t.Context(), id, types.ImageRegistryScanOptions{References: []string{"ghcr.io/library/fixture@" + digest.String()}, Platform: options.Platform})
	require.NoError(t, err)
	require.Nil(t, report)
	denied.Store(true)
	deniedService := newTestService(profiles)
	deniedService.transport = server.Client().Transport
	_, err = deniedService.ProxyScan(t.Context(), id, options)
	require.ErrorIs(t, err, common.ErrForbidden)
}
