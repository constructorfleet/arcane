<script lang="ts">
	import { Tabs as TabsPrimitive } from 'bits-ui';

	import * as Card from '#lib/components/ui/card/index.js';
	import { cn } from '#lib/utils.js';

	let {
		ref = $bindable(null),
		class: className,
		layout,
		children,
		...restProps
	}: TabsPrimitive.ContentProps & { layout?: 'panel' | 'table' } = $props();
</script>

<TabsPrimitive.Content
	bind:ref
	class={cn(
		'h-full min-h-0 ring-offset-background focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:outline-none',
		layout === 'panel' && 'h-auto min-w-0',
		layout === 'table' && 'min-w-0 flex-1',
		className
	)}
	{...restProps}
>
	{#if layout === 'panel' && children}
		<Card.Root>
			<Card.Content>{@render children()}</Card.Content>
		</Card.Root>
	{:else}
		{@render children?.()}
	{/if}
</TabsPrimitive.Content>
