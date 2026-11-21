<script lang="ts">
	import { page } from '$app/state';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import { ArrowLeftIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { previousResourcePath } from '#lib/utils/resource-navigation.js';

	let { fallback }: { fallback: string } = $props();
	const href = $derived(previousResourcePath((page.shallow?.url ?? page.url).href, fallback));
</script>

<ArcaneButton action="base" tone="ghost" {href} class="shrink-0" aria-label={m.resource_previous()} data-sveltekit-replacestate>
	<ArrowLeftIcon class="size-4" />
	{m.resource_prev_short()}
</ArcaneButton>
