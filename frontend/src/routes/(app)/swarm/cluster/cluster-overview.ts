import type { Paginated } from '#lib/types/base.js';
import type { SwarmNode, SwarmServiceSummary } from '#lib/types/swarm.js';
import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';

export async function listClusterResources<T extends { id: string }>(
	fetchPage: (request: SearchPaginationSortRequest) => Promise<Paginated<T>>,
	sortColumn: string
): Promise<T[]> {
	const items = new Map<string, T>();
	let totalPages = 1;
	for (let page = 1; page <= totalPages; page++) {
		const result = await fetchPage({ pagination: { page, limit: 100 }, sort: { column: sortColumn, direction: 'asc' } });
		totalPages = result.pagination.totalPages;
		for (const item of result.data) items.set(item.id, item);
	}
	return [...items.values()];
}

export function hasConnectedAgent(node: SwarmNode): boolean {
	return node.agent?.state === 'connected' && node.agent.connected !== false;
}

export function summarizeNodes(nodes: SwarmNode[]) {
	const isReady = (node: SwarmNode) => node.status.toLowerCase() === 'ready';
	const isActive = (node: SwarmNode) => node.availability.toLowerCase() === 'active';
	const managers = nodes.filter((node) => node.role.toLowerCase() === 'manager');

	return {
		leader: managers.find((node) => node.managerStatus?.toLowerCase() === 'leader'),
		managers: managers.length,
		reachable: managers.filter((node) => node.reachability?.toLowerCase() === 'reachable').length,
		quorum: managers.length ? Math.floor(managers.length / 2) + 1 : 0,
		ready: nodes.filter(isReady).length,
		active: nodes.filter(isActive).length,
		connected: nodes.filter(hasConnectedAgent).length,
		unhealthy: nodes.filter(
			(node) => !isReady(node) || (node.role.toLowerCase() === 'manager' && node.reachability?.toLowerCase() !== 'reachable')
		),
		agentGaps: nodes.filter((node) => !hasConnectedAgent(node))
	};
}

export function hasReplicaShortfall(service: SwarmServiceSummary): boolean {
	return (service.mode === 'replicated' || service.mode === 'global') && service.runningReplicas < service.replicas;
}
