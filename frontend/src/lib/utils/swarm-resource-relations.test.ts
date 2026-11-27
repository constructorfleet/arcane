import { describe, expect, it } from 'vite-plus/test';

import { scopedSwarmMetadata, stackVolumeRelationship, declaredExternalResources } from './swarm-resource-relations';

describe('scoped Swarm metadata relationships', () => {
	it('marks only explicit literal Compose external declarations, respecting overrides', () => {
		const composeContent =
			'networks:\n  shared:\n    external: true\n  renamed:\n    name: actual-network\n    external: true\n  unresolved:\n    name: ${NETWORK}\n    external: true\n  owned:\n    driver: overlay\n';
		expect([...declaredExternalResources({ composeContent, kind: 'networks' })]).toEqual(['shared', 'actual-network']);
		expect([
			...declaredExternalResources({
				composeContent,
				kind: 'networks',
				overrideContent: 'networks:\n  shared:\n    external: false\n'
			})
		]).toEqual(['actual-network']);
		expect([...declaredExternalResources({ composeContent: 'include: [other.yaml]\nnetworks: {}', kind: 'networks' })]).toEqual(
			[]
		);
	});
	it('proves volume consumers with both exact task container and node identity', () => {
		const tasks = [{ nodeId: 'worker-one', containerId: 'container-exact', currentState: 'running' }];
		expect(stackVolumeRelationship({ nodeId: 'worker-one', containers: ['container-exact'] }, 'stack', tasks)).toBe('referenced');
		expect(stackVolumeRelationship({ nodeId: 'worker-two', containers: ['container-exact'] }, 'stack', tasks)).toBeNull();
		expect(stackVolumeRelationship({ nodeId: 'worker-one', containers: ['other-container'] }, 'stack', tasks)).toBeNull();
		expect(
			stackVolumeRelationship({ nodeId: 'worker-one', containers: ['container-exact'] }, 'stack', [
				{ ...tasks[0]!, currentState: 'shutdown' }
			])
		).toBeNull();
		expect(stackVolumeRelationship({ labels: { 'com.docker.stack.namespace': 'stack' } }, 'stack', [])).toBe('owned');
	});
	it('joins exact reference IDs and ownership labels, never display names', () => {
		const rows = scopedSwarmMetadata(
			[
				{ id: 'owned', spec: { Name: 'owned', Labels: { 'com.docker.stack.namespace': 'one' } } },
				{ id: 'external-id', spec: { Name: 'shared' } },
				{ id: 'other-id', spec: { Name: 'shared' } }
			],
			[{ id: 'external-id', name: 'shared', target: '/run/key' }],
			'one'
		);
		expect(rows.map((row) => [row.item.id, row.relationship, row.targets])).toEqual([
			['owned', 'owned', []],
			['external-id', 'referenced', ['/run/key']]
		]);
	});
	it('deduplicates repeated references without manufacturing external ownership', () => {
		const rows = scopedSwarmMetadata(
			[{ id: 'a', spec: { Name: 'a' } }],
			[
				{ id: 'a', name: 'a', target: 'config.yaml' },
				{ id: 'a', name: 'a', target: 'config.yaml' }
			]
		);
		expect(rows).toHaveLength(1);
		expect(rows[0]?.targets).toEqual(['config.yaml']);
		expect(rows[0]?.relationship).toBe('referenced');
	});
});
