import type { Project } from '../../../frontend/src/lib/types/project.js';
import type { SwarmNode, SwarmTaskSummary } from '../../../frontend/src/lib/types/swarm.js';
import type {
	Template,
	TemplateContent,
	TemplateRegistry
} from '../../../frontend/src/lib/types/template.js';

export const contractTimestamp = '2026-11-27T12:00:00Z';
export const automationLabels = Object.fromEntries(
	Array.from({ length: 40 }, (_, index) => [
		`automation.managed.rule.${index}`,
		`automation-label-value-${index}`
	])
);

export const contractNode: SwarmNode = {
	id: 'node-contract',
	hostname: 'worker-contract',
	role: 'manager',
	managerStatus: 'leader',
	availability: 'active',
	status: 'ready',
	runningTasks: 2,
	address: '192.0.2.10',
	engineVersion: '29.0.0',
	platform: 'linux/amd64',
	reachability: 'reachable',
	labels: automationLabels,
	agent: {
		state: 'connected',
		connected: true,
		environmentId: 'node-agent',
		reportedNodeId: 'node-contract',
		bindingKind: 'discovered'
	},
	createdAt: contractTimestamp,
	updatedAt: contractTimestamp
};

const runningTask: SwarmTaskSummary = {
	id: 'task-running',
	name: 'contract_web.1',
	serviceId: 'service-contract',
	serviceName: 'contract_web',
	stackName: 'contract-stack',
	nodeId: contractNode.id,
	nodeName: contractNode.hostname,
	currentState: 'running',
	desiredState: 'running',
	containerId: 'container-contract',
	image: 'registry.example.test/contracts/web:1.0@sha256:0123456789abcdef',
	slot: 1,
	createdAt: contractTimestamp,
	updatedAt: contractTimestamp
};
export const contractTasks: SwarmTaskSummary[] = [
	runningTask,
	{
		...runningTask,
		id: 'task-shutdown',
		name: 'contract_web.1.old',
		currentState: 'shutdown',
		desiredState: 'shutdown'
	},
	{
		...runningTask,
		id: 'task-failed',
		name: 'contract_web.2.failed',
		currentState: 'failed',
		desiredState: 'shutdown',
		error: 'Container exited with status 1'
	}
];

export const contractProject: Project & { labels: Record<string, string> } = {
	id: 'project-contract',
	name: 'contract-project',
	path: '/srv/projects/contract-project',
	projectsRoot: '/srv/projects',
	dirName: 'contract-project',
	relativePath: 'contract-project',
	status: 'running',
	runningCount: 1,
	serviceCount: 1,
	isArchived: false,
	tags: Array.from({ length: 12 }, (_, index) => ({
		name: `category-${index}`,
		color: 'blue',
		sources: ['ui']
	})),
	labels: automationLabels,
	urls: ['https://app.example.test'],
	composeContent: 'name: contract-project\nservices:\n  web:\n    image: nginx:alpine\n',
	envContent: '',
	envFileExists: false,
	composeFileName: 'compose.yaml',
	composeFiles: ['compose.yaml'],
	includeFiles: [],
	services: [],
	runtimeServices: [],
	createdAt: contractTimestamp,
	updatedAt: contractTimestamp
};

export const contractTemplateRegistry: TemplateRegistry = {
	id: 'registry-contract',
	name: 'Approved catalog',
	url: 'https://catalog.example.test/templates.json',
	enabled: true,
	description: 'Contract fixture registry'
};

export const contractTemplates: Template[] = [
	{
		id: 'template-local',
		name: 'Local contract template',
		description: 'Editable Compose configuration',
		content: contractProject.composeContent ?? '',
		envContent: 'APP_MODE=test\n',
		isCustom: true,
		isRemote: false,
		metadata: {
			tags: Array.from({ length: 12 }, (_, index) => `category-${index}`),
			version: '1.0',
			author: 'Operator'
		}
	},
	{
		id: 'template-remote',
		name: 'Remote contract template',
		description: 'Read-only catalog Compose configuration',
		content: contractProject.composeContent ?? '',
		envContent: 'APP_MODE=test\n',
		isCustom: false,
		isRemote: true,
		registryId: contractTemplateRegistry.id,
		registry: contractTemplateRegistry,
		metadata: { tags: ['web', 'production'], version: '2.0', author: 'Catalog maintainer' }
	}
];

export const contractTemplateContents: TemplateContent[] = contractTemplates.map((template) => ({
	template,
	content: template.content,
	envContent: template.envContent ?? '',
	services: ['web'],
	envVariables: [],
	isFolderTemplate: false
}));
