import { describe, expect, it } from 'vite-plus/test';

import type { SwarmNode } from '#lib/types/swarm.js';

import { listClusterResources, summarizeNodes } from './cluster-overview';

describe('Cluster overview', () => {
	it('counts each native resource once across multiple pages', async () => {
		const pages: number[] = [];
		const items = await listClusterResources(async ({ pagination }) => {
			const page = pagination?.page ?? 0;
			pages.push(page);
			return {
				data: page === 1 ? [{ id: 'one' }, { id: 'two' }] : [{ id: 'two' }, { id: 'three' }],
				pagination: { currentPage: page, totalPages: 2, totalItems: 3, itemsPerPage: 100 }
			};
		}, 'hostname');

		expect(pages).toEqual([1, 2]);
		expect(items.map(({ id }) => id)).toEqual(['one', 'two', 'three']);
	});

	it('keeps scheduling, quorum, and agent coverage independent', () => {
		const leader: SwarmNode = {
			id: 'leader',
			hostname: 'swarm01',
			role: 'manager',
			status: 'ready',
			availability: 'drain',
			managerStatus: 'leader',
			reachability: 'reachable',
			agent: { state: 'connected', connected: true },
			createdAt: '',
			updatedAt: ''
		};
		const unreachable: SwarmNode = {
			...leader,
			id: 'manager',
			role: 'manager',
			managerStatus: 'manager',
			reachability: 'unreachable',
			status: 'down',
			availability: 'active',
			agent: { state: 'offline' }
		};
		const worker: SwarmNode = {
			...leader,
			id: 'worker',
			role: 'worker',
			managerStatus: undefined,
			reachability: undefined,
			availability: 'active',
			agent: { state: 'connected', connected: false }
		};

		expect(summarizeNodes([leader, unreachable, worker])).toEqual({
			leader,
			managers: 2,
			reachable: 1,
			quorum: 2,
			ready: 2,
			active: 2,
			connected: 1,
			unhealthy: [unreachable],
			agentGaps: [unreachable, worker]
		});
	});
});
