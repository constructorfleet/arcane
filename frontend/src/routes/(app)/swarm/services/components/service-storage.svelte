<script lang="ts">
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';

	import RelatedResourceTable from '#lib/components/resource-detail/related-resource-table.svelte';
	import * as Card from '#lib/components/ui/card/index.js';
	import { VolumesIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { containerService } from '#lib/services/container-service.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SwarmServiceMount } from '#lib/types/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref } from '#lib/utils/environments.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';
	import { getSwarmTaskContainerTarget } from '#lib/utils/swarm-tasks.js';

	let { mounts, environmentId, serviceId }: { mounts: SwarmServiceMount[]; environmentId: string; serviceId: string } = $props();
	const activeAttachment = createQuery(() => {
		const manager = environmentId;
		const service = serviceId;
		return {
			queryKey: ['swarm-service-volume-attachment', manager, service],
			enabled: mounts.some((mount) => mount.type === 'volume' && mount.source) && hasPermission('swarm-nodes:read', manager),
			queryFn: async ({ signal }) => {
				const { data: tasks, pagination } = await swarmService.getServiceTasks(
					service,
					{ pagination: { page: 1, limit: 2 }, filters: { currentState: 'running', desiredState: 'running' } },
					manager
				);
				const [task] = tasks;
				if (pagination.totalItems !== 1 || tasks.length !== 1 || !task?.nodeId || task.serviceId !== service) return null;
				const target = getSwarmTaskContainerTarget(task, await swarmService.getNode(task.nodeId, manager));
				if (
					!target ||
					!hasPermission('containers:read', target.environmentId) ||
					!hasPermission('volumes:read', target.environmentId)
				)
					return null;
				const container = await containerService.getContainerForEnvironment(target.environmentId, target.containerId, { signal });
				if (container.id !== target.containerId || container.labels?.['com.docker.swarm.task.id'] !== task.id) return null;
				return { environmentId: target.environmentId, mounts: container.mounts };
			},
			retry: false
		};
	});

	function volumeHref(mount: SwarmServiceMount): string | undefined {
		const attachment = activeAttachment.isError ? undefined : activeAttachment.data;
		if (!attachment || mount.type !== 'volume' || !mount.source) return undefined;
		const actual = attachment.mounts.find(
			(entry) => entry.type === 'volume' && entry.name === mount.source && entry.destination === mount.target
		);
		if (!actual?.name) return undefined;
		return resourceDetailHref(
			environmentHref(resolve('/(app)/volumes/[volumeName]', { volumeName: actual.name }), attachment.environmentId)
		);
	}

	const rows = $derived(
		mounts.map((mount, index) => {
			const type = mount.type || 'volume';
			return {
				id: `${mount.target}:${index}`,
				href: volumeHref(mount),
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

<Card.Root>
	<Card.Header icon={VolumesIcon}><Card.Title><h2>{m.storage()}</h2></Card.Title></Card.Header>
	<Card.Content>
		<RelatedResourceTable
			columns={[
				m.common_type(),
				m.containers_mount_label_container(),
				m.common_driver(),
				m.resource_mount_access(),
				m.dashboard_meter_gpu_device()
			]}
			{rows}
			empty={m.containers_no_mounts_configured()}
		/>
	</Card.Content>
</Card.Root>
