<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';

	import RelatedResourceTable from '#lib/components/resource-detail/related-resource-table.svelte';
	import { TabBar } from '#lib/components/tab-bar/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { useUrlTab } from '#lib/hooks/use-url-tab.svelte.js';
	import { ContainersIcon, ImagesIcon, NetworksIcon, VolumesIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { containerService } from '#lib/services/container-service.js';
	import { imageService } from '#lib/services/image-service.js';
	import { networkService } from '#lib/services/network-service.js';
	import { volumeService } from '#lib/services/volume-service.js';
	import type { NetworkSummary } from '#lib/types/network.js';
	import { hasPermission, isInAppPath } from '#lib/utils/auth.js';
	import { environmentHref } from '#lib/utils/environments.js';
	import { bytes } from '#lib/utils/formatting.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';

	let { nodeId, agentId }: { nodeId: string; agentId: string } = $props();

	const tabs = [
		{ value: 'containers', label: m.containers(), icon: ContainersIcon },
		{ value: 'networks', label: m.resource_networks_cap(), icon: NetworksIcon },
		{ value: 'images', label: m.images(), icon: ImagesIcon },
		{ value: 'volumes', label: m.resource_volumes_cap(), icon: VolumesIcon }
	] as const;
	type Resource = (typeof tabs)[number]['value'];
	const available = $derived(tabs.filter(({ value }) => hasPermission(`${value}:list`, agentId)));
	const tab = useUrlTab({
		tabs: () => available,
		defaultTab: () => 'networks',
		param: 'resource',
		history: true,
		ready: () => available.length > 0
	});
	const resource = $derived(tab.value);
	const pageParam = $derived(`${resource}Page`);
	const currentUrl = $derived(page.shallow?.url ?? page.url);
	const pageNumber = $derived(
		Math.max(1, Math.min(1_000_000, Number.parseInt(currentUrl.searchParams.get(pageParam) ?? '1', 10) || 1))
	);

	function inspectHref(path: string, id: string): string {
		return resourceDetailHref(environmentHref(path, id));
	}

	function networkAddresses(network: NetworkSummary) {
		return network.ipam?.config
			?.flatMap((config) =>
				[
					config.subnet && `${m.common_subnet()} ${config.subnet}`,
					config.gateway && `${m.common_gateway()} ${config.gateway}`,
					config.ipRange && `${m.networks_ipam_iprange_label()} ${config.ipRange}`,
					...Object.entries(config.auxAddress ?? {}).map(([label, address]) => `${label} ${address}`)
				].filter(Boolean)
			)
			.join(' · ');
	}

	async function list(kind: Resource, environmentId: string, expectedNode: string, currentPage: number, signal: AbortSignal) {
		const options = { pagination: { page: currentPage, limit: 10 } };
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
		const key = [agentId, nodeId, resource, pageNumber] as const;
		return {
			queryKey: ['node-local-resources', ...key],
			queryFn: ({ signal }) => list(key[2], key[0], key[1], key[3], signal),
			enabled: available.some(({ value }) => value === resource)
		};
	});
	const canInspect = $derived(hasPermission(`${resource}:read`, agentId));
	const lastPage = $derived(resources.data?.pagination.totalPages ?? 1);

	function changePage(next: number) {
		const url = new URL(currentUrl.href);
		url.searchParams.set(pageParam, String(next));
		const href = url.pathname + url.search + url.hash;
		if (isInAppPath(href)) void goto(href, { shallow: true, reset: false, state: page.state });
	}
</script>

{#if available.length}
	<Tabs.Root value={resource} onValueChange={tab.select}>
		<TabBar items={tab.items} />
		<Tabs.Content value={resource} class="mt-4">
			{#if resources.isPending}
				<p class="text-sm text-muted-foreground" role="status">{m.common_loading()}</p>
			{:else if resources.isError}
				<div class="flex items-center gap-2">
					<p class="text-sm text-muted-foreground" role="status">{m.swarm_node_local_resources_unavailable()}</p>
					<Button variant="outline" size="sm" onclick={() => resources.refetch()}>{m.common_retry()}</Button>
				</div>
			{:else if resources.data}
				<RelatedResourceTable
					columns={columns[resource]}
					rows={resources.data.rows.map((row) => ({ ...row, href: canInspect ? inspectHref(row.path, agentId) : undefined }))}
				/>
				{#if pageNumber > 1 || lastPage > 1}
					<div class="mt-3 flex items-center gap-2">
						<Button variant="outline" size="sm" disabled={pageNumber <= 1} onclick={() => changePage(pageNumber - 1)}
							>{m.common_previous()}</Button
						>
						<span class="text-sm text-muted-foreground">{m.common_page({ page: pageNumber })}</span>
						<Button variant="outline" size="sm" disabled={pageNumber >= lastPage} onclick={() => changePage(pageNumber + 1)}
							>{m.common_next()}</Button
						>
					</div>
				{/if}
			{/if}
		</Tabs.Content>
	</Tabs.Root>
{:else}
	<p class="text-sm text-muted-foreground">{m.swarm_node_local_resources_unavailable()}</p>
{/if}
