package network

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"kit.getarcane.app"
	"kit.getarcane.app/concurrency"
	"kit.getarcane.app/docker"
	"kit.getarcane.app/docker/compat"
	"kit.getarcane.app/mapping"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/arcane/v3/internal/database"
	dockerInternal "getarcane.dev/arcane/v3/internal/docker"
	"getarcane.dev/arcane/v3/internal/event"
	"getarcane.dev/arcane/v3/pkg/handlerutil"
	"getarcane.dev/arcane/v3/pkg/httpx"
	"getarcane.dev/arcane/v3/pkg/pagination"
	"getarcane.dev/types/v3"
)

type (
	NetworkService struct {
		db            *database.DB
		dockerService *dockerInternal.DockerClientService
		eventService  *event.EventService

		// Shared, immutable topology build; network mutations bump topologyGen,
		// which retires cached entries and in-flight builds started before them.
		topologyFlight singleflight.Group
		topologyCache  atomic.Pointer[networkTopologyCacheEntry]
		topologyGen    atomic.Uint64
	}

	networkTopologyCacheEntry struct {
		topology *types.NetworkTopology
		builtAt  time.Time
		gen      uint64
	}
)

const (
	networkTopologyCacheTTL     = 5 * time.Second
	networkTopologyBuildTimeout = 30 * time.Second
)

func NewNetworkService(db *database.DB, dockerService *dockerInternal.DockerClientService, eventService *event.EventService) *NetworkService {
	return &NetworkService{
		db:            db,
		dockerService: dockerService,
		eventService:  eventService,
	}
}

func (s *NetworkService) GetNetworkByID(ctx context.Context, id string, order pagination.SortParams) (*types.NetworkInspect, error) {
	inspected, err := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkInspectResult, error) {
		return compat.NetworkInspectWithCompatibility(ctx, dockerClient, id, client.NetworkInspectOptions{Verbose: true})
	})
	if err != nil {
		return nil, fmt.Errorf("network not found: %w", err)
	}
	out, err := mapping.MapOne[network.Inspect, types.NetworkInspect](inspected.Network)
	if err != nil {
		return nil, fmt.Errorf("map network: %w", err)
	}
	info, infoErr := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.SystemInfoResult, error) {
		return dockerClient.Info(ctx, client.InfoOptions{})
	})
	if infoErr != nil {
		slog.WarnContext(ctx, "network cluster identity unavailable", "error", infoErr)
	}
	out.ID = cmp.Or(out.ID, inspected.Network.ID)           // keep: enriches the mapped DTO
	out.ClusterID = kit.FromPtr(info.Info.Swarm.Cluster).ID // keep: enriches the mapped DTO
	out.NodeID = info.Info.Swarm.NodeID                     // keep: enriches the mapped DTO
	for containerID, endpoint := range networkContainerEndpoints(out.Containers) {
		out.ContainersList = append(out.ContainersList, types.NetworkContainerEndpoint{
			ID: containerID, Name: endpoint.Name, EndpointID: endpoint.EndpointID,
			IPv4Address: kit.Ternary(endpoint.IPv4Address.IsValid(), endpoint.IPv4Address.String(), ""),
			IPv6Address: kit.Ternary(endpoint.IPv6Address.IsValid(), endpoint.IPv6Address.String(), ""),
			MacAddress:  endpoint.MacAddress.String(),
		})
	}
	slices.SortFunc(out.ContainersList, func(a, b types.NetworkContainerEndpoint) int {
		if order.Sort == "ip" {
			return kit.Ternary(order.Order == pagination.SortDesc, -1, 1) * kit.CompareAddresses(cmp.Or(a.IPv4Address, a.IPv6Address), cmp.Or(b.IPv4Address, b.IPv6Address))
		}
		return kit.Ternary(order.Order == pagination.SortDesc, -1, 1) * strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return &out, nil
}

func (s *NetworkService) GetNetworkTopology(ctx context.Context) (*types.NetworkTopology, error) {
	gen := s.topologyGen.Load()
	if entry := s.topologyCache.Load(); entry != nil && entry.gen == gen && time.Since(entry.builtAt) < networkTopologyCacheTTL {
		return entry.topology, nil
	}

	// Keyed by generation so callers after a network change never join an older build.
	ch := s.topologyFlight.DoChan(strconv.FormatUint(gen, 10), func() (any, error) {
		buildCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), networkTopologyBuildTimeout)
		defer cancel()

		topology, err := s.buildTopology(buildCtx)
		if err != nil {
			return nil, err
		}
		s.topologyCache.Store(&networkTopologyCacheEntry{
			topology: topology,
			builtAt:  time.Now(),
			gen:      gen,
		})
		return topology, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		topology, ok := res.Val.(*types.NetworkTopology)
		if !ok {
			return nil, errors.New("network topology cache returned unexpected type")
		}
		return topology, nil
	}
}

func (s *NetworkService) buildTopology(ctx context.Context) (*types.NetworkTopology, error) {
	containers, err := s.dockerService.ListContainers(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	containersByID := make(map[string]container.Summary, len(containers))
	for _, rawContainer := range containers {
		containersByID[rawContainer.ID] = rawContainer
	}

	networkList, err := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkListResult, error) {
		return compat.NetworkListWithCompatibility(ctx, dockerClient, client.NetworkListOptions{})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list Docker networks: %w", err)
	}

	inspectedNetworks := make([]network.Inspect, len(networkList.Items))

	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	for i, rawNetwork := range networkList.Items {
		g.Go(func() (workerErr error) {
			defer concurrency.RecoverToError(&workerErr, "network worker")

			inspected, inspectErr := s.dockerService.Call(groupCtx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkInspectResult, error) {
				return compat.NetworkInspectWithCompatibility(ctx, dockerClient, rawNetwork.ID, client.NetworkInspectOptions{})
			})
			if errdefs.IsNotFound(inspectErr) {
				return nil
			}
			if inspectErr != nil {
				return fmt.Errorf("failed to inspect network %s: %w", rawNetwork.Name, inspectErr)
			}

			inspectedNetworks[i] = inspected.Network
			return nil
		})
	}

	if waitErr := g.Wait(); waitErr != nil {
		return nil, waitErr
	}

	nodes := make([]types.NetworkTopologyNode, 0, len(networkList.Items))
	edges := make([]types.NetworkTopologyEdge, 0)
	containerNodeIDs := make(map[string]struct{})
	for i, rawNetwork := range networkList.Items {
		if inspectedNetworks[i].ID == "" {
			continue
		}
		nodes = append(nodes, types.NetworkTopologyNode{
			ID:   rawNetwork.ID,
			Name: rawNetwork.Name,
			Type: types.NetworkTopologyNodeTypeNetwork,
			Metadata: types.NetworkTopologyNodeMetadata{
				Driver:    rawNetwork.Driver,
				Scope:     rawNetwork.Scope,
				IsDefault: docker.IsDefaultNetwork(rawNetwork.Name),
			},
		})

		for containerID, endpoint := range inspectedNetworks[i].Containers {
			if _, exists := containerNodeIDs[containerID]; !exists {
				ctr := containersByID[containerID]
				nodes = append(nodes, types.NetworkTopologyNode{
					ID:   containerID,
					Name: cmp.Or(kit.Ternary(len(ctr.Names) > 0, docker.ContainerNameFromNames(ctr.Names), ctr.ID), endpoint.Name),
					Type: types.NetworkTopologyNodeTypeContainer,
					Metadata: types.NetworkTopologyNodeMetadata{
						Status: string(ctr.State),
						Image:  ctr.Image,
					},
				})
				containerNodeIDs[containerID] = struct{}{}
			}

			edges = append(edges, types.NetworkTopologyEdge{
				ID:          fmt.Sprintf("%s:%s", rawNetwork.ID, containerID),
				Source:      rawNetwork.ID,
				Target:      containerID,
				IPv4Address: kit.Ternary(endpoint.IPv4Address.IsValid(), endpoint.IPv4Address.String(), ""),
				IPv6Address: kit.Ternary(endpoint.IPv6Address.IsValid(), endpoint.IPv6Address.String(), ""),
			})
		}
	}

	slices.SortFunc(nodes, func(a, b types.NetworkTopologyNode) int {
		return cmp.Or(cmp.Compare(a.Type, b.Type), strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)))
	})
	slices.SortFunc(edges, func(a, b types.NetworkTopologyEdge) int {
		return cmp.Or(strings.Compare(a.Source, b.Source), strings.Compare(a.Target, b.Target))
	})

	return &types.NetworkTopology{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func (s *NetworkService) CreateNetwork(ctx context.Context, name string, options client.NetworkCreateOptions, user types.Actor) (*network.CreateResponse, error) {
	if strings.TrimSpace(name) == "" {
		return nil, common.Classify(common.ErrBadRequest, errors.New("network name is required"))
	}
	defer s.eventService.BeginDockerResourceSuppressionWindow("network", "", name)()
	response, err := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkCreateResult, error) {
		return dockerClient.NetworkCreate(ctx, name, options)
	})
	if err != nil {
		s.eventService.LogErrorEvent(ctx, event.ResourceEvent{
			Type:          event.EventTypeNetworkError,
			ResourceType:  "network",
			ResourceName:  name,
			UserID:        user.ID,
			Username:      user.Username,
			EnvironmentID: "0",
			Metadata:      database.JSON{"action": "create", "driver": options.Driver},
		}, err)
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	s.topologyGen.Add(1)

	if logErr := s.eventService.LogResourceEvent(ctx, event.ResourceEvent{
		Type:          event.EventTypeNetworkCreate,
		ResourceType:  "network",
		ResourceID:    response.ID,
		ResourceName:  name,
		UserID:        user.ID,
		Username:      user.Username,
		EnvironmentID: "0",
		Metadata: database.JSON{
			"action": "create",
			"driver": options.Driver,
			"name":   name,
		},
	}); logErr != nil {
		slog.WarnContext(ctx, "could not log network creation action", "error", logErr)
	}

	return &network.CreateResponse{
		ID:      response.ID,
		Warning: strings.Join(response.Warning, "; "),
	}, nil
}

func (s *NetworkService) RemoveNetwork(ctx context.Context, id string, user types.Actor) error {
	networkInfo, err := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkInspectResult, error) {
		return compat.NetworkInspectWithCompatibility(ctx, dockerClient, id, client.NetworkInspectOptions{})
	})
	if err != nil {
		return fmt.Errorf("inspect network before removal: %w", err)
	}
	networkName := networkInfo.Network.Name
	if len(networkContainerEndpoints(networkInfo.Network.Containers)) > 0 {
		return common.Classify(common.ErrConflict, errors.New("network still has container endpoints"))
	}
	if networkInfo.Network.Scope == "swarm" {
		if networkInfo.Network.ID == "" {
			return common.Classify(common.ErrConflict, errors.New("swarm network identity is unavailable"))
		}
		info, infoErr := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.SystemInfoResult, error) {
			return dockerClient.Info(ctx, client.InfoOptions{})
		})
		if infoErr != nil {
			return fmt.Errorf("verify Swarm manager before network removal: %w", infoErr)
		}
		if !info.Info.Swarm.ControlAvailable {
			return common.Classify(common.ErrConflict, errors.New("swarm network removal requires a manager agent; node-local absence of containers is insufficient"))
		}
		services, listErr := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.ServiceListResult, error) {
			return dockerClient.ServiceList(ctx, client.ServiceListOptions{})
		})
		if listErr != nil {
			return fmt.Errorf("verify Swarm network service references: %w", listErr)
		}
		if references := networkServiceReferences(networkInfo.Network.ID, networkName, services.Items); len(references) > 0 {
			return common.Classify(common.ErrConflict, fmt.Errorf("network is referenced by Swarm services: %s", strings.Join(references, ", ")))
		}
	}

	defer s.eventService.BeginDockerResourceSuppressionWindow("network", id, networkName)()
	if _, networkRemoveErr := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkRemoveResult, error) {
		return dockerClient.NetworkRemove(ctx, id, client.NetworkRemoveOptions{})
	}); networkRemoveErr != nil {
		s.eventService.LogErrorEvent(ctx, event.ResourceEvent{
			Type:          event.EventTypeNetworkError,
			ResourceType:  "network",
			ResourceID:    id,
			ResourceName:  networkName,
			UserID:        user.ID,
			Username:      user.Username,
			EnvironmentID: "0",
			Metadata:      database.JSON{"action": "delete"},
		}, networkRemoveErr)
		return fmt.Errorf("failed to remove network: %w", networkRemoveErr)
	}
	s.topologyGen.Add(1)

	if logErr := s.eventService.LogResourceEvent(ctx, event.ResourceEvent{
		Type:          event.EventTypeNetworkDelete,
		ResourceType:  "network",
		ResourceID:    id,
		ResourceName:  networkName,
		UserID:        user.ID,
		Username:      user.Username,
		EnvironmentID: "0",
		Metadata: database.JSON{
			"action":    "delete",
			"networkId": id,
		},
	}); logErr != nil {
		slog.WarnContext(ctx, "could not log network delete action", "error", logErr)
	}

	return nil
}

func (s *NetworkService) ConnectContainer(ctx context.Context, networkID string, req types.NetworkContainerRequest, user types.Actor) error {
	ipv4, ipv4Err := netip.ParseAddr(strings.TrimSpace(req.IPv4Address))
	if ipv4Err != nil && strings.TrimSpace(req.IPv4Address) != "" {
		return common.Classify(common.ErrValidation, fmt.Errorf("invalid IPv4 address %s: %w", strings.TrimSpace(req.IPv4Address), ipv4Err))
	}
	ipv6, ipv6Err := netip.ParseAddr(strings.TrimSpace(req.IPv6Address))
	if ipv6Err != nil && strings.TrimSpace(req.IPv6Address) != "" {
		return common.Classify(common.ErrValidation, fmt.Errorf("invalid IPv6 address %s: %w", strings.TrimSpace(req.IPv6Address), ipv6Err))
	}

	if _, networkConnectErr := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkConnectResult, error) {
		return dockerClient.NetworkConnect(ctx, networkID, client.NetworkConnectOptions{
			Container: req.ContainerID,
			EndpointConfig: &network.EndpointSettings{
				Aliases: req.Aliases,
				IPAMConfig: func() *network.EndpointIPAMConfig {
					if !ipv4.IsValid() && !ipv6.IsValid() {
						return nil
					}
					return &network.EndpointIPAMConfig{
						IPv4Address: ipv4,
						IPv6Address: ipv6,
					}
				}(),
			},
		})
	}); networkConnectErr != nil {
		s.eventService.LogErrorEvent(ctx, event.ResourceEvent{
			Type:          event.EventTypeNetworkError,
			ResourceType:  "network",
			ResourceID:    networkID,
			UserID:        user.ID,
			Username:      user.Username,
			EnvironmentID: "0",
			Metadata: database.JSON{
				"action":      "connect",
				"containerId": req.ContainerID,
			},
		}, networkConnectErr)
		return fmt.Errorf("failed to connect container to network: %w", networkConnectErr)
	}
	s.topologyGen.Add(1)

	if logErr := s.eventService.LogResourceEvent(ctx, event.ResourceEvent{
		Type:          event.EventTypeNetworkConnect,
		ResourceType:  "network",
		ResourceID:    networkID,
		ResourceName:  networkID,
		UserID:        user.ID,
		Username:      user.Username,
		EnvironmentID: "0",
		Metadata: database.JSON{
			"action":      "connect",
			"networkId":   networkID,
			"containerId": req.ContainerID,
		},
	}); logErr != nil {
		slog.WarnContext(ctx, "could not log network connect action", "error", logErr)
	}

	return nil
}

func (s *NetworkService) DisconnectContainer(ctx context.Context, networkID string, req types.NetworkContainerRequest, user types.Actor) error {
	if _, networkDisconnectErr := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkDisconnectResult, error) {
		return dockerClient.NetworkDisconnect(ctx, networkID, client.NetworkDisconnectOptions{
			Container: req.ContainerID,
			Force:     req.Force,
		})
	}); networkDisconnectErr != nil {
		s.eventService.LogErrorEvent(ctx, event.ResourceEvent{
			Type:          event.EventTypeNetworkError,
			ResourceType:  "network",
			ResourceID:    networkID,
			UserID:        user.ID,
			Username:      user.Username,
			EnvironmentID: "0",
			Metadata: database.JSON{
				"action":      "disconnect",
				"containerId": req.ContainerID,
			},
		}, networkDisconnectErr)
		return fmt.Errorf("failed to disconnect container from network: %w", networkDisconnectErr)
	}
	s.topologyGen.Add(1)

	if logErr := s.eventService.LogResourceEvent(ctx, event.ResourceEvent{
		Type:          event.EventTypeNetworkDisconnect,
		ResourceType:  "network",
		ResourceID:    networkID,
		ResourceName:  networkID,
		UserID:        user.ID,
		Username:      user.Username,
		EnvironmentID: "0",
		Metadata: database.JSON{
			"action":      "disconnect",
			"networkId":   networkID,
			"containerId": req.ContainerID,
			"force":       req.Force,
		},
	}); logErr != nil {
		slog.WarnContext(ctx, "could not log network disconnect action", "error", logErr)
	}

	return nil
}

func (s *NetworkService) PruneNetworks(ctx context.Context) (*network.PruneReport, error) {
	report, err := s.dockerService.Call(ctx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkPruneResult, error) {
		return dockerClient.NetworkPrune(ctx, client.NetworkPruneOptions{Filters: make(client.Filters)})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to prune networks: %w", err)
	}
	s.topologyGen.Add(1)

	if logErr := s.eventService.LogResourceEvent(ctx, event.ResourceEvent{
		Type:          event.EventTypeNetworkDelete,
		ResourceType:  "network",
		ResourceName:  "bulk_prune",
		UserID:        types.SystemUser.ID,
		Username:      types.SystemUser.Username,
		EnvironmentID: "0",
		Metadata: database.JSON{
			"action":          "prune",
			"networksDeleted": len(report.Report.NetworksDeleted),
		},
	}); logErr != nil {
		slog.WarnContext(ctx, "could not log network prune action", "error", logErr)
	}

	return &report.Report, nil
}

func (s *NetworkService) ListNetworksPaginated(ctx context.Context, params pagination.QueryParams) ([]types.NetworkSummary, pagination.Response, types.NetworkUsageCounts, error) {
	g, groupCtx := errgroup.WithContext(ctx)
	var containers []container.Summary
	var networkList client.NetworkListResult
	var info client.SystemInfoResult
	g.Go(func() (workerErr error) {
		defer concurrency.RecoverToError(&workerErr, "network identity worker")
		var err error
		info, err = s.dockerService.Call(groupCtx, func(ctx context.Context, dockerClient *client.Client) (client.SystemInfoResult, error) {
			return dockerClient.Info(ctx, client.InfoOptions{})
		})
		if err != nil {
			slog.WarnContext(groupCtx, "network cluster identity unavailable", "error", err)
		}
		return nil
	})
	g.Go(func() (workerErr error) {
		defer concurrency.RecoverToError(&workerErr, "network container list worker")
		var err error
		containers, err = s.dockerService.ListContainers(groupCtx, false)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		return nil
	})
	g.Go(func() (workerErr error) {
		defer concurrency.RecoverToError(&workerErr, "network list worker")
		var err error
		networkList, err = s.dockerService.Call(groupCtx, func(ctx context.Context, dockerClient *client.Client) (client.NetworkListResult, error) {
			return compat.NetworkListWithCompatibility(ctx, dockerClient, client.NetworkListOptions{})
		})
		if err != nil {
			return fmt.Errorf("failed to list Docker networks: %w", err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, pagination.Response{}, types.NetworkUsageCounts{}, err
	}

	inUseByID, inUseByName := docker.BuildNetworkUsageMaps(containers)
	items := make([]types.NetworkSummary, 0, len(networkList.Items))
	inUse := 0
	for _, n := range networkList.Items {
		netDto, mapErr := mapping.MapOne[network.Summary, types.NetworkSummary](n)
		if mapErr != nil {
			return nil, pagination.Response{}, types.NetworkUsageCounts{}, fmt.Errorf("failed to map network: %w", mapErr)
		}
		netDto.ClusterID = kit.FromPtr(info.Info.Swarm.Cluster).ID
		netDto.NodeID = info.Info.Swarm.NodeID
		netDto.InUse = inUseByID[netDto.ID] || inUseByName[netDto.Name] // keep: enriches the mapped DTO
		netDto.IsDefault = docker.IsDefaultNetwork(netDto.Name)         // keep: enriches the mapped DTO
		items = append(items, netDto)
		if netDto.InUse || netDto.IsDefault {
			inUse++
		}
	}
	result := s.buildNetworkPaginationConfig().SearchOrderAndPaginate(items, params)

	return result.Items, pagination.BuildResponse(result.TotalCount, result.TotalAvailable, params), types.NetworkUsageCounts{
		Inuse:  inUse,
		Unused: len(items) - inUse,
		Total:  len(items),
	}, nil
}

func (s *NetworkService) AggregateNetworks(ctx context.Context, params pagination.QueryParams) (types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts], error) {
	collection, err := httpx.CollectEnvironments[types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts]](ctx, url.Values{
		"start": {"0"}, "limit": {"-1"}, "search": nil, "inUse": nil, "label": nil,
	})
	if err != nil {
		return types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts]{}, err
	}
	overview := mergeNetworkOverview(collection.Results)
	paginationConfig := pagination.WithEnvironment(s.buildNetworkPaginationConfig())
	result := paginationConfig.SearchOrderAndPaginate(overview.Items, params)
	if params.Limit > 0 && int64(params.Start) >= result.TotalCount {
		params.Start = int(max(result.TotalCount-1, 0)) / params.Limit * params.Limit
		result = paginationConfig.SearchOrderAndPaginate(overview.Items, params)
	}
	return types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts]{
		Success: true, Data: result.Items, Counts: overview.Counts,
		Pagination:         handlerutil.PaginationResponse(pagination.BuildResponse(result.TotalCount, result.TotalAvailable+overview.Unseen, params)),
		FailedEnvironments: collection.Failures,
	}, nil
}

func (s *NetworkService) AggregateNetworkTopology(ctx context.Context) (types.ApiResponse[types.NetworkTopology], error) {
	collection, err := httpx.CollectEnvironments[types.ApiResponse[types.NetworkTopology]](ctx, nil)
	if err != nil {
		return types.ApiResponse[types.NetworkTopology]{}, err
	}
	nodes := make([]types.NetworkTopologyNode, 0)
	edges := make([]types.NetworkTopologyEdge, 0)
	for _, result := range collection.Results {
		prefix := result.Environment.ID + "/"
		for _, node := range result.Value.Data.Nodes {
			nodes = append(nodes, types.NetworkTopologyNode{
				EnvironmentID:   result.Environment.ID,
				EnvironmentName: result.Environment.Name,
				ID:              prefix + node.ID,
				Name:            node.Name,
				Type:            node.Type,
				Metadata:        node.Metadata,
			})
		}
		for _, edge := range result.Value.Data.Edges {
			edges = append(edges, types.NetworkTopologyEdge{
				ID:          prefix + edge.ID,
				Source:      prefix + edge.Source,
				Target:      prefix + edge.Target,
				IPv4Address: edge.IPv4Address,
				IPv6Address: edge.IPv6Address,
			})
		}
	}
	return types.ApiResponse[types.NetworkTopology]{
		Success: true,
		Data: types.NetworkTopology{
			Nodes: nodes,
			Edges: edges,
		},
		FailedEnvironments: collection.Failures,
	}, nil
}

func (s *NetworkService) AggregateNetworkCounts(ctx context.Context) (types.ApiResponse[types.NetworkUsageCounts], error) {
	scope, ok := httpx.EnvironmentScopeFromContext(ctx)
	if !ok {
		return types.ApiResponse[types.NetworkUsageCounts]{}, errors.New("request has no environment scope")
	}
	listScope := *scope
	listScope.Path = "/networks"
	collection, err := httpx.CollectEnvironments[types.PaginatedWithCounts[types.NetworkSummary, types.NetworkUsageCounts]](
		httpx.WithEnvironmentScope(ctx, &listScope), url.Values{"start": {"0"}, "limit": {"-1"}, "search": nil, "inUse": nil, "label": nil},
	)
	if err != nil {
		return types.ApiResponse[types.NetworkUsageCounts]{}, err
	}
	overview := mergeNetworkOverview(collection.Results)
	return types.ApiResponse[types.NetworkUsageCounts]{Success: true, Data: overview.Counts, Environments: overview.Environments, FailedEnvironments: collection.Failures}, nil
}

func (s *NetworkService) buildNetworkPaginationConfig() pagination.Config[types.NetworkSummary] {
	return pagination.Config[types.NetworkSummary]{
		SearchAccessors: []pagination.SearchAccessor[types.NetworkSummary]{
			func(n types.NetworkSummary) (string, error) { return n.Name, nil },
			func(n types.NetworkSummary) (string, error) { return n.Driver, nil },
			func(n types.NetworkSummary) (string, error) { return n.Scope, nil },
			func(n types.NetworkSummary) (string, error) { return n.ID, nil },
			func(n types.NetworkSummary) (string, error) {
				subnets, gateways := ipamValues(n)
				return strings.Join(append(subnets, gateways...), " "), nil
			},
		},
		SortBindings: []pagination.SortBinding[types.NetworkSummary]{
			{
				Key: "name",
				// Docker allows duplicate network names, so tie-break on ID to keep
				// pagination deterministic.
				Fn: func(a, b types.NetworkSummary) int {
					return cmp.Or(strings.Compare(a.Name, b.Name), strings.Compare(a.ID, b.ID))
				},
			},
			{
				Key: "driver",
				Fn:  func(a, b types.NetworkSummary) int { return strings.Compare(a.Driver, b.Driver) },
			},
			{
				Key: "scope",
				Fn:  func(a, b types.NetworkSummary) int { return strings.Compare(a.Scope, b.Scope) },
			},
			{
				Key: "created",
				Fn:  func(a, b types.NetworkSummary) int { return a.Created.Compare(b.Created) },
			},
			{
				Key: "inUse",
				Fn: func(a, b types.NetworkSummary) int {
					if (a.InUse || a.IsDefault) == (b.InUse || b.IsDefault) {
						return strings.Compare(a.Name, b.Name)
					}
					return kit.Ternary(a.InUse || a.IsDefault, -1, 1)
				},
			},
			// Subnet and gateway compare sorted copies so IPAM entry order does
			// not affect row order; the displayed entries are untouched.
			{
				Key: "subnet",
				Fn: func(a, b types.NetworkSummary) int {
					subnetsA, _ := ipamValues(a)
					subnetsB, _ := ipamValues(b)
					return slices.CompareFunc(
						slices.SortedFunc(slices.Values(subnetsA), kit.CompareSubnets),
						slices.SortedFunc(slices.Values(subnetsB), kit.CompareSubnets),
						kit.CompareSubnets,
					)
				},
			},
			{
				Key: "gateway",
				Fn: func(a, b types.NetworkSummary) int {
					_, gatewaysA := ipamValues(a)
					_, gatewaysB := ipamValues(b)
					return slices.CompareFunc(
						slices.SortedFunc(slices.Values(gatewaysA), kit.CompareAddresses),
						slices.SortedFunc(slices.Values(gatewaysB), kit.CompareAddresses),
						kit.CompareAddresses,
					)
				},
			},
		},
		FilterAccessors: []pagination.FilterAccessor[types.NetworkSummary]{
			{
				Key: "inUse",
				Fn: func(n types.NetworkSummary, filterValue string) bool {
					return kit.Ternary(filterValue == "true", n.InUse || n.IsDefault, filterValue != "false" || (!n.InUse && !n.IsDefault))
				},
			},
			pagination.LabelFilter(func(n types.NetworkSummary) map[string]string {
				return n.Labels
			}),
		},
	}
}

func ipamValues(n types.NetworkSummary) (subnets, gateways []string) {
	for _, cfg := range n.IPAM.Config {
		if cfg.Subnet != "" {
			subnets = append(subnets, cfg.Subnet)
		}
		if cfg.Gateway != "" {
			gateways = append(gateways, cfg.Gateway)
		}
	}
	return subnets, gateways
}
