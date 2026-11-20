import { error, redirect } from '@sveltejs/kit';

import { m } from '#lib/paraglide/messages.js';
import { queryKeys } from '#lib/query/query-keys.js';
import { containerService } from '#lib/services/container-service.js';
import { volumeService } from '#lib/services/volume-service.js';
import { environmentStore } from '#lib/stores/environment.store.svelte.js';
import type { Volume, VolumeConsumer } from '#lib/types/volume.js';
import { extractApiErrorMessage } from '#lib/utils/api.js';
import { hasPermission } from '#lib/utils/auth.js';
import {
	environmentRows,
	readAcrossEnvironments,
	resolveDetailEnvironmentId,
	type EnvironmentFailure,
	type EnvironmentRow
} from '#lib/utils/environments.js';
import { tryCatch } from '#lib/utils/try-catch.js';

import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, parent, url }) => {
	const { queryClient } = await parent();
	const envId = await resolveDetailEnvironmentId(url);

	const { volumeName } = params;
	let logicalVolume: EnvironmentRow<Volume> | undefined;
	let logicalFailures: EnvironmentFailure[] = [];
	let logicalExcluded: string[] = [];
	const cluster = url.searchParams.get('cluster');
	const driver = url.searchParams.get('driver');
	const requestedAgents = new Set((url.searchParams.get('agents') ?? '').split(',').filter(Boolean));
	if (cluster && driver && requestedAgents.size > 0) {
		const agents = environmentStore.available.filter(
			(agent) => requestedAgents.has(agent.id) && agent.enabled && hasPermission('volumes:list', agent.id)
		);
		if (agents.length === 0) error(403, m.common_access_denied());
		// Resolve the logical entry again before reading a physical volume or opening a workspace.
		const observation = await readAcrossEnvironments(agents, (scope, config) =>
			volumeService.getVolumesForEnvironment(scope, { search: volumeName, pagination: { page: 1, limit: -1 } }, config)
		);
		logicalFailures = observation.failures;
		logicalExcluded = observation.excluded.map((agent) => agent.name);
		const includedAgents = agents.filter((agent) => !observation.excluded.some((excluded) => excluded.id === agent.id));
		const page = observation.value;
		const item = environmentRows(page.data, includedAgents).find(
			(item) => item.name === volumeName && item.driver === driver && item.clusterId === cluster
		);
		if (!item) error(404, m.common_not_found_title({ resource: m.resource_volume_cap() }));
		const targetId = item.environmentId ?? includedAgents[0]!.id;
		if (!includedAgents.some((agent) => agent.id === targetId)) error(403, m.common_access_denied());
		if (!hasPermission('volumes:read', targetId)) error(403, m.common_access_denied());
		logicalVolume = item;
		if (!item.logicalId) {
			logicalVolume = {
				...item,
				workspaceState:
					item.mountpoint && ((item.activeConsumers ?? 0) > 0 || (item.runningHelpers ?? 0) > 0) ? 'active' : 'inactive'
			};
		}
		if (targetId !== envId) {
			const target = new URL(url);
			target.searchParams.set('env', targetId);
			redirect(307, `${target.pathname}${target.search}`);
		}
	}

	const operationResult = await tryCatch(
		(async () => {
			const volume = await queryClient.query({
				queryKey: queryKeys.volumes.detail(envId, volumeName),
				queryFn: () => volumeService.getVolumeForEnvironment(envId, volumeName)
			});

			let containersDetailed: VolumeConsumer[] = [];
			if (volume.containers && volume.containers.length > 0) {
				containersDetailed = await Promise.all(
					volume.containers.map(async (id: string) => {
						const operationResult = await tryCatch(
							(async () => {
								const c = await queryClient.query({
									queryKey: queryKeys.containers.detail(envId, id),
									queryFn: () => containerService.getContainerForEnvironment(envId, id)
								});
								const idVal = c?.id || id;
								const nameVal = c?.name || idVal.substring(0, 12);
								const labels = c?.labels ?? {};
								return {
									id: idVal,
									name: nameVal,
									volumeHelper:
										labels['com.getarcaneapp.internal.resource']?.trim().toLowerCase() === 'true' &&
										labels['com.getarcaneapp.volume-helper']?.trim().toLowerCase() === 'true'
								};
							})()
						);
						if (operationResult.error !== null) {
							return {
								id,
								name: id.substring(0, 12),
								volumeHelper: null,
								detailsError: extractApiErrorMessage(operationResult.error)
							};
						} else {
							return operationResult.data;
						}
					})
				);
			}

			return {
				envId,
				volume,
				logicalVolume,
				logicalFailures,
				logicalExcluded,
				containersDetailed
			};
		})()
	);
	if (operationResult.error !== null) {
		const err = operationResult.error;

		console.error('Failed to load volume:', err);
		if ('status' in err && err.status === 404) throw err;
		throw error(500, err.message || m.failed_to_load_volume_details());
	} else {
		return operationResult.data;
	}
};
