package network

import (
	"testing"

	"github.com/stretchr/testify/require"

	"getarcane.dev/arcane/v3/pkg/httpx"
	"getarcane.dev/types/v3"
)

func TestMergeNetworkOverviewPreservesIdentityAndProvenance(t *testing.T) {
	result := func(id string, rows []types.NetworkSummary) httpx.EnvironmentResult[types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts]] {
		return httpx.EnvironmentResult[types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts]]{
			Environment: types.EnvironmentRef{ID: id, Name: "Agent " + id},
			Value:       types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts]{Data: rows},
		}
	}
	overview := mergeNetworkOverview([]httpx.EnvironmentResult[types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts]]{
		result("a", []types.NetworkSummary{
			{ID: "shared", Name: "same", Scope: "swarm", Driver: "overlay", ClusterID: "one", NodeID: "node-a"},
			{ID: "local", Name: "same", Scope: "local", Driver: "bridge"},
			{ID: "unknown", Name: "same", Scope: "swarm", Driver: "overlay", NodeID: "node-a"},
		}),
		result("b", []types.NetworkSummary{
			{ID: "shared", Name: "same", Scope: "swarm", Driver: "overlay", ClusterID: "one", NodeID: "node-b", InUse: true},
			{ID: "shared", Name: "same", Scope: "swarm", Driver: "overlay", ClusterID: "two", NodeID: "other-node"},
			{ID: "local", Name: "same", Scope: "local", Driver: "bridge"},
			{ID: "unknown", Name: "same", Scope: "swarm", Driver: "overlay", NodeID: "node-b"},
		}),
	})
	require.Len(t, overview.Items, 6)
	require.Equal(t, types.NetworkUsageCounts{Inuse: 1, Unused: 5, Total: 6}, overview.Counts)
	require.Equal(t, "a", overview.Items[0].EnvironmentID)
	require.True(t, overview.Items[0].InUse)
	require.Equal(t, []types.NetworkNodeUsage{
		{EnvironmentRef: types.EnvironmentRef{ID: "a", Name: "Agent a"}, NodeID: "node-a"},
		{EnvironmentRef: types.EnvironmentRef{ID: "b", Name: "Agent b"}, NodeID: "node-b", InUse: true},
	}, overview.Items[0].Nodes)
	require.Empty(t, overview.Items[1].Nodes)
	require.Empty(t, overview.Items[2].Nodes)
	require.Len(t, overview.Items[3].Nodes, 1)
	require.Len(t, overview.Environments, 2)
}
