<script lang="ts">
	import { goto } from '$app/navigation';

	import type { TabItem } from '#lib/components/tab-bar/types.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { InspectIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';

	let { href, tabs }: { href: string; tabs: (TabItem & { visible?: boolean })[] } = $props();

	function tabHref(tab: string) {
		const url = new URL(href, 'http://arcane.local');
		url.searchParams.set('tab', tab);
		return `${url.pathname}${url.search}${url.hash}`;
	}
</script>

<DropdownMenu.Sub>
	<DropdownMenu.SubTrigger><InspectIcon class="size-4" />{m.common_view()}</DropdownMenu.SubTrigger>
	<DropdownMenu.SubContent>
		{#each tabs.filter((tab) => tab.visible !== false) as tab (tab.value)}
			{@const Icon = tab.icon ?? InspectIcon}
			<DropdownMenu.Item disabled={tab.disabled} onclick={() => goto(tabHref(tab.value))}>
				<Icon class="size-4" />{tab.label}
			</DropdownMenu.Item>
		{/each}
	</DropdownMenu.SubContent>
</DropdownMenu.Sub>
