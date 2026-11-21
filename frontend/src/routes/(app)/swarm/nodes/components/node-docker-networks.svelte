<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import { Button } from '#lib/components/ui/button/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { networkService } from '#lib/services/network-service.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref } from '#lib/utils/environments.js';

	let { nodeId, agentId }: { nodeId: string; agentId: string } = $props();
	let page = $state(1);
	const request = $derived<SearchPaginationSortRequest>({
		pagination: { page, limit: 10 },
		sort: { column: 'name', direction: 'asc' }
	});
	const networks = createQuery(() => ({
		queryKey: queryKeys.networks.list(agentId, request),
		queryFn: () => networkService.getNetworksForEnvironment(agentId, request)
	}));
	const canReadNetwork = $derived(hasPermission('networks:read', agentId));
	const invalidIdentity = $derived(networks.data?.data.some((network) => network.nodeId !== nodeId));
</script>

<section class="space-y-3 border-t border-border pt-4">
	<h2 class="text-sm font-semibold">{m.swarm_node_docker_networks()}</h2>
	{#if networks.isPending}
		<p class="text-sm text-muted-foreground">{m.common_loading()}</p>
	{:else if networks.isError || invalidIdentity}
		<div class="flex flex-wrap items-center gap-2">
			<p class="text-sm text-muted-foreground">{m.swarm_node_network_list_unavailable()}</p>
			<Button variant="outline" size="sm" disabled={networks.isFetching} onclick={() => networks.refetch()}>
				{m.common_retry()}
			</Button>
		</div>
	{:else}
		{#if networks.data.data.length}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>{m.common_name()}</Table.Head>
						<Table.Head>{m.common_driver()}</Table.Head>
						<Table.Head>{m.common_scope()}</Table.Head>
						<Table.Head>{m.swarm_node_network_ip_configuration()}</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each networks.data.data as network (network.id)}
						<Table.Row>
							<Table.Cell>
								{#if canReadNetwork}
									<a
										class="font-medium text-primary hover:underline"
										href={environmentHref(resolve('/(app)/networks/[networkId]', { networkId: network.id }), agentId)}
										>{network.name}</a
									>
								{:else}
									{network.name}
								{/if}
							</Table.Cell>
							<Table.Cell>{network.driver}</Table.Cell>
							<Table.Cell>{network.scope}</Table.Cell>
							<Table.Cell>
								{#if network.ipam?.config?.length}
									<ul class="space-y-1">
										{#each network.ipam.config as config}
											<li class="space-y-0.5 font-mono text-xs">
												{#if config.subnet}
													<div>{m.common_subnet()} {config.subnet}</div>
												{/if}
												{#if config.gateway}
													<div>{m.common_gateway()} {config.gateway}</div>
												{/if}
												{#if config.ipRange}
													<div>{m.networks_ipam_iprange_label()} {config.ipRange}</div>
												{/if}
												{#each Object.entries(config.auxAddress ?? {}) as [label, address] (label)}
													<div>{label} {address}</div>
												{/each}
											</li>
										{/each}
									</ul>
								{:else}
									{m.common_na()}
								{/if}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{:else}
			<p class="text-sm text-muted-foreground">{m.no_networks_yet()}</p>
		{/if}
		{#if page > 1 || networks.data.pagination.totalPages > 1}
			<div class="flex flex-wrap items-center gap-2">
				<Button variant="outline" size="sm" disabled={page === 1} onclick={() => (page -= 1)}>{m.common_previous()}</Button>
				<span class="text-sm text-muted-foreground">{m.common_page({ page })}</span>
				<Button variant="outline" size="sm" disabled={page >= networks.data.pagination.totalPages} onclick={() => (page += 1)}
					>{m.common_next()}</Button
				>
			</div>
		{/if}
	{/if}
</section>
