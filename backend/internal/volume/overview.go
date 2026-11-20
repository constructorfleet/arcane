package volume

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/api/types/volume"

	"kit.getarcane.app"

	"getarcane.dev/arcane/v3/pkg/httpx"
	"getarcane.dev/arcane/v3/pkg/volumes"
	"getarcane.dev/types/v3"
)

type volumeConsumers struct {
	IDs          []string
	Applications int
	Helpers      int
}

func volumeConsumerMap(containers []container.Summary) map[string]volumeConsumers {
	result := map[string]volumeConsumers{}
	for _, consumer := range containers {
		for _, mounted := range consumer.Mounts {
			if mounted.Type != mount.TypeVolume || mounted.Name == "" {
				continue
			}
			state := result[mounted.Name]
			if slices.Contains(state.IDs, consumer.ID) {
				continue
			}
			state.IDs = append(state.IDs, consumer.ID)
			if volumes.IsHelperContainer(consumer) {
				if consumer.State == container.StateRunning {
					state.Helpers++
				}
			} else if consumer.State == container.StateRunning || consumer.State == container.StatePaused || consumer.State == container.StateRestarting {
				state.Applications++
			}
			result[mounted.Name] = state
		}
	}
	return result
}

func volumeObservation(native volume.Volume, consumers volumeConsumers, node swarm.Info) types.Volume {
	item := types.NewVolume(native)
	item.ClusterID = kit.FromPtr(node.Cluster).ID
	item.NodeID = node.NodeID
	for _, manager := range node.RemoteManagers {
		item.ManagerNodeIDs = append(item.ManagerNodeIDs, manager.NodeID)
	}
	item.Containers = consumers.IDs
	item.ActiveConsumers = consumers.Applications
	item.RunningHelpers = consumers.Helpers
	item.InUse = item.InUse || len(consumers.IDs) > 0
	return item
}

func mergeVolumeOverview(results []httpx.EnvironmentResult[types.PaginatedWithCounts[types.Volume, VolumeUsageCountsData]]) ([]types.Volume, VolumeUsageCountsData, int64) {
	// Workers can identify their Swarm through an observed manager's node ID.
	clusters := map[string]string{}
	for _, result := range results {
		for _, item := range result.Value.Data {
			if item.ClusterID != "" && item.NodeID != "" {
				if previous, exists := clusters[item.NodeID]; exists && previous != item.ClusterID {
					clusters[item.NodeID] = ""
				} else if !exists {
					clusters[item.NodeID] = item.ClusterID
				}
			}
		}
	}
	groups := map[string][]types.Volume{}
	var items []types.Volume
	var unseen int64
	for _, result := range results {
		unseen += max(cmp.Or(result.Value.Pagination.GrandTotalItems, result.Value.Pagination.TotalItems)-int64(len(result.Value.Data)), 0)
		for _, item := range result.Value.Data {
			item.SetEnvironment(result.Environment)
			if item.ClusterID == "" {
				for _, managerID := range item.ManagerNodeIDs {
					cluster := clusters[managerID]
					if cluster != "" {
						if item.ClusterID != "" && item.ClusterID != cluster {
							item.ClusterID = ""
							break
						}
						item.ClusterID = cluster
					}
				}
			}
			if item.Scope != "global" || item.Driver == "" || item.Driver == "local" || item.ClusterID == "" || item.NodeID == "" || item.Name == "" {
				items = append(items, item)
				continue
			}
			key := fmt.Sprintf("global-volume:%q:%q:%q", item.ClusterID, item.Driver, item.Name)
			groups[key] = append(groups[key], item)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(groups)) {
		items = append(items, mergeGlobalVolume(groups[key], key))
	}
	var counts VolumeUsageCountsData
	for _, item := range items {
		counts.Total++
		if item.InUse {
			counts.Inuse++
		} else {
			counts.Unused++
		}
	}
	return items, counts, unseen
}

func mergeGlobalVolume(members []types.Volume, key string) types.Volume {
	slices.SortFunc(members, func(a, b types.Volume) int {
		return cmp.Or(cmp.Compare(a.EnvironmentName, b.EnvironmentName), cmp.Compare(a.EnvironmentID, b.EnvironmentID))
	})
	selected := members[0]
	var applications, helpers []types.Volume
	var nodes []types.VolumeNodeUsage
	inUse := false
	for _, member := range members {
		inUse = inUse || member.InUse
		nodes = append(nodes, types.VolumeNodeUsage{
			EnvironmentRef: member.Environment(), NodeID: member.NodeID, VolumeName: member.Name,
			Mountpoint: member.Mountpoint, Containers: member.Containers, InUse: member.InUse,
			ActiveConsumers: member.ActiveConsumers, RunningHelpers: member.RunningHelpers,
		})
		if member.Mountpoint != "" {
			if member.ActiveConsumers > 0 {
				applications = append(applications, member)
			} else if member.RunningHelpers > 0 {
				helpers = append(helpers, member)
			}
		}
	}
	candidates := applications
	if len(candidates) == 0 {
		candidates = helpers
	}
	state := "inactive"
	if len(candidates) == 1 {
		selected = candidates[0]
		state = "active"
	} else if len(candidates) > 1 {
		state = "ambiguous"
	}
	selected.LogicalID = key
	selected.Nodes = nodes
	selected.InUse = inUse
	selected.WorkspaceState = state
	return selected
}
