package volume

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	acfstypes "kit.getarcane.app/acfs/types"

	"getarcane.dev/types/v3"
)

func TestVolumeACFSCompanionInstallation(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "acfs")
	missing, err := loadVolumeCompanion(filename)
	require.NoError(t, err)
	require.Nil(t, missing)
	content := []byte("packaged executable fixture")
	require.NoError(t, os.WriteFile(filename, content, 0o555))
	binary, err := loadVolumeCompanion(filename)
	require.NoError(t, err)
	require.True(t, path.IsAbs(binary.path))
	require.Equal(t, "/tmp", path.Dir(path.Dir(binary.path)))

	reader := tar.NewReader(bytes.NewReader(binary.archive))
	directory, err := reader.Next()
	require.NoError(t, err)
	require.Equal(t, byte(tar.TypeDir), directory.Typeflag)
	require.EqualValues(t, 0o755, directory.Mode)
	payload, err := reader.Next()
	require.NoError(t, err)
	require.Equal(t, byte(tar.TypeReg), payload.Typeflag)
	require.EqualValues(t, 0o555, payload.Mode)
	require.Zero(t, payload.Uid)
	require.Zero(t, payload.Gid)
	copied, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, content, copied)
	published, err := reader.Next()
	require.NoError(t, err)
	require.Equal(t, byte(tar.TypeLink), published.Typeflag)
	require.Equal(t, payload.Name, published.Linkname)
	require.Equal(t, binary.path, path.Join("/tmp", published.Name))
	_, err = reader.Next()
	require.ErrorIs(t, err, io.EOF)

	var installed bool
	var copies int
	mode := os.FileMode(0o555)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1.41/containers/helper/archive", r.URL.Path)
		switch r.Method {
		case http.MethodHead:
			assert.Equal(t, binary.path, r.URL.Query().Get("path"))
			if !installed {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			stat, marshalErr := json.Marshal(container.PathStat{Name: "acfs", Size: binary.size, Mode: mode})
			assert.NoError(t, marshalErr)
			w.Header().Set("X-Docker-Container-Path-Stat", base64.StdEncoding.EncodeToString(stat))
		case http.MethodPut:
			assert.Equal(t, "/tmp", r.URL.Query().Get("path"))
			assert.Equal(t, "true", r.URL.Query().Get("copyUIDGID"))
			archive, readErr := io.ReadAll(r.Body)
			assert.NoError(t, readErr)
			assert.Equal(t, binary.archive, archive)
			copies++
			installed = true
		default:
			t.Errorf("unexpected Docker request %s", r.Method)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)
	service := &VolumeService{}
	dockerClient := newVolumeServiceTestDockerClient(t, server)
	for range 2 {
		executable, installErr := service.installVolumeCompanion(t.Context(), dockerClient, "helper", binary)
		require.NoError(t, installErr)
		require.Equal(t, binary.path, executable)
	}
	require.Equal(t, 1, copies)
	mode = os.ModeSymlink | 0o555
	_, err = service.installVolumeCompanion(t.Context(), dockerClient, "helper", binary)
	require.ErrorContains(t, err, "unexpected metadata")
	require.Equal(t, 1, copies)

	service.helperByVolume = map[string]*volumeHelper{"fixture": {id: "helper", protocol: acfstypes.ProtocolVersion, listLimit: true}}
	_, err = service.requireVolumeHelperACFS(t.Context(), "fixture", "helper", types.VolumeHelperRequirements{PreserveOwnership: true})
	require.ErrorContains(t, err, "ownership-preserving")
	limited, err := service.requireVolumeHelperACFS(t.Context(), "fixture", "helper", types.VolumeHelperRequirements{})
	require.NoError(t, err)
	require.True(t, limited)
	service.helperByVolume["fixture"].preserveOwner = true
	limited, err = service.requireVolumeHelperACFS(t.Context(), "fixture", "helper", types.VolumeHelperRequirements{PreserveOwnership: true})
	require.NoError(t, err)
	require.True(t, limited)
}
