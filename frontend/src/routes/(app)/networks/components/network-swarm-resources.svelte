<script lang="ts">
	import { page } from '$app/state';
	import { createQuery, keepPreviousData } from '@tanstack/svelte-query';
	import { untrack } from 'svelte';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import { swarmEnvironmentsQueryOptions } from '#lib/components/swarm/swarm-environments.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { networkService } from '#lib/services/network-service.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { collectAcrossEnvironments, fetchAcrossEnvironments } from '#lib/utils/environments.js';
	import { resourceTablePage } from '#lib/utils/resource-navigation.js';

	import SwarmServicesTable from '../../swarm/services/components/services-table.svelte';
	import SwarmTasksTable from '../../swarm/tasks/components/tasks-table.svelte';

	let {
		kind,
		networkId,
		clusterId,
		environmentId
	}: {
		kind: 'services' | 'tasks';
		networkId: string;
		clusterId: string;
		environmentId: string;
	} = $props();

	let requestOptions = $state<SearchPaginationSortRequest>({
		pagination: {
			page: untrack(() => resourceTablePage((page.shallow?.url ?? page.url).searchParams.get('tablePage'))),
			limit: 20
		}
	});
	const permission = $derived(kind === 'services' ? 'swarm-services:list' : 'swarm:read');
	const discovery = createQuery(() => swarmEnvironmentsQueryOptions(permission, { all: true }));
	const manager = createQuery(() => {
		const id = networkId;
		const cluster = clusterId;
		const source = environmentId;
		const inventory = discovery.data;
		return {
			queryKey: ['swarm', 'network-manager', permission, id, cluster, source, inventory?.managers.map((env) => env.id)],
			enabled: !!inventory,
			queryFn: async ({ signal }) => {
				const managers = inventory?.managers ?? [];
				if (cluster) {
					const candidates = managers.filter((env) => inventory?.clusterIds?.get(env.id) === cluster);
					return candidates.find((env) => env.id === source) ?? candidates[0] ?? null;
				}

				// Worker inspect responses can lack a cluster ID. Match the exact network on a readable manager.
				const targets = managers.filter((env) => hasPermission('networks:read', env.id));
				const { values } = await collectAcrossEnvironments(targets, (agent, config) =>
					networkService.getNetworkForEnvironment(agent, id, undefined, { ...config, signal })
				);
				return values.find(({ value }) => value.id === id && value.scope === 'swarm')?.environment ?? null;
			}
		};
	});
	const nodesRequest = { pagination: { page: 1, limit: -1 } };
	const nodes = createQuery(() => {
		const env = manager.data?.id ?? '';
		return {
			queryKey: queryKeys.swarm.nodes(env, nodesRequest),
			enabled: kind === 'tasks' && !!env && hasPermission('swarm-nodes:list', env),
			queryFn: ({ signal }) => swarmService.getNodes(nodesRequest, env, { signal })
		};
	});
	const nodeFilters = $derived(
		[...new Set((nodes.data?.data ?? []).map((node) => node.hostname).filter(Boolean))]
			.sort((a, b) => a.localeCompare(b))
			.map((name) => ({ value: name, label: name }))
	);

	const scopedOptions = $derived({
		...requestOptions,
		filters: { ...requestOptions.filters, networkId }
	});
	const servicesKey = $derived(['swarm', manager.data?.id ?? '', 'network-services', networkId] as const);
	const services = createQuery(() => {
		const env = manager.data;
		const options = $state.snapshot(scopedOptions);
		return {
			queryKey: [...servicesKey, queryKeys.swarm.services(env?.id ?? '', options)],
			enabled: kind === 'services' && !!env,
			placeholderData: keepPreviousData,
			queryFn: ({ signal }) =>
				fetchAcrossEnvironments(env ? [env] : [], options, (agent, request, config) =>
					swarmService.getServices(request, agent, { ...config, signal })
				)
		};
	});
	const tasks = createQuery(() => {
		const env = manager.data;
		const options = $state.snapshot(scopedOptions);
		return {
			queryKey: queryKeys.swarm.tasks(env?.id ?? '', '', options),
			enabled: kind === 'tasks' && !!env,
			placeholderData: keepPreviousData,
			queryFn: ({ signal }) =>
				fetchAcrossEnvironments(env ? [env] : [], options, (agent, request, config) =>
					swarmService.getTasks(request, agent, { ...config, signal })
				)
		};
	});

	const query = $derived(kind === 'services' ? services : tasks);
	const requestFailed = $derived(discovery.isError || manager.isError || query.isError);

	async function retry() {
		await discovery.refetch();
		await manager.refetch();
		if (manager.data) await query.refetch();
	}
</script>

{#if requestFailed || (manager.isSuccess && !manager.data)}
	<Alert.Root variant="warning-subtle" size="sm">
		<Alert.Description>
			{requestFailed ? m.common_action_failed() : m.swarm_cluster_no_managers()}
			<ArcaneButton action="base" size="sm" customLabel={m.common_retry()} onclick={retry} />
		</Alert.Description>
	</Alert.Root>
{:else if kind === 'services' && services.data}
	<SwarmServicesTable
		services={services.data}
		invalidateKey={servicesKey}
		bind:requestOptions
		onRetry={() => services.refetch()}
		persistKey={`arcane-network-services-${networkId}`}
		showEnvironment={false}
	/>
{:else if kind === 'tasks' && tasks.data}
	<SwarmTasksTable
		tasks={tasks.data}
		{nodeFilters}
		bind:requestOptions
		onRetry={() => tasks.refetch()}
		persistKey={`arcane-network-tasks-${networkId}`}
		showEnvironment={false}
	/>
{:else}
	<p class="p-4 text-sm text-muted-foreground" role="status">{m.common_loading()}</p>
{/if}
