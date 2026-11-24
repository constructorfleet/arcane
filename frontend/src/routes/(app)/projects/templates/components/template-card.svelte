<script lang="ts">
	import { resolve } from '$app/paths';
	import { mode } from 'mode-watcher';
	import type { Snippet } from 'svelte';

	import IconImage from '#lib/components/icon-image.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { FolderOpenIcon, GlobeIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { Template } from '#lib/types/template.js';
	import { getThemedIconUrl } from '#lib/utils/docker.js';
	import { withOccurrenceKeys } from '#lib/utils/formatting.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';

	let { template, actions }: { template: Template; actions: Snippet } = $props();

	const MAX_TAGS = 4;

	const registryName = $derived(
		template.registry?.name ?? (template.isRemote ? m.templates_unknown_registry() : m.templates_local_templates())
	);
	const tags = $derived(template.metadata?.tags ?? []);
</script>

<div
	class="group relative flex h-full flex-col gap-3 rounded-xl border border-border/70 bg-card/10 p-4 transition-colors hover:bg-muted/40"
>
	<div class="flex items-start gap-3">
		<div class="relative z-10 shrink-0">
			{@render actions()}
		</div>
		<IconImage
			src={getThemedIconUrl(template.metadata, mode.current) ?? template.metadata?.iconUrl}
			alt={template.name}
			fallback={template.isRemote ? GlobeIcon : FolderOpenIcon}
			class="size-5"
			containerClass="size-9"
		/>
		<div class="min-w-0 flex-1">
			<a
				href={resourceDetailHref(resolve('/(app)/projects/templates/[id]', { id: template.id }))}
				class="block truncate font-medium after:absolute after:inset-0 hover:underline"
			>
				{template.name}
			</a>
			<p class="truncate text-xs text-muted-foreground">
				{template.isRemote ? m.templates_remote() : m.local()} · {registryName}
			</p>
		</div>
	</div>

	{#if template.description}
		<p class="line-clamp-3 text-sm text-muted-foreground">{template.description}</p>
	{:else}
		<p class="text-sm text-muted-foreground italic">{m.common_no_description()}</p>
	{/if}

	{#if tags.length > 0}
		<div class="mt-auto flex flex-wrap gap-1">
			{#each withOccurrenceKeys(tags.slice(0, MAX_TAGS)) as tag (tag.key)}
				<Badge variant="outline">{tag.value}</Badge>
			{/each}
			{#if tags.length > MAX_TAGS}
				<Badge variant="outline">+{tags.length - MAX_TAGS}</Badge>
			{/if}
		</div>
	{/if}
</div>
