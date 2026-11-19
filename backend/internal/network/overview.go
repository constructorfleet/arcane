package network

import (
	"cmp"

	"getarcane.dev/arcane/v3/pkg/httpx"
	"getarcane.dev/types/v3"
)

type networkOverview struct {
	Items        []types.NetworkSummary
	Environments []types.EnvironmentData[types.NetworkUsageCounts]
	Counts       types.NetworkUsageCounts
	Unseen       int64
}

func mergeNetworkOverview(results []httpx.EnvironmentResult[types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts]]) networkOverview {
	overview := networkOverview{}
	shared := map[string]int{}
	for _, result := range results {
		overview.Environments = append(overview.Environments, types.EnvironmentData[types.NetworkUsageCounts]{Environment: result.Environment, Data: result.Value.Counts})
		overview.Unseen += max(cmp.Or(result.Value.Pagination.GrandTotalItems, result.Value.Pagination.TotalItems)-int64(len(result.Value.Data)), 0)
		for _, item := range result.Value.Data {
			item.SetEnvironment(result.Environment)
			// Only a confirmed cluster/network identity can combine node observations.
			if item.Scope != "swarm" || item.Driver != "overlay" || item.ClusterID == "" || item.NodeID == "" || item.ID == "" {
				overview.Items = append(overview.Items, item)
				continue
			}
			member := types.NetworkNodeUsage{EnvironmentRef: result.Environment, NodeID: item.NodeID, InUse: item.InUse || item.IsDefault}
			key := item.ClusterID + "/" + item.ID
			if index, exists := shared[key]; exists {
				overview.Items[index].Nodes = append(overview.Items[index].Nodes, member)
				overview.Items[index].InUse = overview.Items[index].InUse || item.InUse
				continue
			}
			shared[key] = len(overview.Items)
			item.Nodes = []types.NetworkNodeUsage{member} // keep: enriches the node observation with provenance
			overview.Items = append(overview.Items, item)
		}
	}
	for _, item := range overview.Items {
		overview.Counts.Total++
		if item.InUse || item.IsDefault {
			overview.Counts.Inuse++
		} else {
			overview.Counts.Unused++
		}
	}
	return overview
}
