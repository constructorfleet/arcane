<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { useQueryClient, type QueryKey } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';

	import ArcaneTable from '#lib/components/arcane-table/arcane-table.svelte';
	import DetailLinksMenu from '#lib/components/arcane-table/detail-links-menu.svelte';
	import { environmentColumns } from '#lib/components/arcane-table/environment-column.js';
	import type { ColumnSpec, MobileFieldVisibility } from '#lib/components/arcane-table/index.js';
	import { UniversalMobileCard } from '#lib/components/arcane-table/index.js';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import IfPermitted from '#lib/components/if-permitted.svelte';
	import { confirmForceUpdateService } from '#lib/components/swarm/force-update-service.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import {
		ContainersIcon,
		GlobeIcon,
		TrashIcon,
		NetworksIcon,
		SettingsIcon,
		FileTextIcon,
		RestartIcon,
		EditIcon,
		VolumesIcon,
		LayersIcon
	} from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SwarmServiceSummary } from '#lib/types/swarm.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import type { SwarmServicesPaginatedResponse } from '#lib/types/ui/swarm.js';
	import { handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import {
		getSwarmServiceModeLabel,
		getSwarmServiceModeVariant,
		isSwarmServiceModeScalable,
		parseImageRef
	} from '#lib/utils/docker.js';
	import { environmentHref, type EnvironmentPage, type EnvironmentRow } from '#lib/utils/environments.js';
	import { formatRelativeTime } from '#lib/utils/formatting.js';
	import { resourceTableHref } from '#lib/utils/resource-navigation.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	let {
		services,
		invalidateKey,
		requestOptions = $bindable(),
		onRetry,
		persistKey = 'arcane-swarm-services-table',
		showEnvironment = true,
		showStack = true
	}: {
		services: ServicePage;
		invalidateKey: QueryKey;
		requestOptions: SearchPaginationSortRequest;
		onRetry: () => Promise<unknown>;
		persistKey?: string;
		showEnvironment?: boolean;
		showStack?: boolean;
	} = $props();

	const queryClient = useQueryClient();

	type ServiceRow = EnvironmentRow<SwarmServiceSummary>;
	type ServicePage = EnvironmentPage<SwarmServicesPaginatedResponse>;

	function getShortName(name: string, stackName?: string | null): string {
		if (stackName && name.startsWith(`${stackName}_`)) {
			return name.slice(stackName.length + 1);
		}
		return name;
	}

	function modeIconVariant(mode: string): 'emerald' | 'blue' | 'amber' | 'purple' | 'gray' {
		if (mode === 'replicated') return 'blue';
		if (mode === 'global') return 'emerald';
		if (mode === 'replicated-job') return 'amber';
		if (mode === 'global-job') return 'purple';
		return 'gray';
	}

	const isLoading = $state({ remove: false });

	const isAnyLoading = $derived(Object.values(isLoading).some(Boolean));

	function openService(service: ServiceRow, action?: App.PageState['swarmServiceAction']) {
		goto(
			resourceTableHref(
				environmentHref(resolve('/(app)/swarm/services/[serviceId]', { serviceId: service.id }), service.environmentId),
				requestOptions.pagination?.page
			),
			{ state: { swarmServiceAction: action } }
		);
	}

	function handleDelete(service: ServiceRow) {
		openConfirmDialog({
			title: m.common_delete_title({ resource: m.swarm_service() }),
			message: m.common_delete_confirm({ resource: m.swarm_service() }),
			confirm: {
				label: m.common_delete(),
				destructive: true,
				action: async () => {
					isLoading.remove = true;
					await handleApiResultWithCallbacks({
						result: await tryCatch(swarmService.removeService(service.id, service.environmentId)),
						message: m.common_delete_failed({ resource: `${m.swarm_service()} "${service.name}"` }),
						setLoadingState: (v) => (isLoading.remove = v),
						onSuccess: async () => {
							toast.success(m.common_delete_success({ resource: `${m.swarm_service()} "${service.name}"` }));
							await queryClient.invalidateQueries({ queryKey: invalidateKey });
						}
					});
				}
			}
		});
	}

	const columns = $derived([
		{ accessorKey: 'id', title: m.common_id(), hidden: true },
		...(showEnvironment ? environmentColumns<ServiceRow>() : []),
		{ accessorKey: 'name', title: m.swarm_service(), sortable: true, cell: NameCell },
		...(showStack ? [{ accessorKey: 'stackName' as const, title: m.swarm_stack(), cell: StackCell }] : []),
		{ accessorKey: 'mode', title: m.swarm_mode(), sortable: true, cell: ModeCell },
		{ accessorKey: 'replicas', title: m.swarm_replicas(), sortable: true, cell: ReplicasCell },
		{ accessorKey: 'image', title: m.common_image(), sortable: true, cell: ImageCell },
		{ accessorKey: 'updatedAt', id: 'updated', title: m.common_updated(), sortable: true, cell: UpdatedCell },
		{ accessorKey: 'ports', title: m.common_ports(), hidden: true, cell: PortsCell },
		{ accessorKey: 'nodes', title: m.nodes(), hidden: true, cell: NodeCountCell },
		{ accessorKey: 'networks', title: m.resource_networks_cap(), hidden: true, cell: NetworkCountCell },
		{ accessorKey: 'mounts', title: m.common_mounts(), hidden: true, cell: MountCountCell },
		{ accessorKey: 'createdAt', title: m.common_created(), hidden: true, cell: CreatedCell }
	] satisfies ColumnSpec<ServiceRow>[]);

	const mobileFields = $derived([
		...(showStack ? [{ id: 'stackName', label: m.swarm_stack(), defaultVisible: true }] : []),
		{ id: 'mode', label: m.swarm_mode(), defaultVisible: true },
		{ id: 'replicas', label: m.swarm_replicas(), defaultVisible: true },
		{ id: 'image', label: m.common_image(), defaultVisible: true },
		{ id: 'updated', label: m.common_updated(), defaultVisible: true }
	]);

	let mobileFieldVisibility = $state<Record<string, boolean>>({});
</script>

{#snippet NameCell({ item }: { item: ServiceRow })}
	<a
		href={resourceTableHref(
			environmentHref(resolve('/(app)/swarm/services/[serviceId]', { serviceId: item.id }), item.environmentId),
			requestOptions.pagination?.page
		)}
		class="block max-w-56 truncate text-sm font-medium text-primary-text hover:underline"
		title={item.name}
	>
		{getShortName(item.name, item.stackName)}
	</a>
{/snippet}

{#snippet ModeCell({ value }: { value: unknown })}
	<Badge variant={getSwarmServiceModeVariant(String(value ?? ''))} minWidth="20"
		>{getSwarmServiceModeLabel(String(value ?? ''))}</Badge
	>
{/snippet}

{#snippet StackCell({ item }: { item: ServiceRow })}
	{#if item.stackName && hasPermission('swarm-stacks:read', item.environmentId)}
		<a
			class="block max-w-40 truncate text-sm text-primary-text hover:underline"
			title={item.stackName}
			href={resourceTableHref(
				environmentHref(resolve('/(app)/swarm/stacks/[name]', { name: item.stackName }), item.environmentId),
				requestOptions.pagination?.page
			)}>{item.stackName}</a
		>
	{:else}
		<span class="block max-w-40 truncate text-sm text-muted-foreground" title={item.stackName}
			>{item.stackName || m.common_na()}</span
		>
	{/if}
{/snippet}

{#snippet MobileStack(item: ServiceRow)}
	{@render StackCell({ item })}
{/snippet}

{#snippet ReplicasCell({ item }: { item: ServiceRow })}
	{#if item.mode.endsWith('job')}
		<span
			class="font-mono text-sm whitespace-nowrap"
			class:text-success={item.replicas > 0 && item.completedReplicas != null && item.completedReplicas >= item.replicas}
			>{item.completedReplicas ?? m.unavailable()} / {item.replicas} <span class="text-xs">{m.completed()}</span></span
		>
	{:else}
		<span
			class="font-mono text-sm whitespace-nowrap"
			class:text-success={item.statusObserved !== false && item.replicas > 0 && item.runningReplicas >= item.replicas}
			class:text-destructive={item.statusObserved !== false && item.replicas > 0 && item.runningReplicas < item.replicas}
			>{item.statusObserved === false ? m.unavailable() : item.runningReplicas} / {item.replicas}</span
		>
	{/if}
{/snippet}

{#snippet MobileReplicas(item: ServiceRow)}{@render ReplicasCell({ item })}{/snippet}

{#snippet ImageCell({ item }: { item: ServiceRow })}
	{@const image = parseImageRef(item.image)}
	<div class="max-w-64" title={item.image}>
		<span class="block truncate font-mono text-xs">{image.repo || m.common_na()}</span><span
			class="block truncate font-mono text-xs text-muted-foreground"
			>{image.tag}{item.image.includes('@') ? ` · ${item.image.split('@')[1]}` : ''}</span
		>{#if (item.runningImageVersions?.length ?? 0) > 1}<span class="text-xs text-warning">{m.mixed_image_versions()}</span>{/if}
	</div>
{/snippet}

{#snippet UpdatedCell({ item }: { item: ServiceRow })}
	<span class="whitespace-nowrap">{formatRelativeTime(item.updatedAt) || m.common_na()}</span>
{/snippet}
{#snippet CreatedCell({ item }: { item: ServiceRow })}{formatRelativeTime(item.createdAt) || m.common_na()}{/snippet}
{#snippet PortsCell({ item }: { item: ServiceRow })}{item.ports
		.slice(0, 2)
		.map((port) => `${port.publishedPort ?? ''}:${port.targetPort}/${port.protocol}`)
		.join(', ')}{item.ports.length > 2 ? ` +${item.ports.length - 2}` : ''}{/snippet}
{#snippet NodeCountCell({ item }: { item: ServiceRow })}{item.nodes.length}{/snippet}
{#snippet NetworkCountCell({ item }: { item: ServiceRow })}{item.networks.length}{/snippet}
{#snippet MountCountCell({ item }: { item: ServiceRow })}{item.mounts.filter((m) => m.type === 'volume').length} / {item.mounts.filter(
		(m) => m.type === 'bind'
	).length} / {item.mounts.filter((m) => m.type === 'tmpfs').length}{/snippet}

{#snippet ServiceMobileCardSnippet({
	item,
	mobileFieldVisibility
}: {
	item: ServiceRow;
	mobileFieldVisibility: MobileFieldVisibility;
})}
	<UniversalMobileCard
		{item}
		icon={() => ({
			component: ContainersIcon,
			variant: modeIconVariant(item.mode)
		})}
		title={(item: ServiceRow) => getShortName(item.name, item.stackName)}
		badges={[
			(item: ServiceRow) =>
				(mobileFieldVisibility['mode'] ?? true)
					? { variant: getSwarmServiceModeVariant(item.mode), text: getSwarmServiceModeLabel(item.mode) }
					: null
		]}
		fields={[
			{
				label: m.swarm_stack(),
				getValue: (item: ServiceRow) => (item.stackName ? item : null),
				type: 'component',
				component: MobileStack,
				show: showStack && (mobileFieldVisibility['stackName'] ?? true)
			},
			{
				label: m.swarm_replicas(),
				getValue: (item: ServiceRow) => item,
				type: 'component',
				component: MobileReplicas,
				icon: GlobeIcon,
				iconVariant: 'gray' as const,
				show: mobileFieldVisibility['replicas'] ?? true
			},
			{
				label: m.common_image(),
				getValue: (item: ServiceRow) => item.image || m.common_na(),
				type: 'mono',
				show: mobileFieldVisibility['image'] ?? true
			},
			{
				label: m.common_updated(),
				getValue: (item: ServiceRow) => formatRelativeTime(item.updatedAt) || m.common_na(),
				show: mobileFieldVisibility['updated'] ?? true
			}
		]}
		rowActions={RowActions}
	/>
{/snippet}

{#snippet RowActions({ item }: { item: ServiceRow })}
	<RowActionsMenu name={item.name}>
		<IfPermitted perm="swarm-services:read" envId={item.environmentId}>
			<DetailLinksMenu
				href={resourceTableHref(
					environmentHref(resolve('/(app)/swarm/services/[serviceId]', { serviceId: item.id }), item.environmentId),
					requestOptions.pagination?.page
				)}
				tabs={[
					{ value: 'overview', label: m.common_overview() },
					{ value: 'config', label: m.common_configuration(), icon: SettingsIcon },
					{
						value: 'logs',
						label: m.common_logs(),
						icon: FileTextIcon,
						visible: hasPermission('swarm-services:logs', item.environmentId)
					},
					{ value: 'tasks', label: m.tasks(), icon: ContainersIcon, related: true },
					{ value: 'network', label: m.resource_networks_cap(), icon: NetworksIcon, related: true },
					{ value: 'storage', label: m.resource_volumes_cap(), icon: VolumesIcon, related: true },
					{
						value: 'secrets',
						label: m.swarm_secrets_title(),
						icon: FileTextIcon,
						related: true,
						visible: hasPermission('swarm-secrets:list', item.environmentId)
					},
					{
						value: 'configs',
						label: m.swarm_configs_title(),
						icon: FileTextIcon,
						related: true,
						visible: hasPermission('swarm-configs:list', item.environmentId)
					},
					{
						value: 'owner-stack',
						label: m.swarm_stack(),
						icon: LayersIcon,
						related: true,
						visible: !!item.stackName && hasPermission('swarm-stacks:read', item.environmentId),
						href: resourceTableHref(
							environmentHref(resolve('/(app)/swarm/stacks/[name]', { name: item.stackName ?? '' }), item.environmentId),
							requestOptions.pagination?.page
						)
					}
				]}
			/>
		</IfPermitted>
		<IfPermitted perm="swarm-services:update" envId={item.environmentId}>
			<DropdownMenu.Separator />
			<IfPermitted perm="swarm-services:read" envId={item.environmentId}>
				<DropdownMenu.Item onclick={() => openService(item, 'rollback')}
					><RestartIcon class="size-4" />{m.swarm_service_rollback()}</DropdownMenu.Item
				>
				<DropdownMenu.Item onclick={() => openService(item, 'edit')}>
					<EditIcon class="size-4" />
					{m.common_edit()}
				</DropdownMenu.Item>
				{#if isSwarmServiceModeScalable(item.mode)}
					<DropdownMenu.Item onclick={() => openService(item, 'scale')}>
						<LayersIcon class="size-4" />
						{m.swarm_service_scale()}
					</DropdownMenu.Item>
				{/if}
			</IfPermitted>
			<DropdownMenu.Item
				onclick={() =>
					confirmForceUpdateService({
						serviceId: item.id,
						serviceName: item.name,
						environmentId: item.environmentId,
						refresh: onRetry
					})}
			>
				<RestartIcon class="size-4" />
				{m.swarm_service_force_update_short()}
			</DropdownMenu.Item>
		</IfPermitted>
		<IfPermitted perm="swarm-services:delete" envId={item.environmentId}>
			<DropdownMenu.Separator />
			<DropdownMenu.Item variant="destructive" onclick={() => handleDelete(item)} disabled={isAnyLoading}>
				<TrashIcon class="size-4" />
				{m.common_delete()}
			</DropdownMenu.Item>
		</IfPermitted>
	</RowActionsMenu>
{/snippet}

<ArcaneTable
	{persistKey}
	items={services}
	bind:requestOptions
	bind:mobileFieldVisibility
	selectionDisabled={true}
	{onRetry}
	{columns}
	{mobileFields}
	rowActions={RowActions}
	mobileCard={ServiceMobileCardSnippet}
/>
