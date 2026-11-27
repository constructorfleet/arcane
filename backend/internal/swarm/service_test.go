package swarm

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moby/moby/api/types/swarm"
	"github.com/stretchr/testify/require"

	"getarcane.dev/arcane/v3/pkg/httpx"
	"getarcane.dev/arcane/v3/pkg/pagination"
	"getarcane.dev/types/v3"
)

func TestDecodeSwarmSpec_AllowsEmptyObject(t *testing.T) {
	spec, err := decodeSwarmSpec(jsontext.Value(`{}`))
	require.NoError(t, err)
	require.NotNil(t, spec.Labels)
	require.Empty(t, spec.Labels)
}

func TestDecodeSwarmSpec_RejectsNull(t *testing.T) {
	_, err := decodeSwarmSpec(jsontext.Value(`null`))
	require.EqualError(t, err, "swarm spec is required")
}

func swarmScopeContext(t *testing.T, target string, bodies map[string]any) context.Context {
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

func TestAggregateTasksMergesEnvironments(t *testing.T) {
	ctx := swarmScopeContext(t, "/api/agents/a,b,c/swarm/tasks", map[string]any{
		"a": types.Paginated[types.SwarmTaskSummary]{
			Success: true,
			Data:    []types.SwarmTaskSummary{{ID: "t1", Name: "web.1"}},
		},
		"b": types.Paginated[types.SwarmTaskSummary]{
			Success: true,
			Data:    []types.SwarmTaskSummary{{ID: "t2", Name: "api.1"}},
		},
	})

	body, err := (&SwarmService{}).AggregateTasks(ctx, pagination.QueryParams{Params: pagination.Params{Limit: 20}})
	require.NoError(t, err)

	require.Len(t, body.Data, 2)
	require.ElementsMatch(t, []string{"Env a", "Env b"}, []string{body.Data[0].Environment().Name, body.Data[1].Environment().Name})
	require.EqualValues(t, 2, body.Pagination.TotalItems)
	require.Len(t, body.FailedEnvironments, 1)
	require.Equal(t, "c", body.FailedEnvironments[0].Environment.ID)
}

func TestTaskPaginationUsesResolvedTaskNames(t *testing.T) {
	tasks := []types.SwarmTaskSummary{
		types.NewSwarmTaskSummary(swarm.Task{ID: "a", ServiceID: "replicated-id", Slot: 2}, "web", ""),
		types.NewSwarmTaskSummary(swarm.Task{ID: "b", ServiceID: "global-id", NodeID: "node-id"}, "exporter", "swarm03"),
		types.NewSwarmTaskSummary(swarm.Task{ID: "c", ServiceID: "missing-service", NodeID: "missing-node"}, "", ""),
		types.NewSwarmTaskSummary(swarm.Task{ID: "d", ServiceID: "pending-id"}, "pending", ""),
		types.NewSwarmTaskSummary(swarm.Task{ID: "e"}, "", ""),
		types.NewSwarmTaskSummary(swarm.Task{ID: "f", Annotations: swarm.Annotations{Name: "explicit"}, Slot: 3}, "web", ""),
	}
	require.Equal(t, "web.2", tasks[0].Name)
	require.Equal(t, "exporter.swarm03", tasks[1].Name)
	require.Equal(t, "missing-service.missing-node", tasks[2].Name)
	require.Equal(t, "pending", tasks[3].Name)
	require.Equal(t, "e", tasks[4].Name)
	require.Equal(t, "explicit", tasks[5].Name)

	page := taskPagination.SearchOrderAndPaginate(tasks, pagination.QueryParams{
		Params:      pagination.Params{Limit: 10},
		SearchQuery: pagination.SearchQuery{Search: "exporter.swarm03"},
	})
	require.Len(t, page.Items, 1)
	require.Equal(t, "b", page.Items[0].ID)
}

func TestServiceTaskPaginationFiltersBeforeSlicing(t *testing.T) {
	items := make([]types.SwarmTaskSummary, 0, 120)
	for i := range 120 {
		items = append(items, types.SwarmTaskSummary{
			ID: fmt.Sprintf("task-%03d", i), ServiceName: "node-exporter", NodeName: fmt.Sprintf("node-%03d", i),
			CurrentState: "running", DesiredState: "running",
		})
	}
	params := pagination.QueryParams{
		Params:     pagination.Params{Start: 100, Limit: 20},
		SortParams: pagination.SortParams{Sort: "nodeName", Order: pagination.SortAsc},
	}
	page := taskPagination.SearchOrderAndPaginate(items, params)
	require.EqualValues(t, 120, page.TotalCount)
	require.Len(t, page.Items, 20)
	require.Equal(t, "task-100", page.Items[0].ID)

	params.Start = 0
	params.Filters = map[string]string{"nodeName": "node-105,node-115", "currentState": "running", "desiredState": "running"}
	page = taskPagination.SearchOrderAndPaginate(items, params)
	require.EqualValues(t, 120, page.TotalAvailable)
	require.EqualValues(t, 2, page.TotalCount)
	require.Len(t, page.Items, 2)
	require.Equal(t, "task-105", page.Items[0].ID)
	require.Equal(t, "task-115", page.Items[1].ID)
}

func TestTaskUsesNetwork(t *testing.T) {
	running := swarm.Task{NetworksAttachments: []swarm.NetworkAttachment{{Network: swarm.Network{ID: "overlay"}}}}
	pending := swarm.Task{Spec: swarm.TaskSpec{Networks: []swarm.NetworkAttachmentConfig{{Target: "overlay"}}}}
	unrelated := swarm.Task{NetworksAttachments: []swarm.NetworkAttachment{{Network: swarm.Network{ID: "other-overlay"}}}}

	require.True(t, taskUsesNetwork(running, "overlay"))
	require.True(t, taskUsesNetwork(pending, "overlay"))
	require.False(t, taskUsesNetwork(unrelated, "overlay"))
	require.False(t, taskUsesNetwork(swarm.Task{}, "overlay"))
}

func TestAggregateStatusReportsEachEnvironment(t *testing.T) {
	ctx := swarmScopeContext(t, "/api/agents/a,b,c/swarm/status", map[string]any{
		"a": types.ApiResponse[types.SwarmRuntimeStatus]{
			Success: true,
			Data: types.SwarmRuntimeStatus{
				Enabled:   true,
				ClusterID: "cluster-1",
			},
		},
		"b": types.ApiResponse[types.SwarmRuntimeStatus]{
			Success: true,
			Data:    types.SwarmRuntimeStatus{},
		},
	})

	body, err := (&SwarmService{}).AggregateStatus(ctx)
	require.NoError(t, err)

	require.True(t, body.Data.Enabled)
	require.Equal(t, []types.EnvironmentData[types.SwarmRuntimeStatus]{
		{Environment: types.EnvironmentRef{ID: "a", Name: "Env a"}, Data: types.SwarmRuntimeStatus{Enabled: true, ClusterID: "cluster-1"}},
		{Environment: types.EnvironmentRef{ID: "b", Name: "Env b"}, Data: types.SwarmRuntimeStatus{}},
	}, body.Environments)
	require.Len(t, body.FailedEnvironments, 1)
}

func TestTaskHistoryFilteringKeepsDiagnosticsBeforePagination(t *testing.T) {
	items := []types.SwarmTaskSummary{
		{ID: "old", DesiredState: "shutdown", CurrentState: "shutdown"},
		{ID: "retiring", DesiredState: "shutdown", CurrentState: "running"},
		{ID: "failed", DesiredState: "shutdown", CurrentState: "failed"},
		{ID: "rejected", DesiredState: "shutdown", CurrentState: "rejected"},
	}
	result := taskPagination.SearchOrderAndPaginate(items, pagination.QueryParams{Params: pagination.Params{Limit: 1}, Filters: map[string]string{"history": "active"}})
	require.EqualValues(t, 3, result.TotalCount)
	require.NotEqual(t, "old", result.Items[0].ID)
	history := taskPagination.SearchOrderAndPaginate(items, pagination.QueryParams{Params: pagination.Params{Limit: 20}, Filters: map[string]string{"history": "all"}})
	require.Len(t, history.Items, 4)
}
