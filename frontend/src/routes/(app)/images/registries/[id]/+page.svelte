<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';

	import type { ActionButton } from '#lib/components/action-button-group/types.js';
	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import ImagePullSheet from '#lib/components/sheets/image-pull-sheet.svelte';
	import { EmptyState } from '#lib/components/states/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Breadcrumb from '#lib/components/ui/breadcrumb/index.js';
	import { Checkbox } from '#lib/components/ui/checkbox/index.js';
	import { CopyButton } from '#lib/components/ui/copy-button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import * as InputGroup from '#lib/components/ui/input-group/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { AlertIcon, ArrowRightIcon, RegistryIcon, SearchIcon, TagIcon, TrashIcon, DownloadIcon } from '#lib/icons/index.js';
	import { ResourceDetailLayout } from '#lib/layouts/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { containerRegistryService } from '#lib/services/container-registry-service.js';
	import type { PaginationResponse } from '#lib/types/base.js';
	import type { ContainerRegistryRepositoryTagRow } from '#lib/types/ui/containerregistry.js';
	import { cn } from '#lib/utils.js';
	import { extractApiErrorMessage } from '#lib/utils/api.js';
	import { bulkConfirmAndRun, confirmAndRun } from '#lib/utils/bulk-actions.js';
	import { hasShownPermission } from '#lib/utils/environments.js';
	import { bytes, formatRelativeTime } from '#lib/utils/formatting.js';
	import { buildImageReference } from '#lib/utils/registry.js';
	import { debounced } from '#lib/utils/ws.js';

	let { data } = $props();

	const queryClient = useQueryClient();

	const registry = $derived(data.registry);
	const registryLabel = $derived(registry.url || 'docker.io');
	// Repository names already carry any namespace from the registry URL.
	const repository = $derived(data.repository);
	const namespace = $derived(data.namespace);
	const tags = $derived(data.tags);
	const canDeleteTags = $derived(hasShownPermission('registries:delete-tags'));
	const canPullImages = $derived(hasShownPermission('images:pull'));
	let pullOpen = $state(false);
	let pullReference = $state('');

	let selecting = $state(false);
	let selected = $state<string[]>([]);
	let deletingTag = $state<string | null>(null);
	let bulkDeleting = $state(false);
	let refreshing = $state(false);

	// Configured names are relative to the registry URL namespace, like the build flow.
	const configuredNames = $derived(
		(registry.repositoryNames ?? []).map((name) => (registry.namespace ? `${registry.namespace}/${name}` : name))
	);
	const repositories = $derived([
		...new Set(
			data.catalog?.data.map((item) => item.name) ??
				configuredNames.filter((name) => !namespace || name.startsWith(namespace + '/'))
		)
	]);

	function navigate(patch: { repository?: string; namespace?: string; view?: string; search?: string; page?: number }) {
		const target = patch.repository ?? repository;
		const targetNamespace = patch.namespace ?? namespace;
		const view = patch.view ?? (data.allRepositories ? 'repositories' : '');
		const search = patch.search ?? data.search;
		const params = new URLSearchParams({
			...(target ? { repository: target } : {}),
			...(targetNamespace ? { namespace: targetNamespace } : {}),
			...(view === 'repositories' ? { view } : {}),
			...(search ? { search } : {}),
			...((patch.page ?? 1) > 1 ? { page: String(patch.page) } : {})
		});
		selecting = false;
		selected = [];
		goto(`${resolve('/(app)/images/registries/[id]', { id: registry.id })}${params.size > 0 ? `?${params}` : ''}`, {
			reset: false
		});
	}

	const search = debounced((value: string) => navigate({ search: value.trim(), page: 1 }), 300);

	function submitRepositoryQuery(event: SubmitEvent) {
		event.preventDefault();
		const value = String(new FormData(event.currentTarget as HTMLFormElement).get('query') ?? '').trim();
		if (data.catalog) {
			navigate({ search: value, page: 1 });
		} else if (value) {
			navigate({ repository: value, search: '', page: 1 });
		}
	}

	function openNamespace(event: SubmitEvent) {
		event.preventDefault();
		const name = String(new FormData(event.currentTarget as HTMLFormElement).get('namespace') ?? '').trim();
		if (name) navigate({ namespace: name, repository: '', view: 'repositories', search: '', page: 1 });
	}

	function tagReference(tag: string) {
		return buildImageReference(registry.host, '', repository, tag);
	}

	function platformLabel(platform: ContainerRegistryRepositoryTagRow['platforms'][number]) {
		return [platform.os, platform.architecture, platform.variant].filter(Boolean).join('/');
	}

	function tagMeta(tag: ContainerRegistryRepositoryTagRow) {
		if (tag.error) return m.registries_tag_details_unavailable();
		const parts = [bytes.format(tag.size)];
		if (tag.created) parts.push(formatRelativeTime(tag.created));
		return parts.join(' · ');
	}

	function toggleSelected(tag: string, checked: boolean) {
		if (checked) {
			selected = [...selected, tag];
		} else {
			selected = selected.filter((name) => name !== tag);
		}
	}

	// Loader queries outlive their stale windows only until invalidated, so
	// drop every cached page and search variant before rerunning loaders.
	async function refresh() {
		refreshing = true;
		try {
			const registries = queryKeys.containerRegistries;
			await queryClient.invalidateQueries({ queryKey: registries.namespacesPrefix(registry.id), refetchType: 'none' });
			await queryClient.invalidateQueries({ queryKey: registries.repositoriesPrefix(registry.id), refetchType: 'none' });
			if (repository) {
				await queryClient.invalidateQueries({ queryKey: registries.tagsPrefix(registry.id, repository), refetchType: 'none' });
			}
			await refreshAll();
		} catch (err) {
			toast.error(m.common_refresh_failed({ resource: m.common_registry() }), { description: extractApiErrorMessage(err) });
		} finally {
			refreshing = false;
		}
	}

	function handleDeleteOne(tag: string) {
		const reference = tagReference(tag);
		confirmAndRun({
			title: m.registries_delete_tag_title({ tag }),
			message: m.registries_delete_tag_message({ reference }),
			confirmLabel: m.common_delete(),
			destructive: true,
			setLoading: (loading) => (deletingTag = loading ? tag : null),
			run: () => containerRegistryService.deleteTag(registry.id, repository, tag),
			failureMessage: m.registries_delete_tag_failed({ reference }),
			onSuccess: async () => {
				toast.success(m.registries_delete_tag_success({ reference }));
				await refresh();
			}
		});
	}

	function handleDeleteSelected() {
		const ids = selected;
		if (!ids.length) return;
		// Tags sharing a digest are deleted together, so a second request for the same digest would 404.
		const digestByTag = new Map(tags.data.map((tag) => [tag.name, tag.digest]));
		const tagsToDelete = ids.filter((tag, index) => {
			const digest = digestByTag.get(tag);
			return !digest || ids.findIndex((other) => digestByTag.get(other) === digest) === index;
		});

		bulkConfirmAndRun({
			ids: tagsToDelete,
			title: m.registries_delete_tags_selected_title({ count: ids.length }),
			message: m.registries_delete_tags_selected_message(),
			confirmLabel: m.common_delete(),
			destructive: true,
			run: (tag) => containerRegistryService.deleteTag(registry.id, repository, tag),
			messages: {
				success: () => m.registries_bulk_delete_tags_success({ count: ids.length }),
				partial: (success, total, failed) => m.common_bulk_delete_partial({ success, total, failed, resource: m.common_tags() }),
				failure: () => m.registries_bulk_delete_tags_failed({ count: ids.length })
			},
			setLoading: (loading) => (bulkDeleting = loading),
			onComplete: async ({ success }) => {
				if (success > 0) await refresh();
			},
			clearSelection: () => {
				selected = [];
				selecting = false;
			},
			sequential: true
		});
	}

	const actions: ActionButton[] = $derived([
		{
			id: 'refresh',
			action: 'refresh',
			placement: 'secondary',
			iconOnly: true,
			label: m.common_refresh(),
			loading: refreshing,
			disabled: refreshing,
			onclick: refresh
		}
	]);
</script>

{#snippet browsePagination(paging: PaginationResponse)}
	{#if paging.totalPages > 1}
		<div class="flex items-center justify-between text-sm text-muted-foreground">
			<span>{m.common_page_of({ page: paging.currentPage, total: paging.totalPages })}</span>
			<div class="flex gap-2">
				<ArcaneButton
					action="base"
					customLabel={m.common_previous()}
					disabled={paging.currentPage <= 1}
					onclick={() => navigate({ page: paging.currentPage - 1 })}
				/>
				<ArcaneButton
					action="base"
					customLabel={m.common_next()}
					disabled={paging.currentPage >= paging.totalPages}
					onclick={() => navigate({ page: paging.currentPage + 1 })}
				/>
			</div>
		</div>
	{/if}
{/snippet}

{#snippet namespaceNavigation()}
	<div class="flex flex-wrap items-center gap-2">
		<ArcaneButton
			action="base"
			customLabel={m.namespaces_projects()}
			onclick={() => navigate({ namespace: '', repository: '', view: '', search: '', page: 1 })}
		/>
		{#if namespace}<span class="font-mono text-sm">/ {namespace}</span>{/if}
		<ArcaneButton
			action="base"
			customLabel={m.all_repositories()}
			onclick={() => navigate({ namespace: '', repository: '', view: 'repositories', search: '', page: 1 })}
		/>
	</div>
{/snippet}

{#snippet namespaceFallback()}
	<form class="flex flex-wrap items-center gap-2" onsubmit={openNamespace}>
		<InputGroup.Root class="min-w-0 flex-1 md:w-80 md:flex-none">
			<InputGroup.Input name="namespace" aria-label={m.namespace_or_project()} placeholder={m.team_or_project()} />
		</InputGroup.Root>
		<ArcaneButton action="inspect" type="submit" customLabel={m.open_namespace_project()} />
	</form>
	<p class="text-sm text-muted-foreground">{m.project_browsing_without_listing_help()}</p>
{/snippet}

{#snippet namespaceGrid()}
	<div class="space-y-4">
		{@render namespaceNavigation()}
		<h2 class="text-lg font-semibold">{m.namespaces_projects()}</h2>
		<p class="text-sm text-muted-foreground">{m.choose_namespace_project_help()}</p>
		<InputGroup.Root class="md:w-80">
			<InputGroup.Addon><SearchIcon aria-hidden="true" /></InputGroup.Addon>
			<InputGroup.Input
				aria-label={m.common_search()}
				value={data.search}
				placeholder={m.common_search()}
				oninput={(event) => search(event.currentTarget.value)}
			/>
		</InputGroup.Root>
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
			{#each data.namespaces?.data ?? [] as item (item.name)}
				<button
					type="button"
					class="flex items-center gap-3 rounded-xl border border-border/70 bg-card/10 p-4 text-left hover:bg-muted/40"
					onclick={() => navigate({ namespace: item.name, repository: '', view: 'repositories', search: '', page: 1 })}
				>
					<RegistryIcon class="size-5 shrink-0 text-primary" /><span class="min-w-0 flex-1 truncate font-medium">{item.name}</span
					><ArrowRightIcon class="size-4 shrink-0" />
				</button>
			{/each}
		</div>
		{#if data.namespaces?.data.length === 0}<EmptyState
				icon={SearchIcon}
				title={m.common_no_results_found()}
				description={m.common_no_results_hint()}
			/>{/if}
		{#if data.namespaces}{@render browsePagination(data.namespaces.pagination)}{/if}
		{@render namespaceFallback()}
	</div>
{/snippet}

{#snippet repositoryGrid()}
	<div class="space-y-4">
		{@render namespaceNavigation()}
		{#if data.namespaceError}
			<Alert.Root variant="warning-subtle" size="sm" icon={AlertIcon}
				><Alert.Description>{data.namespaceError}</Alert.Description></Alert.Root
			>
		{/if}
		{@render namespaceFallback()}
		<form class="flex flex-wrap items-center gap-2" onsubmit={submitRepositoryQuery}>
			<InputGroup.Root class="min-w-0 flex-1 md:w-80 md:flex-none">
				<InputGroup.Addon>
					<SearchIcon aria-hidden="true" />
				</InputGroup.Addon>
				{#if data.catalog}
					<InputGroup.Input
						name="query"
						value={data.search}
						placeholder={m.common_search()}
						oninput={(e) => search(e.currentTarget.value)}
					/>
				{:else}
					<InputGroup.Input
						name="query"
						aria-label={m.repository_path()}
						placeholder={m.registries_open_repository_placeholder({ namespace: registry.namespace || 'project' })}
					/>
				{/if}
			</InputGroup.Root>
			{#if data.catalogError}
				<ArcaneButton action="inspect" type="submit" customLabel={m.registries_open_repository()} />
			{/if}
		</form>

		{#if !data.catalog}
			<p class="text-sm text-muted-foreground">{m.repository_path_explanation()}</p>
		{/if}

		{#if data.catalogError}
			<Alert.Root variant="warning-subtle" size="sm" icon={AlertIcon}>
				<Alert.Description>{data.catalogError}</Alert.Description>
			</Alert.Root>
		{/if}
		{#if data.catalogError && repositories.length > 0}
			<p class="text-sm text-muted-foreground">{m.repository_list_showing_configured()}</p>
		{/if}

		{#if repositories.length > 0}
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
				{#each repositories as name (name)}
					<button
						type="button"
						class="group flex items-center gap-3 rounded-xl border border-border/70 bg-card/10 p-4 text-left transition-colors hover:bg-muted/40"
						onclick={() => navigate({ repository: name, search: '', page: 1 })}
					>
						<span
							class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary ring-1 ring-primary/20 ring-inset group-hover:bg-primary/15"
						>
							<RegistryIcon class="size-5" />
						</span>
						<span class="min-w-0 flex-1">
							<span class="block truncate font-medium">{name}</span>
							<span class="block truncate text-xs text-muted-foreground">{registry.host}/{name}</span>
						</span>
						<ArrowRightIcon class="size-4 shrink-0 text-muted-foreground" />
					</button>
				{/each}
			</div>
			{#if data.catalog}{@render browsePagination(data.catalog.pagination)}{/if}
			{#if data.catalog && data.catalog.pagination.totalItems > repositories.length}
				<p class="text-xs text-muted-foreground">
					{m.common_showing_of_total({ shown: repositories.length, total: data.catalog.pagination.totalItems })}
				</p>
			{/if}
		{:else if data.catalog}
			<EmptyState icon={SearchIcon} title={m.common_no_results_found()} description={m.common_no_results_hint()} />
		{:else if data.catalogError}
			<EmptyState
				icon={RegistryIcon}
				title={m.repository_list_unavailable()}
				description={m.repository_list_unavailable_description()}
			></EmptyState>
		{/if}
	</div>
{/snippet}

{#snippet tagList()}
	<div class="space-y-4">
		<div class="flex flex-wrap items-center gap-3">
			<Breadcrumb.Root class="min-w-0 flex-1">
				<Breadcrumb.List>
					<Breadcrumb.Item>
						<Breadcrumb.Link onclick={() => navigate({ repository: '', search: '', page: 1 })} class="cursor-pointer">
							{namespace || m.registries_repositories()}
						</Breadcrumb.Link>
					</Breadcrumb.Item>
					<Breadcrumb.Separator />
					<Breadcrumb.Item>
						<Breadcrumb.Page><span class="font-mono">{repository}</span></Breadcrumb.Page>
					</Breadcrumb.Item>
				</Breadcrumb.List>
			</Breadcrumb.Root>
			<CopyButton text={`${registry.host}/${repository}`} class="size-7" title={m.registries_copy_reference()} />
		</div>

		<div class="flex flex-wrap items-center gap-2">
			<InputGroup.Root class="min-w-0 flex-1 md:w-80 md:flex-none">
				<InputGroup.Addon>
					<SearchIcon aria-hidden="true" />
				</InputGroup.Addon>
				<InputGroup.Input value={data.search} placeholder={m.common_search()} oninput={(e) => search(e.currentTarget.value)} />
			</InputGroup.Root>
			{#if canDeleteTags && tags.data.length > 0}
				<div class="ml-auto flex items-center gap-2">
					{#if selecting}
						<ArcaneButton
							action="remove"
							customLabel={m.registries_delete_tags_selected_count({ count: selected.length })}
							disabled={selected.length === 0 || bulkDeleting}
							loading={bulkDeleting}
							onclick={handleDeleteSelected}
						/>
						<ArcaneButton
							action="cancel"
							onclick={() => {
								selecting = false;
								selected = [];
							}}
						/>
					{:else}
						<ArcaneButton action="base" customLabel={m.common_select()} onclick={() => (selecting = true)} />
					{/if}
				</div>
			{/if}
		</div>

		{#if data.tagsError}
			<Alert.Root variant="destructive">
				<AlertIcon class="size-4" />
				<Alert.Title>{m.registries_tags_load_failed({ repository })}</Alert.Title>
				<Alert.Description>{data.tagsError}</Alert.Description>
			</Alert.Root>
		{:else if tags.data.length === 0}
			<EmptyState icon={TagIcon} title={m.common_no_results_found()} description={m.common_no_results_hint()} />
		{:else}
			<div class="divide-y divide-border/60 overflow-hidden rounded-lg border border-border/60">
				{#each tags.data as tag (tag.name)}
					<div
						class={cn(
							'group flex flex-wrap items-center gap-x-3 gap-y-1.5 px-4 py-3 transition-colors hover:bg-muted/40 lg:flex-nowrap',
							selected.includes(tag.name) && 'bg-muted/40'
						)}
					>
						{#if selecting}
							<Checkbox
								checked={selected.includes(tag.name)}
								onCheckedChange={(checked) => toggleSelected(tag.name, checked === true)}
								aria-label={m.common_select_row()}
							/>
						{:else}
							<TagIcon class="size-4 shrink-0 text-muted-foreground" />
						{/if}
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-1">
								<span class="truncate font-medium">{tag.name}</span>
								<span class="opacity-0 group-hover:opacity-100 focus-within:opacity-100">
									<CopyButton text={tagReference(tag.name)} class="size-6" title={m.registries_copy_reference()} />
								</span>
							</div>
							<p class="truncate text-xs text-muted-foreground" title={tag.error}>{tagMeta(tag)}</p>
						</div>
						{#if tag.digest}
							<div class="order-2 flex w-full min-w-0 items-center gap-1 lg:order-none lg:w-auto lg:flex-1">
								<code class="truncate font-mono text-xs text-muted-foreground" title={tag.digest}>{tag.digest}</code>
								<CopyButton text={tag.digest} class="size-6 shrink-0" title={m.images_attestations_digest()} />
							</div>
						{/if}
						{#if tag.platforms.length > 0}
							<div class="order-3 flex w-full flex-wrap gap-1 lg:order-none lg:w-auto lg:justify-end">
								{#each tag.platforms as platform (platform.digest)}
									<Badge variant="outline">{platformLabel(platform)}</Badge>
								{/each}
							</div>
						{/if}
						{#if (canDeleteTags || canPullImages) && !selecting}
							<RowActionsMenu
								name={tag.name}
								triggerClass="order-1 size-8 shrink-0 opacity-0 group-hover:opacity-100 data-[state=open]:opacity-100 lg:order-none"
							>
								{#if canPullImages}
									<DropdownMenu.Item
										onclick={() => {
											pullReference = tagReference(tag.name);
											pullOpen = true;
										}}
									>
										<DownloadIcon class="size-4" />{m.images_pull_image()}
									</DropdownMenu.Item>
								{/if}
								{#if canDeleteTags}
									<DropdownMenu.Item
										variant="destructive"
										onclick={() => handleDeleteOne(tag.name)}
										disabled={deletingTag === tag.name}
									>
										{#if deletingTag === tag.name}
											<Spinner class="size-4" />
										{:else}
											<TrashIcon class="size-4" />
										{/if}
										{m.common_delete()}
									</DropdownMenu.Item>
								{/if}
							</RowActionsMenu>
						{/if}
					</div>
				{/each}
			</div>

			{#if tags.pagination.totalPages > 1}
				<div class="flex items-center justify-between text-sm text-muted-foreground">
					<span>{m.common_page_of({ page: tags.pagination.currentPage, total: tags.pagination.totalPages })}</span>
					<div class="flex gap-2">
						<ArcaneButton
							action="base"
							customLabel={m.common_previous()}
							disabled={tags.pagination.currentPage <= 1}
							onclick={() => navigate({ page: tags.pagination.currentPage - 1 })}
						/>
						<ArcaneButton
							action="base"
							customLabel={m.common_next()}
							disabled={tags.pagination.currentPage >= tags.pagination.totalPages}
							onclick={() => navigate({ page: tags.pagination.currentPage + 1 })}
						/>
					</div>
				</div>
			{/if}
		{/if}
	</div>
{/snippet}

<ResourceDetailLayout backUrl="/images/registries" backLabel={m.registries_title()} title={registryLabel} {actions}>
	{#if repository}
		{@render tagList()}
	{:else if data.namespaces && (data.namespaces.pagination.grandTotalItems ?? data.namespaces.pagination.totalItems) > 0}
		{@render namespaceGrid()}
	{:else}
		{@render repositoryGrid()}
	{/if}
</ResourceDetailLayout>

<ImagePullSheet
	bind:open={pullOpen}
	initialImageRef={pullReference}
	onPullFinished={(success, imageName) => {
		if (success && imageName) {
			toast.success(m.images_pull_success({ repoTag: imageName }));
			void queryClient.invalidateQueries({ queryKey: queryKeys.images.all });
		}
	}}
/>
