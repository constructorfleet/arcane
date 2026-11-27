import { parseDocument } from 'yaml';

import type { SwarmResourceReference } from '#lib/types/swarm.js';

export function declaredExternalResources(options: {
	composeContent: string;
	overrideContent?: string;
	kind: 'networks' | 'volumes' | 'secrets' | 'configs';
}) {
	const definitions = new Map<string, Record<string, unknown>>();
	try {
		for (const content of [options.composeContent, options.overrideContent ?? '']) {
			if (!content.trim()) continue;
			const document = parseDocument(content);
			if (document.errors.length) return new Set<string>();
			const root: unknown = document.toJS({ maxAliasCount: 100 });
			if (!root || typeof root !== 'object' || Array.isArray(root)) return new Set<string>();
			const compose = root as Record<string, unknown>;
			if (compose['include']) return new Set<string>();
			const resources = compose[options.kind];
			if (!resources || typeof resources !== 'object' || Array.isArray(resources)) continue;
			for (const [key, value] of Object.entries(resources)) {
				if (!value || typeof value !== 'object' || Array.isArray(value)) {
					definitions.delete(key);
					continue;
				}
				definitions.set(key, { ...definitions.get(key), ...value });
			}
		}
	} catch {
		return new Set<string>();
	}
	const names = new Set<string>();
	for (const [key, definition] of definitions) {
		if (definition['external'] !== true) continue;
		const name = definition['name'] ?? key;
		if (typeof name === 'string' && name && !name.includes('$')) names.add(name);
	}
	return names;
}

export function stackVolumeRelationship(
	volume: {
		labels?: Record<string, string> | null;
		nodeId?: string;
		containers?: string[];
		nodes?: { nodeId: string; containers: string[] }[];
	},
	stackName: string,
	tasks: { nodeId: string; containerId?: string; currentState: string }[]
) {
	if (volume.labels?.['com.docker.stack.namespace'] === stackName) return 'owned' as const;
	const attachments = volume.nodes ?? [volume];
	const referenced = attachments.some(
		(node) =>
			!!node.nodeId &&
			tasks.some(
				(task) =>
					task.currentState === 'running' &&
					task.nodeId === node.nodeId &&
					!!task.containerId &&
					node.containers?.includes(task.containerId)
			)
	);
	return referenced ? ('referenced' as const) : null;
}

export function scopedSwarmMetadata<T extends { id: string; spec: { Name: string; Labels?: Record<string, string> | null } }>(
	items: T[],
	references: SwarmResourceReference[],
	stackName?: string
) {
	const targets = new Map<string, Set<string>>();
	for (const reference of references) {
		const paths = targets.get(reference.id) ?? new Set<string>();
		if (reference.target) paths.add(reference.target);
		targets.set(reference.id, paths);
	}
	return items.flatMap((item) => {
		const owned = !!stackName && item.spec.Labels?.['com.docker.stack.namespace'] === stackName;
		if (!owned && !targets.has(item.id)) return [];
		return [
			{ item, relationship: owned ? ('owned' as const) : ('referenced' as const), targets: [...(targets.get(item.id) ?? [])] }
		];
	});
}
