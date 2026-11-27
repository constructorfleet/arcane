import type { Page } from '@playwright/test';

import type { Environment } from '../../../frontend/src/lib/types/environment.js';
import type { AccessSurface, PermissionsManifest } from '../../../frontend/src/lib/types/role.js';
import type { SettingsSnapshot } from '../../../frontend/src/lib/types/settings.js';
import type { User } from '../../../frontend/src/lib/types/user.js';

export interface ResourceAPIRequest {
	method: string;
	path: string;
	scope: string;
	url: string;
	search: Record<string, string>;
}

export interface ResourcePageAPIOptions {
	responses: Record<string, unknown>;
	permissions?: string[];
	environments?: Environment[];
}

export const resourceEnvironment: Environment = {
	id: 'manager-a',
	name: 'QA node',
	apiUrl: '',
	status: 'online',
	enabled: true,
	connected: true,
	agentCapabilities: ['swarm']
};

const workerEnvironment: Environment = {
	...resourceEnvironment,
	id: 'node-agent',
	name: 'QA worker',
	enabled: false
};

const routeResources = [
	['dashboard', 'dashboard'],
	['containers', 'containers'],
	['images', 'images'],
	['networks', 'networks'],
	['volumes', 'volumes'],
	['projects', 'projects'],
	['projects.templates', 'projects/templates'],
	['swarm.nodes', 'swarm/nodes'],
	['swarm.tasks', 'swarm/tasks'],
	['swarm.services', 'swarm/services'],
	['swarm.stacks', 'swarm/stacks'],
	['swarm.secrets', 'swarm/secrets'],
	['swarm.configs', 'swarm/configs']
] as const;

const accessSurfaces: AccessSurface[] = routeResources.map(([id, path], index) => ({
	id: `route.${id}`,
	kind: 'route',
	url: `/${path}`,
	label: path,
	accessMode: 'permissions',
	matchMode: 'any-of',
	scopeMode: 'env-plus-global',
	permissions: [path.replaceAll('/', '-') + ':list', path.replaceAll('/', '-') + ':read'],
	fallbackOrder: index
}));

const manifest: PermissionsManifest = { resources: [], accessSurfaces };

const settings: SettingsSnapshot = {
	settings: Object.entries({
		featureSwarmEnabled: 'true',
		featureVulnerabilityManagementEnabled: 'true',
		projectsDirectory: '/srv/projects',
		projectsDirectories: '[{"path":"/srv/projects","enabled":true}]',
		templatesDirectory: '/srv/templates',
		pollingEnabled: 'false',
		frontendCacheMinutes: '10'
	}).map(([key, value]) => {
		let type = 'string';
		if (key === 'projectsDirectories') type = 'json';
		else if (key.endsWith('Enabled')) type = 'boolean';
		return {
			key,
			value,
			type,
			default: value,
			source: 'default',
			locked: false,
			sensitive: false,
			scope: 'environment',
			envName: ''
		};
	}),
	runtime: {
		frontendTracingEnabled: false,
		frontendMetricsEnabled: false,
		frontendLogsEnabled: false
	},
	readOnly: false
};

export function resourcePage<T>(items: T[]) {
	return {
		success: true,
		data: items,
		pagination: {
			currentPage: 1,
			totalPages: 1,
			totalItems: items.length,
			grandTotalItems: items.length,
			itemsPerPage: 20,
			hasNextPage: false
		},
		failedEnvironments: []
	};
}

/** Runs real frontend routes against explicit, isolated API fixtures; never sends writes or external requests. */
export async function installResourcePageAPI(page: Page, options: ResourcePageAPIOptions) {
	const requests: ResourceAPIRequest[] = [];
	const unknownGET: string[] = [];
	const mutations: ResourceAPIRequest[] = [];
	const permissions = options.permissions ?? ['*'];
	const user: User = {
		id: 'contract-operator',
		username: 'contract-operator',
		displayName: 'Contract operator',
		roleAssignments: [],
		permissionsByEnv: { global: permissions },
		isGlobalAdmin: permissions.includes('*'),
		canDelete: false,
		disabled: false,
		timeFormat: '24h',
		preferences: { animationsEnabled: false, glassEffectsEnabled: false },
		requiresPasswordChange: false
	};
	const version = {
		currentVersion: 'v3.0.1-cf.contract',
		displayVersion: 'v3.0.1-cf.contract',
		revision: 'contract-fixture',
		updateAvailable: false,
		enabledFeatures: []
	};
	const environmentItems = options.environments ?? [resourceEnvironment, workerEnvironment];
	const environments = resourcePage(environmentItems);
	const defaults: Record<string, unknown> = {
		'/auth/bootstrap': { success: true, data: { user, version, environments } },
		'/auth/user': { success: true, data: user },
		'/roles/permissions': { success: true, data: manifest },
		'/agents': environments,
		...Object.fromEntries(
			environmentItems.map((environment) => [
				`/agents/${environment.id}`,
				{ success: true, data: environment }
			])
		),
		'/settings': settings,
		'/settings/public': settings,
		'/settings/authentication': settings,
		'/app/version': version,
		'/version': version,
		'/swarm/status': {
			success: true,
			data: { enabled: true, clusterId: 'cluster-contract', controlAvailable: true }
		},
		'/podman/status': {
			success: true,
			data: {
				detected: false,
				podCount: 0,
				rootless: false,
				quadletsSupported: false,
				autoUpdateMode: 'unavailable',
				systemdAvailable: false
			}
		},
		'/variables': { success: true, data: [] },
		'/gitops/syncs': resourcePage([]),
		'/activities': resourcePage([]),
		'/events': resourcePage([]),
		'/notifications': resourcePage([])
	};
	await page.routeWebSocket('**/*', (socket) => socket.close());
	await page.route('**/*', async (route) => {
		const request = route.request();
		const url = new URL(request.url());
		if (!['127.0.0.1', 'localhost'].includes(url.hostname)) {
			await route.abort('blockedbyclient');
			return;
		}
		if (!url.pathname.startsWith('/api/')) {
			await route.continue();
			return;
		}
		const originalPath = url.pathname.slice(4);
		const match = originalPath.match(/^\/agents\/([^/]+)(\/.*)$/);
		const path = match?.[2] ?? originalPath;
		const record: ResourceAPIRequest = {
			method: request.method(),
			path,
			scope: match ? decodeURIComponent(match[1]) : '',
			url: request.url(),
			search: Object.fromEntries(url.searchParams)
		};
		requests.push(record);
		if (record.method !== 'GET' && record.method !== 'HEAD') {
			mutations.push(record);
			await route.abort('blockedbyclient');
			return;
		}
		if (path === '/stream') {
			await route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: ': isolated fixture\n\n'
			});
			return;
		}
		if (path.startsWith('/app/images/')) {
			await route.fulfill({
				status: 200,
				contentType: 'image/svg+xml',
				body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><rect width="20" height="20" fill="#eab308"/></svg>'
			});
			return;
		}
		const body =
			options.responses[originalPath] ??
			options.responses[path] ??
			defaults[originalPath] ??
			defaults[path];
		if (body === undefined) {
			unknownGET.push(path);
			await route.fulfill({
				status: 404,
				json: { title: 'Unmocked conformance request', detail: path, status: 404 }
			});
			return;
		}
		await route.fulfill({ status: 200, json: body });
	});
	return { requests, unknownGET, mutations };
}
