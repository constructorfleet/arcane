import { describe, expect, it } from 'vite-plus/test';

import { consumerMatchesResource, verifiedConsumerServices } from './resource-consumers';

const container = {
	id: 'container-7',
	imageId: 'sha256:image',
	state: 'running',
	mounts: [{ type: 'volume', name: 'data', destination: '/data' }],
	labels: {
		'com.docker.swarm.task.id': 'task-7',
		'com.docker.swarm.service.id': 'service-7',
		'com.docker.swarm.node.id': 'node-7'
	}
};
const task = { id: 'task-7', containerId: 'container-7', serviceId: 'service-7', nodeId: 'node-7', currentState: 'running' };

describe('resource consumers', () => {
	it('uses immutable image identity and exact named volume mounts', () => {
		expect(consumerMatchesResource(container, { imageId: 'sha256:image' })).toBe(true);
		expect(consumerMatchesResource(container, { imageId: 'sha256:other' })).toBe(false);
		expect(consumerMatchesResource(container, { volumeName: 'data' })).toBe(true);
		expect(
			consumerMatchesResource(
				{ ...container, mounts: [{ type: 'bind', name: 'data', source: '/data', destination: '/data' }] },
				{ volumeName: 'data' }
			)
		).toBe(false);
		expect(consumerMatchesResource(container, {})).toBe(false);
	});
	it('requires the exact task, container, service and node join', () => {
		expect(verifiedConsumerServices({ containers: [container], tasks: [task] })).toEqual({
			serviceIds: ['service-7'],
			incomplete: false
		});
		for (const mismatch of [
			{ id: 'other' },
			{ containerId: 'other' },
			{ serviceId: 'other' },
			{ nodeId: 'other' },
			{ currentState: 'shutdown' }
		]) {
			expect(verifiedConsumerServices({ containers: [container], tasks: [{ ...task, ...mismatch }] })).toEqual({
				serviceIds: [],
				incomplete: true
			});
		}
	});
	it('excludes standalone containers and keeps service IDs unique', () => {
		expect(verifiedConsumerServices({ containers: [{ ...container, labels: {} }], tasks: [] })).toEqual({
			serviceIds: [],
			incomplete: false
		});
		expect(verifiedConsumerServices({ containers: [container, container], tasks: [task] })).toEqual({
			serviceIds: ['service-7'],
			incomplete: false
		});
	});
	it('allows verified historical ownership without treating it as a running consumer', () => {
		expect(
			verifiedConsumerServices({ containers: [container], tasks: [{ ...task, currentState: 'shutdown' }], requireRunning: false })
		).toEqual({ serviceIds: ['service-7'], incomplete: false });
	});
});
