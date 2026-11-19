<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { AlertIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { networkService } from '#lib/services/network-service.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import { extractApiErrorMessage } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { collectAcrossEnvironments } from '#lib/utils/environments.js';

	let { networkId, clusterId, agentIds }: { networkId: string; clusterId: string; agentIds: string[] } = $props();
	const requested = $derived([...new Set(agentIds)]);
	const agents = $derived(
		environmentStore.available.filter((agent) => requested.includes(agent.id) && hasPermission('networks:read', agent.id))
	);
	const excluded = $derived(requested.filter((id) => !agents.some((agent) => agent.id === id)));
	const observations = createQuery(() => {
		const expectedNetworkId = networkId;
		const expectedClusterId = clusterId;
		const targets = agents.map(({ id, name }) => ({ id, name }));
		return {
			queryKey: [
				...queryKeys.networks.detail(targets.map((agent) => agent.id).join(','), expectedNetworkId),
				'observations',
				expectedClusterId
			],
			queryFn: async ({ signal }) => {
				await environmentStore.ready;
				return collectAcrossEnvironments(targets, async (agentId, config) => {
					const network = await networkService.getNetworkForEnvironment(agentId, expectedNetworkId, undefined, {
						...config,
						signal,
						timeout: 15_000
					});
					if (
						network.id !== expectedNetworkId ||
						network.clusterId !== expectedClusterId ||
						!network.nodeId ||
						network.scope !== 'swarm' ||
						network.driver !== 'overlay'
					) {
						throw new Error(m.network_identity_unavailable());
					}
					return network;
				});
			},
			retry: false
		};
	});
</script>

<section class="space-y-3" aria-label={m.nodes()}>
	<h2 class="text-lg font-semibold">{m.nodes()}</h2>
	<p class="font-mono text-xs break-all text-muted-foreground">{m.swarm_cluster_id_label()}: {clusterId}</p>
	<Alert.Root variant="warning-subtle" size="sm" icon={AlertIcon}>
		<Alert.Description>{m.network_observation_scope()}</Alert.Description>
	</Alert.Root>
	{#if excluded.length > 0}
		<Alert.Root variant="warning-subtle" size="sm" icon={AlertIcon}>
			<Alert.Description>{m.common_access_denied()}: {excluded.join(', ')}</Alert.Description>
		</Alert.Root>
	{/if}
	{#if observations.isPending}
		<p class="text-sm text-muted-foreground">{m.common_loading()}</p>
	{:else if observations.error}
		<Alert.Root variant="warning-subtle" size="sm" icon={AlertIcon}>
			<Alert.Description>{extractApiErrorMessage(observations.error)}</Alert.Description>
			<ArcaneButton action="refresh" customLabel={m.common_retry()} onclick={() => observations.refetch()} />
		</Alert.Root>
	{:else if observations.data}
		{#if observations.data.failures.length > 0}
			<Alert.Root variant="warning-subtle" size="sm" icon={AlertIcon}>
				<Alert.Description
					>{m.all_environments_unavailable({
						environments: observations.data.failures
							.map(({ environment, message }) => `${environment.name} (${message})`)
							.join(', ')
					})}</Alert.Description
				>
				<ArcaneButton action="refresh" customLabel={m.common_retry()} onclick={() => observations.refetch()} />
			</Alert.Root>
		{/if}
		{#each observations.data.values as { environment, value: network } (environment.id)}
			<Card.Root>
				<Card.Header>
					<Card.Title>{environment.name}</Card.Title>
					<Card.Description><span class="font-mono break-all">{environment.id} · {network.nodeId}</span></Card.Description>
				</Card.Header>
				<Card.Content>
					<div class="space-y-3">
						<div>
							<h3 class="text-sm font-medium">{m.networks_connected_containers_title()}</h3>
							<p class="text-xs text-muted-foreground">
								{m.networks_connected_containers_description({ count: network.containersList.length })}
							</p>
							<ul class="space-y-1 text-sm">
								{#each network.containersList as container (container.id)}
									<li>
										<a
											class="text-primary-text hover:underline"
											href={resolve(`/(app)/containers/[containerId]?env=${encodeURIComponent(environment.id)}`, {
												containerId: container.id
											})}>{container.name || container.id}</a
										> <span class="font-mono text-xs">{container.ipv4Address || container.ipv6Address}</span>
									</li>
								{/each}
							</ul>
						</div>
						<div>
							<h3 class="text-sm font-medium">{m.networks_peers_title()}</h3>
							{#if network.peers?.length}
								<ul class="font-mono text-xs">
									{#each network.peers as peer, index (`${peer.Name}:${peer.IP}:${index}`)}<li>{peer.Name} {peer.IP}</li>{/each}
								</ul>
							{:else}<p class="text-xs text-muted-foreground">{m.none()}</p>{/if}
						</div>
					</div>
				</Card.Content>
			</Card.Root>
		{/each}
	{/if}
</section>
