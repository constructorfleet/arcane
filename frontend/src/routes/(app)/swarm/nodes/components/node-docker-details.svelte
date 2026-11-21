<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';

	import { KeyValueCard } from '#lib/components/resource-detail/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { systemService } from '#lib/services/system-service.js';
	import type { DockerInfo } from '#lib/types/dockerinfo.js';

	let { docker, agentId }: { docker: DockerInfo; agentId: string } = $props();
	const plugins = createQuery(() => ({
		queryKey: queryKeys.system.dockerPlugins(agentId),
		queryFn: () => systemService.listDockerPlugins(agentId)
	}));
	const settings = $derived([
		{ label: m.docker_info_root_dir(), value: docker.DockerRootDir },
		{ label: m.docker_info_api_version_label(), value: docker.apiVersion },
		{ label: m.docker_info_storage_driver_label(), value: docker.Driver },
		{ label: m.docker_info_logging_driver_label(), value: docker.LoggingDriver },
		{ label: m.docker_info_cgroup_driver_label(), value: docker.CgroupDriver },
		{ label: m.docker_info_cgroup_version_label(), value: docker.CgroupVersion },
		{ label: m.docker_info_default_runtime(), value: docker.DefaultRuntime },
		{ label: m.docker_info_debug_label(), value: flag(docker.Debug) },
		{ label: m.docker_info_live_restore_label(), value: flag(docker.LiveRestoreEnabled) },
		{ label: m.docker_info_ipv4_forwarding(), value: flag(docker.IPv4Forwarding) }
	]);

	function flag(value: boolean | undefined) {
		if (value === undefined) return m.common_unknown();
		return value ? m.common_enabled() : m.common_disabled();
	}
</script>

<section class="space-y-3">
	<div>
		<h2 class="text-sm font-semibold">{m.swarm_node_effective_daemon_settings()}</h2>
		<p class="text-sm text-muted-foreground">{m.swarm_node_effective_daemon_description()}</p>
	</div>
	<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
		{#each settings as { label, value } (label)}
			<KeyValueCard {label}>{value || m.common_unknown()}</KeyValueCard>
		{/each}
		{#if docker.DefaultAddressPools?.length}
			<KeyValueCard label={m.docker_info_address_pools_label()} valueFormat="custom">
				<ul class="space-y-1 font-mono text-sm">
					{#each docker.DefaultAddressPools as pool (`${pool.Base}:${pool.Size}`)}
						<li>{m.swarm_node_address_pool({ base: pool.Base, size: pool.Size })}</li>
					{/each}
				</ul>
			</KeyValueCard>
		{/if}
	</div>
</section>

<section class="space-y-3 border-t border-border pt-4">
	<h2 class="text-sm font-semibold">{m.swarm_node_managed_plugins()}</h2>
	{#if plugins.isPending}
		<p class="text-sm text-muted-foreground">{m.common_loading()}</p>
	{:else if plugins.isError}
		<p class="text-sm text-muted-foreground">{m.swarm_node_managed_plugins_unavailable()}</p>
	{:else if plugins.data.length}
		<ul class="divide-y divide-border">
			{#each plugins.data as plugin (plugin.id || plugin.name)}
				<li class="flex flex-wrap items-start justify-between gap-2 py-2">
					<div class="min-w-0">
						<div class="font-medium break-all">{plugin.name}</div>
						{#if plugin.reference && plugin.reference !== plugin.name}
							<div class="font-mono text-xs break-all text-muted-foreground">{plugin.reference}</div>
						{/if}
						{#if plugin.capabilities.length}
							<div class="text-xs text-muted-foreground">{plugin.capabilities.join(', ')}</div>
						{/if}
					</div>
					<Badge variant={plugin.enabled ? 'green' : 'gray'}>
						{plugin.enabled ? m.common_enabled() : m.common_disabled()}
					</Badge>
				</li>
			{/each}
		</ul>
	{:else}
		<p class="text-sm text-muted-foreground">{m.swarm_node_managed_plugins_empty()}</p>
	{/if}
</section>

<section class="space-y-3 border-t border-border pt-4">
	<h2 class="text-sm font-semibold">{m.swarm_node_engine_plugins()}</h2>
	<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
		<KeyValueCard label={m.resource_volumes_cap()}>{docker.Plugins?.Volume?.join(', ') || m.common_na()}</KeyValueCard>
		<KeyValueCard label={m.resource_networks_cap()}>{docker.Plugins?.Network?.join(', ') || m.common_na()}</KeyValueCard>
		<KeyValueCard label={m.common_logs()}>{docker.Plugins?.Log?.join(', ') || m.common_na()}</KeyValueCard>
		<KeyValueCard label={m.docker_info_authorization_plugin()}
			>{docker.Plugins?.Authorization?.join(', ') || m.common_na()}</KeyValueCard
		>
	</div>
</section>
