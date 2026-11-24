<script lang="ts" module>
	export type RelatedResourceRow = {
		id: string;
		name: string;
		values: (string | null | undefined)[];
		href?: string;
		inspect?: { label: string; href?: string }[];
	};
</script>

<script lang="ts">
	import { goto } from '$app/navigation';

	import ArcaneTable from '#lib/components/arcane-table/arcane-table.svelte';
	import { UniversalMobileCard, type ColumnSpec, type MobileFieldVisibility } from '#lib/components/arcane-table/index.js';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { InspectIcon, LayersIcon, type IconType } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { PaginationResponse } from '#lib/types/base.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';

	let {
		columns,
		rows,
		pagination,
		requestOptions = $bindable({ pagination: { page: 1, limit: 10 } }),
		persistKey,
		icon = LayersIcon,
		empty = m.resource_related_empty(),
		loading = false,
		refreshError,
		onRetry
	}: {
		columns: string[];
		rows: RelatedResourceRow[];
		pagination?: PaginationResponse;
		requestOptions?: SearchPaginationSortRequest;
		persistKey?: string;
		icon?: IconType;
		empty?: string;
		loading?: boolean;
		refreshError?: unknown;
		onRetry?: () => Promise<unknown>;
	} = $props();

	const fields = $derived(columns.map((label, index) => ({ id: `detail-${index}`, label, defaultVisible: true })));
	const tableColumns = $derived([
		{ accessorKey: 'name', title: m.common_name(), sortable: !pagination, cell: NameCell },
		...fields.map(({ id, label }, index) => ({
			id,
			title: label,
			accessorFn: (row: RelatedResourceRow) => row.values[index],
			cell: ValueCell
		}))
	] satisfies ColumnSpec<RelatedResourceRow>[]);
	let mobileFieldVisibility = $state<MobileFieldVisibility>({});

	const items = $derived.by(() => {
		if (pagination) return { data: rows, pagination };

		const search = requestOptions.search?.trim().toLowerCase();
		const matches = search
			? rows.filter((row) => [row.name, ...row.values].some((value) => value?.toLowerCase().includes(search)))
			: rows;
		const sorted =
			requestOptions.sort?.column === 'name'
				? matches.toSorted((a, b) => a.name.localeCompare(b.name) * (requestOptions.sort?.direction === 'desc' ? -1 : 1))
				: matches;
		const limit = requestOptions.pagination?.limit ?? 10;
		const totalPages = limit > 0 ? Math.max(1, Math.ceil(sorted.length / limit)) : 1;
		const currentPage = Math.min(requestOptions.pagination?.page ?? 1, totalPages);
		const start = (currentPage - 1) * limit;
		return {
			data: limit > 0 ? sorted.slice(start, start + limit) : sorted,
			pagination: { totalItems: sorted.length, totalPages, currentPage, itemsPerPage: limit }
		};
	});
</script>

{#snippet NameCell({ item }: { item: RelatedResourceRow })}
	{#if item.href}
		<a href={item.href} class="text-primary hover:underline">{item.name}</a>
	{:else}
		{item.name}
	{/if}
{/snippet}

{#snippet ValueCell({ value }: { value: unknown })}
	<span class="block max-w-80 truncate" title={typeof value === 'string' ? value : undefined}>{value || m.common_na()}</span>
{/snippet}

{#snippet actions({ item }: { item: RelatedResourceRow })}
	{#if item.href || item.inspect?.length}
		<RowActionsMenu name={item.name}>
			{#each item.inspect ?? [{ label: m.common_inspect(), href: item.href }] as action (action.href ?? action.label)}
				<DropdownMenu.Item disabled={!action.href} onclick={() => action.href && goto(action.href)}>
					<InspectIcon />{action.label}
				</DropdownMenu.Item>
			{/each}
		</RowActionsMenu>
	{/if}
{/snippet}

{#snippet mobile({ item }: { item: RelatedResourceRow })}
	<UniversalMobileCard
		{item}
		icon={{ component: icon, variant: 'gray' }}
		title={(row) => row.name}
		href={(row) => row.href}
		rowActions={actions}
		fields={fields.map(({ id, label }, index) => ({
			label,
			getValue: (row: RelatedResourceRow) => row.values[index] ?? m.common_na(),
			show: mobileFieldVisibility[id] ?? true
		}))}
	/>
{/snippet}

<ArcaneTable
	{items}
	{persistKey}
	bind:requestOptions
	bind:mobileFieldVisibility
	mobileFields={fields}
	columns={tableColumns}
	selectionDisabled
	withoutFilters
	{loading}
	{refreshError}
	{onRetry}
	rowActions={actions}
	mobileCard={mobile}
>
	{#snippet emptyState()}<p class="p-4 text-sm text-muted-foreground">{empty}</p>{/snippet}
</ArcaneTable>
