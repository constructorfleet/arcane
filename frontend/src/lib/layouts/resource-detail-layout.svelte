<script lang="ts">
	import type { Snippet } from 'svelte';

	import ActionButtonGroup from '#lib/components/action-button-group/action-button-group.svelte';
	import type { ActionButton } from '#lib/components/action-button-group/types.js';
	import EnvironmentDestinationSelect from '#lib/components/form/environment-destination-select.svelte';
	import ResourceHeaderRow from '#lib/components/resource-detail/resource-header-row.svelte';
	import type { IconType } from '#lib/icons/index.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		backUrl?: string;
		backLabel?: string;
		title: string;
		icon?: IconType;
		environmentId?: string;
		subtitle?: string;
		actions?: ActionButton[];
		badges?: Snippet;
		headerExtra?: Snippet;
		children: Snippet;
		class?: string;
	}

	let {
		backUrl,
		title,
		icon: Icon,
		environmentId,
		subtitle,
		actions = [],
		badges,
		headerExtra,
		children,
		class: className = ''
	}: Props = $props();

	let scrollY = $state(0);
	const showFloatingHeader = $derived(scrollY > 120);
</script>

<svelte:window bind:scrollY />

{#snippet detailIdentity(compact: boolean)}
	<div class="flex min-h-9 min-w-0 items-center gap-2">
		{#if Icon}
			<Icon class="size-5 shrink-0 text-primary" aria-hidden="true" />
		{/if}
		{#if environmentId}
			<EnvironmentDestinationSelect value={environmentId} variant="prefix" readonly />
		{/if}
		{#if compact}
			<span class="min-w-0 truncate text-sm font-semibold" {title}>{title}</span>
		{:else}
			<h1 class="min-w-0 truncate font-semibold" {title}>{title}</h1>
		{/if}
		{@render badges?.()}
	</div>
{/snippet}

{#snippet headerActions()}
	<ActionButtonGroup buttons={actions} />
{/snippet}

{#if showFloatingHeader}
	<div class="pointer-events-none fixed inset-x-4 top-4 z-(--arcane-z-page-floating) flex justify-center">
		<div
			class="bubble-shadow-lg pointer-events-auto max-w-full rounded-lg border border-border/60 bg-popover/95 px-4 py-2 backdrop-blur-md supports-backdrop-filter:bg-popover/85"
		>
			<ResourceHeaderRow {backUrl} headerActions={actions.length ? headerActions : undefined} compact>
				{#snippet headerInfo()}{@render detailIdentity(true)}{/snippet}
			</ResourceHeaderRow>
		</div>
	</div>
{/if}

<div class={cn('space-y-6 pb-8', className)}>
	<div class="space-y-3">
		<ResourceHeaderRow {backUrl} headerActions={actions.length ? headerActions : undefined}>
			{#snippet headerInfo()}{@render detailIdentity(false)}{/snippet}
		</ResourceHeaderRow>
		{#if subtitle}
			<p class="truncate text-sm text-muted-foreground" title={subtitle}>{subtitle}</p>
		{/if}
		{@render headerExtra?.()}
	</div>

	{@render children()}
</div>
