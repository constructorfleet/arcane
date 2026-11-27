import { queryOptions } from '@tanstack/svelte-query';

import { swarmManagerQueryOptions } from '#lib/components/swarm/swarm-environments.js';
import { m } from '#lib/paraglide/messages.js';
import { queryKeys } from '#lib/query/query-keys.js';
import { containerService } from '#lib/services/container-service.js';
import { swarmService } from '#lib/services/swarm-service.js';
import type { EnvironmentRef } from '#lib/stores/environment.store.svelte.js';
import type { ContainerListRequestOptions } from '#lib/types/ui/container.js';
import { fetchAcrossEnvironments } from '#lib/utils/environments.js';
import { verifiedConsumerServices } from '#lib/utils/resource-consumers.js';

export function containerListQuery(scope: string, environments: EnvironmentRef[], requestOptions: ContainerListRequestOptions) {
	return queryOptions({
		queryKey: queryKeys.containers.list(scope, requestOptions),
		queryFn: () =>
			fetchAcrossEnvironments(environments, requestOptions, (id, request, config) =>
				containerService.getContainersForEnvironment(id, request, config)
			)
	});
}

export function containerDetailQuery(environmentId: string, containerId: string) {
	return queryOptions({
		queryKey: queryKeys.containers.detail(environmentId, containerId),
		queryFn: () => containerService.getContainerForEnvironment(environmentId, containerId)
	});
}

export function containerOwnershipQuery(scope: { environmentId: string; containerId: string }) {
	return queryOptions({
		queryKey: ['containers', 'ownership', scope.environmentId, scope.containerId],
		staleTime: 15_000,
		queryFn: async ({ client }) => {
			const container = await client.fetchQuery({
				...containerDetailQuery(scope.environmentId, scope.containerId),
				staleTime: 5_000
			});
			const serviceId = container.labels?.['com.docker.swarm.service.id'];
			if (!serviceId) return null;
			const manager = await client.fetchQuery(
				swarmManagerQueryOptions({
					environmentId: scope.environmentId,
					nodeId: container.labels?.['com.docker.swarm.node.id'],
					permission: 'swarm-services:read'
				})
			);
			if (!manager) throw new Error(m.resource_cluster_unverified());
			const [service, tasks] = await Promise.all([
				swarmService.getService(serviceId, manager.id),
				swarmService.getServiceTasks(serviceId, { pagination: { page: 1, limit: -1 } }, manager.id)
			]);
			const verified = verifiedConsumerServices({
				containers: [{ id: container.id, labels: container.labels ?? {} }],
				tasks: tasks.data,
				requireRunning: false
			});
			if (service.id !== serviceId || verified.incomplete || !verified.serviceIds.includes(serviceId))
				throw new Error(m.resource_cluster_unverified());
			const serviceName = typeof service.spec['Name'] === 'string' ? service.spec['Name'] : serviceId;
			const labels = service.spec['Labels'] as Record<string, unknown> | undefined;
			const stackName =
				typeof labels?.['com.docker.stack.namespace'] === 'string' ? labels['com.docker.stack.namespace'] : undefined;
			return { manager, serviceId, serviceName, stackName };
		}
	});
}
