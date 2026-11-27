<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';

	import { DetailSection, KeyValueCard, KeyValueGrid } from '#lib/components/resource-detail/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { AlertIcon, GlobeIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { volumeListQuery } from '#lib/query/volumes.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import type { Volume } from '#lib/types/volume.js';
	import { extractApiErrorMessage } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { rowKey } from '#lib/utils/environments.js';

	let { volume, environmentId }: { volume: Volume; environmentId: string } = $props();
	const eligible = $derived(volume.scope === 'global');
	const agents = $derived(
		environmentStore.available.filter(
			(agent) => agent.id !== environmentId && agent.enabled && hasPermission('volumes:list', agent.id)
		)
	);
	const recordsQuery = createQuery(() => ({
		...volumeListQuery(agents.map((agent) => agent.id).join(','), agents, {
			search: volume.name,
			pagination: { page: 1, limit: -1 }
		}),
		enabled: eligible && !volume.nodes?.length && agents.length > 0,
		retry: false,
		refetchOnWindowFocus: false,
		refetchOnReconnect: false
	}));
	const currentRecord = $derived({
		...volume,
		environmentId,
		environmentName: environmentStore.available.find((agent) => agent.id === environmentId)?.name ?? environmentId
	});
	const observations = $derived.by(() => {
		if (volume.nodes?.length || !eligible) return [currentRecord];
		return [currentRecord, ...(recordsQuery.data?.data ?? [])];
	});
	const records = $derived(
		observations
			.flatMap((record) =>
				record.nodes?.length
					? record.nodes.map((node) => ({
							...record,
							logicalId: undefined,
							nodes: undefined,
							id: node.volumeName,
							name: node.volumeName,
							environmentId: node.id,
							environmentName: node.name,
							mountpoint: node.mountpoint,
							containers: node.containers,
							inUse: node.inUse,
							activeConsumers: node.activeConsumers,
							runningHelpers: node.runningHelpers
						}))
					: [record]
			)
			.filter(
				(record) =>
					record.name === volume.name &&
					record.driver === volume.driver &&
					(!volume.clusterId || record.clusterId === volume.clusterId) &&
					(record.environmentId === environmentId || agents.some((agent) => agent.id === record.environmentId)) &&
					!!record.mountpoint &&
					((record.activeConsumers ?? 0) > 0 || (record.runningHelpers ?? 0) > 0)
			)
			.toSorted((a, b) => a.environmentName.localeCompare(b.environmentName))
	);
</script>

<DetailSection title={m.active_mount_information()} icon={GlobeIcon}>
	{#each records as record (rowKey(record))}
		<KeyValueGrid>
			<KeyValueCard label={m.swarm_node()}>{record.environmentName}</KeyValueCard>
			<KeyValueCard label={m.common_mountpoint()}>{record.mountpoint}</KeyValueCard>
		</KeyValueGrid>
	{:else}
		{#if !eligible || volume.nodes?.length || agents.length === 0 || recordsQuery.isSuccess}
			<p class="text-sm text-muted-foreground">{m.no_active_mounts()}</p>
		{/if}
	{/each}
	{#if eligible && !volume.nodes?.length && agents.length > 0}
		{#if recordsQuery.isPending}
			<p class="text-sm text-muted-foreground" role="status">{m.common_loading()}</p>
		{:else if recordsQuery.error}
			<Alert.Root variant="warning-subtle" icon={AlertIcon}>
				<Alert.Description
					>{m.volume_agent_records_error({ message: extractApiErrorMessage(recordsQuery.error) })}</Alert.Description
				>
			</Alert.Root>
		{/if}
		{#if recordsQuery.data?.failedEnvironments?.length}
			<Alert.Root variant="warning-subtle" icon={AlertIcon}>
				<Alert.Description
					>{m.all_environments_unavailable({
						environments: recordsQuery.data.failedEnvironments
							.map(({ environment, message }) => environment.name + ' (' + message + ')')
							.join(', ')
					})}</Alert.Description
				>
			</Alert.Root>
		{/if}
	{/if}
</DetailSection>
