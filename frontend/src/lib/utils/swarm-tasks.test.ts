import { describe, expect, it, vi } from 'vite-plus/test';

import type { SwarmNode } from '#lib/types/swarm.js';

import { getSwarmTaskContainerTarget } from './swarm-tasks.js';

vi.mock('#lib/utils/formatting.js', () => ({ instantEpochMilliseconds: () => 0 }));

const task = { nodeId: 'worker-node', containerId: 'task-container' };
const node: Pick<SwarmNode, 'id' | 'agent'> = {
	id: 'worker-node',
	agent: {
		state: 'connected',
		connected: true,
		environmentId: 'worker-agent',
		reportedNodeId: 'worker-node',
		bindingKind: 'discovered'
	}
};

describe('task container destination', () => {
	it('targets the verified worker agent rather than the cluster query agent', () => {
		expect(getSwarmTaskContainerTarget(task, node)).toEqual({ environmentId: 'worker-agent', containerId: 'task-container' });
	});

	it.each(['offline', 'ambiguous', 'mismatched', 'pending', 'none'] as const)('refuses %s coverage', (state) => {
		expect(getSwarmTaskContainerTarget(task, { ...node, agent: { ...node.agent, state } })).toBeNull();
	});

	it('refuses stale, missing, or mismatched node identity', () => {
		for (const agent of [
			{ ...node.agent, connected: false },
			{ ...node.agent, reportedNodeId: undefined },
			{ ...node.agent, reportedNodeId: 'other-node' },
			{ ...node.agent, environmentId: '0' },
			{ ...node.agent, environmentId: undefined }
		])
			expect(getSwarmTaskContainerTarget(task, { ...node, agent })).toBeNull();
		expect(getSwarmTaskContainerTarget(task, { ...node, id: 'other-node' })).toBeNull();
	});

	it('does not invent a container or node for unassigned tasks', () => {
		expect(getSwarmTaskContainerTarget({ ...task, containerId: undefined }, node)).toBeNull();
		expect(getSwarmTaskContainerTarget({ ...task, nodeId: '' }, node)).toBeNull();
	});
});
