import type { ContainerDetails, ContainerSummary } from '../../../frontend/src/lib/types/container';
import type {
	ImageDetailSummary,
	ImageRepositoryCopy,
	ImageSummary
} from '../../../frontend/src/lib/types/image';
import type { NetworkInspect, NetworkSummary } from '../../../frontend/src/lib/types/network';
import type { Volume } from '../../../frontend/src/lib/types/volume';

const created = '2026-11-26T12:00:00Z';
export const imageId = 'sha256:1111111111111111111111111111111111111111111111111111111111111111';
const digest = 'sha256:2222222222222222222222222222222222222222222222222222222222222222';
const imageReference = 'registry.example.test/observability/prometheus:3.0';
const labels = {
	'com.docker.swarm.service.id': 'service-1',
	'com.docker.swarm.service.name': 'observability_prometheus',
	'com.docker.swarm.task.id': 'task-1',
	'com.docker.swarm.node.id': 'node-1',
	'com.docker.stack.namespace': 'observability',
	'automation.private': 'DO-NOT-RENDER-IN-TABLES'
};
const mounts = [
	{
		type: 'volume',
		name: 'observability_data',
		source: '/var/lib/docker/volumes/observability_data/_data',
		destination: '/prometheus',
		driver: 'local',
		rw: true
	}
];
const networkSettings = {
	networks: { observability_default: { networkId: 'network-1', ipAddress: '10.0.7.9' } }
};
const hostConfig = {
	networkMode: 'observability_default',
	restartPolicy: 'always',
	memory: 536870912
};

export const containerSummary: ContainerSummary = {
	id: 'container-1',
	names: ['/observability_prometheus.1.task-1'],
	image: imageReference,
	imageId,
	command: '/bin/prometheus',
	created: 1795694400,
	labels,
	state: 'running',
	status: 'Up 2 hours (healthy)',
	ports: [{ privatePort: 9090, publicPort: 9090, type: 'tcp' }],
	hostConfig,
	networkSettings,
	mounts
};
export const containerInspect: ContainerDetails = {
	id: 'container-1',
	name: 'observability_prometheus.1.task-1',
	image: imageReference,
	imageId,
	created,
	state: {
		status: 'running',
		running: true,
		startedAt: created,
		health: { status: 'healthy', failingStreak: 0 }
	},
	config: { cmd: ['/bin/prometheus'] },
	labels,
	ports: containerSummary.ports,
	hostConfig,
	networkSettings,
	mounts
};
export const networkSummary: NetworkSummary = {
	id: 'network-1',
	name: 'observability_default',
	clusterId: 'cluster-1',
	nodeId: 'node-1',
	driver: 'overlay',
	scope: 'swarm',
	created,
	options: {},
	labels: {
		'com.docker.stack.namespace': 'observability',
		'automation.private': 'DO-NOT-RENDER-IN-TABLES'
	},
	ipam: {
		driver: 'default',
		config: [
			{ subnet: '10.0.7.0/24', gateway: '10.0.7.1' },
			{ subnet: '10.0.8.0/24', gateway: '10.0.8.1' },
			{ subnet: '10.0.9.0/24', gateway: '10.0.9.1' }
		]
	},
	inUse: true,
	isDefault: false
};
export const networkInspect: NetworkInspect = {
	id: networkSummary.id,
	name: networkSummary.name,
	clusterId: 'cluster-1',
	nodeId: 'node-1',
	driver: 'overlay',
	scope: 'swarm',
	created,
	options: {},
	labels: networkSummary.labels,
	ipam: networkSummary.ipam,
	configFrom: {},
	containersList: [
		{
			id: 'container-1',
			name: containerInspect.name,
			endpointId: 'endpoint-1',
			ipv4Address: '10.0.7.9/24',
			ipv6Address: '',
			macAddress: '02:42:0a:00:07:09'
		}
	],
	enableIPv4: true,
	enableIPv6: false,
	internal: false,
	attachable: true,
	ingress: false,
	configOnly: false
};
export const volume: Volume = {
	id: 'observability_data',
	name: 'observability_data',
	clusterId: 'cluster-1',
	nodeId: 'node-1',
	driver: 'local',
	scope: 'local',
	createdAt: created,
	options: {},
	labels: {
		'com.docker.stack.namespace': 'observability',
		'automation.private': 'DO-NOT-RENDER-IN-TABLES'
	},
	mountpoint: '/var/lib/docker/volumes/observability_data/_data',
	containers: ['container-1'],
	size: 0,
	inUse: true,
	activeConsumers: 1
};
export const imageRepositoryCopies: ImageRepositoryCopy[] = [
	{
		id: imageId,
		artifactKey: `registry.example.test/observability/prometheus@${digest}|linux/amd64`,
		platform: 'linux/amd64',
		environmentId: 'manager-a',
		environmentName: 'QA node',
		tags: [imageReference, 'registry.example.test/observability/prometheus:latest'],
		repoDigests: [`registry.example.test/observability/prometheus@${digest}`],
		pinnedReferences: [],
		inUse: true
	}
];
export const imageSummary: ImageSummary = {
	id: imageId,
	artifactKey: imageRepositoryCopies[0]!.artifactKey,
	platform: 'linux/amd64',
	repo: 'registry.example.test/observability/prometheus',
	tag: '3.0',
	repoTags: [imageReference, 'registry.example.test/observability/prometheus:latest'],
	repoDigests: [`registry.example.test/observability/prometheus@${digest}`],
	created: 1795694400,
	size: 123000000,
	virtualSize: 123000000,
	labels: {},
	inUse: true,
	runningContainers: 1,
	usedBy: [
		{
			type: 'container',
			id: 'container-1',
			name: containerInspect.name,
			environmentId: 'manager-a',
			environmentName: 'QA node'
		}
	],
	repositoryGroup: { copies: imageRepositoryCopies, digests: [digest] }
};
export const imageDetail: ImageDetailSummary = {
	id: imageId,
	repoTags: imageSummary.repoTags,
	repoDigests: imageSummary.repoDigests,
	created,
	comment: '',
	author: '',
	config: {},
	architecture: 'amd64',
	os: 'linux',
	size: imageSummary.size,
	graphDriver: { name: 'overlay2', data: {} },
	rootFs: { type: 'layers', layers: [] },
	metadata: { lastTagTime: created },
	descriptor: {
		mediaType: 'application/vnd.oci.image.manifest.v1+json',
		digest,
		size: imageSummary.size
	}
};
