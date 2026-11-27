import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';

import { installResourcePageAPI, resourcePage } from './fixtures/api.js';
import { networkSummary, volume } from './fixtures/docker.js';
import { contractNode } from './fixtures/node-projects.js';
import {
	config,
	runtimeTask,
	secret,
	serviceInspect,
	serviceSummary,
	stackInspect,
	stackSource,
	stackSummary
} from './fixtures/swarm.js';

test.use({ baseURL: 'http://127.0.0.1:4176' });

function responses(): Record<string, unknown> {
	return {
		'/swarm/status': { success: true, data: { enabled: true, clusterId: 'cluster-1' } },
		'/swarm/stacks': resourcePage([stackSummary]),
		'/swarm/stacks/observability': { success: true, data: stackInspect },
		'/swarm/stacks/observability/source': { success: true, data: stackSource },
		'/swarm/stacks/observability/services': resourcePage([serviceSummary]),
		'/swarm/stacks/observability/tasks': resourcePage([runtimeTask]),
		'/swarm/services': resourcePage([serviceSummary]),
		'/swarm/services/service-1': { success: true, data: serviceInspect },
		'/swarm/services/service-1/tasks': resourcePage([runtimeTask]),
		'/swarm/secrets': resourcePage([secret]),
		'/swarm/secrets/secret-1': { success: true, data: secret },
		'/swarm/configs': resourcePage([config]),
		'/swarm/configs/config-1': { success: true, data: config },
		'/networks': resourcePage([networkSummary]),
		'/volumes': resourcePage([volume]),
		'/swarm/nodes': resourcePage([]),
		'/swarm/nodes/node-1': {
			success: true,
			data: { ...contractNode, id: 'node-1', hostname: 'QA node' }
		}
	};
}

async function assertReadableLinks(page: import('@playwright/test').Page) {
	const results = await new AxeBuilder({ page })
		.include('main a')
		.withRules(['color-contrast'])
		.analyze();
	expect(results.violations).toEqual([]);
	for (const heading of await page.getByRole('columnheader').all()) {
		const bounds = await heading.boundingBox();
		if (bounds) {
			expect(bounds.x).toBeGreaterThanOrEqual(0);
			expect(bounds.x + bounds.width).toBeLessThanOrEqual(page.viewportSize()!.width + 1);
		}
	}
}

test('actual Stack route defaults to Overview and offers every curated scoped resource tab', async ({
	page
}) => {
	const api = await installResourcePageAPI(page, { responses: responses() });
	await page.goto('/swarm/stacks/observability?env=manager-a');
	await expect(page.getByRole('tab', { name: 'Overview', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	for (const name of ['Services', 'Tasks', 'Networks', 'Volumes', 'Secrets', 'Configs']) {
		await expect(page.getByRole('tab', { name, exact: true })).toBeVisible();
	}
	await test.info().attach('Stack Overview', {
		body: await page.screenshot({ animations: 'disabled' }),
		contentType: 'image/png'
	});
	await assertReadableLinks(page);
	const header = page.getByRole('region', { name: 'Resource information', exact: true });
	await expect(header).toBeVisible();
	const viewport = page.viewportSize()!;
	const bounds = await header.boundingBox();
	expect(bounds).not.toBeNull();
	expect(bounds!.x).toBeGreaterThanOrEqual(0);
	expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(viewport.width + 1);
	expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(
		viewport.width
	);
	await expect(header.getByText('Namespace', { exact: true })).toHaveCount(0);
	await page.getByRole('tab', { name: 'Services', exact: true }).click();
	await expect(
		page.getByText('prometheus', { exact: true }).filter({ visible: true }).first()
	).toBeVisible();
	await page.getByRole('tab', { name: 'Tasks', exact: true }).click();
	await expect(
		page.getByText(runtimeTask.name, { exact: true }).filter({ visible: true }).first()
	).toBeVisible();
	for (const [tab, path, name] of [
		['Networks', '/networks', networkSummary.name],
		['Volumes', '/volumes', volume.name],
		['Secrets', '/swarm/secrets', secret.spec.Name],
		['Configs', '/swarm/configs', config.spec.Name]
	]) {
		await page.getByRole('tab', { name: tab, exact: true }).click();
		await expect(page.getByRole('link', { name, exact: true }).first()).toBeVisible();
		await assertReadableLinks(page);
		expect(api.requests.some((request) => request.path === path)).toBe(true);
	}
	for (const [path, id] of [
		['/swarm/secrets', 'secret-1'],
		['/swarm/configs', 'config-1']
	]) {
		expect(
			api.requests.some(
				(request) =>
					request.path === path &&
					request.search['ids'] === id &&
					request.search['stackName'] === 'observability'
			)
		).toBe(true);
	}
	expect(api.unknownGET).toEqual([]);
	expect(api.mutations).toEqual([]);
});

test('actual Service Overview links its owning Stack and scopes Secrets and Configs by exact reference IDs', async ({
	page
}) => {
	const api = await installResourcePageAPI(page, { responses: responses() });
	await page.goto('/swarm/services/service-1?env=manager-a');
	await expect(page.getByRole('tab', { name: 'Overview', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	await expect(
		page.getByRole('link', { name: 'observability', exact: true }).first()
	).toHaveAttribute('href', /swarm\/stacks\/observability/);
	for (const [tab, path, id, name] of [
		['Secrets', '/swarm/secrets', secret.id, secret.spec.Name],
		['Configs', '/swarm/configs', config.id, config.spec.Name]
	]) {
		await page.getByRole('tab', { name: tab, exact: true }).click();
		await expect(page.getByRole('link', { name, exact: true }).first()).toBeVisible();
		expect(
			api.requests.some((request) => request.path === path && request.search['ids'] === id)
		).toBe(true);
	}
	expect(api.unknownGET).toEqual([]);
	expect(api.mutations).toEqual([]);
});

for (const [kind, item, filter] of [
	['secrets', secret, 'secretId'],
	['configs', config, 'configId']
] as const) {
	test(`actual ${kind} lists omit labels and detail Overview exposes metadata with exact used-by Services`, async ({
		page,
		isMobile
	}) => {
		const api = await installResourcePageAPI(page, { responses: responses() });
		await page.goto(`/swarm/${kind}`);
		await expect(
			page.getByRole('link', { name: item.spec.Name, exact: true }).first()
		).toBeVisible();
		if (!isMobile)
			await expect(page.getByRole('columnheader', { name: 'Labels', exact: true })).toHaveCount(0);
		await expect(page.getByText('automation-label-value-0', { exact: true })).toHaveCount(0);
		await page.getByRole('link', { name: item.spec.Name, exact: true }).first().click();
		await expect(page.getByRole('tab', { name: 'Overview', exact: true })).toHaveAttribute(
			'aria-selected',
			'true'
		);
		await expect(
			page.getByText(item.id, { exact: true }).filter({ visible: true }).first()
		).toBeVisible();
		await page.getByRole('tab', { name: 'Services', exact: true }).click();
		await expect(
			page.getByText('prometheus', { exact: true }).filter({ visible: true }).first()
		).toBeVisible();
		expect(
			api.requests.some(
				(request) => request.path === '/swarm/services' && request.search[filter] === item.id
			)
		).toBe(true);
		expect(api.unknownGET).toEqual([]);
		expect(api.mutations).toEqual([]);
	});
}

test('actual Stack Overview preserves unavailable task counts instead of inventing zero', async ({
	page
}) => {
	const mocked = responses();
	mocked['/swarm/stacks/observability'] = {
		success: true,
		data: { ...stackInspect, runningTasks: null, desiredTasks: null }
	};
	const api = await installResourcePageAPI(page, { responses: mocked });
	await page.goto('/swarm/stacks/observability?env=manager-a');
	await expect(page.getByRole('tab', { name: 'Overview', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	await expect(page.getByRole('link', { name: /Unavailable/, exact: false })).toBeVisible();
	await expect(page.getByRole('link', { name: '0 / 0', exact: true })).toHaveCount(0);
	expect(api.mutations).toEqual([]);
});

test('actual Stack metadata scopes warn when older service summaries omit reference observations', async ({
	page
}) => {
	const mocked = responses();
	const { secrets: _secrets, configs: _configs, ...unobserved } = serviceSummary;
	mocked['/swarm/stacks/observability/services'] = resourcePage([unobserved]);
	const api = await installResourcePageAPI(page, { responses: mocked });
	await page.goto('/swarm/stacks/observability?env=manager-a&tab=secrets');
	await expect(page.getByRole('tab', { name: 'Secrets', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	await expect(page.getByRole('alert').filter({ hasText: 'Unavailable' })).toBeVisible();
	expect(api.mutations).toEqual([]);
});
