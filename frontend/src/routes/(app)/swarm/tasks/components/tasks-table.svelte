<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import ArcaneTable from '#lib/components/arcane-table/arcane-table.svelte';
	import DetailLinksMenu from '#lib/components/arcane-table/detail-links-menu.svelte';
	import { environmentColumns } from '#lib/components/arcane-table/environment-column.js';
	import type { ColumnSpec, FilterOption, MobileFieldVisibility } from '#lib/components/arcane-table/index.js';
	import { UniversalMobileCard } from '#lib/components/arcane-table/index.js';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import { confirmForceUpdateService } from '#lib/components/swarm/force-update-service.js';
	import SwarmTaskContainer from '#lib/components/swarm/swarm-task-container.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { JobsIcon, ConnectionIcon, RestartIcon, InspectIcon, EditIcon, LayersIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { SwarmTaskSummary } from '#lib/types/swarm.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import type { SwarmTasksPaginatedResponse } from '#lib/types/ui/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { parseImageRef } from '#lib/utils/docker.js';
	import { environmentHref, type EnvironmentPage, type EnvironmentRow } from '#lib/utils/environments.js';
	import { formatRelativeTime } from '#lib/utils/formatting.js';
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

	const namedTasks: TaskPage = $derived({
		...tasks,
		data: tasks.data.map((task) => {
			if (task.name?.trim()) return task;
			const service = task.serviceName || task.serviceId;
			const scope = task.slot || task.nodeName || task.nodeId;
			const name = service ? [service, scope].filter(Boolean).join('.') : task.id.slice(0, 12);
			return { ...task, name };
		})
	});

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
		{ accessorKey: 'updatedAt', id: 'updated', title: m.common_updated(), sortable: true, cell: UpdatedCell, hidden: true },
		{ accessorKey: 'error', title: m.common_error(), cell: ErrorCell, hidden: true },
		{ accessorKey: 'createdAt', id: 'created', title: m.common_created(), hidden: true, sortable: true, cell: CreatedCell },
		{ accessorKey: 'image', title: m.common_image(), hidden: true, cell: ImageCell }
	];

	const columns = $derived([
		{ accessorKey: 'id', title: m.common_id(), hidden: true },
		...(showEnvironment
			? environmentColumns<TaskRow>().map((column) => ({ ...column, title: m.query_connection(), hidden: true, sortable: false }))
			: []),
		{ accessorKey: 'name', title: m.common_name(), sortable: true, cell: TaskNameCell },
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
			filterOptions: stateFilters
		},
		{
			accessorKey: 'desiredState',
			title: m.swarm_desired_state(),
			sortable: true,
			cell: DesiredStateCell,
			hidden: true,
			filterOptions: stateFilters
		},
		{ id: 'container', title: m.container(), cell: ContainerCell },
		...serviceColumns
	] satisfies ColumnSpec<TaskRow>[]);

	const mobileFields = $derived([
		...(serviceScoped ? [] : [{ id: 'serviceName', label: m.swarm_service(), defaultVisible: true }]),
		{ id: 'nodeName', label: m.swarm_node(), defaultVisible: true },
		{ id: 'currentState', label: m.swarm_current_state(), defaultVisible: true },
		{ id: 'desiredState', label: m.swarm_desired_state(), defaultVisible: true }
	]);

	let mobileFieldVisibility = $state<Record<string, boolean>>({});
	let verifiedContainers = $state<Record<string, boolean>>({});
	let containerErrors = $state<Record<string, string>>({});
	function containerKey(item: TaskRow) {
		return [item.environmentId, item.id, item.nodeId, item.containerId, item.currentState, item.updatedAt].join(':');
	}
</script>

{#snippet ContainerCell({ item }: { item: TaskRow })}
	<SwarmTaskContainer
		task={item}
		environmentId={item.environmentId}
		tablePage={requestOptions.pagination?.page}
		verified={verifiedContainers[containerKey(item)] ?? false}
		onVerified={(verified) => (verifiedContainers[containerKey(item)] = verified)}
		unavailable={containerErrors[containerKey(item)]}
		onUnavailable={(error) => (containerErrors[containerKey(item)] = error)}
	/>
{/snippet}

{#snippet ServiceCell({ item }: { item: TaskRow })}
	{#if item.serviceId && item.serviceName && hasPermission('swarm-services:read', item.environmentId)}
		<a
			class="text-primary hover:underline"
			href={resourceTableHref(
				environmentHref(
					resolve('/(app)/swarm/services/[serviceId]', { serviceId: item.serviceId }) + '?tab=overview',
					item.environmentId
				),
				requestOptions.pagination?.page
			)}>{item.serviceName}</a
		>
	{:else}
		{item.serviceName}
	{/if}
{/snippet}

{#snippet NodeCell({ item }: { item: TaskRow })}
	<span>{item.nodeName || m.common_unknown()}</span>
{/snippet}

{#snippet MobileService(item: TaskRow)}
	{@render ServiceCell({ item })}
{/snippet}

{#snippet MobileNode(item: TaskRow)}
	{@render NodeCell({ item })}
{/snippet}

{#snippet StateCell({ item }: { item: TaskRow })}
	<div class="space-y-1">
		<Badge variant={getSwarmTaskStateVariant(item.currentState)}>{item.currentState || m.common_unknown()}</Badge>
		<div class="text-xs text-muted-foreground">{m.swarm_desired_state()}: {item.desiredState}</div>
	</div>
{/snippet}

{#snippet DesiredStateCell({ value }: { value: unknown })}
	<Badge variant={getSwarmTaskStateVariant(String(value ?? ''))} minWidth="20">{String(value ?? m.common_unknown())}</Badge>
{/snippet}

{#snippet TaskNameCell({ item }: { item: TaskRow })}
	<div title={item.id}>
		{#if item.name}<div class="truncate font-medium text-foreground" title={item.name}>{item.name}</div>{/if}
		{#if !item.name || (serviceScoped && item.name !== item.id)}
			<div class="font-mono text-xs text-muted-foreground">{item.id.slice(0, 12)}</div>
		{/if}
		{@render ErrorCell({ item })}
	</div>
{/snippet}

{#snippet ImageCell({ item }: { item: TaskRow })}{@const image = parseImageRef(item.image ?? '')}
	<div class="max-w-64" title={item.image}>
		<div class="truncate">{image.repo}</div>
		<div class="truncate text-xs text-muted-foreground">{item.image?.includes('@') ? item.image.split('@')[1] : image.tag}</div>
	</div>{/snippet}

{#snippet CreatedCell({ item }: { item: TaskRow })}{formatRelativeTime(item.createdAt) || m.common_na()}{/snippet}

{#snippet UpdatedCell({ item }: { item: TaskRow })}
	{formatRelativeTime(item.updatedAt) || m.common_na()}
{/snippet}

{#snippet ErrorCell({ item }: { item: TaskRow })}
	{#if item.mixedImageVersions}<span class="wrap-anywhere text-destructive">{m.mixed_image_versions()}</span>{/if}
	{#if item.error}
		<span class="wrap-anywhere text-destructive">{item.error}</span>
	{/if}
{/snippet}

{#snippet TaskActions({ item }: { item: TaskRow })}
	{#snippet VerifiedContainer()}<SwarmTaskContainer
			task={item}
			environmentId={item.environmentId}
			tablePage={requestOptions.pagination?.page}
			verified={verifiedContainers[containerKey(item)] ?? false}
			onVerified={(verified) => (verifiedContainers[containerKey(item)] = verified)}
			unavailable={containerErrors[containerKey(item)]}
			onUnavailable={(error) => (containerErrors[containerKey(item)] = error)}
			menu
		/>{/snippet}
	{@const canReadService = hasPermission('swarm-services:read', item.environmentId)}
	{@const canUpdateService = hasPermission('swarm-services:update', item.environmentId)}
	{#if item.serviceId && (canReadService || (canReadService && canUpdateService) || (item.desiredState === 'running' && canUpdateService))}
		{@const href = resourceTableHref(
			environmentHref(
				resolve('/(app)/swarm/services/[serviceId]', { serviceId: item.serviceId }) + '?tab=overview',
				item.environmentId
			),
			requestOptions.pagination?.page
		)}
		<RowActionsMenu name={item.name || item.id}>
			<DetailLinksMenu
				{href}
				related={verifiedContainers[containerKey(item)] ? VerifiedContainer : undefined}
				tabs={[
					{
						value: 'service',
						label: m.swarm_view_service(),
						icon: InspectIcon,
						related: true,
						visible: canReadService,
						href: resourceTableHref(
							environmentHref(
								resolve('/(app)/swarm/services/[serviceId]', { serviceId: item.serviceId }) + '?tab=overview',
								item.environmentId
							),
							requestOptions.pagination?.page
						)
					},
					{
						value: 'stack',
						label: m.swarm_stack(),
						icon: LayersIcon,
						related: true,
						visible: !!item.stackName && hasPermission('swarm-stacks:read', item.environmentId),
						href: item.stackName
							? resourceTableHref(
									environmentHref(
										resolve('/(app)/swarm/stacks/[name]', { name: item.stackName }) + '?tab=overview',
										item.environmentId
									),
									requestOptions.pagination?.page
								)
							: undefined
					}
				]}
			/>
			{#if canReadService}
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
		title={(item: TaskRow) => item.name || item.id.slice(0, 12)}
		subtitle={(item: TaskRow) => (serviceScoped && item.name && item.name !== item.id ? item.id.slice(0, 12) : null)}
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
				show: mobileFieldVisibility['desiredState'] ?? true
			}
		]}
	>
		{@render ContainerCell({ item })}
		{#if item.mixedImageVersions}<p class="mt-1 text-sm text-destructive">{m.mixed_image_versions()}</p>{/if}
		{#if item.error}
			<p class="mt-1 text-sm wrap-anywhere text-destructive">{item.error}</p>
		{/if}
	</UniversalMobileCard>
{/snippet}

<ArcaneTable
	{persistKey}
	items={namedTasks}
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
