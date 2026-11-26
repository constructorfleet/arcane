<script lang="ts">
	import { createQueries } from '@tanstack/svelte-query';

	import VulnerabilityScanPanel from '#lib/components/vulnerability/vulnerability-scan-panel.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { imageScanResultQuery } from '#lib/query/vulnerabilities.js';
	import type { ImageRepositoryCopy } from '#lib/types/image.js';
	import { extractApiErrorMessage } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { inUseImageTargets } from '#lib/utils/image-scan.js';

	let { copies, isScanning, onScan }: { copies: ImageRepositoryCopy[]; isScanning: boolean; onScan: () => void } = $props();
	const targets = $derived(inUseImageTargets(copies.filter((copy) => hasPermission('vulnerabilities:read', copy.environmentId))));
	const reports = createQueries(() => ({
		queries: targets.map((target) => imageScanResultQuery(target.environmentId, target.id))
	}));
</script>

{#if targets.length === 0}<p class="text-sm text-muted-foreground">{m.security_no_images_to_scan()}</p>{/if}
{#each targets as target, index (target.id)}
	<section class="space-y-3">
		{#if targets.length > 1}<p class="font-mono text-xs text-muted-foreground">{target.repoDigests[0] ?? target.id}</p>{/if}
		{#if reports[index]?.isPending}<p role="status">{m.common_loading()}</p>{/if}
		{#if reports[index]?.error}<p role="alert">{extractApiErrorMessage(reports[index]?.error)}</p>{/if}
		<VulnerabilityScanPanel
			environmentId={target.environmentId}
			scan={reports[index]?.data}
			{isScanning}
			onScan={hasPermission('vulnerabilities:scan', target.environmentId) ? onScan : undefined}
		/>
	</section>
{/each}
