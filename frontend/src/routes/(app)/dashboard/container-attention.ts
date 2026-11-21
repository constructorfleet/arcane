import type { ContainerStatusCounts } from '#lib/types/container.js';
import type { Event } from '#lib/types/event.js';
import { parseInstant } from '#lib/utils/formatting.js';

export interface EnvironmentEvent {
	environmentId: string;
	event: Event;
}

const CONTAINER_EVENT_WINDOW_HOURS = 1;
const CONTAINER_RECOVERY_QUIET_MINUTES = 10;

export function getContainerAttentionFailures(
	events: EnvironmentEvent[],
	containers: ReadonlyMap<string, ContainerStatusCounts>,
	now: Temporal.Instant
) {
	const cutoff = now.subtract({ hours: CONTAINER_EVENT_WINDOW_HOURS });
	const latest = new Map<string, EnvironmentEvent>();
	const failures = new Map<string, number>();
	const started = new Map<string, number>();
	for (const entry of events) {
		const { event } = entry;
		const at = parseInstant(event.timestamp);
		if (!at || Temporal.Instant.compare(at, cutoff) < 0 || !event.resourceId) continue;
		const key = `${entry.environmentId}:${event.resourceId}`;
		if (event.type === 'container.start' || event.type === 'container.restart') {
			started.set(key, Math.max(started.get(key) ?? 0, at.epochMilliseconds));
			continue;
		}
		const failedExit = event.type === 'container.die' && String(event.metadata?.['exitCode'] ?? '0') !== '0';
		if (failedExit) failures.set(key, (failures.get(key) ?? 0) + 1);
		if (!failedExit && event.type !== 'container.oom') continue;
		const existing = latest.get(key);
		if (!existing || (parseInstant(existing.event.timestamp)?.epochMilliseconds ?? 0) < at.epochMilliseconds) {
			latest.set(key, entry);
		}
	}
	const quietSince = now.subtract({ minutes: CONTAINER_RECOVERY_QUIET_MINUTES }).epochMilliseconds;
	const attentionIdsByEnvironment = new Map<string, Set<string>>();
	const healthySwarmSlots = new Set<string>();
	for (const [environmentId, counts] of containers) {
		if (!counts.attention) continue;
		attentionIdsByEnvironment.set(environmentId, new Set(counts.attention.candidateIds));
		for (const slot of counts.attention.healthySwarmSlots) healthySwarmSlots.add(slot);
	}
	const failuresByResource = new Map<string, { key: string; entry: EnvironmentEvent; name: string; count: number }>();
	for (const [key, entry] of latest) {
		const { event, environmentId } = entry;
		const resourceId = event.resourceId ?? '';
		const at = parseInstant(event.timestamp)?.epochMilliseconds ?? 0;
		const attentionIds = attentionIdsByEnvironment.get(environmentId);
		const slot = containers.get(environmentId)?.attention?.swarmSlots[resourceId];
		if (attentionIds) {
			if (!attentionIds.has(resourceId) || (slot && healthySwarmSlots.has(slot))) continue;
		} else {
			const startedAt = started.get(key) ?? 0;
			const recentRepeat = (failures.get(key) ?? 0) > 1 && startedAt > quietSince;
			if (startedAt > at && !recentRepeat) continue;
		}
		const groupKey = slot ? `swarm-slot:${slot}` : key;
		const existing = failuresByResource.get(groupKey);
		const count = (existing?.count ?? 0) + (failures.get(key) ?? 1);
		if (existing && (parseInstant(existing.entry.event.timestamp)?.epochMilliseconds ?? 0) >= at) {
			existing.count = count;
			continue;
		}
		failuresByResource.set(groupKey, {
			key: groupKey,
			entry,
			name: event.resourceName || String(event.metadata?.['name'] ?? resourceId),
			count
		});
	}
	return [...failuresByResource.values()];
}
