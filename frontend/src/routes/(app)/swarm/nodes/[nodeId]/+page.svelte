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
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref, environmentName, readableEnvironments } from '#lib/utils/environments.js';
	import { bytes, capitalizeFirstLetter, formatDateTime } from '#lib/utils/formatting.js';

	import { getSwarmNodeAgentLabel, getSwarmNodeAgentVariant } from '../agent-status';
	import NodeDockerDetails from '../components/node-docker-details.svelte';
	import NodeDockerNetworks from '../components/node-docker-networks.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const node = $derived(data.node);
	const agent = $derived(node.agent);
	const agentId = $derived(
		agent.reportedNodeId === node.id && (agent.bindingKind === 'environment' || agent.bindingKind === 'discovered')
			? agent.environmentId
			: undefined
	);
	const canListNetworks = $derived(!!agentId && readableEnvironments('networks:list', agentId).length > 0);
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
	const urlTab = useUrlTab({ validTabs: () => tabItems.map((tab) => tab.value), defaultTab: () => 'overview' });
	const selectedTab = $derived(urlTab.value);
	const dockerQuery = createQuery(() => ({
		queryKey: queryKeys.system.dockerInfo(agentId ?? ''),
		queryFn: () => systemService.getDockerInfoForEnvironment(agentId!),
		enabled: canReadDocker && (selectedTab === 'host' || selectedTab === 'docker')
	}));
	const docker = $derived(
		canReadDocker && !dockerQuery.isError && dockerQuery.data?.Swarm?.NodeID === node.id ? dockerQuery.data : undefined
	);
	const resourceLinks = $derived(
		(
			[
				{ resource: 'containers', label: m.containers() },
				{ resource: 'images', label: m.images() },
				{ resource: 'volumes', label: m.resource_volumes_cap() },
				{ resource: 'networks', label: m.resource_networks_cap() }
			] as const
		).flatMap(({ resource, label }) =>
			agentId && readableEnvironments(`${resource}:list`, agentId).length
				? [{ href: environmentHref(resolve(`/(app)/${resource}`), agentId), label }]
				: []
		)
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
		<h1 class="min-w-0 truncate text-sm font-semibold">{node.hostname}</h1>
	{/snippet}
	{#snippet headerActions()}
		{#if hasPermission('swarm-nodes:read', data.envId)}
			<a
				class="text-sm font-medium text-primary hover:underline"
				href={environmentHref(resolve('/(app)/swarm/tasks') + '?nodeId=' + encodeURIComponent(node.id), data.envId)}
				>{m.view_tasks()}</a
			>
		{/if}
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
				{#if Object.keys(node.labels ?? {}).length || Object.keys(node.systemLabels ?? {}).length}
					<section class="space-y-2">
						<h2 class="text-sm font-semibold">{m.common_labels()}</h2>
						<div class="flex flex-wrap gap-2">
							{#each Object.entries(node.systemLabels ?? {}) as [key, value] (key)}
								<Badge variant="gray" class="max-w-full break-all">{key}={value}</Badge>
							{/each}
							{#each Object.entries(node.labels ?? {}) as [key, value] (key)}
								<Badge variant="blue" class="max-w-full break-all">{key}={value}</Badge>
							{/each}
						</div>
					</section>
				{/if}
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
			</div>
		</Tabs.Content>
		<Tabs.Content value="resources">
			<div class={contentClass}>
				{#if resourceLinks.length}
					<div class="flex flex-wrap gap-2">
						{#each resourceLinks as { href, label } (href)}
							<a {href} class="rounded-md border border-border px-3 py-2 text-sm font-medium hover:bg-muted">{label}</a>
						{/each}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">{m.swarm_node_local_resources_unavailable()}</p>
				{/if}
				{#if selectedTab === 'resources' && canListNetworks && agentId}
					{#key `${node.id}:${agentId}`}
						<NodeDockerNetworks nodeId={node.id} {agentId} />
					{/key}
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
