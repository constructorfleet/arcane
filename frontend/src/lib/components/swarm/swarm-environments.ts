import { queryKeys } from '#lib/query/query-keys.js';
import { swarmService } from '#lib/services/swarm-service.js';
import { environmentStore, type EnvironmentRef } from '#lib/stores/environment.store.svelte.js';
import { hasPermission } from '#lib/utils/auth.js';
import {
	readableEnvironments,
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
