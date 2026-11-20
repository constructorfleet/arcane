package swarm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"kit.getarcane.app/concurrency"

	"getarcane.dev/types/v3"
)

type cleanupDockerAPI struct {
	failure string
	calls   []string
	removed []string
	mu      sync.Mutex
}

func (api *cleanupDockerAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()
	path := strings.TrimPrefix(req.URL.Path, "/v1.56")
	api.calls = append(api.calls, req.Method+" "+path)
	w.Header().Set("Content-Type", "application/json")
	if path == api.failure {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"message":"synthetic reference inventory unavailable"}`)
		return
	}
	if path == "/services" || path == "/tasks" {
		filter := req.URL.Query().Get("filters")
		if filter != "" && filter != "{}" && filter != "null" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"message":"reference inventory must be unfiltered"}`)
			return
		}
	}
	if req.Method == http.MethodDelete {
		if path == "/configs/current-config" || path == "/secrets/current-secret" {
			w.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(w, `{"message":"resource is in use by current service"}`)
			return
		}
		api.removed = append(api.removed, path)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var body string
	switch path {
	case "/services":
		body = `[
{"ID":"external-service",
 "Spec":{"Name":"another-stack_service","TaskTemplate":{"ContainerSpec":{
  "Configs":[{"ConfigID":"current-config"}],"Secrets":[{"SecretID":"current-secret"}]
 }}},
 "PreviousSpec":{"TaskTemplate":{"ContainerSpec":{
  "Configs":[{"ConfigName":"previous-config"}],"Secrets":[{"SecretName":"previous-secret"}],
  "Privileges":{"CredentialSpec":{"Config":"runtime-config"}}
 }}}},
{"ID":"non-container-service","Spec":{"Name":"non-container"}}]`
	case "/tasks":
		body = `[
{"ID":"completed-task","ServiceID":"external-service","Status":{"State":"complete"},"Spec":{"ContainerSpec":{
 "Configs":[{"ConfigID":"task-config"}],"Secrets":[{"SecretID":"task-secret"}]
}}},
{"ID":"non-container-task","Spec":{}}]`
	case "/configs":
		body = `[
{"ID":"desired-config-id","Spec":{"Name":"desired-config"}},
{"ID":"current-config","Spec":{"Name":"current-config-name"}},
{"ID":"previous-config-id","Spec":{"Name":"previous-config"}},
{"ID":"runtime-config","Spec":{"Name":"runtime-config-name"}},
{"ID":"task-config","Spec":{"Name":"task-config-name"}},
{"ID":"stale-config","Spec":{"Name":"stale-config-name"}}]`
	case "/secrets":
		body = `[
{"ID":"desired-secret-id","Spec":{"Name":"desired-secret"}},
{"ID":"current-secret","Spec":{"Name":"current-secret-name"}},
{"ID":"previous-secret-id","Spec":{"Name":"previous-secret"}},
{"ID":"task-secret","Spec":{"Name":"task-secret-name"}},
{"ID":"stale-secret","Spec":{"Name":"stale-secret-name"}}]`
	default:
		w.WriteHeader(http.StatusNotFound)
		body = `{"message":"unexpected synthetic Docker request"}`
	}
	_, _ = io.WriteString(w, body)
}

func TestCleanupStaleManagedResourcesMatchesCurrentAndLegacyLabels(t *testing.T) {
	t.Parallel()

	listCalls := 0
	removedIDs := make([]string, 0, 2)
	adapter := fileResourceAdapter{
		ResourceType: "config",
		List: func(_ context.Context, filters dockerclient.Filters) ([]staleManagedResource, error) {
			listCalls++
			labels := filters["label"]
			require.True(t, labels[types.SwarmStackNamespaceLabel+"=stack"])

			switch {
			case labels[resourceTypeLabel+"=config"]:
				return []staleManagedResource{
					{ID: "current", Name: "desired"},
					{ID: "both", Name: "stale-both"},
				}, nil
			case labels[legacyResourceTypeLabel+"=config"]:
				return []staleManagedResource{
					{ID: "legacy", Name: "stale-legacy"},
					{ID: "both", Name: "stale-both"},
				}, nil
			default:
				require.FailNow(t, "unexpected managed-resource label filter")
				return nil, nil
			}
		},
		Remove: func(_ context.Context, id string) error {
			removedIDs = append(removedIDs, id)
			return nil
		},
	}

	err := cleanupStaleManagedResources(
		t.Context(),
		"stack",
		map[string]struct{}{"desired": {}},
		adapter,
		true,
		true,
	)
	require.NoError(t, err)
	require.Equal(t, 2, listCalls)
	require.ElementsMatch(t, []string{"both", "legacy"}, removedIDs)

	for _, tc := range []struct {
		name    string
		failure string
	}{
		{name: "keep desired current rollback and historical task dependencies"},
		{name: "defer all cleanup when service inventory fails", failure: "/services"},
		{name: "defer all cleanup when task inventory fails", failure: "/tasks"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &cleanupDockerAPI{failure: tc.failure}
			server := httptest.NewServer(api)
			t.Cleanup(server.Close)
			dockerClient, clientErr := dockerclient.New(
				dockerclient.WithHost(server.URL),
				dockerclient.WithAPIVersion("1.56"),
				dockerclient.WithHTTPClient(server.Client()),
			)
			require.NoError(t, clientErr)
			t.Cleanup(func() { _ = dockerClient.Close() })
			cleanupErr := cleanupStackResources(t.Context(), dockerClient, "stack",
				map[string]resourceMeta{"config": {Name: "desired-config"}},
				map[string]resourceMeta{"secret": {Name: "desired-secret"}},
			)
			require.NoError(t, cleanupErr)
			api.mu.Lock()
			defer api.mu.Unlock()
			if tc.failure != "" {
				require.Empty(t, api.removed)
				require.Contains(t, api.calls, "GET "+tc.failure)
				require.NotContains(t, api.calls, "GET /configs")
				require.NotContains(t, api.calls, "GET /secrets")
				return
			}
			require.ElementsMatch(t, []string{"/configs/stale-config", "/secrets/stale-secret"}, api.removed)
			require.NotContains(t, api.calls, "DELETE /configs/current-config")
			require.NotContains(t, api.calls, "DELETE /secrets/current-secret")
			require.Equal(t, []string{"GET /services", "GET /tasks"}, api.calls[:2])
		})
	}
}

func TestCleanupStaleManagedResourcesFailsWhenResourceStaysInUse(t *testing.T) {
	previousBackoff := staleSwarmResourceRemoveBackoff
	staleSwarmResourceRemoveBackoff = concurrency.Backoff{Base: time.Millisecond}
	t.Cleanup(func() { staleSwarmResourceRemoveBackoff = previousBackoff })

	removeCalls := 0
	adapter := fileResourceAdapter{
		ResourceType: "config",
		List: func(_ context.Context, _ dockerclient.Filters) ([]staleManagedResource, error) {
			return []staleManagedResource{{ID: "stale", Name: "stale"}}, nil
		},
		Remove: func(_ context.Context, _ string) error {
			removeCalls++
			return cerrdefs.ErrConflict.WithMessage("config is in use by task")
		},
	}

	err := cleanupStaleManagedResources(t.Context(), "stack", map[string]struct{}{}, adapter, false, false)
	require.Error(t, err)
	require.Equal(t, staleSwarmResourceRemoveAttempts, removeCalls)

	removeCalls = 0
	err = cleanupStaleManagedResources(t.Context(), "stack", map[string]struct{}{}, adapter, false, true)
	require.NoError(t, err)
	require.Equal(t, 1, removeCalls)
}
