<script module lang="ts">
	import { resolve } from '$app/paths';

	import type { Volume } from '#lib/types/volume.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';

	function volumeDetailHref(item: Volume & { environmentId: string }, tab = 'overview') {
		const params = new URLSearchParams({ env: item.environmentId, tab });
		if (item.logicalId && item.clusterId && item.nodes?.length) {
			params.set('cluster', item.clusterId);
			params.set('driver', item.driver);
			params.set('agents', item.nodes.map((node) => node.id).join(','));
		}
		return resourceDetailHref(resolve(`/(app)/volumes/[volumeName]?${params}`, { volumeName: item.name }));
	}
</script>

<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import type { ColumnVisibilityState } from '@tanstack/table-core';
	import { onMount, untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { SvelteMap } from 'svelte/reactivity';

	import ArcaneTable from '#lib/components/arcane-table/arcane-table.svelte';
	import InUseStatus from '#lib/components/arcane-table/cells/in-use-status.svelte';
	import DetailLinksMenu from '#lib/components/arcane-table/detail-links-menu.svelte';
	import { environmentColumns } from '#lib/components/arcane-table/environment-column.js';
	import type { ColumnSpec, MobileFieldVisibility, BulkAction } from '#lib/components/arcane-table/index.js';
	import { UniversalMobileCard } from '#lib/components/arcane-table/index.js';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import ResourceName from '#lib/components/resource-name.svelte';
	import { EmptyState } from '#lib/components/states/index.js';
	import { swarmEnvironmentsQueryOptions } from '#lib/components/swarm/swarm-environments.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import {
		LayersIcon,
		ContainersIcon,
		TrashIcon,
		BackupIcon,
		VolumesIcon,
		CalendarIcon,
		EditIcon,
		FolderOpenIcon
	} from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import type { APIRequestConfig } from '#lib/services/api-service.js';
	import { emptySettings, settingsService } from '#lib/services/settings-service.js';
	import { volumeService } from '#lib/services/volume-service.js';
	import { environmentStore, type EnvironmentRef } from '#lib/stores/environment.store.svelte.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import type { ResolvedSettings } from '#lib/types/ui/settings.js';
	import type { VolumesPaginatedResponse } from '#lib/types/ui/volume.js';
	import type { VolumeSizeInfo } from '#lib/types/volume.js';
	import { activityToastOptions, extractActivityId } from '#lib/utils/activity-toast.js';
	import { handleApiResultWithCallbacks, extractApiErrorMessage } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { bulkConfirmAndRun } from '#lib/utils/bulk-actions.js';
	import {
		type EnvironmentFailure,
		type EnvironmentPage,
		type EnvironmentRow,
		environmentRows,
		hasShownPermission,
		readableEnvironments,
		readAcrossEnvironments,
		readEachEnvironment,
		rowKey,
		withFailedEnvironments
	} from '#lib/utils/environments.js';
	import { formatRelativeTime } from '#lib/utils/formatting.js';
	import { bytes } from '#lib/utils/formatting.js';
	import { inUseBadge } from '#lib/utils/mobile-card-badges.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	import RenameVolumeDialog from './rename-volume-dialog.svelte';
	import VolumeFilesystemUsage from './volume-filesystem-usage.svelte';

	type VolumeRow = EnvironmentRow<Volume>;

	let {
		volumes,
		selectedIds = $bindable(),
		requestOptions = $bindable(),
		onRetry,
		refreshError,
		showEnvironment = true
	}: {
		showEnvironment?: boolean;
		volumes: EnvironmentPage<VolumesPaginatedResponse>;
		selectedIds: string[];
		requestOptions: SearchPaginationSortRequest;
		onRetry: () => Promise<unknown>;
		refreshError?: unknown;
	} = $props();

	const isLoading = $state({
		removing: false,
		renaming: false
	});
	let volumeToRename = $state<VolumeRow | null>(null);
	let renameDialogOpen = $state(false);
	let renameSession = $state(0);
	let renameEnvironmentId = $state('0');

	const scope = $derived(environmentStore.shownKey);
	const canDeleteVolume = $derived(hasShownPermission('volumes:delete'));
	const canRenameVolume = $derived(hasShownPermission('volumes:rename'));
	let tablePreferences = $state<{ persistCustomSettings(settings: Record<string, unknown>): void }>();
	let customSettings = $state<Record<string, unknown>>({});
	let showInternal = $derived((customSettings['showInternalVolumes'] as boolean) ?? false);

	async function invalidateVolumes() {
		await Promise.all([
			queryClient.invalidateQueries({ queryKey: queryKeys.volumes.listPrefix(scope) }),
			queryClient.invalidateQueries({ queryKey: queryKeys.volumes.sizes(scope) })
		]);
	}

	async function retryFailedEnvironments() {
		await Promise.all([
			onRetry(),
			sizesEnabled && sizesQuery.refetch({ cancelRefetch: false }),
			(backupVolumeNamesQuery.isError || backupVolumeNamesQuery.data?.failures.length) && backupVolumeNamesQuery.refetch()
		]);
	}

	function getCurrentLimit() {
		return requestOptions?.pagination?.limit ?? volumes?.pagination?.itemsPerPage ?? 20;
	}

	function setShowInternal(value: boolean) {
		const currentSetting = (customSettings['showInternalVolumes'] as boolean) ?? false;
		const currentRequest = requestOptions?.includeInternal ?? false;
		if (value === currentSetting && value === currentRequest) return;

		customSettings = { ...customSettings, showInternalVolumes: value };
		tablePreferences?.persistCustomSettings(customSettings);
		requestOptions = {
			...requestOptions,
			includeInternal: value,
			pagination: { page: 1, limit: getCurrentLimit() }
		};
	}

	let columnVisibility = $state<ColumnVisibilityState>({});
	let tablePreferencesReady = $state(false);
	const queryClient = useQueryClient();
	const ownerManagers = createQuery(() => swarmEnvironmentsQueryOptions('swarm-stacks:read', { all: true }));

	async function readBackupVolumeNames(
		environments: EnvironmentRef[],
		fetch: (environmentScope: string, config: APIRequestConfig) => Promise<ResolvedSettings>
	): Promise<{ names: [string, string][]; failures: EnvironmentFailure[] }> {
		if (environments.length === 0) return { names: [], failures: [] };
		const result = await tryCatch(readEachEnvironment(environments, fetch, emptySettings));
		if (result.error !== null) {
			const message = extractApiErrorMessage(result.error);
			return { names: [], failures: environments.map((environment) => ({ environment, message })) };
		}
		return {
			names: result.data.byEnvironment.map(({ environment, data }) => [
				environment.id,
				data.runtime.backupVolumeName || 'arcane-backups'
			]),
			failures: result.data.failedEnvironments
		};
	}

	const backupVolumeNamesQuery = createQuery(() => {
		const environments = readableEnvironments('volumes:list');
		return {
			queryKey: queryKeys.volumes.backupVolumeNames(scope),
			queryFn: async () => {
				const settingsReadable = environments.filter((env) => hasPermission('environment-settings:read', env.id));
				const reads = await Promise.all([
					readBackupVolumeNames(settingsReadable, (environmentScope, config) =>
						settingsService.getSettingsForEnvironment(environmentScope, config)
					),
					readBackupVolumeNames(
						environments.filter((env) => !settingsReadable.includes(env)),
						(environmentScope, config) => settingsService.getPublicSettingsForEnvironment(environmentScope, config)
					)
				]);
				const failures = reads.flatMap((read) => read.failures);
				if (failures.length > 0 && failures.length === environments.length) {
					throw new Error(failures.map(({ environment, message }) => `${environment.name}: ${message}`).join('\n'));
				}
				return { names: new Map(reads.flatMap((read) => read.names)), failures };
			},
			enabled: environments.length > 0,
			refetchOnWindowFocus: false
		};
	});
	const isBackupVolumeName = (name: string, environmentId: string) => {
		const configured = backupVolumeNamesQuery.data?.names.get(environmentId);
		return configured === undefined || name === configured;
	};
	const isBackupVolume = (item: VolumeRow) => isBackupVolumeName(item.name, item.environmentId);
	const sizesEnabled = $derived(tablePreferencesReady && columnVisibility['size'] === true && hasShownPermission('volumes:read'));
	const sizesQuery = createQuery(() => {
		const queryScope = scope;
		const environments = readableEnvironments('volumes:list');
		return {
			queryKey: queryKeys.volumes.sizes(queryScope),
			queryFn: async () => {
				const readable = environments.filter((env) => hasPermission('volumes:read', env.id));
				if (readable.length === 0) return [];
				const { value } = await readAcrossEnvironments(readable, (scope, config) => volumeService.getVolumeSizes(scope, config));
				return environmentRows(value, readable);
			},
			enabled: sizesEnabled,
			refetchOnWindowFocus: false,
			retry: false
		};
	});
	const sizesMap = $derived(
		new Map<string, VolumeSizeInfo>(
			sizesEnabled ? (sizesQuery.data ?? []).map((size) => [rowKey({ ...size, id: size.name }), size]) : []
		)
	);
	const sizesLoading = $derived(sizesEnabled && sizesQuery.isFetching);

	onMount(() => {
		const cache = queryClient.getQueryCache();
		return cache.subscribe((event) => {
			if (event.type !== 'updated') return;
			if (event.query !== cache.find({ queryKey: queryKeys.volumes.sizes(scope), exact: true })) return;
			if (event.action.type === 'success' && sizesEnabled && requestOptions?.sort?.column === 'size') {
				void tryCatch(
					queryClient.invalidateQueries({ queryKey: queryKeys.volumes.listPrefix(scope) }, { throwOnError: true })
				).then((result) =>
					handleApiResultWithCallbacks({ result, message: m.common_refresh_failed({ resource: m.resource_volumes_cap() }) })
				);
			}
		});
	});

	async function handleRemoveVolumeConfirm(name: string, environmentId: string) {
		const safeName = name?.trim() || m.common_unknown();
		if (isBackupVolumeName(safeName, environmentId)) return;
		openConfirmDialog({
			title: m.common_remove_title({ resource: m.resource_volume() }),
			message: m.common_remove_confirm({ resource: `${m.resource_volume()} "${safeName}"` }),
			confirm: {
				label: m.common_remove(),
				destructive: true,
				action: async () => {
					isLoading.removing = true;
					await handleApiResultWithCallbacks({
						result: await tryCatch(volumeService.deleteVolume(safeName, environmentId)),
						message: m.common_remove_failed({ resource: `${m.resource_volume()} "${safeName}"` }),
						setLoadingState: (value) => (isLoading.removing = value),
						onSuccess: async (data) => {
							toast.success(
								m.common_remove_success({ resource: `${m.resource_volume()} "${safeName}"` }),
								activityToastOptions(extractActivityId(data))
							);
							await invalidateVolumes();
						}
					});
				}
			}
		});
	}

	function handleRenameVolume(item: VolumeRow) {
		volumeToRename = item;
		renameEnvironmentId = item.environmentId;
		renameSession += 1;
		renameDialogOpen = true;
	}

	async function handleRenameSubmit(newName: string) {
		if (!volumeToRename) return;
		const oldName = volumeToRename.name;
		isLoading.renaming = true;
		await handleApiResultWithCallbacks({
			result: await tryCatch(volumeService.renameVolume(oldName, { name: newName }, renameEnvironmentId)),
			message: m.volumes_rename_failed({ name: oldName }),
			setLoadingState: (value) => (isLoading.renaming = value),
			onSuccess: async (data) => {
				toast.success(m.volumes_rename_success({ oldName, newName }), activityToastOptions(extractActivityId(data)));
				renameDialogOpen = false;
				await invalidateVolumes();
			}
		});
	}

	const selectedVolumes = new SvelteMap<string, VolumeRow>();
	$effect(() => {
		const keys = new Set(selectedIds ?? []);
		const rows = volumes.data;
		untrack(() => {
			for (const key of selectedVolumes.keys()) if (!keys.has(key)) selectedVolumes.delete(key);
			for (const row of rows) if (keys.has(rowKey(row))) selectedVolumes.set(rowKey(row), row);
		});
	});
	function canDeleteSelectedVolume(key: string) {
		const volume = selectedVolumes.get(key);
		return (
			!!volume &&
			!volume.logicalId &&
			!volume.inUse &&
			hasPermission('volumes:delete', volume.environmentId) &&
			!isBackupVolume(volume)
		);
	}
	function handleDeleteSelected(ids: string[]) {
		// Retain selected metadata across pages and exclude unavailable or protected rows.
		const idsToDelete = ids.filter(canDeleteSelectedVolume);

		bulkConfirmAndRun({
			ids: idsToDelete,
			title: m.volumes_remove_selected_title({ count: idsToDelete.length }),
			message: m.volumes_remove_selected_message({ count: idsToDelete.length }),
			confirmLabel: m.common_remove(),
			destructive: true,
			runInEnvironment: (id, environmentId) => volumeService.deleteVolume(id, environmentId),
			messages: {
				success: (count) => m.common_bulk_remove_success({ count, resource: m.resource_volumes_cap() }),
				partial: (success, total, failed) =>
					m.common_bulk_remove_partial({ success, total, failed, resource: m.resource_volumes_cap() }),
				failure: () => m.common_bulk_remove_failed({ count: idsToDelete.length, resource: m.resource_volumes_cap() })
			},
			setLoading: (loading) => (isLoading.removing = loading),
			onComplete: async (result) => {
				if (result.success > 0) await invalidateVolumes();
			},
			clearSelection: () => (selectedIds = [])
		});
	}

	function stackOwnerHref(item: VolumeRow): string | undefined {
		const name = item.clusterId ? item.labels?.['com.docker.stack.namespace'] : undefined;
		if (!name) return undefined;
		const manager = ownerManagers.data?.managers.find(
			(manager) =>
				ownerManagers.data?.clusterIds?.get(manager.id) === item.clusterId && hasPermission('swarm-stacks:read', manager.id)
		);
		if (!manager) return undefined;
		return resourceDetailHref(
			resolve(
				`/(app)/swarm/stacks/[name]?env=${encodeURIComponent(manager.id)}&cluster=${encodeURIComponent(item.clusterId ?? '')}`,
				{ name }
			)
		);
	}

	const columns = $derived([
		...(showEnvironment
			? environmentColumns<VolumeRow>().map((column) => ({
					...column,
					hidden: false,
					cell: AttachmentCell,
					cellComponent: undefined
				}))
			: []),
		{ accessorKey: 'id', title: m.common_id(), hidden: true },
		{ id: 'owner', title: m.owner(), hidden: true, cell: OwnerCell },
		{ accessorKey: 'name', title: m.common_name(), sortable: true, cell: NameCell },
		{ accessorKey: 'inUse', title: m.common_status(), sortable: true, cell: StatusCell },
		{ accessorKey: 'size', title: m.common_size(), hidden: true, sortable: true, cell: SizeCell, align: 'right' },
		{ id: 'filesystemUsage', title: m.filesystem_disk_usage(), cell: FilesystemUsageCell, align: 'right' },
		{ accessorKey: 'createdAt', title: m.common_created(), hidden: true, sortable: true, cell: CreatedCell },
		{ accessorKey: 'driver', title: m.common_driver(), sortable: true },
		{ accessorKey: 'mountpoint', title: m.common_mountpoint(), hidden: true },
		{ accessorKey: 'scope', title: m.docker_scope(), hidden: true }
	] satisfies ColumnSpec<VolumeRow>[]);

	const mobileFields = [
		{ id: 'id', label: m.common_id(), defaultVisible: false },
		{ id: 'owner', label: m.owner(), defaultVisible: false },
		{ id: 'mountpoint', label: m.common_mountpoint(), defaultVisible: false },
		{ id: 'scope', label: m.docker_scope(), defaultVisible: false },
		{ id: 'inUse', label: m.common_status(), defaultVisible: true },
		{ id: 'size', label: m.common_size(), defaultVisible: false },
		{ id: 'filesystemUsage', label: m.filesystem_disk_usage(), defaultVisible: true },
		{ id: 'createdAt', label: m.common_created(), defaultVisible: false },
		{ id: 'driver', label: m.common_driver(), defaultVisible: true }
	];

	// Keep bulk removal eligibility consistent with the row action.
	const deletableSelectedIds = $derived((selectedIds ?? []).filter(canDeleteSelectedVolume));

	const bulkActions = $derived.by<BulkAction[]>(() => [
		{
			id: 'remove',
			label: m.common_remove_selected_count({ count: deletableSelectedIds.length }),
			action: 'remove',
			onclick: () => handleDeleteSelected(deletableSelectedIds),
			loading: isLoading.removing,
			disabled: !canDeleteVolume || isLoading.removing || deletableSelectedIds.length === 0,
			icon: TrashIcon
		}
	]);

	let mobileFieldVisibility = $state<Record<string, boolean>>({});
</script>

{#snippet OwnerCell({ item }: { item: VolumeRow })}{@const href = stackOwnerHref(item)}{@const name = item.clusterId
		? item.labels?.['com.docker.stack.namespace']
		: undefined}{#if href}<a class="text-primary-text hover:underline" {href}>{name}</a>{:else}<span
			class="text-sm text-muted-foreground">{name ?? m.common_unknown()}</span
		>{/if}
{/snippet}
{#snippet AttachmentCell({ item }: { item: VolumeRow })}<div class="flex flex-col text-sm">
		<span>{item.environmentName}</span><span class="text-xs text-muted-foreground"
			>{item.inUse ? m.common_in_use() : m.common_unused()}</span
		>
	</div>{/snippet}
{#snippet NameCell({ item }: { item: VolumeRow })}
	<ResourceName name={item.name} href={volumeDetailHref(item)} class="font-mono text-xs-plus leading-5 font-medium" />
{/snippet}

{#snippet StatusCell({ item }: { item: VolumeRow })}
	<InUseStatus inUse={item.inUse} />
{/snippet}

{#snippet SizeCell({ item }: { item: VolumeRow })}
	{@const sizeInfo = sizesMap.get(rowKey({ id: item.name, environmentId: item.environmentId, environmentTagged: true }))}
	{#if item.options?.['size']?.trim()}
		<span class="text-sm tabular-nums">{item.options['size'].trim()}</span>
	{:else if item.driver !== 'local'}
		<span class="text-sm text-muted-foreground">-</span>
	{:else if sizeInfo && sizeInfo.size >= 0}
		<span class="text-sm tabular-nums">{bytes.format(sizeInfo.size, { unitSeparator: ' ' })}</span>
	{:else if sizesLoading && sizesMap.size === 0}
		{#if item.size > 0}
			<span class="text-sm tabular-nums">{bytes.format(item.size, { unitSeparator: ' ' })}</span>
		{:else}
			<span class="flex items-center justify-end gap-1 text-sm text-muted-foreground">
				<Spinner class="size-4" />
			</span>
		{/if}
	{:else if item.size > 0}
		<span class="text-sm tabular-nums">{bytes.format(item.size, { unitSeparator: ' ' })}</span>
	{:else if sizesEnabled && sizesQuery.isError}
		<span
			class="text-sm text-muted-foreground"
			title={m.volumes_size_calculation_failed({ error: extractApiErrorMessage(sizesQuery.error) })}>-</span
		>
	{:else}
		<span class="text-sm text-muted-foreground" title={sizesEnabled ? m.volumes_size_not_reported() : undefined}>-</span>
	{/if}
{/snippet}

{#snippet CreatedCell({ value }: { value: unknown })}
	<span title={String(value)}>{formatRelativeTime(String(value))}</span>
{/snippet}

{#snippet FilesystemUsageCell({ item }: { item: VolumeRow })}
	{#if hasPermission('volumes:read', item.environmentId) && (!item.logicalId || item.workspaceState === 'active') && (item.runningHelpers ?? 0) > 0}
		<VolumeFilesystemUsage environmentId={item.environmentId} volumeName={item.name} />
	{:else}
		<span class="text-sm text-muted-foreground">-</span>
	{/if}
{/snippet}

{#snippet VolumeMobileCardSnippet({
	item,
	mobileFieldVisibility
}: {
	item: VolumeRow;
	mobileFieldVisibility: MobileFieldVisibility;
})}
	<UniversalMobileCard
		{item}
		icon={(item) => ({
			component: VolumesIcon,
			variant: item.inUse ? 'emerald' : 'amber'
		})}
		title={(item) => item.name}
		subtitle={(item) => ((mobileFieldVisibility['id'] ?? true) ? item.id : null)}
		badges={[inUseBadge(mobileFieldVisibility['inUse'] ?? true)]}
		fields={[
			{
				label: m.owner(),
				getValue: (item: VolumeRow) =>
					item.clusterId ? (item.labels?.['com.docker.stack.namespace'] ?? m.common_unknown()) : m.common_unknown(),
				show: mobileFieldVisibility['owner'] ?? false
			},
			{
				label: m.common_mountpoint(),
				getValue: (item: VolumeRow) => item.mountpoint,
				show: mobileFieldVisibility['mountpoint'] ?? false
			},
			{ label: m.docker_scope(), getValue: (item: VolumeRow) => item.scope, show: mobileFieldVisibility['scope'] ?? false },
			{
				label: m.common_driver(),
				getValue: (item: VolumeRow) => item.driver,
				icon: VolumesIcon,
				iconVariant: 'gray' as const,
				show: mobileFieldVisibility['driver'] ?? true
			},
			{
				label: m.common_size(),
				getValue: (item: VolumeRow) => ({ item }),
				type: 'component',
				component: SizeCell,
				show: mobileFieldVisibility['size'] ?? false
			},
			{
				label: m.filesystem_disk_usage(),
				getValue: (item: VolumeRow) => ({ item }),
				type: 'component',
				component: FilesystemUsageCell,
				show: mobileFieldVisibility['filesystemUsage'] ?? true
			}
		]}
		footer={(mobileFieldVisibility['createdAt'] ?? false)
			? {
					label: m.common_created(),
					getValue: (item) => formatRelativeTime(String(item.createdAt)),
					icon: CalendarIcon
				}
			: undefined}
		rowActions={RowActions}
		href={() => volumeDetailHref(item)}
	/>
{/snippet}

{#snippet RowActions({ item }: { item: VolumeRow })}
	<RowActionsMenu name={item.name}>
		<DetailLinksMenu
			href={volumeDetailHref(item)}
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
				{ value: 'containers', label: m.containers(), icon: ContainersIcon, related: true },
				{ value: 'services', label: m.services(), icon: LayersIcon, related: true },
				{
					value: 'backups',
					label: m.volumes_nav_backups(),
					icon: BackupIcon,
					visible: hasPermission('volumes:read', item.environmentId)
				},
				{
					value: 'workspace',
					label: m.workspace(),
					icon: FolderOpenIcon,
					visible: hasPermission('volumes:read', item.environmentId),
					disabled: !!item.logicalId && item.workspaceState !== 'active'
				}
			]}
		/>

		{#if canRenameVolume || canDeleteVolume}
			<DropdownMenu.Separator />
		{/if}

		{#if canRenameVolume && hasPermission('volumes:rename', item.environmentId)}
			<DropdownMenu.Item
				onclick={() => handleRenameVolume(item)}
				disabled={!!item.logicalId || item.inUse || isBackupVolume(item) || isLoading.renaming}
			>
				<EditIcon class="size-4" />
				{m.rename()}
			</DropdownMenu.Item>
		{/if}

		{#if canDeleteVolume && hasPermission('volumes:delete', item.environmentId)}
			<DropdownMenu.Item
				variant="destructive"
				onclick={() => handleRemoveVolumeConfirm(item.name, item.environmentId)}
				disabled={!!item.logicalId || item.inUse || isBackupVolume(item)}
			>
				<TrashIcon class="size-4" />
				{m.common_remove()}
			</DropdownMenu.Item>
		{/if}
	</RowActionsMenu>
{/snippet}

<ArcaneTable
	bind:this={tablePreferences}
	persistKey={showEnvironment ? 'arcane-volumes-table' : 'arcane-volume-agent-scoped-table'}
	items={withFailedEnvironments(volumes, backupVolumeNamesQuery.data?.failures)}
	bind:requestOptions
	bind:selectedIds
	bind:mobileFieldVisibility
	bind:customSettings
	bind:columnVisibility
	bind:preferencesReady={
		() => tablePreferencesReady,
		(ready) => {
			const wasReady = tablePreferencesReady;
			tablePreferencesReady = ready;
			if (!ready || wasReady) return;
			const persistedInternal = (customSettings['showInternalVolumes'] as boolean) ?? false;
			if (persistedInternal !== (requestOptions?.includeInternal ?? false)) setShowInternal(persistedInternal);
		}
	}
	hiddenSortFallback={{ column: 'name', direction: 'asc' }}
	{bulkActions}
	onRetry={retryFailedEnvironments}
	{refreshError}
	{columns}
	{mobileFields}
	rowActions={RowActions}
	mobileCard={VolumeMobileCardSnippet}
	customViewOptions={CustomViewOptions}
>
	{#snippet emptyState()}
		<EmptyState icon={VolumesIcon} title={m.no_volumes_yet()} description={m.create_volume_to_keep_data()} />
	{/snippet}
</ArcaneTable>

{#if renameSession > 0}
	{#key renameSession}
		<RenameVolumeDialog
			bind:open={renameDialogOpen}
			volume={volumeToRename}
			isLoading={isLoading.renaming}
			onSubmit={handleRenameSubmit}
		/>
	{/key}
{/if}

{#snippet CustomViewOptions()}
	<DropdownMenu.CheckboxItem checked={showInternal} onCheckedChange={(v) => setShowInternal(!!v)}>
		{`${m.common_show()} ${m.internal()} ${m.resource_volumes_cap()}`}
	</DropdownMenu.CheckboxItem>
{/snippet}
