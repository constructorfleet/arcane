import AxeBuilder from '@axe-core/playwright';
import type { Environment } from '../../frontend/src/lib/types/environment.js';
import { expect, test, type Page } from '@playwright/test';

import { installResourcePageAPI, resourcePage } from './fixtures/api.js';
import {
	containerInspect,
	containerSummary,
	imageDetail,
	imageId,
	imageSummary,
	networkInspect,
	networkSummary,
	volume
} from './fixtures/docker.js';

test.use({ baseURL: 'http://127.0.0.1:4176' });

function responses(): Record<string, unknown> {
	const data = (data: unknown) => ({ success: true, data });
	return {
		'/swarm/status': data({ enabled: true, clusterId: 'cluster-1', controlAvailable: true }),
		'/swarm/nodes/node-1': data({
			id: 'node-1',
			hostname: 'QA node',
			role: 'manager',
			state: 'ready',
			availability: 'active'
		}),
		'/swarm/services': resourcePage([
			{
				id: 'service-1',
				name: 'observability_prometheus',
				stackName: 'observability',
				image: containerSummary.image,
				mode: 'replicated',
				replicas: 1,
				runningReplicas: 1,
				statusObserved: true,
				ports: [],
				nodes: ['node-1'],
				networks: ['observability_default'],
				mounts: [],
				labels: { 'com.docker.stack.namespace': 'observability' },
				createdAt: containerInspect.created,
				updatedAt: containerInspect.created,
				spec: {
					Name: 'observability_prometheus',
					Labels: { 'com.docker.stack.namespace': 'observability' }
				}
			}
		]),
		'/swarm/services/service-1': data({
			id: 'service-1',
			name: 'observability_prometheus',
			spec: {
				Name: 'observability_prometheus',
				Labels: { 'com.docker.stack.namespace': 'observability' }
			}
		}),
		'/swarm/services/service-1/tasks': resourcePage([
			{
				id: 'task-1',
				serviceId: 'service-1',
				nodeId: 'node-1',
				containerId: 'container-1',
				currentState: 'running',
				desiredState: 'running'
			}
		]),
		'/swarm/tasks': resourcePage([
			{
				id: 'task-1',
				serviceId: 'service-1',
				nodeId: 'node-1',
				containerId: 'container-1',
				currentState: 'running',
				desiredState: 'running'
			}
		]),
		'/containers': {
			...resourcePage([containerSummary]),
			counts: {
				runningContainers: 1,
				stoppedContainers: 0,
				totalContainers: 1,
				unhealthyContainers: 0,
				updatesAvailable: 0
			}
		},
		'/containers/counts': data({
			runningContainers: 1,
			stoppedContainers: 0,
			totalContainers: 1,
			unhealthyContainers: 0,
			updatesAvailable: 0
		}),
		'/containers/container-1': data(containerInspect),
		'/networks': { ...resourcePage([networkSummary]), counts: { total: 1, inuse: 1, unused: 0 } },
		'/networks/counts': data({ total: 1, inuse: 1, unused: 0 }),
		'/networks/network-1': data(networkInspect),
		'/volumes': { ...resourcePage([volume]), counts: { total: 1, inuse: 1, unused: 0 } },
		'/volumes/counts': data({ total: 1, inuse: 1, unused: 0 }),
		'/volumes/observability_data': data(volume),
		'/volumes/observability_data/usage': data({ usage: 2048, total: 4096, available: 2048 }),
		'/volume-drivers': data(['local']),
		'/volumes/observability_data/workspace/storage': data(null),
		'/images': resourcePage([imageSummary]),
		'/images/counts': data({
			imagesInuse: 1,
			imagesUnused: 0,
			totalImages: 1,
			totalImageSize: imageSummary.size
		}),
		[`/images/${imageId}`]: data(imageDetail),
		[`/images/${imageId}/vulnerabilities`]: data(null),
		[`/images/${imageId}/attestations`]: data({
			imageRef: imageId,
			subjectDigest: imageId,
			attestations: []
		}),
		[`/images/${encodeURIComponent(imageId)}/history`]: data([]),
		'/vulnerabilities/scanner': data({ available: false, enabled: false }),
		'/vulnerabilities/summary': data({}),
		'/images/updates/summaries': data({}),
		'/swarm/networks/network-1/services': resourcePage([]),
		'/swarm/networks/network-1/tasks': resourcePage([])
	};
}

async function related(page: Page) {
	const trigger = page.getByRole('menuitem', { name: 'Related resources', exact: true });
	await trigger.hover();
	await trigger.focus();
	await page.keyboard.press('ArrowRight');
}

const lists = [
	{
		path: '/containers',
		name: containerSummary.names[0]!,
		columns: ['Name', 'State', 'CPU Usage', 'Memory Usage'],
		absent: ['Labels', 'Created', 'Owner']
	},
	{
		path: '/networks',
		name: networkSummary.name,
		columns: ['Name', 'Source', 'Driver / scope', 'Addressing'],
		absent: ['Labels', 'Nodes', 'Created', 'Query connection']
	},
	{
		path: '/volumes',
		name: volume.name,
		columns: ['Name', 'Status', 'Driver', 'Filesystem disk usage'],
		absent: ['Labels', 'Created', 'Size', 'Mountpoint', 'Docker scope', 'Owner']
	},
	{
		path: '/images',
		name: 'registry.example.test/observability/prometheus',
		columns: ['Image', 'Platform', 'Usage', 'Updates', 'Vulnerabilities'],
		absent: ['Labels', 'Created', 'Used by', 'Digest']
	}
];

test.afterEach(async ({ page }, testInfo) => {
	if (testInfo.status !== testInfo.expectedStatus) return;
	const width = await page.evaluate(() => ({
		scroll: document.documentElement.scrollWidth,
		client: document.documentElement.clientWidth
	}));
	expect(width.scroll).toBeLessThanOrEqual(width.client + 1);
	const path = testInfo.outputPath('actual-resource.png');
	await page.screenshot({ path, fullPage: true, animations: 'disabled' });
	await testInfo.attach('Actual resource route', { path, contentType: 'image/png' });
});

for (const resource of lists) {
	test(`actual ${resource.path} list applies curated default columns without automation labels`, async ({
		page,
		isMobile
	}) => {
		const api = await installResourcePageAPI(page, { responses: responses() });
		await page.goto(resource.path);
		await expect(
			page.getByRole('link', { name: resource.name, exact: true }).first()
		).toBeVisible();
		await expect(page.getByText('DO-NOT-RENDER-IN-TABLES', { exact: true })).toHaveCount(0);
		for (const button of await page.locator('main header').first().getByRole('button').all()) {
			if (!(await button.isVisible())) continue;
			const bounds = await button.boundingBox();
			expect(bounds).not.toBeNull();
			expect(bounds!.x).toBeGreaterThanOrEqual(0);
			expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(page.viewportSize()!.width);
			expect(
				await button.evaluate((element) => {
					const rect = element.getBoundingClientRect();
					const hit = document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2);
					return hit !== null && element.contains(hit);
				})
			).toBe(true);
		}

		if (!isMobile) {
			for (const column of resource.columns)
				await expect(
					page.getByRole('columnheader', { name: new RegExp(`^(Sort by )?${column}$`) })
				).toBeVisible();
			for (const column of resource.absent)
				await expect(
					page.getByRole('columnheader', { name: new RegExp(`^(Sort by )?${column}$`) })
				).toHaveCount(0);
		}
		if (resource.path === '/networks') {
			await expect(page.getByText('10.0.9.0/24', { exact: true })).toHaveCount(0);
			if (isMobile)
				await expect(
					page.getByRole('definition').filter({ hasText: /10\.0\.8\.1 \+1$/ })
				).toBeVisible();
			else await expect(page.getByText('+1', { exact: true }).first()).toBeVisible();
		}
		if (resource.path === '/images') {
			await expect(page.getByText('sha256:222222222222...', { exact: true }).first()).toBeVisible();
			await expect(
				page.getByText(imageSummary.repositoryGroup!.digests![0]!, { exact: true })
			).toHaveCount(0);
			expect(
				api.requests.some(
					(request) => request.path === '/images' && request.search.inUse === 'true'
				)
			).toBe(true);
			expect(
				api.requests.some(
					(request) => request.path === '/images' && request.search.groupByImage === 'true'
				)
			).toBe(true);
		}
		const contrast = await new AxeBuilder({ page })
			.include('main a')
			.withRules(['color-contrast'])
			.analyze();
		expect(contrast.violations).toEqual([]);
		if (!isMobile && resource.path === '/images') {
			const repository = page.getByRole('link', { name: resource.name, exact: true }).first();
			const content = await repository.evaluate((element) => ({
				scroll: element.scrollWidth,
				client: element.clientWidth
			}));
			expect(content.scroll).toBeLessThanOrEqual(content.client + 1);

			const bounds = await repository.evaluate((element) => {
				const own = element.getBoundingClientRect();
				const cell = element.closest('td')!.getBoundingClientRect();
				return { left: own.left, right: own.right, cellLeft: cell.left, cellRight: cell.right };
			});
			expect(bounds.left).toBeGreaterThanOrEqual(bounds.cellLeft);
			expect(bounds.right).toBeLessThanOrEqual(bounds.cellRight);

			for (const label of ['Updates', 'Vulnerabilities']) {
				const header = page.getByRole('columnheader', {
					name: new RegExp(`^(Sort by )?${label}$`)
				});
				const bounds = await header.boundingBox();
				expect(bounds).not.toBeNull();
				expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(page.viewportSize()!.width);
			}
		}

		expect(api.mutations).toEqual([]);
		expect(api.unknownGET).toEqual([]);
	});
}

for (const resource of [
	{ path: '/containers/container-1', section: 'Stack resources' },
	{ path: '/networks/network-1', section: 'Linked resources' },
	{ path: '/volumes/observability_data', section: 'Linked resources' },
	{ path: `/images/${imageId}`, section: 'Linked resources' }
]) {
	test(`actual ${resource.path.split('/')[1]} detail opens Overview and exposes curated resource links`, async ({
		page
	}) => {
		const api = await installResourcePageAPI(page, { responses: responses() });
		await page.goto(`${resource.path}?env=manager-a`);
		await expect(page.getByRole('tab', { name: 'Overview', exact: true })).toHaveAttribute(
			'aria-selected',
			'true'
		);
		await expect(page.getByText(resource.section, { exact: true })).toBeVisible();
		if (!resource.path.startsWith('/images')) {
			const stack = page.getByRole('link', { name: 'observability', exact: true }).first();
			await expect(stack).toBeVisible();
			await expect(stack).toHaveAttribute('href', /swarm\/stacks\/observability.*env=manager-a/);
		}
		if (resource.path.startsWith('/volumes'))
			await expect(page.getByText('Global', { exact: true })).toHaveCount(0);
		if (resource.path.startsWith('/images') || resource.path.startsWith('/volumes')) {
			await page.getByRole('tab', { name: 'Containers', exact: true }).click();
			await expect(
				page.getByRole('link', { name: containerSummary.names[0]!, exact: true }).first()
			).toBeVisible();
			const request = api.requests
				.slice()
				.reverse()
				.find((request) => request.path === '/containers');
			expect(request?.scope).toBe('manager-a');
			if (resource.path.startsWith('/images')) {
				expect(request?.search.imageId).toBe(imageId);
				expect(request?.search.status).toBe('running');
				await expect(page.getByRole('button', { name: 'State', exact: true })).toHaveCount(0);
				await expect(page.getByRole('textbox', { name: 'Search…', exact: true })).toBeVisible();
			} else expect(request?.search.volumeName).toBe(volume.name);
		}
		expect(api.mutations).toEqual([]);
		expect(api.unknownGET).toEqual([]);
	});
}

test('container Related resources lazily resolves exact service ownership and omits removed actions', async ({
	page
}) => {
	const api = await installResourcePageAPI(page, { responses: responses() });
	await page.goto('/containers');
	await expect(
		page.getByRole('link', { name: containerSummary.names[0]!, exact: true }).first()
	).toBeVisible();
	expect(api.requests.some((request) => request.path === '/swarm/services/service-1')).toBe(false);
	await page
		.getByRole('button', { name: `Actions for ${containerSummary.names[0]}`, exact: true })
		.click();
	await expect(page.getByRole('menuitem', { name: /^View details$/i })).toHaveAttribute(
		'href',
		/tab=overview/
	);
	for (const label of ['Commit', 'Release notes', 'Pin to dashboard'])
		await expect(page.getByRole('menuitem', { name: label, exact: true })).toHaveCount(0);
	await related(page);
	await expect(page.getByRole('menuitem', { name: 'Service', exact: true })).toHaveAttribute(
		'href',
		/service-1.*env=manager-a/
	);
	await expect(page.getByRole('menuitem', { name: 'Stack', exact: true })).toHaveAttribute(
		'href',
		/stacks\/observability.*env=manager-a/
	);
	expect(api.mutations).toEqual([]);
	expect(api.unknownGET).toEqual([]);
});

for (const resource of [
	{ path: `/images/${imageId}`, filter: 'imageId', identity: imageId },
	{ path: '/volumes/observability_data', filter: 'volumeName', identity: volume.name }
]) {
	test(`${resource.filter} Services tab resolves consumers through exact running task identities`, async ({
		page,
		isMobile
	}) => {
		const api = await installResourcePageAPI(page, { responses: responses() });
		await page.goto(`${resource.path}?env=manager-a&tab=services`);
		await expect(page.getByRole('tab', { name: 'Services', exact: true })).toHaveAttribute(
			'aria-selected',
			'true'
		);
		if (isMobile)
			await expect(page.getByRole('heading', { name: 'prometheus', exact: true })).toBeVisible();
		else
			await expect(
				page.getByRole('link', { name: 'prometheus', exact: true }).first()
			).toBeVisible();
		expect(
			api.requests.some(
				(request) =>
					request.path === '/containers' &&
					request.search[resource.filter] === resource.identity &&
					request.search.status === 'running'
			)
		).toBe(true);
		expect(api.requests.filter((request) => request.path === '/swarm/tasks')).toHaveLength(1);
		expect(
			api.requests.some(
				(request) => request.path === '/swarm/services' && request.search.serviceIds === 'service-1'
			)
		).toBe(true);
		expect(api.mutations).toEqual([]);
		expect(api.unknownGET).toEqual([]);
	});
}

test('container Overview shows verified owner metadata without unauthorized Stack navigation', async ({
	page
}) => {
	const api = await installResourcePageAPI(page, {
		responses: responses(),
		permissions: ['containers:read', 'containers:list', 'swarm:read', 'swarm-services:read']
	});
	await page.goto('/containers/container-1?env=manager-a');
	await expect(page.getByText('Stack resources', { exact: true })).toBeVisible();
	await expect(page.getByText('observability', { exact: true }).first()).toBeVisible();
	await expect(page.getByRole('link', { name: 'observability', exact: true })).toHaveCount(0);
	const service = page.getByRole('link', { name: 'observability_prometheus', exact: true });
	await expect(service).toHaveAttribute('href', /cluster=cluster-1/);
	expect(api.mutations).toEqual([]);
	expect(api.unknownGET).toEqual([]);
});

test('image Services merges exact consumers from two managers in one cluster without duplicate rows', async ({
	page,
	isMobile
}) => {
	const mocked = responses();
	const managerA: Environment = {
		id: 'manager-a',
		name: 'QA node',
		apiUrl: '',
		status: 'online',
		enabled: true,
		connected: true,
		agentCapabilities: ['swarm']
	};
	const managerB: Environment = { ...managerA, id: 'manager-b', name: 'QA second manager' };
	const clusterStatus = { enabled: true, clusterId: 'cluster-1', controlAvailable: true };
	mocked['/swarm/status'] = {
		success: true,
		data: clusterStatus,
		environments: [
			{ environment: managerA, data: clusterStatus },
			{ environment: managerB, data: clusterStatus }
		]
	};

	const secondContainer = {
		...containerSummary,
		id: 'container-2',
		names: ['/observability_prometheus.2.task-2'],
		labels: {
			...containerSummary.labels,
			'com.docker.swarm.task.id': 'task-2',
			'com.docker.swarm.node.id': 'node-2'
		}
	};
	const copy = imageSummary.repositoryGroup!.copies[0]!;
	mocked['/images'] = resourcePage([
		{
			...imageSummary,
			runningContainers: 2,
			repositoryGroup: {
				...imageSummary.repositoryGroup,
				copies: [copy, { ...copy, environmentId: 'manager-b', environmentName: managerB.name }]
			}
		}
	]);
	mocked['/agents/manager-b/containers'] = resourcePage([secondContainer]);
	mocked['/swarm/tasks'] = resourcePage([
		{
			id: 'task-1',
			serviceId: 'service-1',
			nodeId: 'node-1',
			containerId: 'container-1',
			currentState: 'running',
			desiredState: 'running'
		},
		{
			id: 'task-2',
			serviceId: 'service-1',
			nodeId: 'node-2',
			containerId: 'container-2',
			currentState: 'running',
			desiredState: 'running'
		}
	]);
	const api = await installResourcePageAPI(page, {
		responses: mocked,
		environments: [managerA, managerB]
	});
	await page.goto(`/images/${imageId}?env=manager-a&tab=services`);
	if (isMobile)
		await expect(page.getByRole('heading', { name: 'prometheus', exact: true })).toHaveCount(1);
	else await expect(page.getByRole('link', { name: 'prometheus', exact: true })).toHaveCount(1);
	await expect
		.poll(() => api.requests.filter((request) => request.path === '/swarm/services').length)
		.toBe(1);
	expect(api.requests.filter((request) => request.path === '/swarm/tasks')).toHaveLength(1);
	expect(
		api.requests
			.filter((request) => request.path === '/containers')
			.map((request) => request.scope)
			.sort()
	).toEqual(['manager-a', 'manager-b']);
	expect(api.unknownGET).toEqual([]);
	expect(api.mutations).toEqual([]);
});

test('scoped image Services rejects agents returning unrelated service identities', async ({
	page
}) => {
	const mocked = responses();
	mocked['/swarm/services'] = resourcePage([
		{
			id: 'unrelated-service',
			name: 'unrelated',
			image: containerSummary.image,
			mode: 'replicated',
			replicas: 1,
			runningReplicas: 1,
			ports: [],
			nodes: [],
			networks: [],
			mounts: [],
			labels: {},
			createdAt: containerInspect.created,
			updatedAt: containerInspect.created
		}
	]);
	const api = await installResourcePageAPI(page, { responses: mocked });
	await page.goto(`/images/${imageId}?env=manager-a&tab=services`);
	await expect(page.getByRole('alert').filter({ hasText: 'Unavailable' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'unrelated', exact: true })).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'unrelated', exact: true })).toHaveCount(0);
	expect(
		api.requests.some(
			(request) => request.path === '/swarm/services' && request.search.serviceIds === 'service-1'
		)
	).toBe(true);
	expect(api.unknownGET).toEqual([]);
	expect(api.mutations).toEqual([]);
});
