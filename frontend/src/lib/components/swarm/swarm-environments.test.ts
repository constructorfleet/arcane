import type { QueryFunctionContext } from '@tanstack/svelte-query';
import { afterEach, describe, expect, it, vi } from 'vite-plus/test';

import { collectAcrossEnvironments } from '#lib/utils/environments.js';

import { swarmManagerQueryOptions, type SwarmEnvironments } from './swarm-environments.js';

vi.mock('#lib/stores/environment.store.svelte.js', () => ({ environmentStore: { ready: Promise.resolve() } }));
vi.mock('#lib/services/swarm-service.js', () => ({ swarmService: { getNode: vi.fn() } }));
vi.mock('#lib/utils/auth.js', () => ({ hasPermission: () => true }));
vi.mock('#lib/utils/environments.js', () => ({
	readableEnvironments: () => [],
	readEachEnvironment: vi.fn(),
	sumAcrossEnvironments: vi.fn(),
	collectAcrossEnvironments: vi.fn()
}));
vi.mock('#lib/paraglide/messages.js', () => ({ m: { resource_cluster_unverified: () => 'Unverified cluster' } }));

const first = { id: 'manager-a', name: 'Same name' };
const second = { id: 'manager-b', name: 'Same name' };
const replica = { id: 'manager-a2', name: 'Replica' };
const inventory: SwarmEnvironments = {
	environments: [first, second],
	managers: [first, replica, second],
	clusterIds: new Map([
		[first.id, 'cluster-a'],
		[replica.id, 'cluster-a'],
		[second.id, 'cluster-b']
	]),
	failedEnvironments: [],
	scope: 'manager-a,manager-b'
};
async function resolveManager(scope: Parameters<typeof swarmManagerQueryOptions>[0], discovery = inventory) {
	const options = swarmManagerQueryOptions(scope);
	const client = { fetchQuery: vi.fn().mockResolvedValue(discovery) } as unknown as QueryFunctionContext['client'];
	return options.queryFn({ client, queryKey: options.queryKey, signal: new AbortController().signal, meta: undefined });
}
afterEach(() => vi.clearAllMocks());

describe('resource manager identity', () => {
	it('uses the verified cluster rather than the query connection or display name', async () => {
		const manager = await resolveManager({ environmentId: first.id, clusterId: 'cluster-b', permission: 'swarm-services:list' });
		expect(manager).toEqual({ ...second, clusterId: 'cluster-b' });
		expect(collectAcrossEnvironments).not.toHaveBeenCalled();
	});
	it('keeps the source manager when it belongs to the verified cluster', async () => {
		expect(await resolveManager({ environmentId: replica.id, permission: 'swarm-services:list' })).toEqual({
			...replica,
			clusterId: 'cluster-a'
		});
	});
	it('resolves an exact worker node using at most one query connection per cluster', async () => {
		vi.mocked(collectAcrossEnvironments).mockResolvedValue({
			values: [{ environment: second, value: { id: 'worker-7' } }],
			failures: []
		});
		expect(
			await resolveManager({ environmentId: 'worker-agent', nodeId: 'worker-7', permission: 'swarm-services:list' })
		).toEqual({ ...second, clusterId: 'cluster-b' });
		expect(vi.mocked(collectAcrossEnvironments).mock.calls[0]?.[0]).toEqual([first, second]);
	});
	it('does not invent a manager without cluster or node identity', async () => {
		expect(await resolveManager({ environmentId: 'worker-agent', permission: 'swarm-services:list' })).toBeNull();
		expect(collectAcrossEnvironments).not.toHaveBeenCalled();
	});
	it('bounds manager probes to four at a time even across a large fleet', async () => {
		const managers = Array.from({ length: 11 }, (_, index) => ({ id: `manager-${index}`, name: `Manager ${index}` }));
		vi.mocked(collectAcrossEnvironments).mockResolvedValue({ values: [], failures: [] });
		await resolveManager(
			{ environmentId: 'worker-agent', nodeId: 'worker-7', permission: 'swarm-services:list' },
			{
				...inventory,
				managers,
				environments: managers,
				clusterIds: new Map(managers.map((manager) => [manager.id, `cluster-${manager.id}`]))
			}
		);
		expect(vi.mocked(collectAcrossEnvironments).mock.calls.map(([batch]) => batch.length)).toEqual([4, 4, 3]);
	});
	it('rejects an ambiguous identity instead of choosing the first cluster', async () => {
		vi.mocked(collectAcrossEnvironments).mockResolvedValue({
			values: [
				{ environment: first, value: { id: 'worker-7' } },
				{ environment: second, value: { id: 'worker-7' } }
			],
			failures: []
		});
		await expect(
			resolveManager({ environmentId: 'worker-agent', nodeId: 'worker-7', permission: 'swarm-services:list' })
		).rejects.toThrow(/cluster/i);
	});
	it('keeps failed discovery distinct from an absent relationship', async () => {
		vi.mocked(collectAcrossEnvironments).mockResolvedValue({
			values: [],
			failures: [{ environment: first, message: 'Disconnected' }]
		});
		await expect(
			resolveManager({ environmentId: 'worker-agent', nodeId: 'worker-7', permission: 'swarm-services:list' })
		).rejects.toThrow('Disconnected');
	});
});
