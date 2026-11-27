<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { LayersIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { containerOwnershipQuery } from '#lib/query/containers.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref } from '#lib/utils/environments.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';
	let {
		environmentId,
		containerId,
		presentation = 'links'
	}: { environmentId: string; containerId: string; presentation?: 'links' | 'summary' } = $props();
	const ownership = createQuery(() => ({
		...containerOwnershipQuery({ environmentId, containerId }),
		enabled: presentation === 'links'
	}));
</script>

{#if presentation === 'summary'}
	<span class="block max-w-48 truncate text-sm"
		>{ownership.data?.stackName ?? ownership.data?.serviceName ?? m.common_unknown()}</span
	>
{:else if ownership.data}
	{@const owner = ownership.data}
	{#if hasPermission('swarm-services:read', owner.manager.id)}<DropdownMenu.Item
			>{#snippet child({ props })}<a
					{...props}
					href={resourceDetailHref(
						environmentHref(
							`${resolve('/(app)/swarm/services/[serviceId]', { serviceId: owner.serviceId })}?cluster=${encodeURIComponent(owner.manager.clusterId)}`,
							owner.manager.id
						)
					)}><LayersIcon class="size-4" />{m.swarm_service()}</a
				>{/snippet}</DropdownMenu.Item
		>{/if}
	{#if owner.stackName && hasPermission('swarm-stacks:read', owner.manager.id)}{@const stackName =
			owner.stackName}<DropdownMenu.Item
			>{#snippet child({ props })}<a
					{...props}
					href={resourceDetailHref(
						environmentHref(
							`${resolve('/(app)/swarm/stacks/[name]', { name: stackName })}?cluster=${encodeURIComponent(owner.manager.clusterId)}`,
							owner.manager.id
						)
					)}><LayersIcon class="size-4" />{m.swarm_stack()}</a
				>{/snippet}</DropdownMenu.Item
		>{/if}
{:else}<DropdownMenu.Item disabled
		><LayersIcon class="size-4" /><span role="status">{ownership.isFetching ? m.common_loading() : m.common_unavailable()}</span
		></DropdownMenu.Item
	>{/if}
