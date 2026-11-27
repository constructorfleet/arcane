<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';

	import { m } from '#lib/paraglide/messages.js';
	import { volumeWorkspaceStorageQuery } from '#lib/query/volumes.js';
	import { bytes } from '#lib/utils/formatting.js';

	let { environmentId, volumeName }: { environmentId: string; volumeName: string } = $props();
	const storage = createQuery(() => volumeWorkspaceStorageQuery(environmentId, volumeName));
</script>

{#if storage.data && !storage.isError}
	<span class="text-sm whitespace-nowrap tabular-nums">
		{m.filesystem_usage_ratio({
			used: bytes.format(storage.data.usedBytes, { unitSeparator: '' }) ?? '-',
			total: bytes.format(storage.data.totalBytes, { unitSeparator: '' }) ?? '-'
		})}
	</span>
{:else}
	<span class="text-sm text-muted-foreground">-</span>
{/if}
