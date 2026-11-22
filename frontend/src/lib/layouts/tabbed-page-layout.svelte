<script lang="ts">
	import type { Snippet } from 'svelte';

	import ResourcePrevious from '#lib/components/resource-detail/resource-previous.svelte';
	import { TabBar, type TabItem } from '#lib/components/tab-bar/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		backUrl?: string;
		backLabel?: string;
		tabItems: TabItem[];
		selectedTab: string;
		onTabChange: (value: string) => void;
		headerInfo: Snippet;
		headerActions?: Snippet;
		subHeader?: Snippet;
		tabContent: Snippet<[string]>;
		class?: string;
		showFloatingHeader?: boolean;
	}

	let {
		backUrl,
		tabItems,
		selectedTab,
		onTabChange,
		headerInfo,
		headerActions,
		subHeader,
		tabContent,
		class: className = '',
		showFloatingHeader
	}: Props = $props();

	let scrollTop = $state(0);
	const floatingHeaderVisible = $derived(showFloatingHeader ?? scrollTop > 100);
</script>

{#snippet fullHeader()}
	<div
		class={cn('shrink-0 border-b transition-opacity duration-300', floatingHeaderVisible && 'pointer-events-none opacity-0')}
		inert={floatingHeaderVisible || undefined}
	>
		<div class="max-w-full px-4 pt-3">
			<div class="flex flex-wrap items-center gap-3">
				<div class="flex min-w-0 flex-1 items-center gap-3">
					{#if backUrl}
						<ResourcePrevious fallback={backUrl} />
					{/if}
					<div class="min-w-0 [&_h1]:text-xl [&_h1]:leading-7 [&_h1]:font-semibold sm:[&_h1]:text-2xl">
						{@render headerInfo()}
					</div>
				</div>
				{#if headerActions}
					<div class="flex shrink-0 items-center justify-end gap-2">
						{@render headerActions()}
					</div>
				{/if}
			</div>

			{#if subHeader}
				{@render subHeader()}
			{/if}

			<div class="mt-4 -mb-px">
				<TabBar items={tabItems} />
			</div>
		</div>
	</div>
{/snippet}

<div class={cn('flex h-full min-h-0 flex-col', className)}>
	<Tabs.Root value={selectedTab} onValueChange={onTabChange} class="flex min-h-0 w-full flex-1 flex-col">
		{#if showFloatingHeader === false}
			{@render fullHeader()}
		{/if}

		{#if floatingHeaderVisible}
			<!-- A full-width, click-through strip centres the bubble. Positioning the bubble itself at
			     left-1/2 would cap its shrink-to-fit width at half the viewport. -->
			<div class="pointer-events-none fixed inset-x-4 top-4 z-(--arcane-z-page-floating) flex justify-center">
				<div
					class="bubble-shadow-lg pointer-events-auto max-w-full rounded-lg border border-border/50 bg-popover/90 px-4 py-3 backdrop-blur-md supports-backdrop-filter:bg-popover/80"
				>
					<div class="flex items-center gap-4">
						{#if backUrl}<ResourcePrevious fallback={backUrl} />{/if}
						<div class="min-w-0">
							{@render headerInfo()}
						</div>
						{#if headerActions}
							<div class="h-4 w-px shrink-0 bg-border"></div>
							<div class="flex min-w-9 flex-1 basis-0 items-center justify-end gap-2">
								{@render headerActions()}
							</div>
						{/if}
					</div>
				</div>
			</div>
		{/if}

		<div class="flex min-h-0 flex-1 flex-col overflow-y-auto" onscroll={(event) => (scrollTop = event.currentTarget.scrollTop)}>
			{#if showFloatingHeader !== false}
				{@render fullHeader()}
			{/if}
			<div class="flex min-h-0 flex-1 flex-col px-3 py-5 sm:px-5">
				{@render tabContent(selectedTab)}
			</div>
		</div>
	</Tabs.Root>
</div>
