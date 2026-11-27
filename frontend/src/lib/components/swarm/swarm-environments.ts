import type { QueryFunctionContext } from '@tanstack/svelte-query';

import { m } from '#lib/paraglide/messages.js';
import { queryKeys } from '#lib/query/query-keys.js';
import { swarmService } from '#lib/services/swarm-service.js';
import { environmentStore, type EnvironmentRef } from '#lib/stores/environment.store.svelte.js';
import { hasPermission } from '#lib/utils/auth.js';
import {
	readableEnvironments,
	collectAcrossEnvironments,
	readEachEnvironment,
	sumAcrossEnvironments,
	type EnvironmentFailure
} from '#lib/utils/environments.js';

export type SwarmEnvironments = {
	environments: EnvironmentRef[];
	managers: EnvironmentRef[];
	clusterIds?: ReadonlyMap<string, string | undefined>;
	failedEnvironments: EnvironmentFailure[];
	scope: string;
};

function swarmReadableEnvironments(permission: string): EnvironmentRef[] {
	return readableEnvironments(permission).filter((environment) => hasPermission('swarm:read', environment.id));
}

export async function loadSwarmEnvironments(permission: string, { all = false } = {}): Promise<SwarmEnvironments> {
	await environmentStore.ready;
	const readable = swarmReadableEnvironments(permission);
	const readStatus = all ? readEachEnvironment : sumAcrossEnvironments;
	const status = await readStatus(readable, (scope, config) => swarmService.getSwarmStatus(scope, config), {
		enabled: false
	});
	const clusterIds = new Map(status.byEnvironment.map((entry) => [entry.environment.id, entry.data.clusterId]));
	const managers = readable.filter((environment) => clusterIds.get(environment.id));
	const clusters = new Set<string>();
	const environments = managers.filter((environment) => {
		const clusterId = clusterIds.get(environment.id) ?? '';
		if (clusters.has(clusterId)) return false;
		clusters.add(clusterId);
		return true;
	});
	return {
		environments,
		managers,
		clusterIds,
		failedEnvironments: status.failedEnvironments,
		scope: environments.map((environment) => environment.id).join(',')
	};
}

export function swarmEnvironmentsQueryOptions(permission: string, { all = false } = {}) {
	return {
		queryKey: [
			...queryKeys.swarm.environments(
				swarmReadableEnvironments(permission)
					.map((environment) => environment.id)
					.join(','),
				permission
			),
			all
		],
		queryFn: () => loadSwarmEnvironments(permission, { all })
	};
}
export function swarmManagerQueryOptions(scope: {
	environmentId: string;
	clusterId?: string;
	nodeId?: string;
	permission: string;
}) {
	return {
		queryKey: ['swarm', 'resource-manager', scope.environmentId, scope.clusterId ?? '', scope.nodeId ?? '', scope.permission],
		queryFn: async ({ client, signal }: QueryFunctionContext): Promise<(EnvironmentRef & { clusterId: string }) | null> => {
			const inventory = await client.fetchQuery({
				...swarmEnvironmentsQueryOptions(scope.permission, { all: true }),
				staleTime: 30_000
			});
			const clusterId = scope.clusterId || inventory.clusterIds?.get(scope.environmentId);
			if (clusterId) {
				const candidates = inventory.managers.filter((manager) => inventory.clusterIds?.get(manager.id) === clusterId);
				const manager = candidates.find((candidate) => candidate.id === scope.environmentId) ?? candidates[0];
				return manager ? { ...manager, clusterId } : null;
			}
			if (!scope.nodeId) return null;
			const candidates = inventory.environments.filter((manager) => hasPermission('swarm-nodes:read', manager.id));
			const observations = [];
			for (let start = 0; start < candidates.length; start += 4) {
				observations.push(
					await collectAcrossEnvironments(candidates.slice(start, start + 4), (manager, config) =>
						swarmService.getNode(scope.nodeId!, manager, { ...config, signal })
					)
				);
			}
			const values = observations.flatMap((observation) => observation.values);
			const failures = observations.flatMap((observation) => observation.failures);
			const matches = values.filter(({ value }) => value.id === scope.nodeId);
			if (matches.length > 1) throw new Error(m.resource_cluster_unverified());
			const match = matches[0];
			const resolvedCluster = match && inventory.clusterIds?.get(match.environment.id);
			if (match && resolvedCluster) return { ...match.environment, clusterId: resolvedCluster };
			if (failures.length > 0) throw new Error(failures.map(({ message }) => message).join('\n'));
			return null;
		}
	};
}
