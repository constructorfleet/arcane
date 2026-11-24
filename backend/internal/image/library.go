package image

import (
	"cmp"
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/distribution/reference"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"

	"getarcane.dev/arcane/v3/pkg/pagination"
	"getarcane.dev/types/v3"
)

const swarmTaskIDLabel = "com.docker.swarm.task.id"

func (s *ImageService) runningImageConsumers(ctx context.Context, containers []container.Summary) []container.Summary {
	hasRunningTask := slices.ContainsFunc(containers, func(c container.Summary) bool {
		return c.State == container.StateRunning && c.Labels[swarmTaskIDLabel] != ""
	})
	var tasks []swarm.Task
	if hasRunningTask {
		var taskErr error
		tasks, taskErr = s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) ([]swarm.Task, error) {
			info, err := dockerClient.Info(ctx, client.InfoOptions{})
			if err != nil || !info.Info.Swarm.ControlAvailable {
				return nil, err
			}
			filters := make(client.Filters).Add("node", info.Info.Swarm.NodeID).Add("desired-state", string(swarm.TaskStateRunning))
			result, err := dockerClient.TaskList(ctx, client.TaskListOptions{Filters: filters})
			return append([]swarm.Task{}, result.Items...), err
		})
		if taskErr != nil {
			slog.DebugContext(ctx, "Using local container states for image inventory", "error", taskErr)
			tasks = nil
		}
	}
	return runningImageContainers(containers, tasks)
}

func runningImageContainers(containers []container.Summary, tasks []swarm.Task) []container.Summary {
	active := make(map[string]string, len(tasks))
	for _, task := range tasks {
		if task.DesiredState == swarm.TaskStateRunning && task.Status.State == swarm.TaskStateRunning {
			active[task.ID] = ""
			if task.Spec.ContainerSpec != nil {
				active[task.ID] = task.Spec.ContainerSpec.Image
			}
		}
	}
	var consumers []container.Summary
	for _, c := range containers {
		if c.State != container.StateRunning {
			continue
		}
		if taskID := c.Labels[swarmTaskIDLabel]; taskID != "" && tasks != nil {
			ref, running := active[taskID]
			if !running {
				continue
			}
			c.Image = cmp.Or(ref, c.Image)
		}
		consumers = append(consumers, c)
	}
	return consumers
}

func (s *ImageService) libraryPage(pages []pagination.EnvironmentPage[types.ImageSummary], params pagination.QueryParams) ([]types.ImageSummary, pagination.Response) {
	items := groupImageRepositories(pages, params.Filters["repository"])
	params.Filters = maps.Clone(params.Filters)
	delete(params.Filters, "repository")
	delete(params.Filters, "groupByRepository")
	config := pagination.WithEnvironment(s.getImagePaginationConfig())
	result := config.SearchOrderAndPaginate(items, params)
	if params.Limit > 0 && int64(params.Start) >= result.TotalCount {
		params.Start = int(max(result.TotalCount-1, 0)) / params.Limit * params.Limit
		result = config.SearchOrderAndPaginate(items, params)
	}
	return result.Items, pagination.BuildResponse(result.TotalCount, result.TotalAvailable, params)
}

func groupImageRepositories(pages []pagination.EnvironmentPage[types.ImageSummary], onlyRepository string) []types.ImageSummary {
	groups := map[string][]types.ImageSummary{}
	var items []types.ImageSummary
	for _, page := range pages {
		for _, image := range page.Items {
			image.SetEnvironment(page.Environment)
			byRepository := imageCopiesByRepository(image, onlyRepository)
			for repo, observation := range byRepository {
				groups[repo] = append(groups[repo], observation)
			}
			if len(byRepository) == 0 && onlyRepository == "" {
				items = append(items, image)
			}
		}
	}
	for _, repo := range slices.Sorted(maps.Keys(groups)) {
		items = append(items, mergeImageRepository(groups[repo]))
	}
	return items
}

func imageCopiesByRepository(image types.ImageSummary, onlyRepository string) map[string]types.ImageSummary {
	byRepository := map[string]types.ImageSummary{}
	primary := imageRepositoryName(image.Repo)
	for _, ref := range slices.Concat(image.RepoTags, image.RepoDigests, image.PinnedReferences, image.ActiveReferences) {
		repo := imageRepositoryName(ref)
		if repo == "" || onlyRepository != "" && onlyRepository != repo {
			continue
		}
		observation, exists := byRepository[repo]
		if !exists {
			observation = image
			observation.InUse = slices.ContainsFunc(image.ActiveReferences, func(active string) bool {
				activeRepo := imageRepositoryName(active)
				return activeRepo == repo || activeRepo == "" && repo == primary
			})
			observation.Repo, observation.Tag = repo, "<none>"
			observation.RepoTags, observation.RepoDigests, observation.PinnedReferences = nil, nil, nil
			if repo != primary {
				observation.UpdateInfo = nil
			}
		}
		if _, digest, pinned := strings.Cut(ref, "@"); pinned {
			observation.RepoDigests = append(observation.RepoDigests, repo+"@"+digest)
		} else {
			name, _ := reference.ParseNormalizedNamed(ref)
			tagged := reference.TagNameOnly(name).(reference.NamedTagged)
			observation.RepoTags = append(observation.RepoTags, repo+":"+tagged.Tag())
		}
		byRepository[repo] = observation
	}
	for repo, observation := range byRepository {
		observation.RepoTags = sortedUnique(observation.RepoTags)
		observation.RepoDigests = sortedUnique(observation.RepoDigests)
		if len(observation.RepoTags) != 0 {
			_, tag, _ := strings.CutLast(observation.RepoTags[0], ":")
			observation.Tag = tag
		}
		if repo == primary && image.Tag != "<none>" && slices.Contains(observation.RepoTags, repo+":"+image.Tag) {
			observation.Tag = image.Tag
		}
		byRepository[repo] = observation
	}
	return byRepository
}

func imageRepositoryName(ref string) string {
	if strings.TrimSpace(ref) == "" || strings.HasPrefix(ref, "sha256:") || ref == "<none>" || ref == "<none>:<none>" {
		return ""
	}
	named, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return ""
	}
	return reference.TrimNamed(named).Name()
}

func sortedUnique(values []string) []string {
	slices.Sort(values)
	return slices.Compact(values)
}

func mergeImageRepository(copies []types.ImageSummary) types.ImageSummary {
	slices.SortFunc(copies, func(a, b types.ImageSummary) int {
		if a.InUse != b.InUse {
			if a.InUse {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(b.Created, a.Created), cmp.Compare(a.EnvironmentID, b.EnvironmentID), cmp.Compare(a.ID, b.ID))
	})
	row := copies[0]
	row.LogicalID = "image-repository:" + row.Repo
	row.RepoTags, row.RepoDigests = nil, nil
	group := &types.ImageRepositoryGroup{MinSize: row.Size, MaxSize: row.Size}
	observed := map[string]struct{}{}
	scanned := map[string]struct{}{}
	checked := map[string]struct{}{}
	row.VulnerabilityScan = nil
	for i, observation := range copies {
		group.Copies = append(group.Copies, types.ImageRepositoryCopy{
			EnvironmentTag: observation.EnvironmentTag, ID: observation.ID, Tags: observation.RepoTags, InUse: observation.InUse,
		})
		row.RepoTags = append(row.RepoTags, observation.RepoTags...)
		row.RepoDigests = append(row.RepoDigests, observation.RepoDigests...)
		if !observation.InUse && (row.InUse || i > 0) {
			continue
		}
		key, digest := imageVariant(observation)
		observed[key] = struct{}{}
		group.MinSize = min(group.MinSize, observation.Size)
		group.MaxSize = max(group.MaxSize, observation.Size)
		if digest != "" {
			group.Digests = append(group.Digests, digest)
		}
		if row.InUse && observation.UpdateInfo != nil && observation.Tag != "<none>" {
			checked[key] = struct{}{}
			if observation.UpdateInfo.HasUpdate && (row.UpdateInfo == nil || !row.UpdateInfo.HasUpdate) {
				row.UpdateInfo = observation.UpdateInfo
			}
			if observation.UpdateInfo.Error != "" && group.CheckErrorRef == "" {
				group.CheckErrorRef = observation.Repo + ":" + observation.Tag
			}
		}
		if report := observation.VulnerabilityScan; report != nil {
			group.SawScanError = group.SawScanError || report.Status == types.VulnerabilityScanStatusFailed
			if report.Status == types.VulnerabilityScanStatusCompleted && report.Summary != nil {
				scanned[key] = struct{}{}
				if row.VulnerabilityScan == nil || compareImageScans(report.Summary, row.VulnerabilityScan.Summary) > 0 {
					row.VulnerabilityScan = report
				}
			}
		}
	}
	if !row.InUse || len(checked) < len(observed) && row.UpdateInfo != nil && !row.UpdateInfo.HasUpdate {
		row.UpdateInfo = nil
	}
	row.RepoTags = sortedUnique(row.RepoTags)
	row.RepoDigests = sortedUnique(row.RepoDigests)
	group.Digests = sortedUnique(group.Digests)
	group.Variants = len(observed)
	group.ScannedVariants = len(scanned)
	row.RepositoryGroup = group
	return row
}

func imageVariant(observation types.ImageSummary) (string, string) {
	var pins []string
	for _, ref := range observation.ActiveReferences {
		if _, digest, pinned := strings.Cut(ref, "@"); pinned && imageRepositoryName(ref) == observation.Repo {
			pins = append(pins, digest)
		}
	}
	if pins = sortedUnique(pins); len(pins) == 1 {
		return pins[0], pins[0]
	}
	if len(observation.RepoDigests) == 1 {
		_, digest, _ := strings.Cut(observation.RepoDigests[0], "@")
		return digest, digest
	}
	return observation.ID, ""
}

func compareImageScans(a, b *types.VulnerabilitySeveritySummary) int {
	return cmp.Or(cmp.Compare(a.Critical, b.Critical), cmp.Compare(a.High, b.High), cmp.Compare(a.Total, b.Total))
}
