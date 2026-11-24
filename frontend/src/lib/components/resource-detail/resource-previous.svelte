<script lang="ts">
	import { page } from '$app/state';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import { getNavigationTitleForPath } from '#lib/config/navigation-config.js';
	import { ArrowLeftIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { previousResourcePath } from '#lib/utils/resource-navigation.js';

	let { fallback }: { fallback: string } = $props();
	const current = $derived((page.shallow?.url ?? page.url).href);
	const href = $derived(previousResourcePath(current, fallback));
	const detailLabels: Record<string, () => string> = {
		'/agents': m.resource_environment_cap,
		'/images': m.common_image,
		'/images/registries': m.common_registry,
		'/networks': m.resource_network_cap,
		'/projects': m.project,
		'/projects/templates': () => m.resource_template(),
		'/swarm/nodes': m.swarm_node,
		'/swarm/services': m.swarm_service,
		'/swarm/stacks': m.swarm_stack,
		'/volumes': m.resource_volume_cap
	};
	const label = $derived.by(() => {
		const pathname = new URL(href, current).pathname.replace(/\/$/, '');
		const collection = pathname.slice(0, pathname.lastIndexOf('/'));
		return getNavigationTitleForPath(pathname, true) ?? detailLabels[collection]?.() ?? getNavigationTitleForPath(pathname);
	});
</script>

<ArcaneButton
	action="base"
	tone="ghost"
	{href}
	class="shrink-0"
	aria-label={label ? m.resource_back_to({ destination: label }) : m.resource_previous()}
	data-sveltekit-replacestate
>
	<ArrowLeftIcon class="size-4" />
	{label ?? m.resource_previous()}
</ArcaneButton>
