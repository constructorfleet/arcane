<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import { ArcaneButton } from '#lib/components/arcane-button/index.js';
	import RelatedResourceTable from '#lib/components/resource-detail/related-resource-table.svelte';
	import * as Card from '#lib/components/ui/card/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { containerService } from '#lib/services/container-service.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SwarmServiceMount } from '#lib/types/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref } from '#lib/utils/environments.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';
	import { getSwarmTaskContainerTarget } from '#lib/utils/swarm-tasks.js';

	let { mounts, environmentId, serviceId }: { mounts: SwarmServiceMount[]; environmentId: string; serviceId: string } = $props();
	const activeAttachments = createQuery(() => {
		const manager = environmentId;
		const service = serviceId;
		return {
			queryKey: ['swarm-service-volume-attachment', manager, service],
			enabled: mounts.some((mount) => mount.type === 'volume' && mount.source) && hasPermission('swarm-nodes:read', manager),
			queryFn: async ({ signal }) => {
				const { data: tasks, pagination } = await swarmService.getServiceTasks(
					service,
					{ pagination: { page: 1, limit: 12 }, filters: { currentState: 'running', desiredState: 'running' } },
					manager
				);
				const nodes = new Map<string, ReturnType<typeof swarmService.getNode>>();
				const observations = await Promise.allSettled(
					tasks.map(async (task) => {
						if (!task.nodeId || task.serviceId !== service || task.currentState !== 'running' || task.desiredState !== 'running')
							return null;
						const request = nodes.get(task.nodeId) ?? swarmService.getNode(task.nodeId, manager);
						nodes.set(task.nodeId, request);
						const node = await request;
						const target = getSwarmTaskContainerTarget(task, node);
						if (
							!target ||
							!hasPermission('containers:read', target.environmentId) ||
							!hasPermission('volumes:read', target.environmentId)
						)
							return null;
						const container = await containerService.getContainerForEnvironment(target.environmentId, target.containerId, {
							signal
						});
						if (
							!container.state?.running ||
							container.id !== target.containerId ||
							container.labels?.['com.docker.swarm.task.id'] !== task.id
						)
							return null;
						return { environmentId: target.environmentId, node: node.hostname, mounts: container.mounts };
					})
				);
				return {
					observations: observations.flatMap((observation) =>
						observation.status === 'fulfilled' && observation.value ? [observation.value] : []
					),
					incomplete:
						observations.some((observation) => observation.status === 'rejected') ||
						(pagination?.totalItems ?? tasks.length) > tasks.length
				};
			},
			retry: false
		};
	});

	function volumeLinks(mount: SwarmServiceMount) {
		if (mount.type !== 'volume' || !mount.source) return [];
		const seen = new Set<string>();
		return (activeAttachments.data?.observations ?? []).flatMap((attachment) => {
			const actual = attachment.mounts.find(
				(entry) => entry.type === 'volume' && entry.name === mount.source && entry.destination === mount.target
			);
			if (!actual?.name || seen.has(attachment.environmentId) || !hasPermission('volumes:read', attachment.environmentId))
				return [];
			seen.add(attachment.environmentId);
			return [
				{
					node: attachment.node,
					href: resourceDetailHref(
						environmentHref(resolve('/(app)/volumes/[volumeName]', { volumeName: actual.name }), attachment.environmentId)
					)
				}
			];
		});
	}

	const rows = $derived(
		mounts.map((mount, index) => {
			const type = mount.type || 'volume';
			const links = volumeLinks(mount);
			return {
				id: `${mount.target}:${index}`,
				href: links.length === 1 ? links[0]?.href : undefined,
				inspect:
					type === 'volume'
						? links.length
							? links.map(({ node, href }) => ({
									label: links.length === 1 ? m.common_inspect() : m.swarm_volume_inspect_on_node({ node }),
									href
								}))
							: [
									{
										label:
											activeAttachments.isPending && activeAttachments.isFetching
												? m.common_loading()
												: m.swarm_volume_inspect_unavailable()
									}
								]
						: undefined,
				name: mount.source || (type === 'tmpfs' ? m.containers_mount_type_tmpfs() : m.image_update_auth_anonymous()),
				values: [
					type,
					mount.target,
					mount.volumeDriver,
					mount.readOnly ? m.common_ro() : m.common_rw(),
					mount.devicePath || mount.volumeOptions?.['device']
				]
			};
		})
	);
</script>

{#if activeAttachments.isError || activeAttachments.data?.incomplete}<div role="status" class="p-3 text-sm text-warning">
		{activeAttachments.error?.message ?? m.unavailable()}
		<ArcaneButton action="refresh" onclick={() => activeAttachments.refetch()} />
	</div>{/if}
<Card.Root
	><Card.Header><Card.Title>{m.resource_volumes_cap()}</Card.Title></Card.Header><Card.Content>
		<RelatedResourceTable
			persistKey="arcane-swarm-service-storage-table"
			columns={[
				m.common_type(),
				m.containers_mount_label_container(),
				m.common_driver(),
				m.resource_mount_access(),
				m.dashboard_meter_gpu_device()
			]}
			rows={rows.filter((_, index) => mounts[index]?.type === 'volume')}
			empty={m.containers_no_mounts_configured()}
		/>
	</Card.Content></Card.Root
>
{#if mounts.some((mount) => mount.type !== 'volume')}
	<Card.Root
		><Card.Header><Card.Title>{m.common_mounts()}</Card.Title></Card.Header><Card.Content>
			<RelatedResourceTable
				persistKey="arcane-swarm-service-nonvolume-mounts"
				columns={[
					m.common_type(),
					m.containers_mount_label_container(),
					m.common_driver(),
					m.resource_mount_access(),
					m.dashboard_meter_gpu_device()
				]}
				rows={rows.filter((_, index) => mounts[index]?.type !== 'volume')}
			/>
		</Card.Content></Card.Root
	>
{/if}
