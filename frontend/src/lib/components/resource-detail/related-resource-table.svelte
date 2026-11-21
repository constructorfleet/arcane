<script lang="ts" module>
	export type RelatedResourceRow = {
		id: string;
		name: string;
		values: (string | null | undefined)[];
		href?: string;
	};
</script>

<script lang="ts">
	import { goto } from '$app/navigation';

	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { InspectIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';

	let {
		columns,
		rows,
		empty = m.resource_related_empty()
	}: { columns: string[]; rows: RelatedResourceRow[]; empty?: string } = $props();
</script>

{#if rows.length}
	<div class="overflow-x-auto rounded-lg border border-border">
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head>{m.common_name()}</Table.Head>
					{#each columns as label, index (index)}
						<Table.Head>{label}</Table.Head>
					{/each}
					<Table.Head>{m.common_actions()}</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each rows as row (row.id)}
					{@const href = row.href}
					<Table.Row>
						<Table.Cell class="max-w-64 wrap-anywhere whitespace-normal">
							{#if href}<a {href} class="text-primary hover:underline">{row.name}</a>{:else}{row.name}{/if}
						</Table.Cell>
						{#each row.values as value, index (index)}
							<Table.Cell class="max-w-80 wrap-anywhere whitespace-normal">{value || m.common_na()}</Table.Cell>
						{/each}
						<Table.Cell>
							{#if href}
								<RowActionsMenu name={row.name}>
									<DropdownMenu.Item onclick={() => goto(href)}><InspectIcon />{m.common_inspect()}</DropdownMenu.Item>
								</RowActionsMenu>
							{/if}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	</div>
{:else}
	<p class="text-sm text-muted-foreground">{empty}</p>
{/if}
