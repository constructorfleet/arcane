<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import { KeyValueCard } from '#lib/components/resource-detail/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import { systemService } from '#lib/services/system-service.js';
	import type { EnvironmentRef } from '#lib/stores/environment.store.svelte.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref, readableEnvironments, readEachEnvironment } from '#lib/utils/environments.js';
	import { bytes } from '#lib/utils/formatting.js';

	import { hasConnectedAgent, hasReplicaShortfall, listClusterResources, summarizeNodes } from '../cluster-overview';

	let { clusterId, managers, revision }: { clusterId: string; managers: EnvironmentRef[]; revision: number } = $props();
	const nodeManager = $derived(managers.find((manager) => hasPermission('swarm-nodes:list', manager.id)));
	const serviceManager = $derived(managers.find((manager) => hasPermission('swarm-services:list', manager.id)));
	const nodesQuery = createQuery(() => ({
		queryKey: ['swarm', nodeManager?.id, 'cluster-overview-nodes', revision],
		queryFn: () => listClusterResources((request) => swarmService.getNodes(request, nodeManager!.id), 'hostname'),
		enabled: !!nodeManager
	}));
	const servicesQuery = createQuery(() => ({
		queryKey: ['swarm', serviceManager?.id, 'cluster-overview-services', revision],
		queryFn: () => listClusterResources((request) => swarmService.getServices(request, serviceManager!.id), 'name'),
		enabled: !!serviceManager
	}));
	const nodes = $derived(nodesQuery.isError ? undefined : nodesQuery.data);
	const services = $derived(servicesQuery.isError ? undefined : servicesQuery.data);
	const summary = $derived(nodes ? summarizeNodes(nodes) : undefined);
	const shortfalls = $derived(services?.filter(hasReplicaShortfall) ?? []);
	const connectedAgentIds = $derived(
		new Set(
			(nodes ?? []).flatMap((node) =>
				hasConnectedAgent(node) &&
				(node.agent.bindingKind === 'environment' || node.agent.bindingKind === 'discovered') &&
				node.agent.environmentId
					? [node.agent.environmentId]
					: []
			)
		)
	);
	const diskEnvironments = $derived(readableEnvironments('system:read').filter((env) => connectedAgentIds.has(env.id)));
	const diskQuery = createQuery(() => ({
		queryKey: ['swarm', clusterId, 'cluster-overview-disk', diskEnvironments.map((env) => env.id), revision],
		queryFn: () =>
			readEachEnvironment(diskEnvironments, (id, config) => systemService.getDiskUsage(id, config), {
				unusedImagesBytes: 0,
				buildCacheBytes: 0,
				unusedVolumesBytes: 0,
				stoppedContainersBytes: 0
			}),
		enabled: diskEnvironments.length > 0
	}));
	const diskUsage = $derived(!diskQuery.isError ? (diskQuery.data?.byEnvironment ?? []) : []);
	const reclaimable = $derived(
		diskUsage.reduce((total, { data }) => total + data.unusedImagesBytes + data.buildCacheBytes + data.stoppedContainersBytes, 0)
	);
	const affectedNodes = $derived(new Set([...(summary?.unhealthy ?? []), ...(summary?.agentGaps ?? [])]));

	function nodeHref(nodeId: string) {
		if (!nodeManager || !hasPermission('swarm-nodes:read', nodeManager.id)) return undefined;
		return environmentHref(resolve('/(app)/swarm/nodes/[nodeId]', { nodeId }), nodeManager.id);
	}

	function serviceHref(serviceId: string) {
		if (!serviceManager || !hasPermission('swarm-services:read', serviceManager.id)) return undefined;
		return environmentHref(resolve('/(app)/swarm/services/[serviceId]', { serviceId }), serviceManager.id);
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>{m.cluster()}</Card.Title>
		<Card.Description>{m.swarm_cluster_id_label()} <span class="font-mono break-all">{clusterId}</span></Card.Description>
	</Card.Header>
	<Card.Content>
		<div class="space-y-5">
			<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
				<KeyValueCard label={m.swarm_cluster_ready_nodes()} valueFormat="text">
					{summary && nodes ? m.swarm_cluster_count_of_total({ count: summary.ready, total: nodes.length }) : m.common_na()}
				</KeyValueCard>
				<KeyValueCard label={m.swarm_cluster_active_nodes()} valueFormat="text">
					{summary && nodes ? m.swarm_cluster_count_of_total({ count: summary.active, total: nodes.length }) : m.common_na()}
				</KeyValueCard>
				<KeyValueCard label={m.swarm_cluster_agent_coverage()} valueFormat="text">
					{summary && nodes ? m.swarm_cluster_count_of_total({ count: summary.connected, total: nodes.length }) : m.common_na()}
				</KeyValueCard>
				<KeyValueCard label={m.swarm_cluster_reachable_managers()} valueFormat="text">
					{summary ? m.swarm_cluster_count_of_total({ count: summary.reachable, total: summary.managers }) : m.common_na()}
					{#if summary?.quorum}
						<span class="block text-xs text-muted-foreground">{m.swarm_cluster_quorum_required({ count: summary.quorum })}</span>
					{/if}
				</KeyValueCard>
				<KeyValueCard label={m.swarm_cluster_leader()} valueFormat="text">
					{#if summary?.leader}
						{@const href = nodeHref(summary.leader.id)}
						{#if href}<a {href} class="text-primary hover:underline">{summary.leader.hostname}</a>{:else}{summary.leader
								.hostname}{/if}
					{:else}
						{m.common_na()}
					{/if}
				</KeyValueCard>
				<KeyValueCard label={m.services()} valueFormat="text">{services?.length ?? m.common_na()}</KeyValueCard>
				<KeyValueCard label={m.reclaimable_space()} valueFormat="text">
					{diskUsage.length ? bytes(reclaimable) : m.common_na()}
					{#if diskUsage.length && nodes}
						<span class="block text-xs text-muted-foreground"
							>{m.swarm_cluster_storage_coverage({ count: diskUsage.length, total: nodes.length })}</span
						>
					{/if}
				</KeyValueCard>
			</div>

			{#if (nodeManager && nodesQuery.isPending) || (serviceManager && servicesQuery.isPending) || (diskEnvironments.length && diskQuery.isPending)}
				<p class="text-sm text-muted-foreground">{m.common_loading()}</p>
			{/if}
			{#if nodesQuery.isError || servicesQuery.isError}
				<p class="text-sm text-warning">{m.swarm_cluster_incomplete()}</p>
			{/if}

			<div class="space-y-2 border-t border-border pt-4">
				<h3 class="text-sm font-semibold">{m.needs_attention()}</h3>
				{#if summary?.quorum && summary.reachable < summary.quorum}
					<p class="text-sm text-warning">{m.swarm_cluster_quorum_below_required()}</p>
				{/if}
				{#if affectedNodes.size || shortfalls.length}
					<ul class="space-y-2 text-sm">
						{#each [...affectedNodes].slice(0, 6) as node (node.id)}
							{@const href = nodeHref(node.id)}
							<li class="flex flex-wrap items-baseline gap-x-3">
								{#if href}<a {href} class="font-medium text-primary hover:underline">{node.hostname}</a>{:else}<span
										>{node.hostname}</span
									>{/if}
								{#if summary?.unhealthy.includes(node)}
									<span class="text-muted-foreground">{[node.status, node.reachability].filter(Boolean).join(' · ')}</span>
								{/if}
								{#if !hasConnectedAgent(node)}
									<span class="text-muted-foreground">{m.swarm_cluster_agent_unavailable()}</span>
								{/if}
							</li>
						{/each}
						{#each shortfalls.slice(0, 6) as service (service.id)}
							{@const href = serviceHref(service.id)}
							<li class="flex flex-wrap items-baseline gap-x-3">
								{#if href}<a {href} class="font-medium text-primary hover:underline">{service.name}</a>{:else}<span
										>{service.name}</span
									>{/if}
								<span class="text-muted-foreground"
									>{m.swarm_cluster_replica_shortfall({ running: service.runningReplicas, desired: service.replicas })}</span
								>
							</li>
						{/each}
					</ul>
					{#if affectedNodes.size > 6 || shortfalls.length > 6}
						<p class="text-xs text-muted-foreground">
							{m.swarm_cluster_attention_total({ nodes: affectedNodes.size, services: shortfalls.length })}
						</p>
					{/if}
				{:else if nodes && services}
					<p class="text-sm text-muted-foreground">{m.swarm_cluster_no_attention()}</p>
				{:else}
					<p class="text-sm text-muted-foreground">{m.common_unavailable()}</p>
				{/if}
			</div>
		</div>
	</Card.Content>
</Card.Root>
