<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import { swarmManagerQueryOptions } from '#lib/components/swarm/swarm-environments.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { AlertIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { containerService } from '#lib/services/container-service.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { collectAcrossEnvironments, fetchAcrossEnvironments } from '#lib/utils/environments.js';
	import { consumerMatchesResource, verifiedConsumerServices } from '#lib/utils/resource-consumers.js';

	import ContainerTable from '../../routes/(app)/containers/components/container-table.svelte';
	import ServicesTable from '../../routes/(app)/swarm/services/components/services-table.svelte';

	let {
		kind,
		sources
	}: {
		kind: 'containers' | 'services';
		sources: { environmentId: string; environmentName?: string; imageId?: string; volumeName?: string }[];
	} = $props();
	const queryClient = useQueryClient();
	let selectedIds = $state<string[]>([]);
	let requestOptions = $state<SearchPaginationSortRequest>({ pagination: { page: 1, limit: 20 } });
	const targets = $derived(sources.filter((source) => hasPermission('containers:list', source.environmentId)));
	const environments = $derived(
		targets.map((copy) => ({ id: copy.environmentId, name: copy.environmentName ?? copy.environmentId }))
	);
	const identity = $derived(
		targets.map((copy) => `${copy.environmentId}:${copy.imageId ?? ''}:${copy.volumeName ?? ''}`).join(',')
	);
	const runningQuery = () => ({
		queryKey: ['resources', 'running-consumers', identity],
		queryFn: async () =>
			collectAcrossEnvironments(environments, async (id, config) => {
				const matching = targets.filter((copy) => copy.environmentId === id);
				if (matching.length !== 1) throw new Error(m.common_unavailable());
				const copy = matching[0]!;
				const result = await containerService.getContainersForEnvironment(
					id,
					{
						pagination: { page: 1, limit: -1 },
						filters: {
							...(copy.imageId ? { imageId: copy.imageId } : {}),
							...(copy.volumeName ? { volumeName: copy.volumeName } : {}),
							status: 'running'
						}
					},
					config
				);
				if (result.data.some((container) => container.state !== 'running' || !consumerMatchesResource(container, copy)))
					throw new Error(m.common_unavailable());
				return result.data;
			})
	});
	const containers = createQuery(() => {
		const options = $state.snapshot(requestOptions);
		return {
			queryKey: ['resources', 'scoped-containers', identity, options],
			enabled: kind === 'containers',
			queryFn: () =>
				fetchAcrossEnvironments(environments, options, async (id, request, config) => {
					const matching = targets.filter((copy) => copy.environmentId === id);
					if (matching.length !== 1) throw new Error(m.common_unavailable());
					const copy = matching[0]!;
					const result = await containerService.getContainersForEnvironment(
						id,
						{
							...request,
							filters: {
								...request.filters,
								...(copy.imageId ? { imageId: copy.imageId, status: 'running' } : {}),
								...(copy.volumeName ? { volumeName: copy.volumeName } : {})
							}
						},
						config
					);
					if (
						result.data.some(
							(container) => !consumerMatchesResource(container, copy) || (copy.imageId && container.state !== 'running')
						)
					)
						throw new Error(m.common_unavailable());
					return result;
				})
		};
	});
	const servicesKey = $derived(['resources', 'scoped-services', identity]);
	const services = createQuery(() => {
		const options = $state.snapshot(requestOptions);
		return {
			queryKey: [...servicesKey, options],
			enabled: kind === 'services',
			queryFn: async () => {
				const consumers = await queryClient.fetchQuery(runningQuery());
				if (consumers.failures.length > 0) throw new Error(consumers.failures.map((failure) => failure.message).join('\n'));
				const managers = new Map<string, { id: string; name: string; serviceIds: Set<string> }>();
				for (const observation of consumers.values) {
					const swarmContainers = observation.value.filter((container) => container.labels['com.docker.swarm.service.id']);
					if (swarmContainers.length === 0) continue;
					const nodeIds = new Set(swarmContainers.map((container) => container.labels['com.docker.swarm.node.id']));
					if (nodeIds.size !== 1 || ![...nodeIds][0]) throw new Error(m.common_unavailable());
					const manager = await queryClient.fetchQuery(
						swarmManagerQueryOptions({
							environmentId: observation.environment.id,
							nodeId: [...nodeIds][0],
							permission: 'swarm-services:list'
						})
					);
					if (!manager) throw new Error(m.common_unavailable());
					const resolved = managers.get(manager.clusterId) ?? {
						id: manager.id,
						name: manager.name,
						serviceIds: new Set<string>()
					};
					const tasks = await queryClient.fetchQuery({
						queryKey: ['swarm', manager.clusterId, 'resource-consumer-tasks', identity],
						staleTime: 30_000,
						queryFn: () => swarmService.getTasks({ pagination: { page: 1, limit: -1 } }, resolved.id)
					});
					const verified = verifiedConsumerServices({ containers: swarmContainers, tasks: tasks.data });
					if (verified.incomplete) throw new Error(m.common_unavailable());
					for (const serviceId of verified.serviceIds) resolved.serviceIds.add(serviceId);
					if (resolved.serviceIds.size > 0) managers.set(manager.clusterId, resolved);
				}
				const coordinators = [...managers.values()];
				return fetchAcrossEnvironments(coordinators, options, async (id, request, config) => {
					const serviceIds = coordinators.find((manager) => manager.id === id)!.serviceIds;
					const result = await swarmService.getServices(
						{ ...request, filters: { ...request.filters, serviceIds: [...serviceIds].join(',') } },
						id,
						config
					);
					if (result.data.some((service) => !serviceIds.has(service.id)) || result.pagination.totalItems > serviceIds.size)
						throw new Error(m.common_unavailable());
					return result;
				});
			}
		};
	});
	const excludedSources = $derived(sources.filter((source) => !hasPermission('containers:list', source.environmentId)));
	const failure = $derived(kind === 'containers' ? containers.error : services.error);
</script>

<div class="space-y-4">
	{#if excludedSources.length > 0}<Alert.Root variant="warning-subtle" icon={AlertIcon}
			><Alert.Description
				>{m.common_access_denied()}: {excludedSources
					.map((source) => source.environmentName ?? source.environmentId)
					.join(', ')}</Alert.Description
			></Alert.Root
		>{/if}
	{#if failure}
		<Alert.Root variant="warning-subtle" icon={AlertIcon}
			><Alert.Description
				>{failure.message}<ArcaneButton
					action="refresh"
					size="sm"
					customLabel={m.common_retry()}
					onclick={() => (kind === 'containers' ? containers.refetch() : services.refetch())}
				/></Alert.Description
			></Alert.Root
		>
	{:else if kind === 'containers' && containers.data}
		<ContainerTable
			withoutFilters={sources.every((source) => !!source.imageId)}
			scope={environments.map((environment) => environment.id).join(',')}
			invalidateKey={['resources', 'scoped-containers', identity]}
			containers={containers.data}
			bind:selectedIds
			bind:requestOptions
			onRetry={() => containers.refetch()}
		/>
	{:else if kind === 'services' && services.data}
		<ServicesTable
			services={services.data}
			invalidateKey={servicesKey}
			bind:requestOptions
			onRetry={() => services.refetch()}
			persistKey="arcane-resource-consumer-services"
		/>
	{:else}<p class="p-4 text-sm text-muted-foreground" role="status">{m.common_loading()}</p>{/if}
</div>
