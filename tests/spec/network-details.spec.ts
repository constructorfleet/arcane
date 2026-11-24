import { expect, test } from '../fixtures/test.fixture';

test('shows the network overview and scopes the shared Services and Tasks tables', async ({
	page
}) => {
	const networkId = 'network-details-fixture';
	const agent = '/api/agents/0';
	const network = {
		id: networkId,
		name: 'fixture-overlay',
		clusterId: 'test-cluster',
		scope: 'swarm',
		driver: 'overlay',
		attachable: true,
		labels: { owner: 'qa-fixture' },
		options: { encrypted: '' },
		ipam: { driver: 'default', config: [{ subnet: '10.69.0.0/24' }] },
		containersList: [{ id: 'old-container', name: 'standalone-container-fixture' }],
		peers: [{ Name: 'old-peer-fixture', IP: '10.69.0.2' }]
	};
	const responsePage = <T>(data: T[]) => ({
		data,
		pagination: { totalPages: 1, currentPage: 1, totalItems: data.length, itemsPerPage: 20 }
	});

	await page.route(`**${agent}/networks/${networkId}`, (route) =>
		route.fulfill({ json: { success: true, data: network } })
	);
	await page.route(`**${agent}/swarm/status`, (route) =>
		route.fulfill({
			json: { success: true, data: { enabled: true, clusterId: network.clusterId } }
		})
	);
	await page.route(`**${agent}/swarm/nodes?*`, (route) =>
		route.fulfill({
			json: responsePage([
				{ id: 'n1', hostname: 'swarm01' },
				{ id: 'n2', hostname: 'swarm02' }
			])
		})
	);
	await page.route(`**${agent}/swarm/services?*`, (route) =>
		route.fulfill({
			json: responsePage([
				{
					id: 'fixture-service',
					name: 'edge-service',
					image: 'fixture/edge:v1',
					mode: 'global',
					replicas: 2,
					runningReplicas: 2,
					nodes: ['swarm01', 'swarm02'],
					ports: [],
					labels: {},
					networks: [network.name],
					mounts: []
				}
			])
		})
	);
	await page.route(`**${agent}/swarm/tasks?*`, (route) => {
		const query = new URL(route.request().url()).searchParams;
		const nodes = [
			{ id: 'n1', hostname: 'swarm01' },
			{ id: 'n2', hostname: 'swarm02' }
		];
		const selected = query.get('nodeName');
		const tasks = nodes
			.filter((node) => !selected || selected === node.hostname)
			.map((node) => ({
				id: `task-${node.id}`,
				name: `edge-service.${node.hostname}`,
				serviceId: 'fixture-service',
				serviceName: 'edge-service',
				nodeId: node.id,
				nodeName: node.hostname,
				currentState: 'running',
				desiredState: 'running'
			}));
		return route.fulfill({ json: responsePage(tasks) });
	});

	await page.setViewportSize({ width: 1440, height: 900 });
	await page.goto(`/networks/${networkId}?env=0`);
	const overview = page.getByRole('tabpanel', { name: 'Overview', exact: true });
	await expect(overview.getByText('10.69.0.0/24', { exact: true })).toBeVisible();
	await expect(overview.getByText('qa-fixture', { exact: true })).toBeVisible();
	await expect(overview.getByText('encrypted', { exact: true })).toBeVisible();
	await expect(overview.getByText('standalone-container-fixture')).toHaveCount(0);
	await expect(overview.getByText('old-peer-fixture')).toHaveCount(0);
	await expect(overview.locator('[data-slot=card]').first()).toHaveCSS('border-top-width', '1px');

	const serviceResponse = page.waitForRequest((request) => {
		const url = new URL(request.url());
		return (
			url.pathname === agent + '/swarm/services' && url.searchParams.get('networkId') === networkId
		);
	});
	await page.getByRole('tab', { name: 'Services', exact: true }).click();
	await serviceResponse;
	const services = page.getByRole('tabpanel', { name: 'Services', exact: true });
	const serviceLink = services.getByRole('link', { name: 'edge-service', exact: true });
	await expect(serviceLink).toBeVisible();
	await expect(services.getByRole('columnheader', { name: 'VIP', exact: true })).toHaveCount(0);
	await expect(services).toHaveCSS('border-top-width', '0px');
	const href = new URL((await serviceLink.getAttribute('href'))!, 'http://fixture');
	expect(href.searchParams.get('prev')).toContain('tab=services');

	await page.getByRole('tab', { name: 'Tasks', exact: true }).click();
	const tasks = page.getByRole('tabpanel', { name: 'Tasks', exact: true });
	await expect(tasks.getByText('edge-service.swarm01', { exact: true })).toBeVisible();
	await expect(tasks.getByText('edge-service.swarm02', { exact: true })).toBeVisible();
	await tasks.getByRole('button', { name: 'Node', exact: true }).click();
	const filteredRequest = page.waitForRequest((request) => {
		const url = new URL(request.url());
		return (
			url.pathname === agent + '/swarm/tasks' &&
			url.searchParams.get('networkId') === networkId &&
			url.searchParams.get('nodeName') === 'swarm02'
		);
	});
	await page.getByRole('option', { name: 'swarm02', exact: true }).click();
	await filteredRequest;
	await expect(tasks.getByText('edge-service.swarm01', { exact: true })).toHaveCount(0);
	await expect(tasks.getByText('edge-service.swarm02', { exact: true })).toBeVisible();
});
