<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { AddIcon, EditIcon, TrashIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SwarmNode } from '#lib/types/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';

	import SwarmNodeLabelDialog from './swarm-node-label-dialog.svelte';

	let { node, environmentId }: { node: SwarmNode; environmentId: string } = $props();
	const queryClient = useQueryClient();
	const canEdit = $derived(hasPermission('swarm-nodes:update', environmentId));
	const labels = $derived(Object.entries(node.labels ?? {}).sort(([a], [b]) => a.localeCompare(b)));
	const systemLabels = $derived(Object.entries(node.systemLabels ?? {}).sort(([a], [b]) => a.localeCompare(b)));
	let dialogOpen = $state(false);
	let currentLabel = $state<{ key: string; value: string }>();

	function editLabel(label?: { key: string; value: string }) {
		currentLabel = label;
		dialogOpen = true;
	}

	async function updateLabel(key: string, value?: string) {
		const id = node.id;
		const manager = environmentId;
		const latest = await swarmService.getNode(id, manager);
		const updated = { ...latest.labels };
		if (value === undefined) delete updated[key];
		else updated[key] = value;
		await swarmService.updateNode(id, { labels: updated }, manager);
		toast.success(m.common_update_success({ resource: m.common_labels() }));
		await Promise.all([
			invalidateAll(),
			queryClient.invalidateQueries({ predicate: ({ queryKey }) => queryKey[0] === 'swarm' && queryKey.includes('nodes') })
		]);
	}

	function removeLabel(key: string) {
		const resource = `${m.common_labels()} "${key}"`;
		openConfirmDialog({
			title: m.common_remove_title({ resource }),
			message: m.common_remove_confirm({ resource }),
			confirm: { label: m.common_remove(), destructive: true, action: () => updateLabel(key) }
		});
	}
</script>

<section class="space-y-3 border-t border-border pt-4">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<h2 class="text-sm font-semibold">{m.common_labels()}</h2>
		{#if canEdit}
			<ArcaneButton action="base" tone="outline" size="sm" icon={AddIcon} onclick={() => editLabel()}
				>{m.add_label()}</ArcaneButton
			>
		{/if}
	</div>

	{#if labels.length}
		<ul class="divide-y divide-border/60">
			{#each labels as [key, value] (key)}
				<li class="flex flex-wrap items-center justify-between gap-2 py-2">
					<Badge variant="blue" class="max-w-full break-all">{key}{value ? `=${value}` : ''}</Badge>
					{#if canEdit}
						<div class="flex shrink-0 gap-1">
							<ArcaneButton
								action="base"
								tone="outline"
								size="icon"
								icon={EditIcon}
								aria-label={m.common_edit() + ' ' + key}
								onclick={() => editLabel({ key, value })}
							/>
							<ArcaneButton
								action="base"
								tone="outline"
								size="icon"
								icon={TrashIcon}
								aria-label={m.common_remove() + ' ' + key}
								onclick={() => removeLabel(key)}
							/>
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{:else}
		<p class="text-sm text-muted-foreground">{m.swarm_node_no_labels()}</p>
	{/if}

	{#if systemLabels.length}
		<h3 class="text-xs font-semibold text-muted-foreground">{m.swarm_node_engine_labels()}</h3>
		<div class="flex flex-wrap gap-2">
			{#each systemLabels as [key, value] (key)}
				<Badge variant="gray" class="max-w-full break-all">{key}{value ? `=${value}` : ''}</Badge>
			{/each}
		</div>
	{/if}
</section>

{#if dialogOpen}
	<SwarmNodeLabelDialog bind:open={dialogOpen} initial={currentLabel} onAdd={updateLabel} />
{/if}
