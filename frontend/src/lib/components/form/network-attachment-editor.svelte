<script module lang="ts">
	export type NetworkAttachmentRow = {
		network: string;
		aliases: string;
		ipv4Address: string;
		ipv6Address: string;
		linkLocalAddresses: string;
	};
</script>

<script lang="ts">
	import SearchableSelect from '#lib/components/form/searchable-select.svelte';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { m } from '#lib/paraglide/messages.js';

	import RowListEditor from './row-list-editor.svelte';

	let {
		rows = $bindable([]),
		networks = [],
		disabled = false
	}: {
		rows?: NetworkAttachmentRow[];
		networks?: string[];
		disabled?: boolean;
	} = $props();

	const attached = $derived(new Set(rows.map((row) => row.network)));
	const networkItems = $derived(networks.map((name) => ({ value: name, label: name, disabled: attached.has(name) })));
	const editorId = $props.id();
</script>

<RowListEditor
	bind:rows
	{disabled}
	createRow={() => ({ network: '', aliases: '', ipv4Address: '', ipv6Address: '', linkLocalAddresses: '' })}
>
	{#snippet children(row)}
		{@const networkId = `${editorId}-network-${rows.indexOf(row)}`}
		<div class="grid min-w-0 flex-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
			<div class="space-y-1">
				<Label for={networkId}>{m.resource_network_cap()}</Label>
				<SearchableSelect triggerId={networkId} items={networkItems} bind:value={row.network} {disabled} class="w-full" />
			</div>
			<div class="space-y-1">
				<Label for={`${networkId}-aliases`}>{m.containers_aliases()}</Label>
				<Input id={`${networkId}-aliases`} type="text" bind:value={row.aliases} {disabled} mono title={m.aliases_note()} />
			</div>
			<div class="space-y-1">
				<Label for={`${networkId}-ipv4`}>{m.static_ip()}</Label>
				<Input id={`${networkId}-ipv4`} type="text" bind:value={row.ipv4Address} {disabled} mono />
			</div>
			<div class="space-y-1">
				<Label for={`${networkId}-ipv6`}>{m.static_ipv6_address()}</Label>
				<Input id={`${networkId}-ipv6`} type="text" bind:value={row.ipv6Address} {disabled} mono />
			</div>
			<div class="space-y-1 sm:col-span-2">
				<Label for={`${networkId}-link-local`}>{m.container_network_link_local_addresses()}</Label>
				<Input id={`${networkId}-link-local`} type="text" bind:value={row.linkLocalAddresses} {disabled} mono />
				<p class="text-xs text-muted-foreground">{m.container_network_link_local_hint()}</p>
			</div>
			<p class="text-xs text-muted-foreground sm:col-span-2 lg:col-span-3">{m.container_network_static_ip_hint()}</p>
		</div>
	{/snippet}
</RowListEditor>
