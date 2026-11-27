<script lang="ts">
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';

	import SelectWithLabel from '#lib/components/form/select-with-label.svelte';
	import SheetFooterActions from '#lib/components/sheets/sheet-footer-actions.svelte';
	import * as ResponsiveDialog from '#lib/components/ui/responsive-dialog/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { imageService } from '#lib/services/image-service.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { isEnvironmentOnline } from '#lib/utils/docker.js';

	let {
		open = $bindable(false),
		imageId,
		environmentId,
		repoTags
	}: {
		open: boolean;
		imageId: string;
		environmentId: string;
		repoTags: string[];
	} = $props();
	let target = $state({ imageId: '', environmentId: '', tags: [] as string[] });
	let reference = $state('');
	let pushing = $state(false);
	const agent = $derived(environmentStore.available.find((env) => env.id === target.environmentId));
	const available = $derived(
		!!agent && agent.enabled && isEnvironmentOnline(agent) && hasPermission('images:push', target.environmentId)
	);
	const valid = $derived(available && target.tags.includes(reference));
	$effect(() => {
		if (!open) return;
		untrack(() => {
			const tags = repoTags.filter((tag) => {
				const [host = ''] = tag.split('/');
				return (
					tag.includes('/') &&
					!tag.includes('@') &&
					tag.lastIndexOf(':') > tag.lastIndexOf('/') &&
					(host.includes('.') || host.includes(':') || host === 'localhost')
				);
			});
			target = { imageId, environmentId, tags };
			reference = tags[0] ?? '';
		});
	});
	async function submit() {
		if (!valid || pushing) return;
		const request = {
			operation: 'push' as const,
			imageName: reference,
			imageId: target.imageId,
			environmentId: target.environmentId
		};
		pushing = true;
		open = false;
		const result = await imageService.transferImageStream(request);
		pushing = false;
		if (result.cancelled) return;
		if (!result.success) {
			toast.error(m.image_push_failed(), { description: result.error });
			return;
		}
		toast.success(m.image_pushed_successfully({ reference: request.imageName }));
	}
</script>

<ResponsiveDialog.Root
	{open}
	onOpenChange={(value) => (open = value)}
	title={m.push_image()}
	description={m.push_existing_image_tag()}
	contentClass="sm:max-w-150"
>
	<div class="grid gap-4 py-4">
		<p class="text-sm text-muted-foreground">
			{m.swarm_node()}: {agent?.name ?? m.common_unknown()} <span class="font-mono text-xs">{target.environmentId}</span>
		</p>
		<p class="font-mono text-xs break-all">{m.source_image()}: {target.imageId}</p>
		<SelectWithLabel
			id="push-reference"
			bind:value={reference}
			label={m.registry_reference()}
			options={target.tags.map((tag) => ({ label: tag, value: tag }))}
		/>
		{#if !available}<p class="text-sm text-destructive">{m.source_agent_push_unavailable()}</p>{/if}
	</div>
	{#snippet footer()}
		<SheetFooterActions
			bind:open
			submitAction="base"
			submitLabel={m.push()}
			submitDisabled={!valid || pushing}
			submitLoading={pushing}
			onSubmit={submit}
		/>
	{/snippet}
</ResponsiveDialog.Root>
