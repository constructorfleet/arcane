package services

import (
	"context"
	stdjson "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/libtnb/sqlite"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"getarcane.dev/arcane/v3/internal/config"
	"getarcane.dev/arcane/v3/internal/database"
	"getarcane.dev/arcane/v3/internal/docker"
	"getarcane.dev/arcane/v3/internal/settings"
	"getarcane.dev/arcane/v3/pkg/httpx"
	"getarcane.dev/arcane/v3/pkg/pagination"
	"getarcane.dev/types/v3"
)

func newSwarmTestDockerClient(t *testing.T, server *httptest.Server) *client.Client {
	t.Helper()
	cli, err := client.New(
		client.WithHost(server.URL),
		client.WithAPIVersion("1.41"),
		client.WithHTTPClient(server.Client()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func TestForceUpdateServicePreservesCurrentSpecAndRegistryAuth(t *testing.T) {
	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{Name: "service-1", Labels: map[string]string{"owner": "fixture"}},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: &swarm.ContainerSpec{Image: "registry.example.test/example:stable"},
			ForceUpdate:   3,
		},
		Mode:         swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: new(uint64(3))}},
		UpdateConfig: &swarm.UpdateConfig{Parallelism: 1, Order: swarm.UpdateOrderStopFirst},
	}
	golden := spec
	golden.TaskTemplate.ForceUpdate = 4 // keep: models the single expected change
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1.41/services/service-1":
			assert.NoError(t, stdjson.NewEncoder(w).Encode(swarm.Service{
				ID: "service-1", Meta: swarm.Meta{Version: swarm.Version{Index: 7}}, Spec: spec,
			}))
		case r.Method == http.MethodPost && r.URL.Path == "/v1.41/services/service-1/update":
			assert.Equal(t, "7", r.URL.Query().Get("version"))
			assert.Equal(t, "spec", r.URL.Query().Get("registryAuthFrom"))
			assert.Empty(t, r.URL.Query().Get("rollback"))
			var request swarm.ServiceSpec
			if assert.NoError(t, stdjson.NewDecoder(r.Body).Decode(&request)) {
				assert.Equal(t, golden, request)
			}
			_, _ = w.Write([]byte(`{"Warnings":["accepted"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	service := NewService(newSettingsDockerService(t, newSwarmTestDockerClient(t, server)), func(context.Context) error { return nil }, nil)

	result, err := service.ForceUpdateService(t.Context(), "service-1")
	require.NoError(t, err)
	require.Equal(t, []string{"accepted"}, result.Warnings)
}

func TestService_ServiceModes(t *testing.T) {
	ctx := t.Context()
	replicas := uint64(5)
	maxConcurrent := uint64(2)

	tests := []struct {
		assertMode func(*testing.T, swarm.ServiceMode)
		name       string
		modeJSON   string
		wantErr    bool
	}{
		{
			name:     "replicated",
			modeJSON: `{"Replicated":{}}`,
			assertMode: func(t *testing.T, mode swarm.ServiceMode) {
				t.Helper()
				require.NotNil(t, mode.Replicated)
				require.NotNil(t, mode.Replicated.Replicas)
				require.Equal(t, replicas, *mode.Replicated.Replicas)
				require.Nil(t, mode.ReplicatedJob)
			},
		},
		{
			name:     "replicated job",
			modeJSON: `{"ReplicatedJob":{"MaxConcurrent":2,"TotalCompletions":0}}`,
			assertMode: func(t *testing.T, mode swarm.ServiceMode) {
				t.Helper()
				require.Nil(t, mode.Replicated)
				require.NotNil(t, mode.ReplicatedJob)
				require.NotNil(t, mode.ReplicatedJob.TotalCompletions)
				require.Equal(t, replicas, *mode.ReplicatedJob.TotalCompletions)
				require.NotNil(t, mode.ReplicatedJob.MaxConcurrent)
				require.Equal(t, maxConcurrent, *mode.ReplicatedJob.MaxConcurrent)
			},
		},
		{
			name:     "zero replicas",
			modeJSON: `{"Replicated":{"Replicas":0}}`,
			assertMode: func(t *testing.T, mode swarm.ServiceMode) {
				t.Helper()
				require.NotNil(t, mode.Replicated)
				require.NotNil(t, mode.Replicated.Replicas)
				require.Equal(t, replicas, *mode.Replicated.Replicas)
			},
		},
		{
			name:     "global",
			modeJSON: `{"Global":{}}`,
			wantErr:  true,
		},
		{
			name:     "global job",
			modeJSON: `{"GlobalJob":{}}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalls := 0
			var updatedSpec swarm.ServiceSpec

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/v1.41/info":
					if !assert.NoError(t, json.MarshalWrite(w, system.Info{
						Swarm: swarm.Info{
							LocalNodeState:   swarm.LocalNodeStateActive,
							ControlAvailable: true,
						},
					})) {
						return
					}
				case r.Method == http.MethodGet && r.URL.Path == "/v1.41/services/service-1":
					_, _ = fmt.Fprintf(w, `{"ID":"service-1","Version":{"Index":7},"Spec":{"Name":"service-1","Mode":%s}}`, tt.modeJSON)
				case r.Method == http.MethodGet && (r.URL.Path == "/v1.41/nodes" || r.URL.Path == "/v1.41/tasks"):
					_, _ = w.Write([]byte(`[]`))
				case r.Method == http.MethodPost && r.URL.Path == "/v1.41/services/service-1/update":
					updateCalls++
					if !assert.Equal(t, "7", r.URL.Query().Get("version")) {
						return
					}
					if !assert.NoError(t, json.UnmarshalRead(r.Body, &updatedSpec)) {
						return
					}
					if !assert.NoError(t, json.MarshalWrite(w, map[string]any{"Warnings": []string{"updated"}})) {
						return
					}
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)

			dockerService := newSettingsDockerService(t, newSwarmTestDockerClient(t, server))
			svc := NewService(dockerService, func(context.Context) error { return nil }, nil)

			inspected, err := svc.GetService(ctx, "service-1")
			require.NoError(t, err)
			encoded, err := json.Marshal(inspected)
			require.NoError(t, err)
			var wire struct {
				Spec struct {
					Mode jsontext.Value
				} `json:"spec"`
			}
			require.NoError(t, json.Unmarshal(encoded, &wire))
			require.JSONEq(t, tt.modeJSON, string(wire.Spec.Mode))

			resp, err := svc.ScaleService(ctx, "service-1", replicas)
			if tt.wantErr {
				require.Error(t, err)
				require.True(t, errdefs.IsInvalidArgument(err), "expected invalid argument, got %v", err)
				require.Equal(t, 0, updateCalls)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, []string{"updated"}, resp.Warnings)
			require.Equal(t, 1, updateCalls)
			tt.assertMode(t, updatedSpec.Mode)
		})
	}
}

func servicesScopeContext(t *testing.T, target string, bodies map[string]any) context.Context {
	t.Helper()
	return httpx.WithEnvironmentScope(t.Context(), &httpx.EnvironmentScope{
		IDs:     []string{"a", "b", "c"},
		Request: httptest.NewRequest(http.MethodGet, target, http.NoBody),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, ok := bodies[strings.Split(strings.TrimPrefix(r.URL.Path, "/api/agents/"), "/")[0]]
			if !ok {
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(`{"detail":"environment offline"}`))
				return
			}
			_ = json.MarshalWrite(w, body)
		}),
		Resolve: func(_ context.Context, id string) (types.EnvironmentRef, error) {
			return types.EnvironmentRef{ID: id, Name: "Env " + id}, nil
		},
	})
}

func TestListServicesFiltersByNetworkBeforePagination(t *testing.T) {
	services := []swarm.Service{
		{ID: "unrelated", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "aaa-unrelated"}}},
		{ID: "dnsrr", Spec: swarm.ServiceSpec{
			Annotations:  swarm.Annotations{Name: "dnsrr"},
			TaskTemplate: swarm.TaskSpec{Networks: []swarm.NetworkAttachmentConfig{{Target: "overlay"}}},
		}},
		{
			ID: "vip", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "vip"}},
			Endpoint: swarm.Endpoint{VirtualIPs: []swarm.EndpointVirtualIP{{NetworkID: "overlay"}}},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1.41/services":
			assert.NoError(t, stdjson.NewEncoder(w).Encode(services))
		case "/v1.41/tasks", "/v1.41/nodes", "/v1.41/networks":
			_, _ = w.Write([]byte("[]"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	service := NewService(newSettingsDockerService(t, newSwarmTestDockerClient(t, server)), func(context.Context) error { return nil }, nil)
	params := pagination.QueryParams{
		Params:     pagination.Params{Start: 1, Limit: 1},
		Filters:    map[string]string{"networkId": "overlay"},
		SortParams: pagination.SortParams{Sort: "name", Order: pagination.SortAsc},
	}

	items, page, err := service.ListServicesPaginated(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "vip", items[0].ID)
	require.EqualValues(t, 2, page.TotalItems)
	require.Equal(t, "overlay", params.Filters["networkId"])
}

func TestAggregateMergesEnvironments(t *testing.T) {
	ctx := servicesScopeContext(t, "/api/agents/a,b,c/swarm/services", map[string]any{
		"a": types.Paginated[types.SwarmServiceSummary]{
			Success: true,
			Data:    []types.SwarmServiceSummary{{ID: "s1", Name: "web"}},
		},
		"b": types.Paginated[types.SwarmServiceSummary]{
			Success: true,
			Data:    []types.SwarmServiceSummary{{ID: "s2", Name: "api"}},
		},
	})

	body, err := (&Service{}).Aggregate(ctx, pagination.QueryParams{Params: pagination.Params{Limit: 20}})
	require.NoError(t, err)

	require.Len(t, body.Data, 2)
	require.ElementsMatch(t, []string{"Env a", "Env b"}, []string{body.Data[0].Environment().Name, body.Data[1].Environment().Name})
	require.EqualValues(t, 2, body.Pagination.TotalItems)
	require.Len(t, body.FailedEnvironments, 1)
	require.Equal(t, "c", body.FailedEnvironments[0].Environment.ID)
}

func newSettingsDockerService(t *testing.T, dockerClient *client.Client) *docker.DockerClientService {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, gormDB.AutoMigrate(&settings.SettingVariable{}))
	db := &database.DB{DB: gormDB}
	settingsService, err := settings.NewSettingsService(settings.SettingsServiceParams{
		Ctx:    t.Context(),
		DB:     db,
		Config: &config.Config{},
		Env:    settings.Env{},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, settingsService.Stop(context.WithoutCancel(t.Context()))) })
	return docker.NewDockerClientService(t.Context(), db, settingsService, docker.ClientOptions{}).WithClient(dockerClient)
}

func TestRunningImageVersions(t *testing.T) {
	const first = "repo/app:old@sha256:aaaa"
	const second = "repo/app:new@sha256:bbbb"
	services := []swarm.Service{
		{ID: "mixed", Spec: swarm.ServiceSpec{Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{}}}},
		{ID: "same", Spec: swarm.ServiceSpec{Mode: swarm.ServiceMode{Replicated: &swarm.ReplicatedService{}}}},
		{ID: "global", Spec: swarm.ServiceSpec{Mode: swarm.ServiceMode{Global: &swarm.GlobalService{}}}},
	}
	tasks := []swarm.Task{
		{ServiceID: "mixed", DesiredState: swarm.TaskStateShutdown, Status: swarm.TaskStatus{State: swarm.TaskStateRunning}, Spec: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: first}}},
		{ServiceID: "mixed", Status: swarm.TaskStatus{State: swarm.TaskStateRunning}, Spec: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: second}}},
		{ServiceID: "same", Status: swarm.TaskStatus{State: swarm.TaskStateRunning}, Spec: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: first}}},
		{ServiceID: "same", Status: swarm.TaskStatus{State: swarm.TaskStateRunning}, Spec: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: "repo/app:alias@sha256:aaaa"}}},
		{ServiceID: "same", Status: swarm.TaskStatus{State: swarm.TaskStateShutdown}, Spec: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: second}}},
		{ServiceID: "global", Status: swarm.TaskStatus{State: swarm.TaskStateRunning}, Spec: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: first}}},
		{ServiceID: "global", Status: swarm.TaskStatus{State: swarm.TaskStateRunning}, Spec: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: second}}},
	}
	require.Equal(t, map[string][]string{"mixed": {second, first}}, RunningImageVersions(services, tasks))
	summaries := []types.SwarmServiceSummary{{ID: "mixed", RunningImageVersions: []string{first, second}}, {ID: "same"}}
	page := servicePagination.SearchOrderAndPaginate(summaries, pagination.QueryParams{Filters: map[string]string{"mixedImageVersions": "true"}})
	require.Len(t, page.Items, 1)
	require.Equal(t, "mixed", page.Items[0].ID)
}
