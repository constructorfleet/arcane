<script lang="ts">
	import type { Snippet } from 'svelte';

	import type { TabItem } from '#lib/components/tab-bar/types.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { ExternalLinkIcon, InspectIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';

	let {
		href,
		tabs,
		disabled = false,
		details
	}: {
		href: string;
		tabs: (TabItem & { visible?: boolean; related?: boolean })[];
		disabled?: boolean;
		details?: Snippet;
	} = $props();

	const visibleTabs = $derived.by(() => {
		const destinations = new Set<string>();
		return tabs.filter((tab) => {
			if (tab.visible === false || destinations.has(tab.value)) return false;
			destinations.add(tab.value);
			return true;
		});
	});
	const overview = $derived(visibleTabs.find((tab) => tab.value === 'overview'));
	const detailTabs = $derived(visibleTabs.filter((tab) => tab.value !== 'overview' && !tab.related));
	const relatedTabs = $derived(visibleTabs.filter((tab) => tab.value !== 'overview' && tab.related));

	function tabHref(tab: string) {
		const url = new URL(href, 'http://arcane.local');
		url.searchParams.set('tab', tab);
		return `${url.pathname}${url.search}${url.hash}`;
	}
</script>

{#if overview}
	<DropdownMenu.Item disabled={disabled || overview.disabled}>
		{#snippet child({ props })}
			<a {...props} href={disabled || overview.disabled ? undefined : tabHref('overview')}>
				<InspectIcon class="size-4" />{m.common_view_details()}
			</a>
		{/snippet}
	</DropdownMenu.Item>
{/if}

{#snippet tabLinks(items: typeof visibleTabs)}
	{#each items as tab (tab.value)}
		{@const Icon = tab.icon ?? InspectIcon}
		<DropdownMenu.Item disabled={disabled || tab.disabled}>
			{#snippet child({ props })}
				<a {...props} href={disabled || tab.disabled ? undefined : tabHref(tab.value)}>
					<Icon class="size-4" />{tab.label}
				</a>
			{/snippet}
		</DropdownMenu.Item>
	{/each}
{/snippet}

{#if detailTabs.length > 0 || details}
	<DropdownMenu.Sub>
		<DropdownMenu.SubTrigger {disabled}><InspectIcon class="size-4" />{m.details()}</DropdownMenu.SubTrigger>
		<DropdownMenu.SubContent>
			{@render tabLinks(detailTabs)}
			{@render details?.()}
		</DropdownMenu.SubContent>
	</DropdownMenu.Sub>
{/if}

{#if relatedTabs.length > 0}
	<DropdownMenu.Sub>
		<DropdownMenu.SubTrigger {disabled}><ExternalLinkIcon class="size-4" />{m.related_resources()}</DropdownMenu.SubTrigger>
		<DropdownMenu.SubContent>
			{@render tabLinks(relatedTabs)}
		</DropdownMenu.SubContent>
	</DropdownMenu.Sub>
{/if}
