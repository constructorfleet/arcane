import 'temporal-polyfill/global';
import { describe, expect, it, vi } from 'vite-plus/test';

import type { ContainerStatusCounts } from '#lib/types/container.js';

import { getContainerAttentionFailures, type EnvironmentEvent } from './container-attention.js';

vi.mock('#lib/utils/formatting.js', () => ({ parseInstant: (value: string) => Temporal.Instant.from(value) }));

const now = Temporal.Instant.from('2026-11-21T09:30:00Z');

function exit(environmentId: string, resourceId: string, minute: number): EnvironmentEvent {
	const timestamp = now.subtract({ minutes: minute }).toString();
	return {
		environmentId,
		event: {
			id: `${environmentId}:${resourceId}:${minute}`,
			type: 'container.die',
			severity: 'error',
			title: 'Container exited',
			resourceId,
			resourceName: resourceId,
			metadata: { exitCode: '1' },
			timestamp,
			createdAt: timestamp
		}
	};
}

function states(attention: ContainerStatusCounts['attention']): ContainerStatusCounts {
	return {
		runningContainers: 0,
		stoppedContainers: 0,
		totalContainers: 0,
		unhealthyContainers: 0,
		updatesAvailable: 0,
		attention
	};
}

describe('Dashboard container attention', () => {
	it('uses agent-local current state when restart events are absent', () => {
		const containers = new Map([
			['swarm01', states({ candidateIds: [], healthySwarmSlots: [], swarmSlots: {} })],
			['swarm02', states({ candidateIds: ['same-id'], healthySwarmSlots: [], swarmSlots: {} })]
		]);
		const warnings = getContainerAttentionFailures(
			[exit('swarm01', 'same-id', 5), exit('swarm02', 'same-id', 4)],
			containers,
			now
		);

		expect(warnings.map((item) => item.key)).toEqual(['swarm02:same-id']);
	});

	it('clears a slot after a healthy replacement and groups unresolved attempts', () => {
		const containers = new Map([
			[
				'swarm01',
				states({
					candidateIds: ['replaced', 'older', 'newer'],
					swarmSlots: { replaced: 'service:1', older: 'service:2', newer: 'service:2' },
					healthySwarmSlots: []
				})
			],
			['swarm02', states({ candidateIds: [], swarmSlots: {}, healthySwarmSlots: ['service:1', 'other-service:2'] })]
		]);
		const warnings = getContainerAttentionFailures(
			[exit('swarm01', 'replaced', 5), exit('swarm01', 'newer', 2), exit('swarm01', 'older', 3)],
			containers,
			now
		);

		expect(warnings).toHaveLength(1);
		expect(warnings[0]).toMatchObject({
			key: 'swarm-slot:service:2',
			count: 2,
			entry: { event: { timestamp: now.subtract({ minutes: 2 }).toString() } }
		});
	});

	it('keeps event-only behavior during a mixed-version agent rollout', () => {
		const warnings = getContainerAttentionFailures(
			[exit('swarm01', 'failed', 5)],
			new Map([['swarm01', states(undefined)]]),
			now
		);

		expect(warnings.map((item) => item.key)).toEqual(['swarm01:failed']);
	});
});
