import type { QueryFunctionContext } from '@tanstack/svelte-query';
import { afterEach, describe, expect, it, vi } from 'vite-plus/test';

import type { ContainerDetails } from '#lib/types/container.js';

import { containerDetailQuery, containerOwnershipQuery } from './containers';

vi.mock('@tanstack/svelte-query', () => ({ queryOptions: (options: unknown) => options }));

const mocks = vi.hoisted(() => ({ getContainerForEnvironment: vi.fn(), getService: vi.fn(), getServiceTasks: vi.fn() }));
vi.mock('#lib/services/container-service.js', () => ({
	containerService: { getContainerForEnvironment: mocks.getContainerForEnvironment }
}));
vi.mock('#lib/services/swarm-service.js', () => ({
	swarmService: { getService: mocks.getService, getServiceTasks: mocks.getServiceTasks }
}));
vi.mock('#lib/components/swarm/swarm-environments.js', () => ({
	swarmManagerQueryOptions: () => ({
		queryKey: ['test-manager'],
		queryFn: () => ({ id: 'manager-1', name: 'Manager', clusterId: 'cluster-1' })
	})
}));
vi.mock('#lib/utils/environments.js', () => ({ fetchAcrossEnvironments: vi.fn() }));
vi.mock('#lib/paraglide/messages.js', () => ({ m: { resource_cluster_unverified: () => 'Unverified cluster' } }));

const labels = {
	'com.docker.swarm.service.id': 'service-1',
	'com.docker.swarm.task.id': 'task-1',
	'com.docker.swarm.node.id': 'node-1'
};
const container: ContainerDetails = {
	id: 'container-1',
	name: 'example',
	image: 'example:1',
	imageId: 'sha256:example',
	created: '2026-11-26T12:00:00Z',
	state: { status: 'exited', running: false },
	config: {},
	hostConfig: {},
	networkSettings: { networks: {} },
	ports: [],
	mounts: [],
	labels
};
const task = {
	id: 'task-1',
	name: 'example.1',
	containerId: 'container-1',
	nodeId: 'node-1',
	nodeName: 'node',
	serviceId: 'service-1',
	serviceName: 'example',
	desiredState: 'shutdown',
	currentState: 'shutdown',
	createdAt: container.created,
	updatedAt: container.created
};
const service = {
	id: 'service-1',
	version: { index: 1 },
	createdAt: container.created,
	updatedAt: container.created,
	endpoint: {},
	spec: { Name: 'observability_prometheus', Labels: { 'com.docker.stack.namespace': 'observability' } }
};
const pagination = { totalPages: 1, totalItems: 1, currentPage: 1, itemsPerPage: -1, grandTotalItems: 1, hasNextPage: false };
function ownershipQuery(observed: ContainerDetails = container) {
	const options = containerOwnershipQuery({ environmentId: 'agent-1', containerId: 'container-1' });
	const fetchQuery = vi
		.fn()
		.mockResolvedValueOnce(observed)
		.mockResolvedValueOnce({ id: 'manager-1', name: 'Manager', clusterId: 'cluster-1' });
	const client = { fetchQuery } as unknown as QueryFunctionContext['client'];
	if (typeof options.queryFn !== 'function') throw new Error('Ownership query requires a query function');
	const result = options.queryFn({ client, queryKey: options.queryKey, signal: new AbortController().signal, meta: undefined });
	return { result, fetchQuery };
}
afterEach(() => vi.clearAllMocks());

describe('container owner resolution', () => {
	it('uses the cached inspect query and verifies the task chain before exposing service and stack', async () => {
		mocks.getService.mockResolvedValue(service);
		mocks.getServiceTasks.mockResolvedValue({ data: [task], pagination });
		const { result, fetchQuery } = ownershipQuery();
		await expect(result).resolves.toMatchObject({
			serviceId: 'service-1',
			serviceName: 'observability_prometheus',
			stackName: 'observability',
			manager: { id: 'manager-1', clusterId: 'cluster-1' }
		});
		expect(fetchQuery).toHaveBeenNthCalledWith(
			1,
			expect.objectContaining({ queryKey: containerDetailQuery('agent-1', 'container-1').queryKey, staleTime: 5_000 })
		);
		expect(mocks.getContainerForEnvironment).not.toHaveBeenCalled();
		expect(mocks.getServiceTasks).toHaveBeenCalledWith('service-1', { pagination: { page: 1, limit: -1 } }, 'manager-1');
	});
	it('refuses service ownership when the observed container does not match the task', async () => {
		mocks.getService.mockResolvedValue({ ...service, spec: {} });
		mocks.getServiceTasks.mockResolvedValue({
			data: [{ ...task, containerId: 'other-container' }],
			pagination
		});
		await expect(ownershipQuery().result).rejects.toThrow();
	});
	it('does not invent a Swarm owner for standalone containers', async () => {
		const { result, fetchQuery } = ownershipQuery({ ...container, labels: {} });
		await expect(result).resolves.toBeNull();
		expect(fetchQuery).toHaveBeenCalledTimes(1);
		expect(mocks.getService).not.toHaveBeenCalled();
	});
});
