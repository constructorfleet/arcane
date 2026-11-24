<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';

	import { ActionButtonGroup, type ActionButton } from '#lib/components/action-button-group/index.js';
	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import { DetailSection, KeyValueCard, KeyValueGrid } from '#lib/components/resource-detail/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { useUrlTab } from '#lib/hooks/use-url-tab.svelte.js';
	import { AlertIcon, JobsIcon, LayersIcon, NetworksIcon, SettingsIcon, TagIcon, type IconType } from '#lib/icons/index.js';
	import TabbedPageLayout from '#lib/layouts/tabbed-page-layout.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { networkService } from '#lib/services/network-service.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import type { NetworkInspect, NetworkIPAMConfig } from '#lib/types/network.js';
	import { activityToastOptions, extractActivityId } from '#lib/utils/activity-toast.js';
	import { handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { isSwarmGatewayBridge } from '#lib/utils/docker.js';
	import { collectAcrossEnvironments } from '#lib/utils/environments.js';
	import { formatDateTimeShort } from '#lib/utils/formatting.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	import NetworkSwarmResources from '../components/network-swarm-resources.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let errorMessage = $state('');

	let isRemoving = $state(false);

	const network = $derived(data.network);
	const shortId = $derived(network?.id?.substring(0, 12) ?? m.common_unknown());
	const createdDate = $derived(network?.created ? formatDateTimeShort(network.created) : m.common_unknown());

	const localContainerCount = $derived(network?.containersList.length ?? 0);
	const inUse = $derived(localContainerCount > 0);
	const observationAgents = $derived((page.url.searchParams.get('agents') ?? '').split(',').filter(Boolean));
	const observationCluster = $derived(page.url.searchParams.get('cluster') ?? '');
	const requested = $derived([...new Set(observationAgents)]);
	const agents = $derived(
		environmentStore.available.filter((agent) => requested.includes(agent.id) && hasPermission('networks:read', agent.id))
	);
	const excluded = $derived(requested.filter((id) => !agents.some((agent) => agent.id === id)));
	const hasObservations = $derived(requested.length > 0 && observationCluster !== '');
	const observations = createQuery(() => {
		const expectedNetworkId = network?.id ?? '';
		const expectedClusterId = observationCluster;
		const targets = agents.map(({ id, name }) => ({ id, name }));
		return {
			queryKey: [
				...queryKeys.networks.detail(targets.map((agent) => agent.id).join(','), expectedNetworkId),
				'observations',
				expectedClusterId
			],
			enabled: hasObservations && agents.length > 0,
			queryFn: async ({ signal }) => {
				await environmentStore.ready;
				return collectAcrossEnvironments(targets, async (agentId, config) => {
					const observedNetwork = await networkService.getNetworkForEnvironment(agentId, expectedNetworkId, undefined, {
						...config,
						signal,
						timeout: 15_000
					});
					if (
						observedNetwork.id !== expectedNetworkId ||
						observedNetwork.clusterId !== expectedClusterId ||
						!observedNetwork.nodeId ||
						observedNetwork.scope !== 'swarm' ||
						observedNetwork.driver !== 'overlay'
					) {
						throw new Error(m.network_identity_unavailable());
					}
					return observedNetwork;
				});
			},
			retry: false
		};
	});
	const observedInUse = $derived(observations.data?.values.some(({ value }) => value.containersList.length > 0) ?? false);
	const observationsIncomplete = $derived(
		hasObservations &&
			(excluded.length > 0 ||
				observations.isPending ||
				observations.isFetching ||
				!!observations.error ||
				!observations.data ||
				observations.data.failures.length > 0)
	);
	const removalBlocked = $derived(inUse || observedInUse || observationsIncomplete);

	const swarmScoped = $derived(network?.scope === 'swarm');
	const swarmGateway = $derived(!!network && isSwarmGatewayBridge(network));
	const isPredefined = $derived(network?.name === 'bridge' || network?.name === 'host' || network?.name === 'none');

	const urlTab = useUrlTab({
		tabs: () => [
			{ value: 'overview', label: m.common_overview(), icon: NetworksIcon },
			{ value: 'services', label: m.services(), icon: LayersIcon, visible: swarmScoped },
			{ value: 'tasks', label: m.tasks(), icon: JobsIcon, visible: swarmScoped }
		],
		defaultTab: () => 'overview',
		history: true
	});

	function triggerRemove() {
		if (inUse || observedInUse) {
			toast.error(m.network_has_observed_consumers());
			return;
		}
		if (observationsIncomplete) {
			toast.error(m.network_observations_not_ready());
			return;
		}
		if (isPredefined) {
			toast.error(m.networks_cannot_delete_default({ name: network?.name ?? m.common_unknown() }));
			console.warn('Cannot remove predefined network');
			return;
		}

		if (!network?.id) {
			toast.error(m.networks_missing_id ? m.networks_missing_id() : m.error_occurred());
			return;
		}

		// The dialog outlives navigation, so it removes the network it was opened for.
		const target = { id: network.id, environmentId: data.envId };
		openConfirmDialog({
			title: m.common_remove_title({ resource: m.resource_network() }),
			message: [
				m.networks_remove_confirm_message({ name: network?.name ?? shortId }),
				m.network_query_agent_target({
					agent: environmentStore.available.find((agent) => agent.id === target.environmentId)?.name ?? target.environmentId,
					agentId: target.environmentId
				}),
				...(swarmScoped ? [m.network_remove_swarm_warning()] : [])
			].join('\n\n'),
			confirm: {
				label: m.common_remove(),
				destructive: true,
				action: async () => {
					isRemoving = true;
					await handleApiResultWithCallbacks({
						result: await tryCatch(networkService.deleteNetwork(target.id, target.environmentId)),
						message: m.networks_remove_failed({ name: network?.name ?? shortId }),
						setLoadingState: (value) => (isRemoving = value),
						onSuccess: async (data) => {
							toast.success(
								m.networks_remove_success({ name: network?.name ?? shortId }),
								activityToastOptions(extractActivityId(data))
							);
							goto(resolve('/(app)/networks'));
						},
						onError: (error) => {
							errorMessage = error?.message ?? m.error_occurred();
							toast.error(errorMessage);
						}
					});
				}
			}
		});
	}

	const actions: ActionButton[] = $derived([
		{
			id: 'remove',
			action: 'remove',
			group: 'danger',
			destructive: true,
			placement: 'secondary',
			label: m.common_remove(),
			loading: isRemoving,
			disabled: isRemoving || isPredefined || removalBlocked,
			onclick: triggerRemove
		}
	]);
</script>

{#snippet ipamSubnet(config: NetworkIPAMConfig)}
	<KeyValueGrid>
		{#if config.subnet}
			<KeyValueCard label={m.common_subnet()} valueTitle={m.common_click_to_select()}>{config.subnet}</KeyValueCard>
		{/if}
		{#if config.gateway}
			<KeyValueCard label={m.common_gateway()} valueTitle={m.common_click_to_select()}>{config.gateway}</KeyValueCard>
		{/if}
		{#if config.ipRange}
			<KeyValueCard label={m.networks_ipam_iprange_label()} valueTitle={m.common_click_to_select()}>
				{config.ipRange}
			</KeyValueCard>
		{/if}
		{#if config.auxAddress && Object.keys(config.auxAddress).length > 0}
			{#each Object.entries(config.auxAddress) as [name, addr] (name)}
				<KeyValueCard label={name} valueTitle={m.common_click_to_select()}>{addr}</KeyValueCard>
			{/each}
		{/if}
	</KeyValueGrid>
{/snippet}

{#snippet networkIpam(network: NetworkInspect)}
	{#if network.ipam && (network.ipam.driver || network.ipam.config?.length || Object.keys(network.ipam.options ?? {}).length)}
		<DetailSection title={m.networks_ipam_title()} icon={SettingsIcon}>
			<div class="space-y-3">
				{#each network.ipam.config ?? [] as config (config)}
					{@render ipamSubnet(config)}
				{/each}
			</div>

			{#if network.ipam.driver}
				<div class="flex items-center gap-2">
					<span class="text-sm font-medium text-muted-foreground">{m.networks_ipam_driver_label()}:</span>
					<Badge variant="cyan" minWidth="20">{network.ipam.driver}</Badge>
				</div>
			{/if}

			{#if network.ipam.options && Object.keys(network.ipam.options).length > 0}
				<div>
					<p class="mb-2 text-sm font-medium text-muted-foreground">{m.networks_ipam_options_label()}</p>
					<div class="divide-y divide-border/50">
						{#each Object.entries(network.ipam.options) as [key, value] (key)}
							<div class="flex justify-between py-1.5 font-mono text-xs">
								<span class="text-muted-foreground">{key}:</span>
								<span>{value}</span>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</DetailSection>
	{/if}
{/snippet}

{#snippet keyValueSection(title: string, icon: IconType, values: Record<string, string> | null | undefined)}
	{#if values && Object.keys(values).length > 0}
		<DetailSection {title} {icon}>
			<KeyValueGrid>
				{#each Object.entries(values) as [key, value] (key)}
					<KeyValueCard label={key} valueTitle={m.common_click_to_select()}>{value}</KeyValueCard>
				{/each}
			</KeyValueGrid>
		</DetailSection>
	{/if}
{/snippet}

<TabbedPageLayout backUrl="/networks" tabItems={urlTab.items} selectedTab={urlTab.value} onTabChange={urlTab.select}>
	{#snippet headerInfo()}
		<div class="flex min-h-9 min-w-0 items-center gap-2">
			<NetworksIcon class="size-5 shrink-0 text-primary" />
			<h1 class="min-w-0 truncate font-semibold" title={network.name}>{network.name}</h1>
			<Badge variant="secondary" mono title={network.id}>{m.common_id()}: {shortId}</Badge>
			{#if network.labels?.['com.docker.stack.namespace']}
				<Badge variant="secondary">{m.swarm_stack_namespace({ namespace: network.labels['com.docker.stack.namespace'] })}</Badge>
			{/if}
			{#if swarmGateway}
				<Badge variant="blue">{m.network_swarm_gateway()}</Badge>
			{:else if !swarmScoped}
				<Badge variant={inUse ? 'green' : 'amber'}>
					{inUse ? m.networks_in_use_count({ count: localContainerCount }) : m.common_unused()}
				</Badge>
			{/if}
			{#if isPredefined}
				<Badge variant="blue">{m.networks_predefined()}</Badge>
			{/if}
			<Badge variant="purple">{network.driver ?? m.common_unknown()}</Badge>
		</div>
	{/snippet}
	{#snippet headerActions()}<ActionButtonGroup buttons={actions} />{/snippet}

	{#snippet subHeader()}
		{#if errorMessage}
			<Alert.Root variant="destructive" class="mt-3" icon={AlertIcon}>
				<Alert.Title>{m.common_action_failed()}</Alert.Title>
				<Alert.Description>{errorMessage}</Alert.Description>
			</Alert.Root>
		{/if}
	{/snippet}

	{#snippet tabContent()}
		<Tabs.Content value="overview" layout="panel">
			<div class="space-y-6">
				{#if hasObservations && !observations.isFetching && observationsIncomplete}
					<Alert.Root variant="warning-subtle" size="sm" icon={AlertIcon}>
						<Alert.Description>
							{m.network_observations_not_ready()}
							{#if agents.length > 0}
								<ArcaneButton action="base" size="sm" customLabel={m.common_retry()} onclick={() => observations.refetch()} />
							{/if}
						</Alert.Description>
					</Alert.Root>
				{/if}
				<DetailSection
					title={m.common_details_title({ resource: m.resource_network_cap() })}
					icon={NetworksIcon}
					class="border-t-0 pt-0"
				>
					<KeyValueGrid>
						<KeyValueCard label={m.common_id()}>{network.id}</KeyValueCard>
						<KeyValueCard label={m.common_scope()} valueFormat="text">{network.scope ?? m.common_unknown()}</KeyValueCard>
						<KeyValueCard label={m.common_created()} valueFormat="text">{createdDate}</KeyValueCard>
					</KeyValueGrid>
					{#if network.attachable || network.internal || network.ingress || network.enableIPv6 || network.configOnly}
						<div class="flex flex-wrap gap-2">
							{#if network.attachable}<Badge variant="green">{m.attachable()}</Badge>{/if}
							{#if network.internal}<Badge variant="blue">{m.internal()}</Badge>{/if}
							{#if network.ingress}<Badge variant="cyan">{m.ingress()}</Badge>{/if}
							{#if network.enableIPv6}<Badge variant="indigo">{m.ipv6_enabled()}</Badge>{/if}
							{#if network.configOnly}<Badge variant="pink">{m.config_only()}</Badge>{/if}
						</div>
					{/if}
				</DetailSection>
				{@render networkIpam(network)}
				{@render keyValueSection(m.common_labels(), TagIcon, network.labels)}
				{@render keyValueSection(m.networks_options_title(), SettingsIcon, network.options)}
			</div>
		</Tabs.Content>
		{#if swarmScoped}
			{@const clusterId = network.clusterId ?? ''}
			<Tabs.Content value="services" layout="table">
				{#if urlTab.value === 'services'}
					{#key `${data.envId}:${network.id}`}
						<NetworkSwarmResources kind="services" networkId={network.id} {clusterId} environmentId={data.envId} />
					{/key}
				{/if}
			</Tabs.Content>
			<Tabs.Content value="tasks" layout="table">
				{#if urlTab.value === 'tasks'}
					{#key `${data.envId}:${network.id}`}
						<NetworkSwarmResources kind="tasks" networkId={network.id} {clusterId} environmentId={data.envId} />
					{/key}
				{/if}
			</Tabs.Content>
		{/if}
	{/snippet}
</TabbedPageLayout>
