<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { containerService } from '#lib/services/container-service.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import type { SwarmTaskSummary } from '#lib/types/swarm.js';
	import { extractApiErrorMessage, handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { resourceTableHref } from '#lib/utils/resource-navigation.js';
	import { getSwarmTaskContainerTarget } from '#lib/utils/swarm-tasks.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	let { task, environmentId, tablePage }: { task: SwarmTaskSummary; environmentId: string; tablePage?: number } = $props();
	const key = $derived(`${environmentId}:${task.id}:${task.nodeId}:${task.containerId ?? ''}`);
	let operation = $state({ key: '', loading: false, error: '' });
	const loading = $derived(operation.key === key && operation.loading);
	const error = $derived(operation.key === key ? operation.error : '');
	const canResolve = $derived(hasPermission('swarm-nodes:read', environmentId));

	async function openContainer() {
		if (loading || !task.nodeId || !task.containerId || !canResolve) return;
		const requestedKey = key;
		const selected = { ...task };
		const source = environmentId;
		operation = { key: requestedKey, loading: true, error: '' };
		const result = await tryCatch(
			(async () => {
				await environmentStore.ready;
				const node = await swarmService.getNode(selected.nodeId, source);
				const target = getSwarmTaskContainerTarget(selected, node);
				if (!target) throw new Error(m.task_node_agent_unavailable());
				if (!hasPermission('containers:read', target.environmentId)) throw new Error(m.common_access_denied());
				const container = await containerService.getContainerForEnvironment(target.environmentId, target.containerId, {
					timeout: 15_000
				});
				if (container.id !== target.containerId || container.labels?.['com.docker.swarm.task.id'] !== selected.id) {
					throw new Error(m.task_container_identity_mismatch());
				}
				if (requestedKey !== key) return;
				await goto(
					resourceTableHref(
						resolve(`/(app)/containers/[containerId]?env=${encodeURIComponent(target.environmentId)}`, {
							containerId: target.containerId
						}),
						tablePage
					)
				);
			})()
		);
		if (requestedKey !== key) return;
		await handleApiResultWithCallbacks({
			result,
			message: m.common_load_failed({ resource: m.container() }),
			setLoadingState: (value) => {
				if (requestedKey === key) operation = { ...operation, loading: value };
			},
			onError: (failure) => {
				if (requestedKey === key) operation = { key: requestedKey, loading: false, error: extractApiErrorMessage(failure) };
			}
		});
	}
</script>

<div class="flex flex-col items-start gap-1">
	{#if !task.nodeId || !task.containerId}
		<span class="text-xs text-muted-foreground">{m.task_has_no_container()}</span>
	{:else if !canResolve}
		<span class="text-xs text-muted-foreground">{m.common_access_denied()}</span>
	{:else}
		<ArcaneButton
			action="base"
			tone="outline"
			size="sm"
			customLabel={m.view_task_container()}
			onclick={openContainer}
			{loading}
			disabled={loading}
		/>
	{/if}
	{#if error}
		<span class="max-w-72 text-xs break-words text-destructive" role="status">{m.task_container_unavailable({ error })}</span>
	{/if}
</div>
