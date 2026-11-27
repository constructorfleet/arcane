import type {
	SwarmConfig,
	SwarmSecret,
	SwarmServiceInspect,
	SwarmServiceSummary,
	SwarmStackInspect,
	SwarmStackSource,
	SwarmStackSummary,
	SwarmTaskSummary
} from '../../../frontend/src/lib/types/swarm';
import { automationLabels, contractTimestamp } from './node-projects';
import { imageId } from './docker';

export const serviceSummary: SwarmServiceSummary = {
	id: 'service-1',
	name: 'observability_prometheus',
	image: 'registry.example.test/observability/prometheus:3.0',
	mode: 'replicated',
	replicas: 5,
	runningReplicas: 4,
	statusObserved: true,
	completedReplicas: 0,
	ports: [],
	createdAt: contractTimestamp,
	updatedAt: contractTimestamp,
	labels: { ...automationLabels, 'com.docker.stack.namespace': 'observability' },
	stackName: 'observability',
	nodes: ['QA node'],
	networks: ['observability_default'],
	networkIds: ['network-1'],
	mounts: [{ type: 'volume', source: 'observability_data', target: '/prometheus' }],
	secrets: [{ id: 'secret-1', name: 'api-key', target: 'api-key' }],
	configs: [{ id: 'config-1', name: 'prometheus-config', target: '/etc/prometheus.yml' }]
};
export const serviceInspect: SwarmServiceInspect = {
	id: serviceSummary.id,
	createdAt: contractTimestamp,
	updatedAt: contractTimestamp,
	version: { Index: 1 },
	nodes: serviceSummary.nodes,
	mounts: serviceSummary.mounts,
	endpoint: { Ports: [], VirtualIPs: [{ NetworkID: 'network-1', Addr: '10.0.7.2/24' }] },
	networkDetails: {
		'network-1': {
			id: 'network-1',
			name: 'observability_default',
			driver: 'overlay',
			scope: 'swarm',
			internal: false,
			attachable: true,
			ingress: false,
			enableIPv4: true,
			enableIPv6: false,
			configOnly: false
		}
	},
	spec: {
		Name: serviceSummary.name,
		Labels: serviceSummary.labels,
		Mode: { Replicated: { Replicas: 5 } },
		TaskTemplate: {
			ContainerSpec: {
				Image: serviceSummary.image,
				Env: ['APP=1'],
				Secrets: [{ SecretID: 'secret-1', SecretName: 'api-key', File: { Name: 'api-key' } }],
				Configs: [
					{
						ConfigID: 'config-1',
						ConfigName: 'prometheus-config',
						File: { Name: '/etc/prometheus.yml' }
					}
				],
				Mounts: [{ Type: 'volume', Source: 'observability_data', Target: '/prometheus' }]
			},
			Networks: [{ Target: 'network-1' }]
		}
	}
};
export const stackSummary: SwarmStackSummary = {
	id: 'observability',
	name: 'observability',
	namespace: 'observability',
	services: 1,
	runningTasks: 4,
	desiredTasks: 5,
	hasSource: true,
	clusterId: 'cluster-1',
	createdAt: contractTimestamp,
	updatedAt: contractTimestamp
};
export const stackInspect: SwarmStackInspect = stackSummary;
export const secret: SwarmSecret = {
	id: 'secret-1',
	createdAt: contractTimestamp,
	updatedAt: contractTimestamp,
	version: { Index: 1 },
	spec: {
		Name: 'api-key',
		Labels: { ...automationLabels, 'com.docker.stack.namespace': 'observability' }
	}
};
export const config: SwarmConfig = {
	id: 'config-1',
	createdAt: contractTimestamp,
	updatedAt: contractTimestamp,
	version: { Index: 1 },
	spec: {
		Name: 'prometheus-config',
		Labels: { ...automationLabels, 'com.docker.stack.namespace': 'observability' }
	}
};
export const runtimeTask: SwarmTaskSummary = {
	id: 'task-1',
	name: 'observability_prometheus.1',
	serviceId: 'service-1',
	serviceName: serviceSummary.name,
	stackName: stackSummary.name,
	nodeId: 'node-1',
	nodeName: 'QA node',
	currentState: 'running',
	desiredState: 'running',
	containerId: 'container-1',
	image: imageId,
	slot: 1,
	createdAt: contractTimestamp,
	updatedAt: contractTimestamp
};
export const stackSource: SwarmStackSource = {
	name: 'observability',
	composeContent:
		'services:\n  prometheus:\n    image: registry.example.test/observability/prometheus:3.0\n',
	envContent: ''
};
