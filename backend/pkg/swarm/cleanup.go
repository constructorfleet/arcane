package swarm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/swarm"
	dockerclient "github.com/moby/moby/client"

	"kit.getarcane.app"
	"kit.getarcane.app/concurrency"

	"getarcane.dev/types/v3"
)

type (
	staleManagedResource struct {
		ID   string
		Name string
	}
)

const (
	// Swarm tears tasks down asynchronously, so a config, secret, or network can
	// stay referenced for a short window after the owning services are removed.
	staleSwarmResourceRemoveAttempts = 5
)

var staleSwarmResourceRemoveBackoff = concurrency.Backoff{Base: 500 * time.Millisecond}

func cleanupStackResources(
	ctx context.Context,
	dockerClient *dockerclient.Client,
	stackName string,
	configMetaByKey map[string]resourceMeta,
	secretMetaByKey map[string]resourceMeta,
) error {
	retainedConfigs, retainedSecrets, inventoryErr := referencedStackResources(ctx, dockerClient)
	if inventoryErr != nil {
		slog.WarnContext(ctx, "Automatic stack resource cleanup deferred", "stack", stackName, "error", inventoryErr)
		return nil
	}
	for _, meta := range configMetaByKey {
		retainedConfigs[meta.Name] = struct{}{}
	}
	for _, meta := range secretMetaByKey {
		retainedSecrets[meta.Name] = struct{}{}
	}
	if err := cleanupStaleManagedResources(
		ctx,
		stackName,
		retainedConfigs,
		configFileResourceAdapter(dockerClient),
		true,
		true,
	); err != nil {
		return err
	}

	if err := cleanupStaleManagedResources(
		ctx,
		stackName,
		retainedSecrets,
		secretFileResourceAdapter(dockerClient),
		true,
		true,
	); err != nil {
		return err
	}

	return nil
}

func referencedStackResources(ctx context.Context, dockerClient *dockerclient.Client) (map[string]struct{}, map[string]struct{}, error) {
	services, err := dockerClient.ServiceList(ctx, dockerclient.ServiceListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list service resource references: %w", err)
	}
	tasks, err := dockerClient.TaskList(ctx, dockerclient.TaskListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list task resource references: %w", err)
	}
	configs := make(map[string]struct{})
	secrets := make(map[string]struct{})
	for _, service := range services.Items {
		retainTaskResourceReferences(service.Spec.TaskTemplate, configs, secrets)
		retainTaskResourceReferences(kit.FromPtr(service.PreviousSpec).TaskTemplate, configs, secrets)
	}
	for _, task := range tasks.Items {
		retainTaskResourceReferences(task.Spec, configs, secrets)
	}
	return configs, secrets, nil
}

func retainTaskResourceReferences(spec swarm.TaskSpec, configs, secrets map[string]struct{}) {
	container := kit.FromPtr(spec.ContainerSpec)
	credentialSpec := kit.FromPtr(kit.FromPtr(container.Privileges).CredentialSpec)
	if credentialSpec.Config != "" {
		configs[credentialSpec.Config] = struct{}{}
	}
	for _, reference := range container.Configs {
		if reference.ConfigID != "" {
			configs[reference.ConfigID] = struct{}{}
		}
		if reference.ConfigName != "" {
			configs[reference.ConfigName] = struct{}{}
		}
	}
	for _, reference := range container.Secrets {
		if reference.SecretID != "" {
			secrets[reference.SecretID] = struct{}{}
		}
		if reference.SecretName != "" {
			secrets[reference.SecretName] = struct{}{}
		}
	}
}

func collectStaleManagedResources[T any](items []T, convert func(T) staleManagedResource) []staleManagedResource {
	resources := make([]staleManagedResource, 0, len(items))
	for _, item := range items {
		resources = append(resources, convert(item))
	}
	return resources
}

func cleanupStaleManagedResources(
	ctx context.Context,
	stackName string,
	retainedResources map[string]struct{},
	adapter fileResourceAdapter,
	managedOnly bool,
	tolerateInUse bool,
) error {
	resourceTypeLabels := []string{""}
	if managedOnly {
		resourceTypeLabels = []string{resourceTypeLabel, legacyResourceTypeLabel}
	}

	resources := make([]staleManagedResource, 0)
	seenResourceIDs := make(map[string]struct{})
	for _, typeLabel := range resourceTypeLabels {
		filters := make(dockerclient.Filters)
		filters.Add("label", fmt.Sprintf("%s=%s", types.SwarmStackNamespaceLabel, stackName))
		if typeLabel != "" {
			filters.Add("label", typeLabel+"="+adapter.ResourceType)
		}

		listedResources, err := adapter.List(ctx, filters)
		if err != nil {
			return err
		}
		for _, resource := range listedResources {
			if _, seen := seenResourceIDs[resource.ID]; seen {
				continue
			}
			seenResourceIDs[resource.ID] = struct{}{}
			resources = append(resources, resource)
		}
	}

	for _, resource := range resources {
		if _, ok := retainedResources[resource.Name]; ok {
			continue
		}
		if _, ok := retainedResources[resource.ID]; ok {
			continue
		}
		if err := removeStaleSwarmResource(ctx, tolerateInUse, func(ctx context.Context) error {
			return adapter.Remove(ctx, resource.ID)
		}); err != nil {
			return fmt.Errorf("failed to remove stale stack %s %s: %w", adapter.ResourceType, resource.Name, err)
		}
	}

	return nil
}

func removeStaleSwarmResource(ctx context.Context, tolerateInUse bool, remove func(context.Context) error) error {
	attempts := kit.Ternary(tolerateInUse, 1, staleSwarmResourceRemoveAttempts)

	var err error
	for attempt := range attempts {
		err = remove(ctx)
		if err == nil || cerrdefs.IsNotFound(err) {
			return nil
		}
		if !isStaleSwarmResourceStillInUse(err) {
			return err
		}
		if attempt == attempts-1 {
			break
		}
		if sleepErr := staleSwarmResourceRemoveBackoff.Sleep(ctx, attempt); sleepErr != nil {
			return errors.Join(sleepErr, err)
		}
	}

	return kit.Ternary(tolerateInUse, nil, err)
}

func RemoveStackResources(ctx context.Context, dockerClient *dockerclient.Client, stackName string) error {
	stackName = strings.TrimSpace(stackName)
	if stackName == "" {
		return invalidStackError(errors.New("stack name is required"))
	}

	emptyDesiredSet := map[string]struct{}{}
	if err := cleanupStaleManagedResources(ctx, stackName, emptyDesiredSet, configFileResourceAdapter(dockerClient), false, false); err != nil {
		return err
	}
	if err := cleanupStaleManagedResources(ctx, stackName, emptyDesiredSet, secretFileResourceAdapter(dockerClient), false, false); err != nil {
		return err
	}

	filters := make(dockerclient.Filters).Add("label", fmt.Sprintf("%s=%s", types.SwarmStackNamespaceLabel, stackName))
	networksResult, err := dockerClient.NetworkList(ctx, dockerclient.NetworkListOptions{Filters: filters})
	if err != nil {
		return fmt.Errorf("failed to list stack networks: %w", err)
	}
	networks := networksResult.Items
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})
	for _, stackNetwork := range networks {
		if stackNetwork.Ingress {
			continue
		}
		if removeStaleSwarmResourceErr := removeStaleSwarmResource(ctx, false, func(ctx context.Context) error {
			_, networkRemoveErr := dockerClient.NetworkRemove(ctx, stackNetwork.ID, dockerclient.NetworkRemoveOptions{})
			return networkRemoveErr
		}); removeStaleSwarmResourceErr != nil {
			return fmt.Errorf("failed to remove stack network %s: %w", stackNetwork.Name, removeStaleSwarmResourceErr)
		}
	}

	return nil
}

func isStaleSwarmResourceStillInUse(err error) bool {
	if cerrdefs.IsConflict(err) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "in use")
}
