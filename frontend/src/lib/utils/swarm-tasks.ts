import type { ContainerDetails } from '#lib/types/container.js';
import type { SwarmNode, SwarmTaskSummary } from '#lib/types/swarm.js';
import { instantEpochMilliseconds } from '#lib/utils/formatting.js';

const SWARM_TASK_STATE_ORDER: Record<string, number> = {
	running: 0,
	starting: 1,
	pending: 2,
	ready: 3,
	complete: 4,
	shutdown: 5,
	failed: 6,
	rejected: 7,
	orphaned: 8,
	remove: 9
};

const taskStateVariants = new Map<string, 'green' | 'amber' | 'red'>([
	['running', 'green'],
	['pending', 'amber'],
	['starting', 'amber'],
	['failed', 'red'],
	['rejected', 'red']
]);

export function getSwarmTaskStateVariant(state: string): 'green' | 'amber' | 'red' | 'gray' {
	return taskStateVariants.get(state) ?? 'gray';
}

export function getSwarmTaskIconVariant(state: string): 'emerald' | 'amber' | 'red' | 'gray' {
	const variant = getSwarmTaskStateVariant(state);
	return variant === 'green' ? 'emerald' : variant;
}

export function sortSwarmTasks(raw: SwarmTaskSummary[]): SwarmTaskSummary[] {
	const updatedMs = new Map(raw.map((t) => [t, instantEpochMilliseconds(t.updatedAt) ?? 0]));
	return [...raw].sort((a, b) => {
		const stateA = SWARM_TASK_STATE_ORDER[a.currentState] ?? 99;
		const stateB = SWARM_TASK_STATE_ORDER[b.currentState] ?? 99;
		if (stateA !== stateB) return stateA - stateB;
		return (updatedMs.get(b) ?? 0) - (updatedMs.get(a) ?? 0);
	});
}

export function getSwarmTaskContainerTarget(
	task: Pick<SwarmTaskSummary, 'nodeId' | 'containerId'>,
	node: Pick<SwarmNode, 'id' | 'agent'>
): { environmentId: string; containerId: string } | null {
	const agent = node.agent;
	if (
		!task.containerId ||
		!task.nodeId ||
		node.id !== task.nodeId ||
		agent.state !== 'connected' ||
		agent.connected === false ||
		agent.reportedNodeId !== task.nodeId ||
		!agent.environmentId ||
		agent.environmentId === '0'
	)
		return null;
	return { environmentId: agent.environmentId, containerId: task.containerId };
}

export function isSwarmTaskContainerRunning(
	task: Pick<SwarmTaskSummary, 'id' | 'containerId' | 'currentState'>,
	container: Pick<ContainerDetails, 'id' | 'state' | 'labels'>
): boolean {
	return (
		task.currentState === 'running' &&
		!!task.containerId &&
		container.id === task.containerId &&
		container.labels?.['com.docker.swarm.task.id'] === task.id &&
		container.state.running === true &&
		container.state.status === 'running'
	);
}
