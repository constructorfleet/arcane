import { DEFAULT_NETWORK_NAMES } from '#lib/constants.js';
import type { NetworkSummary } from '#lib/types/network.js';
import { isSwarmGatewayBridge } from '#lib/utils/docker.js';

export function networkSource(
	network: NetworkSummary
): { kind: 'stack'; name: string; clusterId: string } | { kind: 'docker' | 'system' | 'unknown' } {
	const namespace = network.labels?.['com.docker.stack.namespace'];
	if (namespace && network.clusterId && network.scope === 'swarm')
		return { kind: 'stack', name: namespace, clusterId: network.clusterId };
	if (isSwarmGatewayBridge(network) || (network.isDefault && network.scope === 'swarm' && network.name === 'ingress'))
		return { kind: 'system' };
	if (network.isDefault && network.scope === 'local' && DEFAULT_NETWORK_NAMES.has(network.name)) return { kind: 'docker' };
	return { kind: 'unknown' };
}
