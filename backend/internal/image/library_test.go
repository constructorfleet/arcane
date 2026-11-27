package image

import (
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/swarm"
	"github.com/stretchr/testify/require"

	"getarcane.dev/arcane/v3/pkg/pagination"
	"getarcane.dev/types/v3"
)

func TestImageLibraryGroupsBeforeSearchAndPagination(t *testing.T) {
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	pages := []pagination.EnvironmentPage[types.ImageSummary]{
		{
			Environment: types.EnvironmentRef{ID: "one", Name: "swarm01"},
			Items: []types.ImageSummary{
				{
					ID: "active", Platform: "linux/amd64", Repo: "alpine", Tag: "latest", ActiveReferences: []string{"alpine:latest"},
					RepoTags:    []string{"alpine:latest", "registry.example/project/base:current"},
					RepoDigests: []string{"alpine@" + digest}, InUse: true, Size: 10, Created: 10,
					UpdateInfo: &types.ImageUpdateInfo{CurrentDigest: digest},
					VulnerabilityScan: &types.VulnerabilityScanSummary{
						Status: types.VulnerabilityScanStatusCompleted, Summary: &types.VulnerabilitySeveritySummary{High: 2, Total: 2},
					},
				},
				{ID: "dangling", Repo: "<none>", Tag: "<none>"},
			},
		},
		{
			Environment: types.EnvironmentRef{ID: "two", Name: "swarm02"},
			Items: []types.ImageSummary{
				{
					ID: "cached", Platform: "linux/amd64", Repo: "alpine", Tag: "old", RepoTags: []string{"docker.io/library/alpine:old"},
					Size: 500, Created: 20, InUse: true, UpdateInfo: &types.ImageUpdateInfo{HasUpdate: true, Error: "old task"},
				},
				{ID: "pinned", Platform: "linux/amd64", Repo: "registry.example/library/alpine", RepoDigests: []string{"registry.example/library/alpine@" + digest}},
			},
		},
	}

	service := &ImageService{}
	params := pagination.QueryParams{
		Params:     pagination.Params{Limit: 1},
		SortParams: pagination.SortParams{Sort: "repo", Order: pagination.SortAsc},
		Filters:    map[string]string{"repository": "alpine"},
		Search:     "latest",
	}
	rows, response := service.libraryPage(pages, params)
	require.Len(t, rows, 1)
	require.EqualValues(t, 1, response.TotalItems)
	row := rows[0]
	require.Equal(t, "active", row.ID)
	require.Equal(t, "one", row.EnvironmentID)
	require.Equal(t, []string{"docker.io/library/alpine:latest"}, row.RepoTags)
	require.False(t, row.UpdateInfo.HasUpdate)
	require.Empty(t, row.RepositoryGroup.CheckErrorRef)
	require.Equal(t, []string{digest}, row.RepositoryGroup.Digests)
	require.Len(t, row.RepositoryGroup.Copies, 1)
	require.Equal(t, []string{"docker.io/library/alpine@" + digest}, row.RepositoryGroup.Copies[0].RepoDigests)

	all := groupImageRepositories(pages, "")
	require.Len(t, all, 4)
	require.Equal(t, "dangling", all[0].ID)
	require.Nil(t, all[0].RepositoryGroup)
	require.Equal(t, "docker.io/library/alpine", all[2].Repo)
	require.Equal(t, "active", all[2].ID)
	require.Equal(t, "registry.example/library/alpine", all[3].Repo)
	require.Nil(t, all[3].UpdateInfo)
	require.False(t, all[3].InUse)
}

func TestImageLibrarySeparatesVersionsAndMergesReplicas(t *testing.T) {
	const ref = "example.com/project/image:latest@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	containers := []container.Summary{
		{ID: "old-task", State: container.StateExited, Labels: map[string]string{swarmTaskIDLabel: "old"}},
		{ID: "retiring-task", State: container.StateRunning, Labels: map[string]string{swarmTaskIDLabel: "retiring"}},
		{ID: "current-task", State: container.StateRunning, Labels: map[string]string{swarmTaskIDLabel: "current"}},
		{ID: "standalone", State: container.StateRunning},
	}
	tasks := []swarm.Task{
		{ID: "retiring", DesiredState: swarm.TaskStateShutdown, Status: swarm.TaskStatus{State: swarm.TaskStateRunning}},
		{
			ID: "current", DesiredState: swarm.TaskStateRunning, Status: swarm.TaskStatus{State: swarm.TaskStateRunning},
			Spec: swarm.TaskSpec{ContainerSpec: &swarm.ContainerSpec{Image: ref}},
		},
	}
	running := runningImageContainers(containers, tasks)
	require.Len(t, running, 3)
	require.Equal(t, "current-task", running[1].ID)
	require.Equal(t, ref, running[1].Image)
	require.Equal(t, "standalone", running[2].ID)

	olderScan := &types.VulnerabilityScanSummary{
		Status: types.VulnerabilityScanStatusCompleted, ScanTime: time.Unix(100, 0),
		Summary: &types.VulnerabilitySeveritySummary{High: 10, Total: 10},
	}
	newerScan := &types.VulnerabilityScanSummary{
		Status: types.VulnerabilityScanStatusCompleted, ScanTime: time.Unix(200, 0),
		Summary: &types.VulnerabilitySeveritySummary{High: 2, Total: 2},
	}
	rows := groupImageRepositories([]pagination.EnvironmentPage[types.ImageSummary]{{Items: []types.ImageSummary{
		{
			ID: "v1", Platform: "linux/amd64", Repo: "example.com/project/image", Tag: "latest", InUse: true, Size: 10, RepoTags: []string{"example.com/project/image:latest"},
			ActiveReferences:  []string{ref},
			UpdateInfo:        &types.ImageUpdateInfo{HasUpdate: true},
			VulnerabilityScan: olderScan,
		},
		{
			ID: "v2", Platform: "linux/amd64", Repo: "example.com/project/image", Tag: "edge", InUse: true, Size: 20, RepoTags: []string{"example.com/project/image:edge"},
			ActiveReferences: []string{"example.com/project/image:edge"},
			UpdateInfo:       &types.ImageUpdateInfo{Error: "registry check failed"},
		},
		{
			ID: "z-node-id", Platform: "linux/amd64", Repo: "example.com/project/image", Tag: "latest", InUse: true, Size: 10,
			ActiveReferences: []string{ref}, VulnerabilityScan: newerScan,
		},
	}}}, "")
	require.Len(t, rows, 2)
	grouped := rows[0]
	require.Equal(t, "z-node-id", grouped.ID)
	require.Len(t, grouped.RepositoryGroup.Copies, 2)
	require.NotEqual(t, grouped.LogicalID, rows[1].LogicalID)
	require.True(t, grouped.UpdateInfo.HasUpdate)
	require.Empty(t, grouped.RepositoryGroup.CheckErrorRef)
	require.Equal(t, newerScan, grouped.VulnerabilityScan)
	require.Equal(t, "example.com/project/image:edge", rows[1].RepositoryGroup.CheckErrorRef)
	detail, response := (&ImageService{}).libraryPage([]pagination.EnvironmentPage[types.ImageSummary]{{Items: rows}}, pagination.QueryParams{
		Params: pagination.Params{Limit: 1}, Filters: map[string]string{"repository": "example.com/project/image", "imageId": "v2", "inUse": "true"},
	})
	require.Len(t, detail, 1)
	require.Equal(t, "v2", detail[0].ID)
	require.EqualValues(t, 1, response.TotalItems)
}
