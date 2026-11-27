<script lang="ts">
	import { afterNavigate, goto, refreshAll, replaceState } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { toast } from 'svelte-sonner';

	import { ActionButtonGroup, type ActionButton } from '#lib/components/action-button-group/index.js';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import ServiceEditorDialog from '#lib/components/dialogs/service-editor-dialog.svelte';
	import ResourceNotFound from '#lib/components/resource-not-found.svelte';
	import SheetFooterActions from '#lib/components/sheets/sheet-footer-actions.svelte';
	import { confirmForceUpdateService } from '#lib/components/swarm/force-update-service.js';
	import SwarmKvScopedPanel from '#lib/components/swarm/swarm-kv-scoped-panel.svelte';
	import { type TabItem } from '#lib/components/tab-bar/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import * as ResponsiveDialog from '#lib/components/ui/responsive-dialog/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { useUrlTab } from '#lib/hooks/use-url-tab.svelte.js';
	import {
		ContainersIcon,
		FileTextIcon,
		JobsIcon,
		SettingsIcon,
		NetworksIcon,
		VolumesIcon,
		EditIcon,
		LayersIcon,
		RedeployIcon,
		TrashIcon
	} from '#lib/icons/index.js';
	import TabbedPageLayout from '#lib/layouts/tabbed-page-layout.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type {
		SwarmServiceInspect,
		SwarmResourceReference,
		SwarmServiceModeSpec,
		SwarmServiceMount,
		SwarmServiceNetworkDetail,
		SwarmServicePort
	} from '#lib/types/swarm.js';
	import type {
		RawServiceNetworkAttachment,
		RawServiceVirtualIP,
		RawSwarmServiceMount,
		RawSwarmServicePort,
		ServiceNetworkAttachment,
		ServiceVirtualIP
	} from '#lib/types/ui/swarm.js';
	import { handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { getSwarmServiceModeFromSpec, isSwarmServiceModeScalable } from '#lib/utils/docker.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	import ScopedTasksPanel from '../../tasks/components/scoped-tasks-panel.svelte';
	import ServiceConfiguration from '../components/service-configuration.svelte';
	import ServiceLogsPanel from '../components/service-logs-panel.svelte';
	import ServiceNetwork from '../components/service-network.svelte';
	import ServiceOverview from '../components/service-overview.svelte';
	import ServiceStorage from '../components/service-storage.svelte';

	let { data } = $props();
	let service = $derived(data?.service as SwarmServiceInspect);

	let scaleDialogOpen = $state(false);
	let userScaleReplicas = $state<number | null>(null);
	let userScaleReplicasServiceId = $state<string | null>(null);
	const isLoading = $state({ update: false, rollback: false, remove: false, scale: false });

	let editOpen = $state(false);

	type ServiceContainerSpecShape = {
		Secrets?: { SecretID: string; SecretName: string; File?: { Name: string } }[];
		Configs?: { ConfigID: string; ConfigName: string; File?: { Name: string } }[];
		Image?: string;
		Env?: string[];
		Mounts?: RawSwarmServiceMount[];
		Command?: string[];
		Args?: string[];
		Dir?: string;
		User?: string;
		Hostname?: string;
	};

	type ServiceSpecShape = {
		Name?: string;
		Labels?: Record<string, string>;
		Mode?: SwarmServiceModeSpec;
		TaskTemplate?: {
			ContainerSpec?: ServiceContainerSpecShape;
			Networks?: RawServiceNetworkAttachment[];
		};
	};

	const spec = $derived(service?.spec as ServiceSpecShape | undefined);
	const containerSpec = $derived(spec?.TaskTemplate?.ContainerSpec);
	const serviceName = $derived(spec?.Name || '');
	const serviceImage = $derived(containerSpec?.Image || '');
	const secretReferences = $derived<SwarmResourceReference[]>(
		(containerSpec?.Secrets ?? []).map((ref) => ({ id: ref.SecretID, name: ref.SecretName, target: ref.File?.Name }))
	);
	const configReferences = $derived<SwarmResourceReference[]>(
		(containerSpec?.Configs ?? []).map((ref) => ({ id: ref.ConfigID, name: ref.ConfigName, target: ref.File?.Name }))
	);

	const serviceMode = $derived(getSwarmServiceModeFromSpec(spec?.Mode));
	const showsReplicas = $derived(isSwarmServiceModeScalable(serviceMode));

	const desiredReplicas = $derived.by(() => {
		if (serviceMode === 'global') return 0;
		if (serviceMode === 'global-job') return 0;
		if (serviceMode === 'replicated-job') {
			return spec?.Mode?.ReplicatedJob?.TotalCompletions ?? spec?.Mode?.ReplicatedJob?.MaxConcurrent ?? 1;
		}
		return spec?.Mode?.Replicated?.Replicas ?? 1;
	});
	const scaleReplicas = $derived.by(() => {
		if (userScaleReplicasServiceId === service?.id && userScaleReplicas !== null) return userScaleReplicas;
		return desiredReplicas;
	});

	const envVars = $derived(containerSpec?.Env || []);
	const labels = $derived(spec?.Labels || {});

	function normalizeMount(mount: RawSwarmServiceMount): SwarmServiceMount {
		return {
			type: mount.type ?? mount.Type ?? 'volume',
			source: mount.source ?? mount.Source ?? '',
			target: mount.target ?? mount.Target ?? '',
			readOnly: mount.readOnly ?? mount.ReadOnly ?? false,
			volumeDriver: mount.volumeDriver ?? mount.VolumeDriver,
			volumeOptions: mount.volumeOptions ?? mount.VolumeOptions,
			devicePath: mount.devicePath ?? mount.DevicePath
		};
	}

	function normalizePort(port: RawSwarmServicePort): SwarmServicePort {
		return {
			protocol: port.protocol ?? port.Protocol ?? 'tcp',
			targetPort: port.targetPort ?? port.TargetPort ?? 0,
			publishedPort: port.publishedPort ?? port.PublishedPort ?? undefined,
			publishMode: port.publishMode ?? port.PublishMode ?? undefined
		};
	}

	function normalizeNetworkAttachment(network: RawServiceNetworkAttachment): ServiceNetworkAttachment {
		return {
			target: network.target ?? network.Target ?? '',
			aliases: network.aliases ?? network.Aliases ?? []
		};
	}

	function normalizeVirtualIP(vip: RawServiceVirtualIP): ServiceVirtualIP {
		return {
			networkID: vip.networkID ?? vip.NetworkID ?? '',
			addr: vip.addr ?? vip.Addr ?? ''
		};
	}

	const mounts = $derived<SwarmServiceMount[]>(service?.mounts ?? (containerSpec?.Mounts ?? []).map(normalizeMount));
	const command = $derived(containerSpec?.Command || []);
	const args = $derived(containerSpec?.Args || []);
	const workingDir = $derived(containerSpec?.Dir || '');
	const user = $derived(containerSpec?.User || '');
	const hostname = $derived(containerSpec?.Hostname || '');

	const endpointPorts = $derived<SwarmServicePort[]>(
		((service?.endpoint?.['Ports'] as RawSwarmServicePort[] | undefined) ?? []).map(normalizePort)
	);
	const specNetworks = $derived<ServiceNetworkAttachment[]>((spec?.TaskTemplate?.Networks ?? []).map(normalizeNetworkAttachment));
	const virtualIPs = $derived<ServiceVirtualIP[]>(
		((service?.endpoint?.['VirtualIPs'] as RawServiceVirtualIP[] | undefined) ?? []).map(normalizeVirtualIP)
	);
	const networkDetails = $derived<Record<string, SwarmServiceNetworkDetail>>(service?.networkDetails ?? {});

	const hasEnvVars = $derived(envVars.length > 0);
	const hasLabels = $derived(Object.keys(labels).length > 0);
	const hasAdvancedConfig = $derived(command.length > 0 || args.length > 0 || !!workingDir || !!user || !!hostname);
	const showConfiguration = $derived(hasEnvVars || hasLabels || hasAdvancedConfig);

	const envId = $derived(data.envId);
	const canUpdateService = $derived(hasPermission('swarm-services:update', envId));
	const canDeleteService = $derived(hasPermission('swarm-services:delete', envId));
	const canScaleService = $derived(canUpdateService && showsReplicas);

	const serviceActions = $derived<ActionButton[]>([
		...(canScaleService
			? [
					{
						id: 'scale',
						action: 'base' as const,
						placement: 'primary' as const,
						label: m.swarm_service_scale(),
						icon: LayersIcon,
						disabled: isLoading.scale,
						onclick: () => (scaleDialogOpen = true)
					}
				]
			: []),
		...(canUpdateService
			? [
					{
						id: 'edit',
						action: 'edit' as const,
						placement: canScaleService ? ('secondary' as const) : ('primary' as const),
						label: m.common_edit(),
						icon: EditIcon,
						disabled: isLoading.update,
						onclick: openEdit
					},
					{
						id: 'force-update',
						action: 'redeploy' as const,
						group: 'deploy' as const,
						label: m.swarm_service_force_update_short(),
						icon: RedeployIcon,
						onclick: () =>
							confirmForceUpdateService({ serviceId: service.id, serviceName, environmentId: envId, refresh: refreshData })
					},
					{
						id: 'rollback',
						action: 'redeploy' as const,
						group: 'deploy' as const,
						label: m.swarm_service_rollback(),
						icon: RedeployIcon,
						disabled: isLoading.rollback,
						onclick: handleRollback
					}
				]
			: []),
		...(canDeleteService
			? [
					{
						id: 'delete',
						action: 'remove' as const,
						group: 'danger' as const,
						destructive: true,
						label: m.common_delete(),
						icon: TrashIcon,
						disabled: isLoading.remove,
						onclick: handleDelete
					}
				]
			: [])
	]);
	const canViewServiceLogs = $derived(hasPermission('swarm-services:logs', envId));

	const tabItems = $derived<TabItem[]>([
		{ value: 'overview', label: m.common_overview(), icon: ContainersIcon },
		...(canViewServiceLogs ? [{ value: 'logs', label: m.common_logs(), icon: FileTextIcon }] : []),
		{ value: 'tasks', label: m.tasks(), icon: JobsIcon },
		...(showConfiguration ? [{ value: 'config', label: m.common_configuration(), icon: SettingsIcon }] : []),
		{ value: 'network', label: m.resource_networks_cap(), icon: NetworksIcon },
		{ value: 'storage', label: m.resource_volumes_cap(), icon: VolumesIcon },
		...(hasPermission('swarm-secrets:list', envId)
			? [{ value: 'secrets', label: m.swarm_secrets_title(), icon: FileTextIcon }]
			: []),
		...(hasPermission('swarm-configs:list', envId)
			? [{ value: 'configs', label: m.swarm_configs_title(), icon: FileTextIcon }]
			: [])
	]);

	const urlTab = useUrlTab({ history: true, validTabs: () => tabItems.map((tab) => tab.value), defaultTab: () => 'overview' });
	const selectedTab = $derived(urlTab.value);

	function onTabChange(value: string) {
		urlTab.select(value);
	}

	function onScaleReplicasInput(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		userScaleReplicasServiceId = service?.id ?? null;
		userScaleReplicas = Number(input.value);
	}

	async function refreshData() {
		await refreshAll();
	}

	const editVersion = $derived(service?.version?.index ?? service?.version?.Index ?? 0);
	const editSpec = $derived(JSON.stringify(spec ?? {}, null, 2));

	function openEdit() {
		editOpen = true;
	}

	afterNavigate(() => {
		const { swarmServiceAction, ...state } = page.state;
		if (!swarmServiceAction) return;

		void replaceState('', state);
		if (swarmServiceAction === 'edit' && canUpdateService) openEdit();
		if (swarmServiceAction === 'rollback' && canUpdateService) handleRollback();
		if (swarmServiceAction === 'scale' && canScaleService) scaleDialogOpen = true;
	});

	async function handleUpdate(payload: { spec: Record<string, unknown>; options?: Record<string, unknown> }) {
		if (!service?.id) return;
		isLoading.update = true;
		await handleApiResultWithCallbacks({
			result: await tryCatch(swarmService.updateService(service.id, { version: editVersion, ...payload }, envId)),
			message: m.common_update_failed({ resource: `${m.swarm_service()} "${serviceName}"` }),
			setLoadingState: (v) => (isLoading.update = v),
			onSuccess: async () => {
				toast.success(m.common_update_success({ resource: `${m.swarm_service()} "${serviceName}"` }));
				editOpen = false;
				await refreshData();
			}
		});
	}

	function handleRollback() {
		openConfirmDialog({
			title: m.swarm_service_rollback_title(),
			message: m.swarm_service_rollback_confirm({ name: serviceName }),
			confirm: {
				label: m.swarm_service_rollback(),
				destructive: false,
				action: async () => {
					isLoading.rollback = true;
					await handleApiResultWithCallbacks({
						result: await tryCatch(swarmService.rollbackService(service.id, envId)),
						message: m.swarm_service_rollback_failed({ name: serviceName }),
						setLoadingState: (v) => (isLoading.rollback = v),
						onSuccess: async () => {
							toast.success(m.swarm_service_rollback_success({ name: serviceName }));
							await refreshData();
						}
					});
				}
			}
		});
	}

	function handleDelete() {
		openConfirmDialog({
			title: m.common_delete_title({ resource: m.swarm_service() }),
			message: m.common_delete_confirm({ resource: m.swarm_service() }),
			confirm: {
				label: m.common_delete(),
				destructive: true,
				action: async () => {
					isLoading.remove = true;
					await handleApiResultWithCallbacks({
						result: await tryCatch(swarmService.removeService(service.id, envId)),
						message: m.common_delete_failed({ resource: `${m.swarm_service()} "${serviceName}"` }),
						setLoadingState: (v) => (isLoading.remove = v),
						onSuccess: async () => {
							toast.success(m.common_delete_success({ resource: `${m.swarm_service()} "${serviceName}"` }));
							goto(resolve('/(app)/swarm/services'));
						}
					});
				}
			}
		});
	}

	async function handleScale() {
		if (!service?.id || !canScaleService) return;
		const replicas = Math.max(0, Number(scaleReplicas) || 0);
		isLoading.scale = true;
		await handleApiResultWithCallbacks({
			result: await tryCatch(swarmService.scaleService(service.id, { replicas }, envId)),
			message: m.common_update_failed({ resource: `${m.swarm_service()} "${serviceName}"` }),
			setLoadingState: (v) => (isLoading.scale = v),
			onSuccess: async () => {
				toast.success(m.swarm_service_scale_success({ name: serviceName, replicas }));
				scaleDialogOpen = false;
				await refreshData();
			}
		});
	}
</script>

{#if service}
	<TabbedPageLayout backUrl="/swarm/services" backLabel={m.common_back()} {tabItems} {selectedTab} {onTabChange}>
		{#snippet headerInfo()}
			<div class="flex min-h-9 min-w-0 items-center gap-2">
				<div data-slot="swarm-service-icon" class="flex size-9 shrink-0 items-center justify-center rounded-full bg-primary/10">
					<ContainersIcon class="size-5 text-primary" />
				</div>
				<h1 class="min-w-0 truncate text-sm font-semibold" title={serviceName}>
					{serviceName}
				</h1>
			</div>
		{/snippet}

		{#snippet headerActions()}
			{#if serviceActions.length > 0}
				<ActionButtonGroup buttons={serviceActions} />
			{/if}
		{/snippet}

		{#snippet tabContent(_activeTab)}
			<Tabs.Content value="overview">
				<ServiceOverview {service} {serviceName} {serviceImage} {serviceMode} {desiredReplicas} {labels} environmentId={envId} />
			</Tabs.Content>

			<Tabs.Content value="logs" class="h-full">
				{#if selectedTab === 'logs'}
					<ServiceLogsPanel environmentId={envId} serviceId={service.id} />
				{/if}
			</Tabs.Content>

			<Tabs.Content value="tasks" layout="table">
				{#if selectedTab === 'tasks'}
					{#key JSON.stringify([envId, service.id])}
						<ScopedTasksPanel environmentId={envId} resourceId={service.id} scope="service" />
					{/key}
				{/if}
			</Tabs.Content>

			{#if showConfiguration}
				<Tabs.Content value="config">
					<ServiceConfiguration
						{envVars}
						{labels}
						{command}
						{args}
						{workingDir}
						{user}
						{hostname}
						{hasEnvVars}
						{hasLabels}
						{hasAdvancedConfig}
					/>
				</Tabs.Content>
			{/if}

			<Tabs.Content value="network" layout="table">
				<ServiceNetwork ports={endpointPorts} networks={specNetworks} {virtualIPs} {networkDetails} environmentId={envId} />
			</Tabs.Content>

			<Tabs.Content value="storage" layout="table">
				<ServiceStorage {mounts} environmentId={envId} serviceId={service.id} />
			</Tabs.Content>
			<Tabs.Content value="secrets" layout="table"
				>{#if selectedTab === 'secrets'}<SwarmKvScopedPanel
						kind="secrets"
						environmentId={envId}
						references={secretReferences}
					/>{/if}</Tabs.Content
			>
			<Tabs.Content value="configs" layout="table"
				>{#if selectedTab === 'configs'}<SwarmKvScopedPanel
						kind="configs"
						environmentId={envId}
						references={configReferences}
					/>{/if}</Tabs.Content
			>
		{/snippet}
	</TabbedPageLayout>

	{#if canScaleService}
		<ResponsiveDialog.Root
			bind:open={scaleDialogOpen}
			title={m.swarm_service_scale()}
			description={m.swarm_service_scale_description({ name: serviceName })}
		>
			<form
				class="grid gap-2 py-4"
				onsubmit={(event) => {
					event.preventDefault();
					void handleScale();
				}}
			>
				<Label for="swarm-service-scale-replicas">{m.swarm_replicas()}</Label>
				<Input
					id="swarm-service-scale-replicas"
					type="number"
					min="0"
					step="1"
					value={scaleReplicas}
					oninput={onScaleReplicasInput}
					disabled={isLoading.scale}
				/>
			</form>

			{#snippet footer()}
				<SheetFooterActions
					onCancel={() => (scaleDialogOpen = false)}
					submitAction="base"
					submitLabel={m.swarm_service_scale()}
					submitLoading={isLoading.scale}
					submitDisabled={isLoading.scale}
					onSubmit={() => void handleScale()}
				/>
			{/snippet}
		</ResponsiveDialog.Root>
	{/if}

	{#if editOpen}
		<ServiceEditorDialog
			bind:open={editOpen}
			title={`${m.common_edit()} ${m.swarm_service()}`}
			description={m.common_edit_description()}
			submitLabel={m.common_save()}
			initialSpec={editSpec}
			isLoading={isLoading.update}
			onSubmit={handleUpdate}
		/>
	{/if}
{:else}
	<ResourceNotFound
		resource={m.swarm_service()}
		resourceListTitle={m.services()}
		backHref="/swarm/services"
		onRetry={refreshData}
	/>
{/if}
