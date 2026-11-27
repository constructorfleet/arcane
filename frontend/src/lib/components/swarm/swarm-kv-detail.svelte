<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import { ActionButtonGroup } from '#lib/components/action-button-group/index.js';
	import { KeyValueCard } from '#lib/components/resource-detail/index.js';
	import SwarmKvFormSheet from '#lib/components/sheets/swarm-kv-form-sheet.svelte';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { useUrlTab } from '#lib/hooks/use-url-tab.svelte.js';
	import { InspectIcon, LayersIcon } from '#lib/icons/index.js';
	import TabbedPageLayout from '#lib/layouts/tabbed-page-layout.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import type { SwarmKvSummary } from '#lib/types/ui/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref, fetchAcrossEnvironments } from '#lib/utils/environments.js';
	import { formatRelativeTime } from '#lib/utils/formatting.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';
	import type { SwarmKvResource } from '#lib/utils/swarm-kv.js';

	import SwarmServicesTable from '../../../routes/(app)/swarm/services/components/services-table.svelte';
	let { resource, item, environmentId }: { resource: SwarmKvResource; item: SwarmKvSummary; environmentId: string } = $props();
	let editOpen = $state(false);
	let sheetEnvironmentId = $state('');
	const canListServices = $derived(hasPermission('swarm-services:list', environmentId));
	const tabs = useUrlTab({
		tabs: () => [
			{ value: 'overview', label: m.common_overview(), icon: InspectIcon },
			{ value: 'services', label: m.services(), icon: LayersIcon, visible: canListServices }
		],
		defaultTab: () => 'overview',
		history: true
	});
	let requestOptions = $state<SearchPaginationSortRequest>({ pagination: { page: 1, limit: 20 } });
	const options = $derived({
		...requestOptions,
		filters: { ...requestOptions.filters, [resource.kind === 'secrets' ? 'secretId' : 'configId']: item.id }
	});
	const services = createQuery(() => ({
		queryKey: ['swarm', environmentId, 'metadata-services', resource.kind, item.id, options],
		enabled: canListServices && tabs.value === 'services',
		queryFn: () =>
			fetchAcrossEnvironments([{ id: environmentId, name: environmentId }], options, (id, request, config) =>
				swarmService.getServices(request, id, config)
			)
	}));
	const stackName = $derived(item.spec.Labels?.['com.docker.stack.namespace']);
</script>

<TabbedPageLayout
	backUrl={resource.kind === 'secrets' ? '/swarm/secrets' : '/swarm/configs'}
	tabItems={tabs.items}
	selectedTab={tabs.value}
	onTabChange={tabs.select}
>
	{#snippet headerInfo()}<div class="flex min-w-0 items-center gap-2">
			<resource.icon class="size-5" />
			<h1 class="truncate font-semibold">{item.spec.Name}</h1>
		</div>{/snippet}
	{#snippet headerActions()}{#if hasPermission(resource.permissions.update, environmentId)}<ActionButtonGroup
				buttons={[
					{
						id: 'edit',
						action: 'edit',
						label: m.common_edit(),
						onclick: () => {
							sheetEnvironmentId = environmentId;
							editOpen = true;
						}
					}
				]}
			/>{/if}{/snippet}
	{#snippet tabContent()}
		<Tabs.Content value="overview">
			<Card.Root
				><Card.Header><Card.Title>{m.details()}</Card.Title></Card.Header><Card.Content
					><div class="grid gap-4 sm:grid-cols-2">
						<KeyValueCard label={m.common_name()}>{item.spec.Name}</KeyValueCard>
						<KeyValueCard label={m.common_id()}>{item.id}</KeyValueCard>
						<KeyValueCard label={m.common_created()}>{formatRelativeTime(item.createdAt)}</KeyValueCard>
						<KeyValueCard label={m.common_updated()}>{formatRelativeTime(item.updatedAt)}</KeyValueCard>
						{#if item.spec.Driver?.Name}<KeyValueCard label={m.common_driver()}>{item.spec.Driver.Name}</KeyValueCard>{/if}
					</div></Card.Content
				></Card.Root
			>
			{#if stackName}<Card.Root
					><Card.Header><Card.Title>{m.linked_resources()}</Card.Title></Card.Header><Card.Content>
						<KeyValueCard label={m.owned_by()} valueFormat="custom"
							>{#if hasPermission('swarm-stacks:read', environmentId)}<a
									class="text-primary-text hover:underline"
									href={resourceDetailHref(
										environmentHref(resolve('/(app)/swarm/stacks/[name]', { name: stackName }), environmentId)
									)}>{stackName}</a
								>{:else}{stackName}{/if}</KeyValueCard
						>
					</Card.Content></Card.Root
				>{/if}
			{#if item.spec.Labels && Object.keys(item.spec.Labels).length}<Card.Root
					><Card.Header><Card.Title>{m.common_labels()}</Card.Title></Card.Header><Card.Content
						><pre class="overflow-auto text-xs break-all whitespace-pre-wrap">{JSON.stringify(
								item.spec.Labels,
								null,
								2
							)}</pre></Card.Content
					></Card.Root
				>{/if}
		</Tabs.Content>
		<Tabs.Content value="services" layout="table"
			>{#if services.data}<SwarmServicesTable
					services={services.data}
					invalidateKey={['swarm', environmentId]}
					bind:requestOptions
					onRetry={() => services.refetch()}
					showEnvironment={false}
				/>{:else}<p role="status">{services.error?.message ?? m.common_loading()}</p>{/if}</Tabs.Content
		>
	{/snippet}
</TabbedPageLayout>
{#if editOpen}<SwarmKvFormSheet
		bind:open={editOpen}
		{resource}
		bind:environmentId={sheetEnvironmentId}
		existing={item}
		onSaved={async () => {
			editOpen = false;
			await refreshAll();
		}}
	/>{/if}
