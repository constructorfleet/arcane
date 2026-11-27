<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { untrack } from 'svelte';

	import RelatedResourceTable from '#lib/components/resource-detail/related-resource-table.svelte';
	import { TabBar } from '#lib/components/tab-bar/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { useUrlTab } from '#lib/hooks/use-url-tab.svelte.js';
	import { ContainersIcon, ImagesIcon, NetworksIcon, VolumesIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { containerService } from '#lib/services/container-service.js';
	import { imageService } from '#lib/services/image-service.js';
	import { networkService } from '#lib/services/network-service.js';
	import { volumeService } from '#lib/services/volume-service.js';
	import type { NetworkSummary } from '#lib/types/network.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref } from '#lib/utils/environments.js';
	import { bytes } from '#lib/utils/formatting.js';
	import { resourceTableHref, resourceTablePage } from '#lib/utils/resource-navigation.js';

	let { nodeId, agentId }: { nodeId: string; agentId?: string } = $props();

	const tabs = [
		{ value: 'containers', label: m.containers(), icon: ContainersIcon },
		{ value: 'networks', label: m.resource_networks_cap(), icon: NetworksIcon },
		{ value: 'images', label: m.images(), icon: ImagesIcon },
		{ value: 'volumes', label: m.resource_volumes_cap(), icon: VolumesIcon }
	] as const;
	type Resource = (typeof tabs)[number]['value'];
	const available = $derived(tabs.filter(({ value }) => !!agentId && hasPermission(`${value}:list`, agentId)));
	const tab = useUrlTab({
		tabs: () => available,
		defaultTab: () => 'networks',
		param: 'resource',
		history: true,
		ready: () => available.length > 0
	});
	const resource = $derived(tab.value);
	type LocalResource = Resource;

	function initialRequest(kind: LocalResource): SearchPaginationSortRequest {
		const url = page.shallow?.url ?? page.url;
		const current = url.searchParams.get('resource') ?? 'networks';
		const requestedPage = (current === kind ? url.searchParams.get('tablePage') : null) ?? url.searchParams.get(`${kind}Page`);
		return { pagination: { page: resourceTablePage(requestedPage), limit: 10 } };
	}

	const requests = $state(
		untrack(() => ({
			containers: initialRequest('containers'),
			networks: initialRequest('networks'),
			images: initialRequest('images'),
			volumes: initialRequest('volumes')
		}))
	);

	function networkAddresses(network: NetworkSummary) {
		const configs = network.ipam?.config ?? [];
		const displayed = configs.slice(0, 2).flatMap((config) => {
			const auxiliary = Object.entries(config.auxAddress ?? {});
			return [
				config.subnet && m.common_subnet() + ' ' + config.subnet,
				config.gateway && m.common_gateway() + ' ' + config.gateway,
				config.ipRange && m.networks_ipam_iprange_label() + ' ' + config.ipRange,
				...auxiliary.slice(0, 2).map(([label, address]) => label + ' ' + address),
				auxiliary.length > 2 ? '+' + (auxiliary.length - 2) : ''
			];
		});
		if (configs.length > 2) displayed.push('+' + (configs.length - 2));
		return displayed.filter(Boolean).join(' · ');
	}

	async function list(
		kind: Resource,
		environmentId: string | undefined,
		expectedNode: string,
		options: SearchPaginationSortRequest,
		signal: AbortSignal
	) {
		if (!environmentId) return undefined;
		switch (kind) {
			case 'containers': {
				const { data, pagination } = await containerService.getContainersForEnvironment(environmentId, options, { signal });
				return {
					pagination,
					rows: data.map((item) => ({
						id: item.id,
						name: item.names[0]?.replace(/^\//, '') || item.id.slice(0, 12),
						values: [item.image, item.state],
						path: resolve('/(app)/containers/[containerId]', { containerId: item.id })
					}))
				};
			}
			case 'networks': {
				const { data, pagination } = await networkService.getNetworksForEnvironment(environmentId, options, { signal });
				if (data.some((item) => item.nodeId !== expectedNode)) throw new Error(m.swarm_node_network_list_unavailable());
				return {
					pagination,
					rows: data.map((item) => ({
						id: item.id,
						name: item.name,
						values: [item.driver, item.scope, networkAddresses(item)],
						path: resolve('/(app)/networks/[networkId]', { networkId: item.id })
					}))
				};
			}
			case 'images': {
				const { data, pagination } = await imageService.getImagesForEnvironment(environmentId, options, { signal });
				return {
					pagination,
					rows: data.map((item) => ({
						id: item.id,
						name: item.repoTags?.[0] || item.id,
						values: [bytes(item.size)],
						path: resolve('/(app)/images/[imageId]', { imageId: item.id })
					}))
				};
			}
			case 'volumes': {
				const { data, pagination } = await volumeService.getVolumesForEnvironment(environmentId, options, { signal });
				if (data.some((item) => item.nodeId && item.nodeId !== expectedNode))
					throw new Error(m.swarm_node_local_resources_unavailable());
				return {
					pagination,
					rows: data.map((item) => ({
						id: item.name,
						name: item.name,
						values: [item.driver, item.scope],
						path: resolve('/(app)/volumes/[volumeName]', { volumeName: encodeURIComponent(item.name) })
					}))
				};
			}
		}
	}

	const columns = {
		containers: [m.common_image(), m.common_status()],
		networks: [m.common_driver(), m.common_scope(), m.swarm_node_network_ip_configuration()],
		images: [m.common_size()],
		volumes: [m.common_driver(), m.common_scope()]
	};
	const resources = createQuery(() => {
		const options = $state.snapshot(requests[resource]);
		const key = [agentId, nodeId, resource, options] as const;
		return {
			queryKey: ['node-local-resources', ...key],
			queryFn: ({ signal }) => list(key[2], key[0], key[1], key[3], signal),
			enabled: !!agentId && available.some(({ value }) => value === resource)
		};
	});
	const canInspect = $derived(!!agentId && hasPermission(`${resource}:read`, agentId));
	const icons = { containers: ContainersIcon, networks: NetworksIcon, images: ImagesIcon, volumes: VolumesIcon };
</script>

{#if available.length}
	<Tabs.Root value={resource} onValueChange={tab.select}>
		<TabBar items={tab.items} />
		<Tabs.Content value={resource} class="mt-4">
			{#if agentId}
				{#key resource}
					<RelatedResourceTable
						columns={columns[resource]}
						icon={icons[resource]}
						persistKey={`arcane-node-local-${resource}-table`}
						bind:requestOptions={requests[resource]}
						pagination={resources.data?.pagination ?? { currentPage: 1, itemsPerPage: 10, totalItems: 0, totalPages: 1 }}
						rows={(resources.data?.rows ?? []).map((row) => ({
							...row,
							href: canInspect
								? resourceTableHref(environmentHref(row.path, agentId), requests[resource].pagination?.page)
								: undefined
						}))}
						loading={resources.isPending}
						refreshError={resources.error}
						onRetry={() => resources.refetch()}
					/>
				{/key}
			{/if}
		</Tabs.Content>
	</Tabs.Root>
{:else}
	<p class="text-sm text-muted-foreground">{m.swarm_node_local_resources_unavailable()}</p>
{/if}
