<script lang="ts">
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	import EnvironmentDestinationSelect from '#lib/components/form/environment-destination-select.svelte';
	import FormInput from '#lib/components/form/form-input.svelte';
	import SheetFooterActions from '#lib/components/sheets/sheet-footer-actions.svelte';
	import * as ResponsiveDialog from '#lib/components/ui/responsive-dialog/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { imageService } from '#lib/services/image-service.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { isEnvironmentOnline, parseImageRef } from '#lib/utils/docker.js';
	import { destinationEnvironmentId } from '#lib/utils/environments.js';
	import { createForm, preventDefault } from '#lib/utils/settings.svelte.js';

	type ImagePullFormProps = {
		open: boolean;
		initialImageRef?: string;
		onPullFinished?: (success: boolean, imageName?: string, error?: string, environmentId?: string) => void;
	};

	let { open = $bindable(false), initialImageRef = '', onPullFinished = () => {} }: ImagePullFormProps = $props();

	const formSchema = z.object({
		imageRef: z.string().min(1, m.images_image_required()),
		tag: z.string().optional().default('latest')
	});

	const form = createForm<typeof formSchema>(formSchema, {
		imageRef: '',
		tag: 'latest'
	});

	let isPulling = $state(false);
	let pickedEnvironmentId = $state('');
	const destinations = $derived(
		environmentStore.available.filter((env) => env.enabled && isEnvironmentOnline(env) && hasPermission('images:pull', env.id))
	);
	const environmentId = $derived(pickedEnvironmentId);
	const target = $derived(environmentStore.available.find((env) => env.id === environmentId));
	const canPull = $derived(destinations.some((env) => env.id === environmentId));
	$effect(() => {
		if (!open) return;
		const ref = initialImageRef;
		untrack(() => {
			pickedEnvironmentId = destinationEnvironmentId('', 'images:pull', destinations);
			if (ref.includes('@')) {
				form.reset({ imageRef: ref, tag: '' });
			} else {
				const { repo, tag } = parseImageRef(ref);
				form.reset({ imageRef: repo, tag });
			}
		});
	});

	async function handleSubmit() {
		if (!canPull) {
			toast.error(m.select_connected_image_pull_agent());
			return;
		}
		const data = form.validate();
		if (!data) return;

		isPulling = true;

		let imageName = data.imageRef.trim();
		let imageTag = data.tag?.trim() || 'latest';

		if (imageName.includes(':')) {
			const lastColonIndex = imageName.lastIndexOf(':');
			const possibleTag = imageName.substring(lastColonIndex + 1).trim();
			if (possibleTag && !possibleTag.includes('/')) {
				imageName = imageName.substring(0, lastColonIndex);
				imageTag = possibleTag;
			}
		}

		const fullImageName = `${imageName}:${imageTag}`;
		const targetEnvironmentId = environmentId;
		open = false;
		isPulling = false;
		void drainPullStream(fullImageName, targetEnvironmentId);
	}

	async function drainPullStream(fullImageName: string, targetEnvironmentId: string) {
		const result = await imageService.transferImageStream({
			operation: 'pull',
			imageName: fullImageName,
			environmentId: targetEnvironmentId
		});
		if (result.cancelled) return;
		if (!result.success) {
			const message = result.error || m.images_pull_unexpected_error();
			toast.error(message);
			onPullFinished(false, fullImageName, message, targetEnvironmentId);
			return;
		}
		onPullFinished(true, fullImageName, undefined, targetEnvironmentId);
	}

	function handleOpenChange(newOpenState: boolean) {
		if (!newOpenState && isPulling) {
			return;
		}

		open = newOpenState;
		pickedEnvironmentId = '';
		form.reset();
	}
</script>

<ResponsiveDialog.Root
	{open}
	onOpenChange={handleOpenChange}
	variant="sheet"
	title={m.images_pull_image()}
	description={m.images_pull_description()}
	contentClass="sm:max-w-150"
>
	<form onsubmit={preventDefault(handleSubmit)} class="grid gap-4 py-6">
		<EnvironmentDestinationSelect
			bind:value={() => environmentId, (value) => (pickedEnvironmentId = value)}
			permission="images:pull"
			environments={destinations}
			showSingle
			variant="field"
		/>
		<p class="text-sm text-muted-foreground">
			{m.swarm_node()}: {target?.name ?? m.common_unknown()} <span class="font-mono text-xs">{environmentId}</span>
		</p>
		{#if !canPull}<p class="text-sm text-destructive">{m.select_connected_image_pull_agent()}</p>{/if}
		<FormInput
			label={m.images_image_name_label()}
			type="text"
			placeholder={m.images_image_name_placeholder()}
			description={m.images_image_name_description()}
			bind:input={form.inputs.imageRef}
		/>
		<FormInput
			label={m.tag()}
			type="text"
			placeholder={m.images_tag_latest()}
			description={m.images_tag_description()}
			bind:input={form.inputs.tag}
		/>
	</form>

	{#snippet footer()}
		<SheetFooterActions bind:open submitAction="pull" onSubmit={handleSubmit} submitDisabled={isPulling || !canPull} />
	{/snippet}
</ResponsiveDialog.Root>
