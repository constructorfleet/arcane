<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';

	import ArcaneTable from '#lib/components/arcane-table/arcane-table.svelte';
	import DetailLinksMenu from '#lib/components/arcane-table/detail-links-menu.svelte';
	import { environmentColumns } from '#lib/components/arcane-table/environment-column.js';
	import type { ColumnSpec, MobileFieldVisibility } from '#lib/components/arcane-table/index.js';
	import { UniversalMobileCard } from '#lib/components/arcane-table/index.js';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { CalendarIcon, EditIcon, TrashIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { Paginated } from '#lib/types/base.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import type { SwarmKvSummary } from '#lib/types/ui/swarm.js';
	import { handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import type { EnvironmentPage, EnvironmentRow } from '#lib/utils/environments.js';
	import { environmentHref } from '#lib/utils/environments.js';
	import { formatRelativeTime } from '#lib/utils/formatting.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';
	import type { SwarmKvResource } from '#lib/utils/swarm-kv.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	let {
		resource,
		items,
		scope,
		requestOptions = $bindable(),
		onRetry,
		onEdit
	}: {
		resource: SwarmKvResource;
		items: KvPage;
		scope: string;
		requestOptions: SearchPaginationSortRequest;
		onRetry: () => Promise<unknown>;
		onEdit: (item: KvRow) => void;
	} = $props();

	type KvRow = EnvironmentRow<SwarmKvSummary>;
	type KvPage = EnvironmentPage<Paginated<SwarmKvSummary>>;

	let isDeleting = $state(false);
	let mobileFieldVisibility = $state<Record<string, boolean>>({});

	const queryClient = useQueryClient();
	const managerIds = $derived(
		[...new Set(items.data.map((item) => item.environmentId))].filter((id) => hasPermission('swarm-services:list', id)).sort()
	);
	const usage = createQuery(() => ({
		queryKey: ['swarm', 'service-reference-inventory', managerIds],
		enabled: managerIds.length > 0,
		queryFn: async () => {
			const results = await Promise.allSettled(
				managerIds.map(async (id) => ({ id, page: await swarmService.getServices({ pagination: { page: 1, limit: -1 } }, id) }))
			);
			return new Map(
				results.flatMap((result) => (result.status === 'fulfilled' ? [[result.value.id, result.value.page.data] as const] : []))
			);
		}
	}));
	function usageCount(item: KvRow) {
		const services = usage.data?.get(item.environmentId);
		if (!services || services.some((service) => service[resource.kind] === undefined)) return undefined;
		return services.filter((service) => service[resource.kind]?.some((ref) => ref.id === item.id)).length;
	}
	function detailHref(item: KvRow) {
		return resourceDetailHref(
			environmentHref(
				resource.kind === 'secrets'
					? resolve('/(app)/swarm/secrets/[id]', { id: item.id })
					: resolve('/(app)/swarm/configs/[id]', { id: item.id }),
				item.environmentId
			)
		);
	}

	function handleDelete(item: KvRow) {
		const name = item.spec.Name;
		openConfirmDialog({
			title: m.common_delete_title({ resource: resource.resource() }),
			message: resource.deleteConfirm(name),
			confirm: {
				label: m.common_delete(),
				destructive: true,
				action: async () => {
					isDeleting = true;
					await handleApiResultWithCallbacks({
						result: await tryCatch(swarmService.removeKvItem(resource.kind, item.id, item.environmentId)),
						message: resource.deleteFailed(name),
						setLoadingState: (value) => (isDeleting = value),
						onSuccess: async () => {
							toast.success(resource.deleteSuccess(name));
							await queryClient.invalidateQueries({ queryKey: queryKeys.swarm.listPrefix(scope, resource.kind) });
						}
					});
				}
			}
		});
	}

	const columns = $derived([
		{ accessorKey: 'id', title: m.common_id(), hidden: true },
		...environmentColumns<KvRow>(),
		{ id: 'name', accessorFn: (item) => item.spec.Name, title: m.common_name(), sortable: true, cell: NameCell },
		{
			id: 'owner',
			title: m.source(),
			accessorFn: (item) => item.spec.Labels?.['com.docker.stack.namespace'] ?? '',
			cell: OwnerCell
		},
		{ id: 'usedBy', title: m.used_by(), cell: UsedByCell },
		...(resource.kind === 'secrets'
			? [
					{
						id: 'driver',
						accessorFn: (item: KvRow) => item.spec.Driver?.Name ?? '',
						title: m.common_driver(),
						cell: DriverCell,
						hidden: true
					}
				]
			: []),

		{ accessorKey: 'createdAt', title: m.common_created(), sortable: true, cell: TimeCell, hidden: true },
		{ accessorKey: 'updatedAt', title: m.common_updated(), sortable: true, cell: TimeCell }
	] satisfies ColumnSpec<KvRow>[]);

	const mobileFields = [{ id: 'updatedAt', label: m.common_updated(), defaultVisible: true }];
</script>

{#snippet NameCell({ item }: { item: KvRow })}
	{#if hasPermission(resource.permissions.read, item.environmentId)}<a
			class="block max-w-56 truncate font-medium hover:underline"
			title={item.spec.Name}
			href={detailHref(item)}>{item.spec.Name}</a
		>{:else}{item.spec.Name}{/if}
{/snippet}

{#snippet OwnerCell({ item }: { item: KvRow })}
	{@const owner = item.spec.Labels?.['com.docker.stack.namespace']}
	{#if owner && hasPermission('swarm-stacks:read', item.environmentId)}<a
			class="block max-w-40 truncate text-primary-text hover:underline"
			title={owner}
			href={resourceDetailHref(environmentHref(resolve('/(app)/swarm/stacks/[name]', { name: owner }), item.environmentId))}
			>{owner}</a
		>{:else}{owner ?? m.common_unknown()}{/if}
{/snippet}
{#snippet UsedByCell({ item }: { item: KvRow })}{usageCount(item) ?? m.unavailable()}{/snippet}

{#snippet DriverCell({ item }: { item: KvRow })}
	<span class="text-sm text-muted-foreground">{item.spec.Driver?.Name || '-'}</span>
{/snippet}

{#snippet TimeCell({ value }: { value: unknown })}
	<span class="text-sm">{formatRelativeTime(String(value ?? '')) || m.common_unknown()}</span>
{/snippet}

{#snippet MobileCard({ item, mobileFieldVisibility }: { item: KvRow; mobileFieldVisibility: MobileFieldVisibility })}
	<UniversalMobileCard
		{item}
		icon={{ component: resource.icon, variant: 'purple' }}
		title={(item: KvRow) => item.spec.Name}
		footer={(mobileFieldVisibility['updatedAt'] ?? true)
			? {
					label: m.common_updated(),
					getValue: (item: KvRow) => formatRelativeTime(item.updatedAt) || m.common_unknown(),
					icon: CalendarIcon
				}
			: undefined}
		fields={[
			{ label: m.source(), getValue: (item: KvRow) => item.spec.Labels?.['com.docker.stack.namespace'] ?? m.common_unknown() },
			{ label: m.used_by(), getValue: (item: KvRow) => String(usageCount(item) ?? m.unavailable()) }
		]}
		rowActions={RowActions}
		href={(item: KvRow) => (hasPermission(resource.permissions.read, item.environmentId) ? detailHref(item) : undefined)}
	/>
{/snippet}

{#snippet RowActions({ item }: { item: KvRow })}
	<RowActionsMenu name={item.spec.Name}>
		{#if hasPermission(resource.permissions.read, item.environmentId)}<DetailLinksMenu
				href={detailHref(item)}
				tabs={[
					{ value: 'overview', label: m.common_overview() },
					{
						value: 'services',
						label: m.services(),
						related: true,
						visible: hasPermission('swarm-services:list', item.environmentId)
					},
					{
						value: 'owner-stack',
						label: m.swarm_stack(),
						related: true,
						visible: !!item.spec.Labels?.['com.docker.stack.namespace'] && hasPermission('swarm-stacks:read', item.environmentId),
						href: resourceDetailHref(
							environmentHref(
								resolve('/(app)/swarm/stacks/[name]', { name: item.spec.Labels?.['com.docker.stack.namespace'] ?? '' }),
								item.environmentId
							)
						)
					}
				]}
			/>{/if}
		{#if hasPermission(resource.permissions.update, item.environmentId)}
			<DropdownMenu.Item onclick={() => onEdit(item)}>
				<EditIcon class="size-4" />
				{m.common_edit()}
			</DropdownMenu.Item>
		{/if}
		{#if hasPermission(resource.permissions.delete, item.environmentId)}
			<DropdownMenu.Separator />
			<DropdownMenu.Item variant="destructive" onclick={() => handleDelete(item)} disabled={isDeleting}>
				<TrashIcon class="size-4" />
				{m.common_delete()}
			</DropdownMenu.Item>
		{/if}
	</RowActionsMenu>
{/snippet}

<ArcaneTable
	persistKey={resource.persistKey}
	{items}
	bind:requestOptions
	bind:mobileFieldVisibility
	selectionDisabled={true}
	{onRetry}
	{columns}
	{mobileFields}
	rowActions={RowActions}
	mobileCard={MobileCard}
/>
