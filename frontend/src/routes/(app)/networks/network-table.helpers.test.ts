import { describe, expect, it, vi } from 'vite-plus/test';

import type { NetworkSummary } from '#lib/types/network.js';

import { networkSource } from './network-table.helpers';

vi.mock('#lib/paraglide/messages.js', () => ({ m: {} }));
vi.mock('#lib/utils/formatting.js', () => ({ instantEpochMilliseconds: vi.fn() }));

const network: NetworkSummary = {
	id: 'network-7',
	name: 'database',
	driver: 'overlay',
	scope: 'swarm',
	clusterId: 'cluster-7',
	labels: { 'com.docker.stack.namespace': 'observability' },
	options: {},
	ipam: {},
	inUse: true,
	isDefault: false,
	created: ''
};

describe('network provenance', () => {
	it('requires cluster identity for ownership and never calls unowned networks external', () => {
		expect(networkSource(network)).toEqual({ kind: 'stack', name: 'observability', clusterId: 'cluster-7' });
		expect(networkSource({ ...network, clusterId: undefined })).toEqual({ kind: 'unknown' });
		expect(networkSource({ ...network, labels: {} })).toEqual({ kind: 'unknown' });
	});
	it('distinguishes built-in Docker networks from identified Swarm infrastructure', () => {
		expect(networkSource({ ...network, labels: {}, name: 'bridge', driver: 'bridge', scope: 'local', isDefault: true })).toEqual({
			kind: 'docker'
		});
		expect(networkSource({ ...network, labels: {}, name: 'ingress', isDefault: true })).toEqual({ kind: 'system' });
		expect(networkSource({ ...network, labels: {}, name: 'docker_gwbridge', driver: 'bridge', scope: 'local' })).toEqual({
			kind: 'system'
		});
		expect(networkSource({ ...network, labels: {}, name: 'bridge', isDefault: false })).toEqual({ kind: 'unknown' });
	});
});
