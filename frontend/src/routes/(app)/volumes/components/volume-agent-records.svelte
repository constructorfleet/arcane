<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import { DetailSection } from '#lib/components/resource-detail/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { AlertIcon, GlobeIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { volumeListQuery } from '#lib/query/volumes.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import type { Volume } from '#lib/types/volume.js';
	import { extractApiErrorMessage } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref, rowKey } from '#lib/utils/environments.js';

	let {
		volume,
		environmentId,
		hasWorkspaceChanges = false
	}: { volume: Volume; environmentId: string; hasWorkspaceChanges?: boolean } = $props();
	const eligible = $derived(volume.scope === 'global');
	const agents = $derived(
		environmentStore.available.filter(
			(agent) => agent.id !== environmentId && agent.enabled && hasPermission('volumes:list', agent.id)
		)
	);
	const recordsQuery = createQuery(() => ({
		...volumeListQuery(agents.map((agent) => agent.id).join(','), agents, {
			search: volume.name,
			pagination: { page: 1, limit: -1 }
		}),
		enabled: eligible && agents.length > 0,
		retry: false,
		refetchOnWindowFocus: false,
		refetchOnReconnect: false
	}));
	const records = $derived(
		(recordsQuery.data?.data ?? [])
			.filter(
				(record) =>
					record.name === volume.name &&
					record.driver === volume.driver &&
					agents.some((agent) => agent.id === record.environmentId)
			)
			.toSorted((a, b) => Number(!!b.mountpoint) - Number(!!a.mountpoint) || a.environmentName.localeCompare(b.environmentName))
	);
</script>

{#if eligible}
	<DetailSection title={m.volume_agent_records_title()} icon={GlobeIcon}>
		<p class="text-sm text-muted-foreground">{m.volume_agent_records_description()}</p>
		{#if agents.length === 0}
			<p class="text-sm text-muted-foreground">{m.volume_agent_records_no_agents()}</p>
		{:else}
			<ArcaneButton action="refresh" disabled={recordsQuery.isFetching} onclick={() => recordsQuery.refetch()} />
			{#if recordsQuery.isPending}
				<p class="text-sm text-muted-foreground" role="status">{m.common_loading()}</p>
			{:else if recordsQuery.error}
				<Alert.Root variant="warning-subtle" icon={AlertIcon}>
					<Alert.Description>
						{m.volume_agent_records_error({ message: extractApiErrorMessage(recordsQuery.error) })}
					</Alert.Description>
				</Alert.Root>
			{:else}
				{#if recordsQuery.data?.failedEnvironments?.length}
					<Alert.Root variant="warning-subtle" icon={AlertIcon}>
						<Alert.Description>
							{m.all_environments_unavailable({
								environments: recordsQuery.data.failedEnvironments
									.map(({ environment, message }) => `${environment.name} (${message})`)
									.join(', ')
							})}
						</Alert.Description>
					</Alert.Root>
				{/if}
				{#if recordsQuery.data?.excludedEnvironments?.length}
					<Alert.Root variant="warning-subtle" icon={AlertIcon}>
						<Alert.Description>
							{m.volume_agent_records_excluded({
								agents: recordsQuery.data.excludedEnvironments.map((agent) => agent.name).join(', ')
							})}
						</Alert.Description>
					</Alert.Root>
				{/if}
				{#each records as record (rowKey(record))}
					{@const overviewHref = environmentHref(
						`${resolve('/(app)/volumes/[volumeName]', { volumeName: record.name })}?tab=overview`,
						record.environmentId
					)}
					<div class="space-y-2 rounded-lg border p-4">
						<p class="text-sm font-medium">{record.environmentName}</p>
						<p class="text-sm break-all">
							{m.volume_agent_records_mountpoint()}: <code>{record.mountpoint || m.common_unknown()}</code>
						</p>
						<p class="text-sm">{m.volume_agent_records_references({ count: record.containers?.length ?? 0 })}</p>
						{#if hasPermission('volumes:read', record.environmentId)}
							<ArcaneButton
								action="base"
								tone="outline"
								href={overviewHref}
								data-sveltekit-reload
								onclick={(event) => {
									if (
										!hasWorkspaceChanges ||
										event.button !== 0 ||
										event.metaKey ||
										event.ctrlKey ||
										event.shiftKey ||
										event.altKey
									)
										return;
									event.preventDefault();
									openConfirmDialog({
										title: m.common_unsaved_changes(),
										message: m.volume_agent_records_discard_warning(),
										confirm: {
											label: m.volume_agent_records_discard_open(),
											destructive: true,
											action: async () => {
												window.location.assign(overviewHref);
											}
										}
									});
								}}
							>
								{m.volume_agent_records_view()}
							</ArcaneButton>
						{:else}
							<p class="text-sm text-muted-foreground">{m.common_access_denied()}</p>
						{/if}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">{m.volume_agent_records_none()}</p>
				{/each}
				<p class="text-xs text-muted-foreground">{m.volume_agent_records_reference_note()}</p>
			{/if}
		{/if}
	</DetailSection>
{/if}
