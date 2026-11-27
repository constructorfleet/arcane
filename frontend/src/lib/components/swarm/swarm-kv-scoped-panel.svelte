<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import RelatedResourceTable from '#lib/components/resource-detail/related-resource-table.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SwarmResourceReference } from '#lib/types/swarm.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import type { SwarmKvKind } from '#lib/types/ui/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref } from '#lib/utils/environments.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';
	import { scopedSwarmMetadata } from '#lib/utils/swarm-resource-relations.js';
	let {
		kind,
		environmentId,
		references,
		stackName,
		incomplete = false,
		externalNames = new Set<string>()
	}: {
		kind: SwarmKvKind;
		environmentId: string;
		references: SwarmResourceReference[];
		stackName?: string;
		incomplete?: boolean;
		externalNames?: Set<string>;
	} = $props();
	let requestOptions = $state<SearchPaginationSortRequest>({ pagination: { page: 1, limit: 20 } });
	const ids = $derived([...new Set(references.map((ref) => ref.id))].sort().join(','));
	const query = createQuery(() => ({
		queryKey: ['swarm', environmentId, kind, 'relationship-metadata', ids, stackName, requestOptions],
		enabled: hasPermission(`swarm-${kind}:list`, environmentId) && (!!ids || !!stackName),
		queryFn: ({ signal }) =>
			swarmService.getKvItems(kind, { ...requestOptions, filters: { ids, stackName: stackName ?? '' } }, environmentId, {
				signal
			})
	}));
	const rows = $derived(
		scopedSwarmMetadata(query.data?.data ?? [], references, stackName).map(({ item, relationship, targets }) => ({
			id: item.id,
			name: item.spec.Name,
			values: [
				relationship === 'owned' ? m.owned() : externalNames.has(item.spec.Name) ? m.external_reference() : m.referenced(),
				targets.slice(0, 2).join(', ') + (targets.length > 2 ? ` +${targets.length - 2}` : '')
			],
			href: hasPermission(`swarm-${kind}:read`, environmentId)
				? resourceDetailHref(
						environmentHref(
							kind === 'secrets'
								? resolve('/(app)/swarm/secrets/[id]', { id: item.id })
								: resolve('/(app)/swarm/configs/[id]', { id: item.id }),
							environmentId
						)
					)
				: undefined
		}))
	);
	const missing = $derived(
		query.isSuccess && new Set(references.map((ref) => ref.id)).size > (query.data?.pagination?.totalItems ?? 0)
	);
</script>

{#if incomplete || query.isError || (query.isSuccess && missing)}
	<Alert.Root variant="warning-subtle"
		><Alert.Description
			>{query.error?.message ?? m.unavailable()}
			<ArcaneButton action="refresh" onclick={() => query.refetch()} /></Alert.Description
		></Alert.Root
	>
{/if}
<RelatedResourceTable
	columns={[m.relationship(), m.targets()]}
	{rows}
	pagination={query.data?.pagination}
	bind:requestOptions
	loading={query.isPending && query.isFetching}
	refreshError={query.error}
	onRetry={() => query.refetch()}
/>
