<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import ArcaneTable from '#lib/components/arcane-table/arcane-table.svelte';
	import { environmentColumns } from '#lib/components/arcane-table/environment-column.js';
	import type { ColumnSpec, FilterOption, MobileFieldVisibility } from '#lib/components/arcane-table/index.js';
	import { UniversalMobileCard } from '#lib/components/arcane-table/index.js';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import { confirmForceUpdateService } from '#lib/components/swarm/force-update-service.js';
	import SwarmTaskContainer from '#lib/components/swarm/swarm-task-container.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { JobsIcon, ConnectionIcon, RestartIcon, InspectIcon, EditIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { SwarmTaskSummary } from '#lib/types/swarm.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import type { SwarmTasksPaginatedResponse } from '#lib/types/ui/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref, type EnvironmentPage, type EnvironmentRow } from '#lib/utils/environments.js';
	import { formatDateTimeShort } from '#lib/utils/formatting.js';
	import { resourceTableHref } from '#lib/utils/resource-navigation.js';
	import { getSwarmTaskIconVariant, getSwarmTaskStateVariant } from '#lib/utils/swarm-tasks.js';

	let {
		tasks,
		requestOptions = $bindable(),
		onRetry,
		persistKey = 'arcane-swarm-tasks-table',
		showEnvironment = true,
		serviceScoped = false,
		nodeFilters = [],
		loading = false,
		refreshError
	}: {
		tasks: TaskPage;
		requestOptions: SearchPaginationSortRequest;
		onRetry: () => Promise<unknown>;
		persistKey?: string;
		showEnvironment?: boolean;
		serviceScoped?: boolean;
		nodeFilters?: FilterOption[];
		loading?: boolean;
		refreshError?: unknown;
	} = $props();

	type TaskRow = EnvironmentRow<SwarmTaskSummary>;
	type TaskPage = EnvironmentPage<SwarmTasksPaginatedResponse>;

	const stateFilters = [
		'new',
		'allocated',
		'pending',
		'assigned',
		'accepted',
		'preparing',
		'ready',
		'starting',
		'running',
		'complete',
		'shutdown',
		'failed',
		'rejected',
		'remove',
		'orphaned'
	].map((state) => ({ label: state, value: state }));

	const serviceColumns: ColumnSpec<TaskRow>[] = [
		{ accessorKey: 'updatedAt', id: 'updated', title: m.common_updated(), sortable: true, cell: UpdatedCell },
		{ accessorKey: 'error', title: m.common_error(), cell: ErrorCell }
	];

	const columns = $derived([
		{ accessorKey: 'id', title: m.common_id(), hidden: true },
		...(showEnvironment ? environmentColumns<TaskRow>() : []),
		{ accessorKey: 'name', title: m.common_name(), sortable: true, cell: serviceScoped ? TaskNameCell : undefined },
		{ id: 'container', title: m.container(), cell: ContainerCell },
		...(serviceScoped
			? []
			: [{ accessorKey: 'serviceName', title: m.swarm_service(), sortable: true, cell: ServiceCell } as const]),
		{
			accessorKey: 'nodeName',
			title: m.swarm_node(),
			cell: NodeCell,
			sortable: true,
			filterOptions: nodeFilters.length ? nodeFilters : undefined
		},
		{
			accessorKey: 'currentState',
			title: m.swarm_current_state(),
			sortable: true,
			cell: StateCell,
			filterOptions: serviceScoped ? stateFilters : undefined
		},
		{
			accessorKey: 'desiredState',
			title: m.swarm_desired_state(),
			sortable: true,
			cell: DesiredStateCell,
			filterOptions: serviceScoped ? stateFilters : undefined
		},
		...(serviceScoped ? serviceColumns : [])
	] satisfies ColumnSpec<TaskRow>[]);

	const mobileFields = $derived([
		...(serviceScoped ? [] : [{ id: 'serviceName', label: m.swarm_service(), defaultVisible: true }]),
		{ id: 'nodeName', label: m.swarm_node(), defaultVisible: true },
		{ id: 'currentState', label: m.swarm_current_state(), defaultVisible: true },
		{ id: 'desiredState', label: m.swarm_desired_state(), defaultVisible: serviceScoped }
	]);

	let mobileFieldVisibility = $state<Record<string, boolean>>({});
</script>

{#snippet ContainerCell({ item }: { item: TaskRow })}
	<SwarmTaskContainer task={item} environmentId={item.environmentId} tablePage={requestOptions.pagination?.page} />
{/snippet}

{#snippet ServiceCell({ item }: { item: TaskRow })}
	{#if item.serviceId && item.serviceName && hasPermission('swarm-services:read', item.environmentId)}
		<a
			class="text-primary hover:underline"
			href={resourceTableHref(
				environmentHref(resolve('/(app)/swarm/services/[serviceId]', { serviceId: item.serviceId }), item.environmentId),
				requestOptions.pagination?.page
			)}>{item.serviceName}</a
		>
	{:else}
		{item.serviceName}
	{/if}
{/snippet}

{#snippet NodeCell({ item }: { item: TaskRow })}
	{#if item.nodeId && item.nodeName && hasPermission('swarm-nodes:read', item.environmentId)}
		<a
			class="text-primary hover:underline"
			href={resourceTableHref(
				environmentHref(resolve('/(app)/swarm/nodes/[nodeId]', { nodeId: item.nodeId }), item.environmentId),
				requestOptions.pagination?.page
			)}>{item.nodeName}</a
		>
	{:else}
		{item.nodeName}
	{/if}
{/snippet}

{#snippet MobileService(item: TaskRow)}
	{@render ServiceCell({ item })}
{/snippet}

{#snippet MobileNode(item: TaskRow)}
	{@render NodeCell({ item })}
{/snippet}

{#snippet StateCell({ value }: { value: unknown })}
	<Badge variant={getSwarmTaskStateVariant(String(value ?? ''))} minWidth="20">{String(value ?? m.common_unknown())}</Badge>
{/snippet}

{#snippet DesiredStateCell({ value }: { value: unknown })}
	<Badge variant={getSwarmTaskStateVariant(String(value ?? ''))} minWidth="20">{String(value ?? m.common_unknown())}</Badge>
{/snippet}

{#snippet TaskNameCell({ item }: { item: TaskRow })}
	<div title={item.id}>
		{#if item.name}<div class="truncate" title={item.name}>{item.name}</div>{/if}
		<div class="font-mono text-xs text-muted-foreground">{item.id.slice(0, 12)}</div>
	</div>
{/snippet}

{#snippet UpdatedCell({ item }: { item: TaskRow })}
	{formatDateTimeShort(item.updatedAt) || m.common_na()}
{/snippet}

{#snippet ErrorCell({ item }: { item: TaskRow })}
	{#if item.error}
		<span class="wrap-anywhere text-destructive">{item.error}</span>
	{/if}
{/snippet}

{#snippet TaskActions({ item }: { item: TaskRow })}
	{@const canReadService = hasPermission('swarm-services:read', item.environmentId)}
	{@const canUpdateService = hasPermission('swarm-services:update', item.environmentId)}
	{#if item.serviceId && (canReadService || (item.desiredState === 'running' && canUpdateService))}
		{@const href = resourceTableHref(
			environmentHref(resolve('/(app)/swarm/services/[serviceId]', { serviceId: item.serviceId }), item.environmentId),
			requestOptions.pagination?.page
		)}
		<RowActionsMenu name={item.name || item.id} triggerClass="relative size-8 p-0" iconClass="">
			{#if canReadService}
				<DropdownMenu.Item onclick={() => goto(href)}>
					<InspectIcon class="size-4" />
					{m.swarm_view_service()}
				</DropdownMenu.Item>
				{#if canUpdateService}
					<DropdownMenu.Item onclick={() => goto(href, { state: { swarmServiceAction: 'edit' } })}>
						<EditIcon class="size-4" />
						{m.swarm_edit_service()}
					</DropdownMenu.Item>
				{/if}
			{/if}
			{#if item.desiredState === 'running' && canUpdateService}
				<DropdownMenu.Item
					onclick={() =>
						confirmForceUpdateService({
							serviceId: item.serviceId,
							serviceName: item.serviceName || item.serviceId,
							environmentId: item.environmentId,
							refresh: onRetry
						})}
				>
					<RestartIcon class="size-4" />
					{m.swarm_service_force_update()}
				</DropdownMenu.Item>
			{/if}
		</RowActionsMenu>
	{/if}
{/snippet}

{#snippet TaskMobileCardSnippet({ item, mobileFieldVisibility }: { item: TaskRow; mobileFieldVisibility: MobileFieldVisibility })}
	<UniversalMobileCard
		{item}
		class={serviceScoped ? 'my-1 rounded-lg border border-border bg-card p-3' : undefined}
		icon={() => ({
			component: JobsIcon,
			variant: getSwarmTaskIconVariant(item.currentState)
		})}
		title={(item: TaskRow) => (serviceScoped ? item.name || item.id.slice(0, 12) : item.name)}
		subtitle={(item: TaskRow) => (serviceScoped ? item.id.slice(0, 12) : null)}
		badges={[
			(item: TaskRow) =>
				(mobileFieldVisibility['currentState'] ?? true)
					? {
							variant: getSwarmTaskStateVariant(item.currentState),
							text: serviceScoped ? `${m.swarm_current_state()}: ${item.currentState}` : item.currentState
						}
					: null
		]}
		rowActions={TaskActions}
		fields={[
			{
				label: m.swarm_service(),
				getValue: (item: TaskRow) => (item.serviceName ? item : null),
				type: 'component',
				component: MobileService,
				show: !serviceScoped && (mobileFieldVisibility['serviceName'] ?? true)
			},
			{
				label: m.swarm_node(),
				getValue: (item: TaskRow) => (item.nodeName ? item : null),
				type: 'component',
				component: MobileNode,
				icon: ConnectionIcon,
				iconVariant: 'gray' as const,
				show: mobileFieldVisibility['nodeName'] ?? true
			},
			{
				label: m.swarm_desired_state(),
				getValue: (item: TaskRow) => item.desiredState,
				icon: ConnectionIcon,
				iconVariant: 'gray' as const,
				show: mobileFieldVisibility['desiredState'] ?? serviceScoped
			}
		]}
	>
		<SwarmTaskContainer task={item} environmentId={item.environmentId} tablePage={requestOptions.pagination?.page} />
		{#if serviceScoped && item.error}
			<p class="mt-1 text-sm wrap-anywhere text-destructive">{item.error}</p>
		{/if}
	</UniversalMobileCard>
{/snippet}

<ArcaneTable
	{persistKey}
	items={tasks}
	bind:requestOptions
	bind:mobileFieldVisibility
	selectionDisabled={true}
	{onRetry}
	{loading}
	listOnly={serviceScoped}
	{refreshError}
	{columns}
	{mobileFields}
	rowActions={TaskActions}
	mobileCard={TaskMobileCardSnippet}
/>
