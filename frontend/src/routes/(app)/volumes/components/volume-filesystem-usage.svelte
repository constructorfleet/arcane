<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';

	import { m } from '#lib/paraglide/messages.js';
	import { volumeWorkspaceStorageQuery } from '#lib/query/volumes.js';
	import { bytes } from '#lib/utils/formatting.js';

	let { environmentId, volumeName }: { environmentId: string; volumeName: string } = $props();
	const storage = createQuery(() => volumeWorkspaceStorageQuery(environmentId, volumeName));
</script>

{#if storage.isError}
	<span class="text-sm text-muted-foreground">{m.common_unavailable()}</span>
{:else if storage.data}
	<span class="text-sm tabular-nums">
		{m.filesystem_used_of_total({
			used: bytes.format(storage.data.usedBytes, { unitSeparator: ' ' }) ?? m.common_unavailable(),
			total: bytes.format(storage.data.totalBytes, { unitSeparator: ' ' }) ?? m.common_unavailable()
		})}
	</span>
{:else}
	<span class="text-sm text-muted-foreground">{storage.isPending ? m.common_loading() : m.common_unavailable()}</span>
{/if}
