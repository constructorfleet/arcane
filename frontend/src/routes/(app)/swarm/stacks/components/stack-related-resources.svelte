<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import RelatedResourceTable from '#lib/components/resource-detail/related-resource-table.svelte';
	import { swarmManagerQueryOptions } from '#lib/components/swarm/swarm-environments.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { networkService } from '#lib/services/network-service.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import { volumeService } from '#lib/services/volume-service.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import type { SwarmServiceSummary } from '#lib/types/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { fetchAcrossEnvironments, environmentHref } from '#lib/utils/environments.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';
	import { stackVolumeRelationship } from '#lib/utils/swarm-resource-relations.js';
	let {
		kind,
		environmentId,
		stackName,
		services,
		externalNames = new Set<string>()
	}: {
		kind: 'networks' | 'volumes';
		environmentId: string;
		stackName: string;
		services: SwarmServiceSummary[];
		externalNames?: Set<string>;
	} = $props();
	const manager = createQuery(() => swarmManagerQueryOptions({ environmentId, permission: 'swarm-stacks:read' }));
	const inventory = createQuery(() => ({
		queryKey: ['swarm', environmentId, 'stack-related', stackName, kind, manager.data?.clusterId, [...externalNames].sort()],
		enabled: !!manager.data,
		queryFn: async () => {
			const cluster = manager.data!.clusterId;
			if (kind === 'networks') {
				const page = await networkService.getNetworksForEnvironment(environmentId, { pagination: { page: 1, limit: -1 } });
				const ids = new Set(services.flatMap((service) => service.networkIds ?? []));
				return {
					rows: page.data
						.filter(
							(item) =>
								(!item.clusterId || item.clusterId === cluster) &&
								(item.labels?.['com.docker.stack.namespace'] === stackName || ids.has(item.id))
						)
						.map((item) => ({
							id: item.id,
							name: item.name,
							values: [
								item.labels?.['com.docker.stack.namespace'] === stackName
									? m.owned()
									: externalNames.has(item.name)
										? m.external_reference()
										: m.referenced(),
								`${item.driver} / ${item.scope}`
							],
							href: hasPermission('networks:read', environmentId)
								? resourceDetailHref(
										environmentHref(resolve('/(app)/networks/[networkId]', { networkId: item.id }), environmentId)
									)
								: undefined
						})),
					incomplete: services.some((service) => service.networkIds === undefined)
				};
			}
			const environments = environmentStore.available.filter((env) => hasPermission('volumes:list', env.id));
			const [page, tasks] = await Promise.all([
				fetchAcrossEnvironments(environments, { pagination: { page: 1, limit: -1 } }, (id, request, config) =>
					volumeService.getVolumesForEnvironment(id, request, config)
				),
				swarmService.getStackTasks(stackName, { pagination: { page: 1, limit: -1 } }, environmentId)
			]);
			const rows = page.data
				.filter((item) => item.clusterId === cluster)
				.flatMap((item) => {
					const relationship = stackVolumeRelationship(item, stackName, tasks.data);
					if (!relationship) return [];
					const params = new URLSearchParams({ env: item.environmentId, tab: 'overview' });
					if (item.logicalId && item.nodes?.length) {
						params.set('cluster', cluster);
						params.set('driver', item.driver);
						params.set('agents', item.nodes.map((node) => node.id).join(','));
					}
					return [
						{
							id: `${item.environmentId}:${item.id}`,
							name: item.name,
							values: [
								relationship === 'owned' ? m.owned() : externalNames.has(item.name) ? m.external_reference() : m.referenced(),
								item.driver
							],
							href: hasPermission('volumes:read', item.environmentId)
								? resourceDetailHref(`${resolve('/(app)/volumes/[volumeName]', { volumeName: item.name })}?${params}`)
								: undefined
						}
					];
				});
			return { rows, incomplete: !!page.failedEnvironments?.length || page.data.some((item) => !item.clusterId) };
		}
	}));
</script>

{#if manager.isError || inventory.isError || inventory.data?.incomplete || (manager.isSuccess && !manager.data)}<Alert.Root
		variant="warning-subtle"
		><Alert.Description
			>{manager.error?.message ?? inventory.error?.message ?? m.unavailable()}
			<ArcaneButton
				action="refresh"
				onclick={async () => {
					await manager.refetch();
					await inventory.refetch();
				}}
			/></Alert.Description
		></Alert.Root
	>{/if}
<RelatedResourceTable
	columns={[m.relationship(), m.common_driver()]}
	rows={inventory.data?.rows ?? []}
	loading={inventory.isFetching}
	refreshError={inventory.error}
	onRetry={() => inventory.refetch()}
/>
