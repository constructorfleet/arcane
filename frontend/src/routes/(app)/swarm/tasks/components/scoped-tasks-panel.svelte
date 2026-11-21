<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { untrack } from 'svelte';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import type { SwarmTasksPaginatedResponse } from '#lib/types/ui/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentName, environmentRows, type EnvironmentPage } from '#lib/utils/environments.js';
	import { resourceTablePage } from '#lib/utils/resource-navigation.js';

	import SwarmTasksTable from './tasks-table.svelte';

	let { environmentId, resourceId, scope }: { environmentId: string; resourceId: string; scope: 'node' | 'service' } = $props();
	let requestOptions = $state<SearchPaginationSortRequest>({
		pagination: {
			page: untrack(() => resourceTablePage((page.shallow?.url ?? page.url).searchParams.get('tablePage'))),
			limit: 20
		},
		sort: { column: 'updated', direction: 'desc' }
	});
	const nodeRequest: SearchPaginationSortRequest = { pagination: { page: 1, limit: -1 } };
	const nodesQuery = createQuery(() => {
		const agent = environmentId;
		return {
			queryKey: queryKeys.swarm.nodes(agent, nodeRequest),
			queryFn: () => swarmService.getNodes(nodeRequest, agent),
			enabled: scope === 'service' && !!agent && hasPermission('swarm-nodes:read', agent)
		};
	});
	const nodeFilters = $derived(
		[...new Set((nodesQuery.data?.data ?? []).map((node) => node.hostname).filter(Boolean))]
			.sort((a, b) => a.localeCompare(b))
			.map((name) => ({ value: name, label: name }))
	);
	const tasksQuery = createQuery(() => {
		const agent = environmentId;
		const id = resourceId;
		const kind = scope;
		const options = $state.snapshot(requestOptions);
		return {
			queryKey:
				kind === 'node' ? queryKeys.swarm.nodeTasks(agent, id, options) : queryKeys.swarm.serviceTasks(agent, id, options),
			queryFn: ({ signal }) =>
				kind === 'node'
					? swarmService.getNodeTasks(id, options, agent, { signal })
					: swarmService.getServiceTasks(id, options, agent),
			enabled: !!id && hasPermission(kind === 'node' ? 'swarm-nodes:read' : 'swarm-services:read', agent)
		};
	});
	const result = $derived(tasksQuery.data);
	const tasks: EnvironmentPage<SwarmTasksPaginatedResponse> = $derived({
		data: environmentRows(result?.data ?? [], [{ id: environmentId, name: environmentName(environmentId) }]),
		pagination: result?.pagination ?? {
			totalItems: 0,
			totalPages: 1,
			currentPage: requestOptions.pagination?.page ?? 1,
			itemsPerPage: requestOptions.pagination?.limit ?? 20
		}
	});
</script>

<div class="flex flex-col gap-3">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<h2 class="text-sm font-semibold">
			{#if result}
				{scope === 'service'
					? m.swarm_service_tasks_count({ count: result.pagination.grandTotalItems ?? result.pagination.totalItems })
					: m.tasks()}
			{:else}
				{m.tasks()}
			{/if}
		</h2>
		<ArcaneButton action="refresh" onclick={() => tasksQuery.refetch()} disabled={tasksQuery.isFetching}>
			{m.common_refresh()}
		</ArcaneButton>
	</div>
	<SwarmTasksTable
		{tasks}
		bind:requestOptions
		showEnvironment={false}
		serviceScoped={scope === 'service'}
		{nodeFilters}
		persistKey={`arcane-swarm-${scope}-tasks-table`}
		loading={tasksQuery.isFetching && !result}
		refreshError={tasksQuery.error}
		onRetry={() => tasksQuery.refetch()}
	/>
</div>
