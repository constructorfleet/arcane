<script lang="ts">
	import { mode } from 'mode-watcher';

	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';
	import { customLogoHash } from '#lib/utils/branding.js';
	import { getApplicationLogo } from '#lib/utils/docker.js';

	let { isCollapsed }: { isCollapsed: boolean } = $props();

	const logoUrl = $derived(getApplicationLogo(!isCollapsed, mode.current === 'dark', { custom: customLogoHash(!isCollapsed) }));
</script>

<div
	class={cn('flex h-11 shrink-0 items-center transition-all duration-300', isCollapsed ? 'justify-center' : 'justify-start pl-2')}
>
	<div class={cn('relative flex shrink-0 items-center justify-center', isCollapsed ? '' : 'flex-col')}>
		<img
			src={logoUrl}
			alt={m.layout_title()}
			class={cn('object-contain drop-shadow-sm transition-all duration-300', isCollapsed ? 'size-6' : 'h-9 w-auto max-w-40')}
			width={isCollapsed ? 24 : 160}
			height={isCollapsed ? 24 : 40}
		/>
	</div>
</div>
