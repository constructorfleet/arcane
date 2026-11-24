<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';

	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Spinner } from '#lib/components/ui/spinner/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { imageService } from '#lib/services/image-service.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { bytes, formatDate, formatDateTime } from '#lib/utils/formatting.js';

	let { imageId, environmentId: imageEnvironmentId }: { imageId: string; environmentId: string } = $props();

	const historyQuery = createQuery(() => {
		const environmentId = imageEnvironmentId;
		const requestedImageId = imageId;
		return {
			queryKey: queryKeys.images.history(environmentId ?? '', requestedImageId),
			queryFn: async () => {
				await environmentStore.ready;
				return imageService.getImageHistory(requestedImageId, environmentId);
			},
			enabled: !!environmentId && !!requestedImageId && hasPermission('images:read', environmentId)
		};
	});
	const history = $derived(historyQuery.data ?? []);
	const loading = $derived(historyQuery.isPending);
	const error = $derived.by(() => {
		if (historyQuery.error) return m.images_history_load_failed();
		return null;
	});
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>{m.images_history_title()}</Card.Title>
		<Card.Description>{m.images_history_description()}</Card.Description>
	</Card.Header>
	<Card.Content>
		{#if loading}
			<div class="flex items-center gap-2 py-8 text-sm text-muted-foreground">
				<Spinner class="size-4" />
				{m.images_history_loading()}
			</div>
		{:else if error}
			<p class="py-8 text-sm text-destructive">{error}</p>
		{:else if history.length === 0}
			<p class="py-8 text-sm text-muted-foreground">{m.images_history_empty()}</p>
		{:else}
			<table class="w-full table-fixed text-sm">
				<thead class="border-b text-left text-xs text-muted-foreground uppercase">
					<tr>
						<th scope="col" class="py-2 pr-2 font-medium">{m.common_command()}</th>
						<th scope="col" class="w-19 py-2 pr-2 font-medium wrap-anywhere">{m.common_created()}</th>
						<th scope="col" class="w-18 py-2 text-right font-medium wrap-anywhere">{m.common_size()}</th>
					</tr>
				</thead>
				<tbody>
					{#each history as item, index (`${item.id}-${index}`)}
						<tr class="border-b align-top last:border-0">
							<td class="py-3 pr-3">
								<div class="font-mono text-xs wrap-anywhere whitespace-pre-wrap">{item.createdBy || m.common_na()}</div>
								{#if item.tags?.length}
									<div class="mt-1.5 flex min-w-0 flex-wrap gap-1" role="group" aria-label={m.common_tags()}>
										{#each item.tags as tag (tag)}
											<Badge variant="secondary" class="max-w-full wrap-anywhere whitespace-normal">{tag}</Badge>
										{/each}
									</div>
								{/if}
							</td>
							<td class="py-3 pr-2 text-xs wrap-anywhere tabular-nums">
								{#if item.created}
									{@const instant = Temporal.Instant.fromEpochMilliseconds(item.created * 1000)}
									<time datetime={instant.toString()} title={formatDateTime(instant)}
										>{formatDate(instant, { dateStyle: 'short' })}</time
									>
								{:else}
									{m.common_na()}
								{/if}
							</td>
							<td class="py-3 text-right text-xs wrap-anywhere tabular-nums">{bytes.format(item.size)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</Card.Content>
</Card.Root>
