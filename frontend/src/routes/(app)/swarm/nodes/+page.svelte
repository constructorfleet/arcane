<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { untrack } from 'svelte';

	import { swarmEnvironmentsQueryOptions } from '#lib/components/swarm/swarm-environments.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { AlertIcon, UsersIcon } from '#lib/icons/index.js';
	import { ResourcePageLayout, type StatCardConfig } from '#lib/layouts/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { swarmNodeListQuery } from '#lib/query/swarm.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import { handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { createRefreshActionButtons } from '#lib/utils/resource-actions.js';
	import { ResourceListPageState } from '#lib/utils/resource-list-page.svelte.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	import SwarmNodesTable from './components/nodes-table.svelte';

	let { data } = $props();

	const pageState = new ResourceListPageState<SearchPaginationSortRequest>(untrack(() => data.requestOptions));
	const swarmQuery = createQuery(() => swarmEnvironmentsQueryOptions('swarm-nodes:list'));
	const swarm = $derived(swarmQuery.data ?? data.swarm);

	const nodesQuery = createQuery(() => {
		const scope = swarm.scope;
		return {
			...swarmNodeListQuery(swarm, pageState.requestOptions),
			placeholderData: (previous, query) => (query?.queryKey[1] === scope ? previous : undefined),
			select: (value) => ({ envId: scope, value })
		};
	});
	const nodes = $derived(nodesQuery.data?.value ?? data.nodes);
	const isRefreshing = $derived(nodesQuery.isFetching && !nodesQuery.isPending);

	async function refresh() {
		await swarmQuery.refetch();
		await handleApiResultWithCallbacks({
			result: await tryCatch(nodesQuery.refetch({ throwOnError: true })),
			message: m.common_refresh_failed({ resource: m.nodes() })
		});
	}

	const totalNodes = $derived(nodes?.pagination?.totalItems ?? nodes?.data?.length ?? 0);
	const uncoveredNodes = $derived(
		(nodes?.data ?? []).filter((node) => node.agent?.state !== 'connected' || node.agent?.connected === false)
	);
	const uncoveredNodeCount = $derived(uncoveredNodes.length);

	const actionButtons = $derived(
		createRefreshActionButtons({
			refreshLabel: m.common_refresh(),
			onRefresh: refresh,
			refreshing: isRefreshing
		})
	);

	const statCards: StatCardConfig[] = $derived([
		{
			title: m.swarm_nodes_total(),
			value: totalNodes,
			icon: UsersIcon,
			iconColor: 'text-info'
		}
	]);
</script>

<ResourcePageLayout title={m.nodes()} subtitle={m.swarm_nodes_subtitle()} {actionButtons} {statCards}>
	{#snippet mainContent()}
		<div class="space-y-4">
			{#if uncoveredNodeCount > 0}
				<Alert.Root variant="warning-subtle">
					<AlertIcon class="size-4 text-warning" />
					<Alert.Title>{m.swarm_node_agent_warning_title()}</Alert.Title>
					<Alert.Description>{m.swarm_node_agent_warning_description({ count: uncoveredNodeCount })}</Alert.Description>
				</Alert.Root>
			{/if}

			<SwarmNodesTable
				{nodes}
				scope={swarm.scope}
				bind:requestOptions={pageState.requestOptions}
				onRetry={() => nodesQuery.refetch()}
			/>
		</div>
	{/snippet}
</ResourcePageLayout>
