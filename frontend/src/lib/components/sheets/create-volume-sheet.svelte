<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { z } from 'zod/v4';

	import EnvironmentDestinationSelect from '#lib/components/form/environment-destination-select.svelte';
	import FormInput from '#lib/components/form/form-input.svelte';
	import KeyValueEditor, { type KeyValueRow } from '#lib/components/form/key-value-editor.svelte';
	import SheetFooterActions from '#lib/components/sheets/sheet-footer-actions.svelte';
	import * as Accordion from '#lib/components/ui/accordion/index.js';
	import * as ResponsiveDialog from '#lib/components/ui/responsive-dialog/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { volumeService } from '#lib/services/volume-service.js';
	import type { VolumeCreate } from '#lib/types/ui/volume.js';
	import { destinationEnvironmentId } from '#lib/utils/environments.js';
	import { keyValueRowsToRecord, parseKeyValuePairs } from '#lib/utils/form-parsers.js';
	import { createForm, preventDefault } from '#lib/utils/settings.svelte.js';

	import SelectWithLabel from '../form/select-with-label.svelte';

	type CreateVolumeFormProps = {
		open: boolean;
		onSubmit: (data: VolumeCreate, environmentId: string) => void;
		isLoading: boolean;
	};

	let { open = $bindable(false), onSubmit, isLoading }: CreateVolumeFormProps = $props();

	let pickedEnvironmentId = $state('');
	const environmentId = $derived(destinationEnvironmentId(pickedEnvironmentId, 'volumes:create'));

	const manualDriver = '__manual__';
	let driverChoice = $state('local');
	let driverOptions = $state<KeyValueRow[]>([]);
	const optionKeys = $derived(driverOptions.map((row) => row.key.trim()).filter(Boolean));
	const invalidOptions = $derived(
		driverOptions.some((row) => !row.key.trim() && row.value !== '') || new Set(optionKeys).size !== optionKeys.length
	);
	const driversQuery = createQuery(() => {
		const id = environmentId;
		return {
			queryKey: ['volume-drivers', id],
			queryFn: ({ signal }) => volumeService.listVolumeDrivers(id, { signal }),
			enabled: open && !!id,
			retry: false
		};
	});
	const drivers = $derived([
		{ value: 'local', label: m.local() },
		...(driversQuery.data ?? []).filter((name) => name !== 'local').map((name) => ({ value: name, label: name })),
		{ value: manualDriver, label: m.volume_driver_manual() }
	]);

	const formSchema = z.object({
		volumeName: z.string().trim().min(1, m.volume_name_required()),
		volumeDriver: z.string().trim().min(1, m.common_driver_required()),
		volumeLabels: z.string().optional().default('')
	});

	const form = createForm<typeof formSchema>(formSchema, {
		volumeName: '',
		volumeDriver: 'local',
		volumeLabels: ''
	});

	function handleSubmit() {
		if (!environmentId || isLoading || invalidOptions) return;
		const data = form.validate();
		if (!data) return;

		const options = keyValueRowsToRecord(driverOptions);
		const labels = parseKeyValuePairs(data.volumeLabels || '');

		const volumeOptions: VolumeCreate = {
			name: data.volumeName.trim(),
			driver: data.volumeDriver,
			options: Object.keys(options).length ? options : undefined,
			labels: Object.keys(labels).length ? labels : undefined
		};

		onSubmit(volumeOptions, environmentId);
	}

	function changeDriver(value: string) {
		driverChoice = value;
		form.inputs.volumeDriver.value = value === manualDriver ? '' : value;
	}

	function handleOpenChange(newOpenState: boolean) {
		open = newOpenState;
		if (!newOpenState) {
			pickedEnvironmentId = '';
			form.reset();
			driverChoice = 'local';
			driverOptions = [];
		}
	}
</script>

<ResponsiveDialog.Root
	{open}
	onOpenChange={handleOpenChange}
	variant="sheet"
	title={m.create_volume_title()}
	description={m.create_volume_description()}
	contentClass="sm:max-w-150"
>
	<form onsubmit={preventDefault(handleSubmit)} class="grid gap-4 py-6">
		<EnvironmentDestinationSelect
			bind:value={
				() => environmentId,
				(value) => {
					if (value === environmentId) return;
					pickedEnvironmentId = value;
					changeDriver('local');
					driverOptions = [];
				}
			}
			permission="volumes:create"
			variant="field"
		/>
		<FormInput
			label={m.volume_name_label()}
			id="volume-name"
			type="text"
			placeholder={m.volume_name_placeholder()}
			description={m.volume_name_description()}
			disabled={isLoading}
			bind:input={form.inputs.volumeName}
		/>

		<SelectWithLabel
			id="driver-select"
			value={driverChoice}
			onValueChange={changeDriver}
			disabled={isLoading}
			label={m.volume_driver_label()}
			description={m.volume_driver_description()}
			options={drivers}
			placeholder={m.select_a_driver_placeholder()}
		/>

		{#if driversQuery.isFetching}
			<p class="text-xs text-muted-foreground" role="status">{m.volume_driver_discovery_loading()}</p>
		{:else if driversQuery.isError}
			<p class="text-xs text-muted-foreground" role="status">{m.volume_driver_discovery_unavailable()}</p>
		{/if}
		{#if driverChoice === manualDriver}
			<FormInput label={m.volume_driver_name()} bind:input={form.inputs.volumeDriver} disabled={isLoading} />
		{/if}

		<fieldset class="space-y-2">
			<legend class="text-sm font-medium">{m.common_driver_options()}</legend>
			<p class="text-xs text-muted-foreground">
				{driverChoice === 'local' ? m.volume_driver_local_options() : m.volume_driver_plugin_options()}
			</p>
			<KeyValueEditor bind:rows={driverOptions} disabled={isLoading} />
			{#if invalidOptions}
				<p class="text-xs text-destructive" role="alert">{m.volume_driver_option_keys()}</p>
			{/if}
		</fieldset>

		<Accordion.Root type="single" class="w-full">
			<Accordion.Item value="advanced">
				<Accordion.Trigger>{m.volume_advanced_settings()}</Accordion.Trigger>
				<Accordion.Content>
					<div class="space-y-4 pt-4">
						<FormInput
							label={m.common_labels()}
							type="textarea"
							placeholder={m.volume_labels_placeholder()}
							description={m.volumes_labels_description()}
							disabled={isLoading}
							rows={3}
							bind:input={form.inputs.volumeLabels}
						/>
					</div>
				</Accordion.Content>
			</Accordion.Item>
		</Accordion.Root>
	</form>

	{#snippet footer()}
		<SheetFooterActions
			bind:open
			cancelDisabled={isLoading}
			submitDisabled={isLoading || !environmentId || invalidOptions}
			submitLoading={isLoading}
			onSubmit={handleSubmit}
			submitLabel={m.common_create_button({ resource: m.resource_volume_cap() })}
		/>
	{/snippet}
</ResponsiveDialog.Root>
