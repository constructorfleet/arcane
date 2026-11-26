<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { mode } from 'mode-watcher';
	import { toast } from 'svelte-sonner';

	import ArcaneTable from '#lib/components/arcane-table/arcane-table.svelte';
	import type { ColumnSpec, TableViewMode } from '#lib/components/arcane-table/arcane-table.types.svelte.js';
	import { templateTypeFilters } from '#lib/components/arcane-table/data.js';
	import DetailLinksMenu from '#lib/components/arcane-table/detail-links-menu.svelte';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import IconImage from '#lib/components/icon-image.svelte';
	import IfPermitted from '#lib/components/if-permitted.svelte';
	import EmptyState from '#lib/components/states/empty-state.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import {
		TemplateIcon,
		InspectIcon,
		FolderOpenIcon,
		GlobeIcon,
		TrashIcon,
		DownloadIcon,
		MoveToFolderIcon
	} from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { templateService } from '#lib/services/template-service.js';
	import type { Paginated } from '#lib/types/base.js';
	import type { Template } from '#lib/types/template.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import { handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { getThemedIconUrl } from '#lib/utils/docker.js';
	import { hasShownPermission } from '#lib/utils/environments.js';
	import { withOccurrenceKeys } from '#lib/utils/formatting.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	import TemplateCard from './template-card.svelte';

	let {
		templates,
		requestOptions = $bindable()
	}: {
		templates: Paginated<Template>;
		requestOptions: SearchPaginationSortRequest;
	} = $props();

	const queryClient = useQueryClient();

	let deletingId = $state<string | null>(null);
	let downloadingId = $state<string | null>(null);

	let viewMode = $state<TableViewMode>('grid');
	const canCreateTemplate = $derived(hasShownPermission('templates:create'));
	const canDeleteTemplate = $derived(hasShownPermission('templates:delete'));

	const columns = $derived([
		{ accessorKey: 'name', title: m.common_name(), sortable: true, card: 'title', cell: TemplateName },
		{ accessorKey: 'description', title: m.common_description(), sortable: true, card: 'field' },
		{
			id: 'type',
			accessorFn: (item) => item.isRemote,
			title: m.common_type(),
			filterOptions: templateTypeFilters,
			card: 'badge',
			cell: TemplateSource
		},
		{ id: 'registry', accessorFn: (item) => item.registry?.name, title: m.common_registry(), card: 'context' },
		{ id: 'tags', accessorFn: (item) => item.metadata?.tags, title: m.common_tags(), card: 'badge', cell: TemplateTags },
		{ id: 'author', accessorFn: (item) => item.metadata?.author, title: m.common_author(), card: 'hidden', hidden: true },
		{ id: 'version', accessorFn: (item) => item.metadata?.version, title: m.common_version(), card: 'hidden', hidden: true }
	] satisfies ColumnSpec<Template>[]);

	async function handleDeleteTemplate(template: Template) {
		openConfirmDialog({
			title: m.common_delete_title({ resource: m.resource_template() }),
			message: m.common_delete_confirm({ resource: `${m.resource_template()} "${template.name}"` }),
			confirm: {
				label: m.templates_delete_template(),
				destructive: true,
				action: async () => {
					deletingId = template.id;

					const result = await tryCatch(templateService.deleteTemplate(template.id));
					await handleApiResultWithCallbacks({
						result,
						message: m.common_delete_failed({ resource: `${m.resource_template()} "${template.name}"` }),
						setLoadingState: (value) => (value ? null : (deletingId = null)),
						onSuccess: async () => {
							toast.success(m.common_delete_success({ resource: `${m.resource_template()} "${template.name}"` }));
							await queryClient.invalidateQueries({ queryKey: queryKeys.templates.all });
							deletingId = null;
						}
					});
				}
			}
		});
	}

	async function handleDownloadTemplate(template: Template) {
		downloadingId = template.id;

		const result = await tryCatch(templateService.download(template.id));
		await handleApiResultWithCallbacks({
			result,
			message: m.templates_download_failed(),
			setLoadingState: (value) => (value ? null : (downloadingId = null)),
			onSuccess: async () => {
				toast.success(m.templates_downloaded_success({ name: template.name }));
				await queryClient.invalidateQueries({ queryKey: queryKeys.templates.all });
				downloadingId = null;
			}
		});
	}
</script>

{#snippet TemplateName({ item }: { item: Template })}
	<a
		class="flex min-w-0 items-center gap-2 font-medium hover:underline"
		href={resourceDetailHref(resolve('/(app)/projects/templates/[id]', { id: item.id }))}
	>
		<IconImage
			src={getThemedIconUrl(item.metadata, mode.current) ?? item.metadata?.iconUrl}
			fallback={item.isRemote ? GlobeIcon : FolderOpenIcon}
			class="size-4"
			containerClass="size-7 shrink-0"
		/>
		<span class="truncate" title={item.name}>{item.name}</span>
	</a>
{/snippet}

{#snippet TemplateSource({ item }: { item: Template })}
	<Badge variant={item.isRemote ? 'blue' : 'gray'}>{item.isRemote ? m.templates_remote() : m.local()}</Badge>
{/snippet}

{#snippet TemplateTags({ item }: { item: Template })}
	<div class="flex flex-wrap gap-1">
		{#each withOccurrenceKeys(item.metadata?.tags ?? []) as tag (tag.key)}
			<Badge variant="outline">{tag.value}</Badge>
		{/each}
	</div>
{/snippet}

{#snippet TemplateActions({ item }: { item: Template })}
	{@const localVersion =
		item.isRemote && item.metadata?.remoteUrl
			? templates.data.find((template) => !template.isRemote && template.metadata?.remoteUrl === item.metadata?.remoteUrl)
			: undefined}
	<RowActionsMenu name={item.name}>
		{#if localVersion}<DropdownMenu.Item
				onclick={() => goto(resourceDetailHref(resolve('/(app)/projects/templates/[id]', { id: localVersion.id })))}
				>{m.templates_view_local_version()}</DropdownMenu.Item
			>{/if}
		<DetailLinksMenu
			href={resourceDetailHref(resolve('/(app)/projects/templates/[id]', { id: item.id }))}
			tabs={[
				{ value: 'overview', label: m.common_overview() },
				{ value: 'compose', label: m.compose() },
				{ value: 'env', label: m.common_environment_variables(), visible: !!item.envContent || !!item.metadata?.envUrl }
			]}
		/>
		<DropdownMenu.Item onclick={() => goto(resourceDetailHref(resolve('/(app)/projects/templates/[id]', { id: item.id })))}>
			<InspectIcon class="size-4" />
			{m.common_view_details()}
		</DropdownMenu.Item>

		<IfPermitted perm="projects:create">
			<DropdownMenu.Item onclick={() => goto(resolve(`/(app)/projects/new?templateId=${encodeURIComponent(item.id)}`))}>
				<MoveToFolderIcon class="size-4" />
				{m.compose_create_project()}
			</DropdownMenu.Item>
		</IfPermitted>

		{#if (item.isRemote && canCreateTemplate) || (!item.isRemote && canDeleteTemplate)}
			<DropdownMenu.Separator />
		{/if}

		{#if item.isRemote && canCreateTemplate && !localVersion}
			<DropdownMenu.Item onclick={() => handleDownloadTemplate(item)} disabled={downloadingId === item.id}>
				{#if downloadingId === item.id}
					<Spinner class="size-4" />
				{:else}
					<DownloadIcon class="size-4" />
				{/if}
				{m.templates_download()}
			</DropdownMenu.Item>
		{:else if !item.isRemote && canDeleteTemplate}
			<DropdownMenu.Item variant="destructive" onclick={() => handleDeleteTemplate(item)} disabled={deletingId === item.id}>
				{#if deletingId === item.id}
					<Spinner class="size-4" />
				{:else}
					<TrashIcon class="size-4" />
				{/if}
				{m.templates_delete_template()}
			</DropdownMenu.Item>
		{/if}
	</RowActionsMenu>
{/snippet}

<ArcaneTable
	items={templates}
	{columns}
	selectionDisabled
	allowMobileList
	persistKey="arcane-template-gallery"
	bind:requestOptions
	bind:viewMode
	rowActions={TemplateActions}
>
	{#snippet mobileCard({ item })}
		<TemplateCard template={item}>
			{#snippet actions()}{@render TemplateActions({ item })}{/snippet}
		</TemplateCard>
	{/snippet}
	{#snippet emptyState()}
		<EmptyState
			icon={TemplateIcon}
			title={m.templates_no_templates()}
			actionLabel={canCreateTemplate ? m.templates_create_template() : undefined}
			actionHref={canCreateTemplate ? '/projects/templates/create' : undefined}
		/>
	{/snippet}
</ArcaneTable>
