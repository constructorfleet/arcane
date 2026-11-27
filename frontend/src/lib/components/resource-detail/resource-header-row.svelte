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

<div class="-m-1 flex min-h-11 min-w-0 flex-wrap items-center gap-3 p-1 sm:flex-nowrap" data-slot="resource-header-row">
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
			'-order-1 -my-1 min-w-0 basis-full overflow-x-auto py-1 whitespace-nowrap sm:order-0 sm:min-w-24 sm:flex-1 sm:basis-0',
			compact ? '[&_h1]:text-sm' : '[&_h1]:text-xl [&_h1]:leading-7 sm:[&_h1]:text-2xl'
		)}
	>
		{@render headerInfo()}
	</div>
	{#if headerActions}
		<div class="ml-auto flex min-w-0 flex-wrap items-center justify-end gap-2 sm:shrink-0" data-slot="resource-header-actions">
			{@render headerActions()}
		</div>
	{/if}
</div>
