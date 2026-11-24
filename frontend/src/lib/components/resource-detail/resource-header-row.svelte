<script lang="ts">
	import type { Snippet } from 'svelte';

	import ResourcePrevious from '#lib/components/resource-detail/resource-previous.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';

	let {
		backUrl,
		headerInfo,
		headerActions,
		compact = false
	}: {
		backUrl?: string;
		headerInfo: Snippet;
		headerActions?: Snippet;
		compact?: boolean;
	} = $props();
</script>

<div class="-m-1 flex min-h-11 min-w-0 items-center gap-3 overflow-x-auto p-1" data-slot="resource-header-row">
	{#if backUrl}
		<ResourcePrevious fallback={backUrl} />
	{/if}
	<!-- svelte-ignore a11y_no_noninteractive_tabindex (The overflow region supports keyboard scrolling.) -->
	<div
		role="region"
		aria-label={m.resource_header_information()}
		tabindex="0"
		data-slot="resource-header-information"
		class={cn(
			'-my-1 min-w-24 flex-1 overflow-x-auto py-1 whitespace-nowrap',
			compact ? '[&_h1]:text-sm' : '[&_h1]:text-xl [&_h1]:leading-7 sm:[&_h1]:text-2xl'
		)}
	>
		{@render headerInfo()}
	</div>
	{#if headerActions}
		<div class="flex shrink-0 items-center justify-end gap-2">
			{@render headerActions()}
		</div>
	{/if}
</div>
