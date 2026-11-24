<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import { KeyValueCard } from '#lib/components/resource-detail/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import { useUrlTab } from '#lib/hooks/use-url-tab.svelte.js';
	import { AgentConnectionIcon, ContainersIcon, CpuIcon, LayersIcon, UsersIcon } from '#lib/icons/index.js';
	import TabbedPageLayout from '#lib/layouts/tabbed-page-layout.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { systemService } from '#lib/services/system-service.js';
	import { environmentName, readableEnvironments } from '#lib/utils/environments.js';
	import { bytes, capitalizeFirstLetter, formatDateTime } from '#lib/utils/formatting.js';

	import { getSwarmNodeAgentLabel, getSwarmNodeAgentVariant } from '../agent-status';
	import NodeDockerDetails from '../components/node-docker-details.svelte';
	import NodeLabels from '../components/node-labels.svelte';
	import NodeLocalResources from '../components/node-local-resources.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const node = $derived(data.node);
	const agent = $derived(node.agent);
	const agentId = $derived(
		agent.reportedNodeId === node.id && (agent.bindingKind === 'environment' || agent.bindingKind === 'discovered')
			? agent.environmentId
			: undefined
	);
	const canReadDocker = $derived(
		!!agentId &&
			agent.state === 'connected' &&
			agent.connected !== false &&
			readableEnvironments('system:read', agentId).length > 0
	);
	const tabItems = $derived([
		{ value: 'overview', label: m.common_overview(), icon: UsersIcon },
		{ value: 'host', label: m.swarm_node_hardware_host(), icon: CpuIcon },
		{ value: 'docker', label: 'Docker', icon: ContainersIcon },
		{ value: 'resources', label: m.swarm_node_local_resources_tab(), icon: LayersIcon },
		{ value: 'agent', label: m.agent(), icon: AgentConnectionIcon }
	]);
	const urlTab = useUrlTab({ history: true, validTabs: () => tabItems.map((tab) => tab.value), defaultTab: () => 'overview' });
	const selectedTab = $derived(urlTab.value);
	const dockerQuery = createQuery(() => ({
		queryKey: queryKeys.system.dockerInfo(agentId ?? ''),
		queryFn: () => systemService.getDockerInfoForEnvironment(agentId!),
		enabled: canReadDocker && (selectedTab === 'host' || selectedTab === 'docker')
	}));
	const docker = $derived(
		canReadDocker && !dockerQuery.isError && dockerQuery.data?.Swarm?.NodeID === node.id ? dockerQuery.data : undefined
	);
	const contentClass = 'space-y-4 rounded-lg border border-border p-4 sm:p-6';
	const fieldClass = 'grid gap-3 sm:grid-cols-2 lg:grid-cols-3';
</script>

<TabbedPageLayout
	backUrl={resolve('/(app)/swarm/nodes')}
	backLabel={m.nodes()}
	{tabItems}
	{selectedTab}
	onTabChange={urlTab.select}
>
	{#snippet headerInfo()}
		<div class="flex min-h-9 items-center gap-2">
			<UsersIcon class="size-5 shrink-0 text-primary" />
			<h1 class="shrink-0 font-semibold">{node.hostname}</h1>
		</div>
	{/snippet}
	{#snippet tabContent()}
		<Tabs.Content value="overview">
			<div class={contentClass}>
				<div class={fieldClass}>
					<KeyValueCard label={m.common_role()} valueFormat="text">{capitalizeFirstLetter(node.role)}</KeyValueCard>
					<KeyValueCard label={m.common_status()}>
						<Badge variant={node.status.toLowerCase() === 'ready' ? 'green' : 'amber'}>{node.status}</Badge>
					</KeyValueCard>
					<KeyValueCard label={m.swarm_availability()} valueFormat="text">{capitalizeFirstLetter(node.availability)}</KeyValueCard
					>
					<KeyValueCard label={m.swarm_node_identity()}>{node.id}</KeyValueCard>
					<KeyValueCard label={m.swarm_node_manager_connection()}>{environmentName(data.envId)}</KeyValueCard>
					{#if node.address}
						<KeyValueCard label={m.swarm_node_address()}>{node.address}</KeyValueCard>
					{/if}
					{#if node.role.toLowerCase() === 'manager'}
						<KeyValueCard label={m.swarm_node_manager_status()}>{node.managerStatus || m.common_unknown()}</KeyValueCard>
						<KeyValueCard label={m.swarm_node_reachability()}>{node.reachability || m.common_unknown()}</KeyValueCard>
					{/if}
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="host">
			<div class={contentClass}>
				<div class={fieldClass}>
					<KeyValueCard label={m.platform()}>{node.platform || m.common_unknown()}</KeyValueCard>
					{#if docker}
						<KeyValueCard label={m.swarm_node_os()}>{docker.OperatingSystem || m.common_unknown()}</KeyValueCard>
						<KeyValueCard label={m.swarm_node_kernel()}>{docker.KernelVersion || m.common_unknown()}</KeyValueCard>
						<KeyValueCard label={m.swarm_node_logical_cpus()}>{docker.NCPU}</KeyValueCard>
						<KeyValueCard label={m.swarm_node_memory()}>{bytes(docker.MemTotal)}</KeyValueCard>
					{/if}
				</div>
				{#if docker}
					<p class="text-sm text-muted-foreground">{m.swarm_node_engine_reported()}</p>
				{:else}
					{@render engineUnavailable()}
				{/if}
			</div>
		</Tabs.Content>
		<Tabs.Content value="docker">
			<div class={contentClass}>
				<KeyValueCard label={m.swarm_engine_version()}>{node.engineVersion || m.common_unknown()}</KeyValueCard>
				{#if docker && agentId}
					<NodeDockerDetails {docker} {agentId} />
				{:else}
					{@render engineUnavailable()}
				{/if}
				{#key `${data.envId}:${node.id}`}<NodeLabels {node} environmentId={data.envId} />{/key}
			</div>
		</Tabs.Content>
		<Tabs.Content value="resources">
			<div class={contentClass}>
				{#if selectedTab === 'resources'}
					{#key `${data.envId}:${node.id}`}<NodeLocalResources
							nodeId={node.id}
							managerId={data.envId}
							agentId={agent.state === 'connected' && agent.connected !== false ? agentId : undefined}
						/>{/key}
				{:else}
					<p class="text-sm text-muted-foreground">{m.swarm_node_local_resources_unavailable()}</p>
				{/if}
			</div>
		</Tabs.Content>
		<Tabs.Content value="agent">
			<div class={contentClass}>
				<div class={fieldClass}>
					<KeyValueCard label={m.common_status()} valueFormat="custom">
						<Badge variant={getSwarmNodeAgentVariant(agent.state)}>{getSwarmNodeAgentLabel(agent.state)}</Badge>
					</KeyValueCard>
					{#if agent.environmentName}
						<KeyValueCard label={m.agent()}>{agent.environmentName}</KeyValueCard>
					{/if}
					{#if agent.environmentId}
						<KeyValueCard label={m.swarm_node_agent_environment_id()}>{agent.environmentId}</KeyValueCard>
					{/if}
					{#if agent.lastHeartbeat}
						<KeyValueCard label={m.swarm_node_last_heartbeat()}>{formatDateTime(agent.lastHeartbeat)}</KeyValueCard>
					{/if}
				</div>
			</div>
		</Tabs.Content>
	{/snippet}
</TabbedPageLayout>

{#snippet engineUnavailable()}
	<p class="text-sm text-muted-foreground">
		{canReadDocker && dockerQuery.isPending ? m.common_loading() : m.swarm_node_engine_unavailable()}
	</p>
{/snippet}
