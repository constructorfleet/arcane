<script lang="ts">
	import DetailLinksMenu from '../src/lib/components/arcane-table/detail-links-menu.svelte';
	import RowActionsMenu from '../src/lib/components/arcane-table/row-actions-menu.svelte';
	import type { TabItem } from '../src/lib/components/tab-bar/types';
	import * as DropdownMenu from '../src/lib/components/ui/dropdown-menu/index';
	import { InspectIcon, NetworksIcon } from '../src/lib/icons/index';

	const scenario = new URLSearchParams(location.search).get('scenario') ?? 'service';
	let releaseNotesOpen = $state(false);
	const disabled = scenario === 'unavailable';
	const tabs: (TabItem & { visible?: boolean; related?: boolean; href?: string })[] =
		scenario === 'overview-only'
			? [
					{ value: 'overview', label: 'Overview' },
					{ value: 'tasks', label: 'Tasks', related: true, visible: false }
				]
			: [
					{ value: 'overview', label: 'Overview', icon: InspectIcon },
					{ value: 'config', label: 'Configuration' },
					{ value: 'logs', label: 'Logs', visible: scenario !== 'restricted' },
					{ value: 'tasks', label: 'Tasks', related: true },
					{ value: 'network', label: 'Networks', related: true, icon: NetworksIcon },
					{ value: 'storage', label: 'Volumes', related: true, disabled: true }
				];
	if (scenario === 'no-overview') tabs.shift();
	if (scenario === 'duplicates')
		tabs.push(
			{ value: 'config', label: 'Duplicate configuration', related: true },
			{ value: 'overview', label: 'Duplicate overview' }
		);
	if (scenario === 'owners')
		tabs.push(
			{ value: 'stack', label: 'Owning stack', related: true, href: '/swarm/stacks/observability?env=manager&tab=overview' },
			{
				value: 'duplicate-stack',
				label: 'Duplicate stack',
				related: true,
				href: '/swarm/stacks/observability?tab=overview&env=manager'
			}
		);
</script>

{#snippet imageDetails()}
	<DropdownMenu.Item onclick={() => (releaseNotesOpen = true)}><InspectIcon class="size-4" />Release notes</DropdownMenu.Item>
{/snippet}

<main class="min-h-screen bg-background p-4 text-foreground">
	<h1 class="mb-4 text-xl">Resource navigation acceptance fixture</h1>
	<table class="w-full">
		<thead><tr><th class="text-left">Service</th><th class="text-right">Actions</th></tr></thead>
		<tbody
			><tr
				><td>observability_prometheus</td><td class="text-right">
					<RowActionsMenu name="observability_prometheus">
						<DetailLinksMenu
							href="/swarm/services/service-7?env=node-2&cluster=cluster-9&agents=node-2%2Cnode-3&prev=%2Fswarm%2Fstacks%2Fobservability%3Fenv%3Dmanager%26tab%3Dservices%26tablePage%3D2&tablePage=3#resource"
							{tabs}
							{disabled}
							details={scenario === 'image' ? imageDetails : undefined}
						/>
					</RowActionsMenu>
				</td></tr
			></tbody
		>
	</table>
	{#if releaseNotesOpen}<p role="status">Release notes requested</p>{/if}
</main>
