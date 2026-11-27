<script module lang="ts">
	import { resolve } from '$app/paths';

	import type { NetworkSummary } from '#lib/types/network.js';
	import type { NetworksPaginatedResponse } from '#lib/types/ui/network.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';

	function networkDetailHref(item: NetworkSummary & { environmentId: string }) {
		const params = new URLSearchParams({ env: item.environmentId });
		if (item.clusterId && item.nodes?.length) {
			params.set('cluster', item.clusterId);
			params.set('agents', item.nodes.map((node) => node.id).join(','));
		}
		return resourceDetailHref(resolve(`/(app)/networks/[networkId]?${params}`, { networkId: item.id }));
	}

	function ipamValues(item: NetworkSummary, key: 'subnet' | 'gateway'): string[] {
		return (item.ipam?.config ?? []).map((c) => c[key]).filter((v): v is string => !!v);
	}
</script>

<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';

	import ArcaneTable from '#lib/components/arcane-table/arcane-table.svelte';
	import DetailLinksMenu from '#lib/components/arcane-table/detail-links-menu.svelte';
	import { environmentColumns } from '#lib/components/arcane-table/environment-column.js';
	import type { ColumnSpec, BulkAction } from '#lib/components/arcane-table/index.js';
	import { UniversalMobileCard } from '#lib/components/arcane-table/index.js';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import { EmptyState } from '#lib/components/states/index.js';
	import { swarmEnvironmentsQueryOptions } from '#lib/components/swarm/swarm-environments.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { DEFAULT_NETWORK_NAMES } from '#lib/constants.js';
	import { NetworksIcon, TrashIcon, LayersIcon, JobsIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { networkService } from '#lib/services/network-service.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import { activityToastOptions, extractActivityId } from '#lib/utils/activity-toast.js';
	import { handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { bulkConfirmAndRun } from '#lib/utils/bulk-actions.js';
	import { isSwarmGatewayBridge } from '#lib/utils/docker.js';
	import { type EnvironmentPage, type EnvironmentRow, hasShownPermission, rowKey } from '#lib/utils/environments.js';
	import { capitalizeFirstLetter, formatRelativeTime } from '#lib/utils/formatting.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	import { networkSource } from '../network-table.helpers';
	type FieldVisibility = Record<string, boolean>;
	type NetworkRow = EnvironmentRow<NetworkSummary>;
	type NetworkPage = EnvironmentPage<NetworksPaginatedResponse>;

	let {
		networks,
		selectedIds = $bindable(),
		requestOptions = $bindable(),
		onRetry,
		refreshError,
		showEnvironment = true
	}: {
		showEnvironment?: boolean;
		networks: NetworkPage;
		selectedIds: string[];
		requestOptions: SearchPaginationSortRequest;
		onRetry: () => Promise<unknown>;
		refreshError?: unknown;
	} = $props();

	const queryClient = useQueryClient();
	const sourceManagers = createQuery(() => swarmEnvironmentsQueryOptions('swarm-stacks:read', { all: true }));

	const isLoading = $state({
		remove: false
	});

	async function handleDeleteNetwork(item: NetworkRow) {
		const { id, name, environmentId } = item;
		const safeName = name?.trim() || m.common_unknown();
		if (item.inUse) {
			toast.error(m.network_has_observed_consumers());
			return;
		}
		if (DEFAULT_NETWORK_NAMES.has(name)) {
			toast.error(m.networks_cannot_delete_default({ name: safeName }));
			return;
		}
		openConfirmDialog({
			title: m.common_delete_title({ resource: m.resource_network() }),
			message: [
				m.networks_delete_confirm_message({ name: safeName }),
				m.network_query_agent_target({ agent: item.environmentName, agentId: environmentId }),
				...(item.scope === 'swarm' ? [m.swarm_network_usage_warning()] : [])
			].join('\n\n'),
			confirm: {
				label: m.common_delete(),
				destructive: true,
				action: async () => {
					isLoading.remove = true;
					await handleApiResultWithCallbacks({
						result: await tryCatch(networkService.deleteNetwork(id, environmentId)),
						message: m.common_delete_failed({ resource: `${m.resource_network()} "${safeName}"` }),
						setLoadingState: (value) => (isLoading.remove = value),
						onSuccess: async (data) => {
							toast.success(
								m.common_delete_success({ resource: `${m.resource_network()} "${safeName}"` }),
								activityToastOptions(extractActivityId(data))
							);
							await queryClient.invalidateQueries({ queryKey: queryKeys.networks.listPrefix(environmentStore.shownKey) });
						}
					});
				}
			}
		});
	}

	function handleDeleteSelectedNetworks(ids: string[]) {
		const selectedNetworkList = networks.data.filter((n) => ids.includes(rowKey(n)));
		if (selectedNetworkList.some((item) => item.inUse)) {
			toast.error(m.network_has_observed_consumers());
			return;
		}
		const defaultNetworks = selectedNetworkList.filter((n) => n.isDefault);

		if (defaultNetworks.length > 0) {
			const names = defaultNetworks.map((n) => n.name ?? m.common_unknown()).join(', ');
			toast.error(m.networks_cannot_delete_default_many({ names }));
			return;
		}

		bulkConfirmAndRun({
			ids,
			title: m.networks_delete_selected_title({ count: ids.length }),
			message: [
				m.networks_delete_selected_message({ count: ids.length }),
				...(selectedNetworkList.some((item) => item.scope === 'swarm') ? [m.swarm_network_usage_warning()] : [])
			].join('\n\n'),
			confirmLabel: m.common_delete(),
			destructive: true,
			runInEnvironment: (id, environmentId) => networkService.deleteNetwork(id, environmentId),
			messages: {
				success: (count) => m.common_bulk_delete_success({ count, resource: m.resource_networks_cap() }),
				partial: (success, total, failed) =>
					m.common_bulk_delete_partial({ success, total, failed, resource: m.resource_networks_cap() }),
				failure: () => m.common_bulk_delete_failed({ count: ids.length, resource: m.resource_networks_cap() })
			},
			setLoading: (loading) => (isLoading.remove = loading),
			onComplete: async (result) => {
				if (result.success > 0)
					await queryClient.invalidateQueries({ queryKey: queryKeys.networks.listPrefix(environmentStore.shownKey) });
			},
			clearSelection: () => (selectedIds = [])
		});
	}

	const isAnyLoading = $derived(Object.values(isLoading).some((l) => l));
	const canDeleteNetwork = $derived(hasShownPermission('networks:delete'));

	function getNetworkStatusBadge(
		item: NetworkRow,
		visible: boolean
	): { variant: 'gray' | 'green' | 'amber'; text: string } | null {
		if (!visible) return null;
		if (isSwarmGatewayBridge(item)) return { variant: 'gray', text: m.network_swarm_gateway() };
		if (item.isDefault || DEFAULT_NETWORK_NAMES.has(item.name)) {
			return { variant: 'gray', text: m.networks_predefined() };
		}
		if (item.inUse) return { variant: 'green', text: m.common_in_use() };
		return { variant: 'amber', text: m.common_unused() };
	}

	function sourceText(item: NetworkSummary): string {
		const source = networkSource(item);
		switch (source.kind) {
			case 'stack':
				return source.name;
			case 'docker':
				return m.docker();
			case 'system':
				return m.system();
			default:
				return m.common_unknown();
		}
	}

	function stackOwnerHref(item: NetworkRow): string | undefined {
		const source = networkSource(item);
		if (source.kind !== 'stack') return undefined;
		const manager = sourceManagers.data?.managers.find(
			(manager) =>
				sourceManagers.data?.clusterIds?.get(manager.id) === source.clusterId && hasPermission('swarm-stacks:read', manager.id)
		);
		if (!manager) return undefined;
		return resourceDetailHref(
			resolve(
				`/(app)/swarm/stacks/[name]?env=${encodeURIComponent(manager.id)}&cluster=${encodeURIComponent(source.clusterId)}`,
				{ name: source.name }
			)
		);
	}

	const columns = $derived([
		...(showEnvironment
			? environmentColumns<NetworkRow>().map((column) => ({ ...column, title: m.query_connection(), hidden: true }))
			: []),
		{ accessorKey: 'id', title: m.common_id(), cell: IdCell, hidden: true },
		{ accessorKey: 'name', title: m.common_name(), sortable: true, cell: NameCell },
		{ accessorKey: 'inUse', title: m.common_status(), sortable: true, cell: StatusCell },
		{ id: 'source', title: m.source(), cell: SourceCell },
		{ accessorKey: 'driver', title: m.driver_scope(), sortable: true, cell: DriverCell },
		{ id: 'addressing', title: m.addressing(), cell: AddressingCell },
		{ accessorKey: 'created', title: m.common_created(), hidden: true, sortable: true, cell: CreatedCell }
	] satisfies ColumnSpec<NetworkRow>[]);

	const mobileFields = [
		{ id: 'id', label: m.common_id(), defaultVisible: false },
		{ id: 'inUse', label: m.common_status(), defaultVisible: true },
		{ id: 'source', label: m.source(), defaultVisible: true },
		{ id: 'driver', label: m.driver_scope(), defaultVisible: true },
		{ id: 'addressing', label: m.addressing(), defaultVisible: true },
		{ id: 'created', label: m.common_created(), defaultVisible: false }
	];

	const bulkActions = $derived.by<BulkAction[]>(() => {
		if (!canDeleteNetwork) return [];
		return [
			{
				id: 'remove',
				label: m.common_remove_selected_count({ count: selectedIds?.length ?? 0 }),
				action: 'remove',
				onclick: handleDeleteSelectedNetworks,
				loading: isLoading.remove,
				disabled: isLoading.remove,
				icon: TrashIcon
			}
		];
	});

	let mobileFieldVisibility = $state<Record<string, boolean>>({});
</script>

{#snippet NameCell({ item }: { item: NetworkRow })}
	<a class="font-medium hover:underline" href={networkDetailHref(item)}>{item.name}</a>
{/snippet}

{#snippet IdCell({ item }: { item: NetworkRow })}
	<span class="truncate font-mono text-sm">{String(item.id)}</span>
{/snippet}

{#snippet SourceCell({ item }: { item: NetworkRow })}{@const href = stackOwnerHref(item)}{#if href}<a
			class="text-sm text-primary-text hover:underline"
			{href}>{sourceText(item)}</a
		>{:else}<span class="text-sm text-muted-foreground">{sourceText(item)}</span>{/if}
{/snippet}
{#snippet DriverCell({ item }: { item: NetworkRow })}
	<div class="flex flex-col text-sm">
		<span>{capitalizeFirstLetter(item.driver)}</span><span class="text-xs text-muted-foreground"
			>{capitalizeFirstLetter(item.scope)}</span
		>
	</div>
{/snippet}
{#snippet AddressingCell({ item }: { item: NetworkRow })}
	{@const configs = item.ipam?.config ?? []}
	<div class="flex flex-col gap-1 font-mono text-xs">
		{#each configs.slice(0, 2) as config, index (index)}<div>
				<div>{config.subnet ?? '—'}</div>
				<div class="text-muted-foreground">{config.gateway ?? '—'}</div>
			</div>{/each}{#if configs.length > 2}<span class="text-muted-foreground">+{configs.length - 2}</span
			>{/if}{#if configs.length === 0}—{/if}
	</div>
{/snippet}
{#snippet CreatedCell({ item }: { item: NetworkRow })}<span title={item.created}>{formatRelativeTime(item.created)}</span
	>{/snippet}

{#snippet StatusCell({ item }: { item: NetworkRow })}
	{@const status = getNetworkStatusBadge(item, true)}
	{#if status}<Badge
			variant={status.variant}
			minWidth="20"
			title={item.scope === 'swarm' && !item.isDefault ? m.network_list_usage_scope() : undefined}>{status.text}</Badge
		>{/if}
{/snippet}

{#snippet NetworkMobileCardSnippet({ item, mobileFieldVisibility }: { item: NetworkRow; mobileFieldVisibility: FieldVisibility })}
	<UniversalMobileCard
		{item}
		icon={(item: NetworkRow) => ({
			component: NetworksIcon,
			variant: item.inUse ? 'emerald' : 'amber'
		})}
		title={(item: NetworkRow) => item.name}
		subtitle={(item: NetworkRow) => ((mobileFieldVisibility['id'] ?? true) ? item.id : null)}
		badges={[(item: NetworkRow) => getNetworkStatusBadge(item, mobileFieldVisibility['inUse'] ?? true)]}
		fields={[
			{
				label: m.source(),
				getValue: (item: NetworkRow) => sourceText(item),
				show: mobileFieldVisibility['source'] ?? true
			},
			{
				label: m.driver_scope(),
				getValue: (item: NetworkRow) => `${capitalizeFirstLetter(item.driver)} / ${capitalizeFirstLetter(item.scope)}`,
				show: mobileFieldVisibility['driver'] ?? true
			},
			{
				label: m.addressing(),
				getValue: (item: NetworkRow) =>
					(item.ipam?.config ?? [])
						.slice(0, 2)
						.map((config) => `${config.subnet ?? '—'} / ${config.gateway ?? '—'}`)
						.join(', ') + (ipamValues(item, 'subnet').length > 2 ? ` +${ipamValues(item, 'subnet').length - 2}` : '') || '—',
				show: mobileFieldVisibility['addressing'] ?? true
			},
			{
				label: m.common_created(),
				getValue: (item: NetworkRow) => formatRelativeTime(item.created),
				show: mobileFieldVisibility['created'] ?? false
			}
		]}
		rowActions={RowActions}
		href={() => networkDetailHref(item)}
	/>
{/snippet}

{#snippet RowActions({ item }: { item: NetworkRow })}
	<RowActionsMenu name={item.name}>
		<DetailLinksMenu
			disabled={isAnyLoading}
			href={networkDetailHref(item)}
			tabs={[
				{ value: 'overview', label: m.common_overview() },
				{
					value: 'owning-stack',
					label: m.swarm_stack(),
					icon: LayersIcon,
					related: true,
					href: stackOwnerHref(item),
					visible: !!stackOwnerHref(item)
				},
				{ value: 'services', label: m.services(), icon: LayersIcon, related: true, visible: item.scope === 'swarm' },
				{ value: 'tasks', label: m.tasks(), icon: JobsIcon, related: true, visible: item.scope === 'swarm' }
			]}
		/>

		{#if hasPermission('networks:delete', item.environmentId)}
			<DropdownMenu.Separator />

			<DropdownMenu.Item
				variant="destructive"
				onclick={() => handleDeleteNetwork(item)}
				disabled={isAnyLoading || item.inUse || item.isDefault || DEFAULT_NETWORK_NAMES.has(item.name)}
			>
				<TrashIcon class="size-4" />
				{m.common_delete()}
			</DropdownMenu.Item>
		{/if}
	</RowActionsMenu>
{/snippet}

<ArcaneTable
	persistKey={showEnvironment ? 'arcane-networks-table' : 'arcane-network-agent-scoped-table'}
	items={networks}
	bind:requestOptions
	bind:selectedIds
	bind:mobileFieldVisibility
	{bulkActions}
	selectionDisabled={!canDeleteNetwork}
	{onRetry}
	{refreshError}
	{columns}
	{mobileFields}
	rowActions={RowActions}
	mobileCard={NetworkMobileCardSnippet}
>
	{#snippet emptyState()}
		<EmptyState icon={NetworksIcon} title={m.no_networks_yet()} description={m.create_network_to_connect_containers()} />
	{/snippet}
</ArcaneTable>
