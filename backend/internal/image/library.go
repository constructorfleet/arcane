package image

import (
	"cmp"
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"

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
			filters := make(client.Filters).Add("node", info.Info.Swarm.NodeID)
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
		if task.Status.State == swarm.TaskStateRunning {
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
	if imageID := params.Filters["imageId"]; imageID != "" {
		items = slices.DeleteFunc(items, func(item types.ImageSummary) bool {
			source := params.Filters["imageEnvironmentId"]
			if item.RepositoryGroup == nil {
				return item.ID != imageID || source != "" && item.EnvironmentID != source
			}
			return !slices.ContainsFunc(item.RepositoryGroup.Copies, func(copy types.ImageRepositoryCopy) bool {
				return copy.ID == imageID && (source == "" || copy.EnvironmentID == source)
			})
		})
	}
	params.Filters = maps.Clone(params.Filters)
	delete(params.Filters, "repository")
	delete(params.Filters, "groupByImage")
	delete(params.Filters, "imageId")
	delete(params.Filters, "imageEnvironmentId")
	config := pagination.WithEnvironment(s.getImagePaginationConfig())
	result := config.SearchOrderAndPaginate(items, params)
	if params.Limit > 0 && int64(params.Start) >= result.TotalCount {
		params.Start = int(max(result.TotalCount-1, 0)) / params.Limit * params.Limit
		result = config.SearchOrderAndPaginate(items, params)
	}
	return result.Items, pagination.BuildResponse(result.TotalCount, result.TotalAvailable, params)
}

func groupImageRepositories(pages []pagination.EnvironmentPage[types.ImageSummary], onlyRepository string) []types.ImageSummary {
	if onlyRepository != "" {
		onlyRepository = imageRepositoryName(onlyRepository)
		if onlyRepository == "" {
			return nil
		}
	}
	groups := map[string][]types.ImageSummary{}
	var items []types.ImageSummary
	for _, page := range pages {
		for _, image := range page.Items {
			image.SetEnvironment(page.Environment)
			byRepository := imageCopiesByRepository(image, onlyRepository)
			for _, observation := range byRepository {
				key := observation.ArtifactKey
				groups[key] = append(groups[key], observation)
			}
			if len(byRepository) == 0 && onlyRepository == "" {
				if image.InUse {
					key := "local:" + image.ID + "|" + image.Platform
					groups[key] = append(groups[key], image)
				} else {
					items = append(items, image)
				}
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
			observation.RepoTags, observation.RepoDigests = nil, nil
			observation.PinnedReferences = slices.DeleteFunc(slices.Clone(image.ActiveReferences), func(active string) bool {
				return imageRepositoryName(active) != repo || !strings.Contains(active, "@")
			})
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
		if len(image.ActiveReferences) > 0 && !observation.InUse {
			delete(byRepository, repo)
			continue
		}
		key, _ := imageVariant(observation)
		observation.ArtifactKey = "image:" + repo + "@" + key
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
	if slices.ContainsFunc(copies, func(copy types.ImageSummary) bool { return copy.InUse }) {
		copies = slices.DeleteFunc(copies, func(copy types.ImageSummary) bool { return !copy.InUse })
	}
	slices.SortFunc(copies, func(a, b types.ImageSummary) int {
		if a.InUse != b.InUse {
			if a.InUse {
				return -1
			}
			return 1
		}
		switch {
		case a.VulnerabilityScan != nil && b.VulnerabilityScan != nil:
			if freshness := b.VulnerabilityScan.ScanTime.Compare(a.VulnerabilityScan.ScanTime); freshness != 0 {
				return freshness
			}
		case a.VulnerabilityScan != nil:
			return -1
		case b.VulnerabilityScan != nil:
			return 1
		}
		return cmp.Or(cmp.Compare(b.Created, a.Created), cmp.Compare(a.EnvironmentID, b.EnvironmentID), cmp.Compare(a.ID, b.ID))
	})
	row := copies[0]
	row.RunningContainers, row.UsedBy = runningRepositoryConsumers(copies)
	row.LogicalID = row.ArtifactKey
	if row.Repo == "<none>" || row.Repo == "" {
		row.LogicalID = "image-local:" + row.ID + "|" + row.Platform
	}
	row.RepoTags, row.RepoDigests = nil, nil
	group := &types.ImageRepositoryGroup{}
	checked := false
	for i, observation := range copies {
		group.Copies = append(group.Copies, types.ImageRepositoryCopy{
			ArtifactKey: observation.ArtifactKey, Platform: observation.Platform, EnvironmentTag: observation.EnvironmentTag, ID: observation.ID, Tags: observation.RepoTags, InUse: observation.InUse,
			RepoDigests: observation.RepoDigests, PinnedReferences: observation.PinnedReferences, Hardened: observation.Hardened,
		})
		row.RepoTags = append(row.RepoTags, observation.RepoTags...)
		row.RepoDigests = append(row.RepoDigests, observation.RepoDigests...)
		if !observation.InUse && (row.InUse || i > 0) {
			continue
		}
		_, digest := imageVariant(observation)
		if digest != "" {
			group.Digests = append(group.Digests, digest)
		}
		if row.InUse && observation.UpdateInfo != nil && observation.Tag != "<none>" {
			checked = true
			if observation.UpdateInfo.HasUpdate && (row.UpdateInfo == nil || !row.UpdateInfo.HasUpdate) {
				row.UpdateInfo = observation.UpdateInfo
			}
			if observation.UpdateInfo.Error != "" && group.CheckErrorRef == "" {
				group.CheckErrorRef = observation.Repo + ":" + observation.Tag
			}
		}
	}
	if !row.InUse || !checked && row.UpdateInfo != nil && !row.UpdateInfo.HasUpdate {
		row.UpdateInfo = nil
	}
	row.RepoTags = sortedUnique(row.RepoTags)
	row.RepoDigests = sortedUnique(row.RepoDigests)
	group.Digests = sortedUnique(group.Digests)
	row.RepositoryGroup = group
	return row
}

func runningRepositoryConsumers(copies []types.ImageSummary) (*int, []types.ImageUsedBy) {
	runningContainers := 0
	countedCopies := make(map[string]bool)
	consumerIdentities := make(map[string]bool)
	var consumers []types.ImageUsedBy
	for _, observation := range copies {
		if !observation.InUse {
			continue
		}
		if observation.RunningContainers == nil {
			return nil, nil
		}
		copyIdentity := observation.EnvironmentID + "\x00" + observation.ID
		if !countedCopies[copyIdentity] {
			runningContainers += *observation.RunningContainers
			countedCopies[copyIdentity] = true
		}
		for _, consumer := range observation.UsedBy {
			identity := observation.EnvironmentID + "\x00" + consumer.Type + "\x00" + cmp.Or(consumer.ID, consumer.Name)
			if consumerIdentities[identity] {
				continue
			}
			consumerIdentities[identity] = true
			consumers = append(consumers, types.ImageUsedBy{
				EnvironmentTag: observation.EnvironmentTag, Type: consumer.Type, Name: consumer.Name, ID: consumer.ID,
			})
		}
	}
	return new(runningContainers), consumers
}

func imageVariant(observation types.ImageSummary) (string, string) {
	if observation.Platform == "" {
		return observation.ID + "|" + observation.EnvironmentID, ""
	}
	var pins []string
	for _, ref := range observation.ActiveReferences {
		if _, digest, pinned := strings.Cut(ref, "@"); pinned && imageRepositoryName(ref) == observation.Repo {
			pins = append(pins, digest)
		}
	}
	if pins = sortedUnique(pins); len(pins) == 1 {
		return pins[0] + "|" + observation.Platform, pins[0]
	}
	if len(observation.RepoDigests) == 1 {
		_, digest, _ := strings.Cut(observation.RepoDigests[0], "@")
		return digest + "|" + observation.Platform, digest
	}
	return observation.ID + "|" + observation.Platform, ""
}

// imagePlatforms reads immutable local metadata once, without contacting a registry.
func (s *ImageService) imagePlatforms(ctx context.Context, consumers []container.Summary) map[string]string {
	ids := make(map[string]struct{}, len(consumers))
	for _, consumer := range consumers {
		ids[consumer.ImageID] = struct{}{}
	}
	ordered := slices.Sorted(maps.Keys(ids))
	values := make([]string, len(ordered))
	var group errgroup.Group
	group.SetLimit(8)
	for i, id := range ordered {
		group.Go(func() error {
			value, _, err := s.platforms.GetOrLoad(id, func(id string) (string, bool, error) {
				inspected, err := s.dockerService.Call(ctx, func(ctx context.Context, cli *client.Client) (client.ImageInspectResult, error) {
					return cli.ImageInspect(ctx, id)
				})
				if err != nil {
					return "", false, err
				}
				if inspected.Os == "" || inspected.Architecture == "" {
					return "", false, nil
				}
				platform := platforms.Format(platforms.Normalize(v1.Platform{OS: inspected.Os, Architecture: inspected.Architecture, Variant: inspected.Variant}))
				return platform, true, nil
			})
			if err != nil {
				slog.DebugContext(ctx, "Image platform unavailable", "imageId", id, "error", err)
				return nil
			}
			values[i] = value
			return nil
		})
	}
	_ = group.Wait()
	result := make(map[string]string, len(ordered))
	for i, id := range ordered {
		result[id] = values[i]
	}
	return result
}
