package image

import (
	"testing"

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
					ID: "active", Repo: "alpine", Tag: "latest", ActiveRepositories: []string{"docker.io/library/alpine"},
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
					ID: "cached", Repo: "alpine", Tag: "old", RepoTags: []string{"docker.io/library/alpine:old"},
					Size: 500, Created: 20, UpdateInfo: &types.ImageUpdateInfo{HasUpdate: true, Error: "idle cached copy"},
				},
				{ID: "pinned", Repo: "registry.example/library/alpine", RepoDigests: []string{"registry.example/library/alpine@" + digest}},
			},
		},
	}

	service := &ImageService{}
	params := pagination.QueryParams{
		Params:     pagination.Params{Limit: 1},
		SortParams: pagination.SortParams{Sort: "repo", Order: pagination.SortAsc},
		Filters:    map[string]string{"repository": "docker.io/library/alpine"},
		Search:     "old",
	}
	rows, response := service.libraryPage(pages, params)
	require.Len(t, rows, 1)
	require.EqualValues(t, 1, response.TotalItems)
	row := rows[0]
	require.Equal(t, "active", row.ID)
	require.Equal(t, "one", row.EnvironmentID)
	require.Equal(t, []string{"docker.io/library/alpine:latest", "docker.io/library/alpine:old"}, row.RepoTags)
	require.False(t, row.UpdateInfo.HasUpdate)
	require.Empty(t, row.RepositoryGroup.CheckErrorRef)
	require.Equal(t, []string{digest}, row.RepositoryGroup.Digests)
	require.EqualValues(t, 10, row.RepositoryGroup.MaxSize)
	require.Equal(t, 1, row.RepositoryGroup.Variants)
	require.Equal(t, 1, row.RepositoryGroup.ScannedVariants)
	require.Len(t, row.RepositoryGroup.Copies, 2)
	require.Equal(t, "two", row.RepositoryGroup.Copies[1].EnvironmentID)

	all := groupImageRepositories(pages, "")
	require.Len(t, all, 4)
	require.Equal(t, "dangling", all[0].ID)
	require.Nil(t, all[0].RepositoryGroup)
	require.Equal(t, "registry.example/library/alpine", all[2].Repo)
	require.Equal(t, "pinned", all[2].ID)
	require.Equal(t, "registry.example/project/base", all[3].Repo)
	require.Nil(t, all[3].UpdateInfo)
	require.False(t, all[3].InUse)
}

func TestImageLibrarySurfacesActiveCheckFailures(t *testing.T) {
	grouped := mergeImageRepository([]types.ImageSummary{
		{
			ID: "v1", Repo: "example.com/project/image", Tag: "latest", InUse: true, Size: 10,
			UpdateInfo: &types.ImageUpdateInfo{HasUpdate: true},
		},
		{
			ID: "v2", Repo: "example.com/project/image", Tag: "edge", InUse: true, Size: 20,
			UpdateInfo: &types.ImageUpdateInfo{Error: "registry check failed"},
		},
		{ID: "v2", Repo: "example.com/project/image", Tag: "edge", InUse: true, Size: 20},
	})
	require.True(t, grouped.UpdateInfo.HasUpdate)
	require.Equal(t, "example.com/project/image:edge", grouped.RepositoryGroup.CheckErrorRef)
	require.Equal(t, 2, grouped.RepositoryGroup.Variants)
	require.EqualValues(t, 10, grouped.RepositoryGroup.MinSize)
	require.EqualValues(t, 20, grouped.RepositoryGroup.MaxSize)
}
