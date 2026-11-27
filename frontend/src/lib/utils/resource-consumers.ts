import type { ContainerSummary } from '#lib/types/container.js';
import type { SwarmTaskSummary } from '#lib/types/swarm.js';

type Consumer = Pick<ContainerSummary, 'id' | 'imageId' | 'mounts' | 'labels'>;
type RuntimeTask = Pick<SwarmTaskSummary, 'id' | 'containerId' | 'serviceId' | 'nodeId' | 'currentState'>;

export function consumerMatchesResource(
	container: Pick<Consumer, 'imageId' | 'mounts'>,
	scope: { imageId?: string; volumeName?: string }
): boolean {
	if (!scope.imageId && !scope.volumeName) return false;
	if (scope.imageId && container.imageId !== scope.imageId) return false;
	if (scope.volumeName && !container.mounts.some((mount) => mount.type === 'volume' && mount.name === scope.volumeName))
		return false;
	return true;
}

export function verifiedConsumerServices(scope: {
	containers: Pick<Consumer, 'id' | 'labels'>[];
	tasks: RuntimeTask[];
	requireRunning?: boolean;
}): { serviceIds: string[]; incomplete: boolean } {
	const serviceIds = new Set<string>();
	const tasksById = new Map(scope.tasks.map((task) => [task.id, task]));
	let incomplete = false;
	for (const container of scope.containers) {
		const serviceId = container.labels['com.docker.swarm.service.id'];
		if (!serviceId) continue;
		const task = tasksById.get(container.labels['com.docker.swarm.task.id'] ?? '');
		if (
			!task ||
			task.containerId !== container.id ||
			task.serviceId !== serviceId ||
			task.nodeId !== container.labels['com.docker.swarm.node.id'] ||
			(scope.requireRunning !== false && task.currentState.toLowerCase() !== 'running')
		) {
			incomplete = true;
			continue;
		}
		serviceIds.add(task.serviceId);
	}
	return { serviceIds: [...serviceIds], incomplete };
}
