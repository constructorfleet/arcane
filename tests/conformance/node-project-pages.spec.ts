import { expect, test, type Page, type TestInfo } from '@playwright/test';

import { installResourcePageAPI, resourcePage } from './fixtures/api.js';
import {
	contractNode,
	contractProject,
	contractTasks,
	contractTemplateContents,
	contractTemplateRegistry,
	contractTemplates
} from './fixtures/node-projects.js';

test.use({ baseURL: 'http://127.0.0.1:4176' });

function responses(): Record<string, unknown> {
	return {
		'/swarm/nodes': resourcePage([contractNode]),
		'/swarm/nodes/node-contract': { success: true, data: contractNode },
		'/swarm/nodes/node-contract/tasks': resourcePage(contractTasks),
		'/swarm/tasks': resourcePage(
			contractTasks.filter(
				(task) => task.currentState !== 'shutdown' || task.desiredState !== 'shutdown'
			)
		),
		'/projects': resourcePage([contractProject]),
		'/projects/counts': {
			success: true,
			data: {
				runningProjects: 1,
				stoppedProjects: 0,
				totalProjects: 1,
				archivedProjects: 0,
				updatesAvailable: 0
			}
		},
		'/projects/tags': { success: true, data: contractProject.tags },
		'/projects/project-contract': { success: true, data: contractProject },
		'/projects/project-contract/compose': { success: true, data: contractProject },
		'/projects/project-contract/runtime': { success: true, data: contractProject },
		'/projects/project-contract/updates': { success: true, data: contractProject },
		'/projects/project-contract/workspace': {
			success: true,
			data: { files: [], fileTreeRevision: 'fixture', fileTreeTruncated: false }
		},
		'/templates': resourcePage(contractTemplates),
		'/templates/all': { success: true, data: contractTemplates },
		'/templates/registries': { success: true, data: [contractTemplateRegistry] },
		...Object.fromEntries(
			contractTemplateContents.map((content) => [
				`/templates/${content.template.id}/content`,
				{ success: true, data: content }
			])
		)
	};
}

async function capture(page: Page, testInfo: TestInfo, name: string) {
	if (!['desktop-light', 'mobile-dark'].includes(testInfo.project.name)) return;
	const geometry = await page.evaluate(() => ({
		width: document.documentElement.clientWidth,
		scroll: document.documentElement.scrollWidth
	}));
	expect(geometry.scroll).toBeLessThanOrEqual(geometry.width + 1);
	const path = testInfo.outputPath(name + '.png');
	await page.screenshot({ path, fullPage: true, animations: 'disabled' });
	await testInfo.attach(name, { path, contentType: 'image/png' });
}

async function expectUsableMobileHeader(page: Page, isMobile: boolean) {
	if (!isMobile) return;
	const geometry = await page
		.locator('[data-slot="resource-header-row"]')
		.first()
		.evaluate((header) => {
			const information = header.querySelector('[data-slot="resource-header-information"]');
			const actions = header.lastElementChild;
			return {
				viewport: document.documentElement.clientWidth,
				identityWidth: information?.getBoundingClientRect().width ?? 0,
				titleWidth: information?.querySelector('h1')?.getBoundingClientRect().width ?? 0,
				actionsRight: actions?.getBoundingClientRect().right ?? 0
			};
		});
	expect(geometry.identityWidth).toBeGreaterThanOrEqual(geometry.viewport * 0.7);
	expect(geometry.titleWidth).toBeGreaterThanOrEqual(96);
	expect(geometry.actionsRight).toBeLessThanOrEqual(geometry.viewport);
}

async function openRelated(page: Page) {
	const related = page.getByRole('menuitem', { name: 'Related resources', exact: true });
	await related.hover();
	await related.focus();
	await page.keyboard.press('ArrowRight');
}

test('actual Nodes route shows colored elected roles and curated node-scoped Tasks', async ({
	page,
	isMobile
}, testInfo) => {
	const api = await installResourcePageAPI(page, { responses: responses() });
	await page.goto('/swarm/nodes');
	await expect(
		page.getByRole('link', { name: contractNode.hostname, exact: true }).first()
	).toBeVisible();
	await expect(
		page.getByText('Leader', { exact: true }).filter({ visible: true }).first()
	).toBeVisible();
	await expect(
		page.getByText('Leader', { exact: true }).filter({ visible: true }).first()
	).toHaveClass(/bg-mint/);
	await expect(
		page.getByText('Manager', { exact: true }).filter({ visible: true }).first()
	).toHaveClass(/bg-purple/);
	if (!isMobile) {
		await expect(page.getByRole('columnheader', { name: /Scheduling/ })).toBeVisible();
		await expect(page.getByRole('columnheader', { name: /Engine/ })).toHaveCount(0);
	}
	await capture(page, testInfo, 'nodes-list');
	await expect(page.getByText('automation-label-value-0', { exact: true })).toHaveCount(0);
	await page
		.getByRole('button', { name: `Actions for ${contractNode.hostname}`, exact: true })
		.click();
	await expect(page.getByRole('menuitem', { name: 'View Details', exact: true })).toHaveAttribute(
		'href',
		/tab=overview/
	);
	await openRelated(page);
	await expect(page.getByRole('menuitem', { name: 'Tasks', exact: true })).toHaveAttribute(
		'href',
		/tab=tasks/
	);
	await page.getByRole('menuitem', { name: 'Tasks', exact: true }).click();
	await expect(page.getByRole('tab', { name: 'Tasks', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	await expect
		.poll(() => api.requests.some((request) => request.path === '/swarm/nodes/node-contract/tasks'))
		.toBe(true);
	expect(api.unknownGET).toEqual([]);
	expect(api.mutations).toEqual([]);
});

test('actual Tasks route keeps node plain, combines states, retains diagnostics and gates container navigation', async ({
	page,
	isMobile
}) => {
	const mocked = responses();
	const api = await installResourcePageAPI(page, { responses: mocked });
	await page.goto('/swarm/tasks');
	await expect(
		page.getByText('contract_web.1', { exact: true }).filter({ visible: true }).first()
	).toBeVisible();
	await expect(page.getByRole('link', { name: contractNode.hostname, exact: true })).toHaveCount(0);
	await expect(
		page
			.getByText('Container exited with status 1', { exact: true })
			.filter({ visible: true })
			.first()
	).toBeVisible();
	await expect(page.getByRole('button', { name: 'View container', exact: true })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Verify container', exact: true })).toHaveCount(1);
	if (!isMobile) {
		await expect(
			page.getByRole('columnheader', { name: 'Desired State', exact: true })
		).toHaveCount(0);
	}
	await page.getByRole('button', { name: 'Actions for contract_web.1', exact: true }).click();
	await openRelated(page);
	await expect(page.getByRole('menuitem', { name: 'View service', exact: true })).toHaveAttribute(
		'href',
		/service-contract/
	);
	await expect(page.getByRole('menuitem', { name: 'Stack', exact: true })).toHaveAttribute(
		'href',
		/contract-stack/
	);
	await expect(page.getByRole('menuitem', { name: 'View container', exact: true })).toHaveCount(0);
	await page.keyboard.press('Escape');
	await page.keyboard.press('Escape');
	mocked['/swarm/tasks'] = resourcePage(contractTasks);
	await page.getByRole('switch', { name: 'Show shutdown history', exact: true }).click();
	await expect
		.poll(() =>
			api.requests.some(
				(request) => request.path === '/swarm/tasks' && request.search['history'] === 'all'
			)
		)
		.toBe(true);
	await expect(
		page.getByText('contract_web.1.old', { exact: true }).filter({ visible: true }).first()
	).toBeVisible();
	expect(api.unknownGET).toEqual([]);
	expect(api.mutations).toEqual([]);
});

test('actual Project opens Overview with metadata below its clean title and Services remains explicitly scoped', async ({
	page,
	isMobile
}, testInfo) => {
	const api = await installResourcePageAPI(page, { responses: responses() });
	await page.goto('/projects/project-contract?env=manager-a');
	await expect(page.getByRole('tab', { name: 'Overview', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	await expect(
		page.getByText('/srv/projects/contract-project', { exact: true }).first()
	).toBeVisible();
	await expect(page.getByText('Configuration source', { exact: true })).toBeVisible();
	await expectUsableMobileHeader(page, isMobile);
	await capture(page, testInfo, 'project-overview');
	await page.getByRole('tab', { name: /Services/ }).click();
	await expect(page).toHaveURL(/tab=services/);
	expect(api.mutations).toEqual([]);
	expect(api.unknownGET).toEqual([]);
});

test('actual Projects list hides path, created and user tags by default and links names to Overview', async ({
	page,
	isMobile
}) => {
	const api = await installResourcePageAPI(page, { responses: responses() });
	await page.goto('/projects');
	const name = page.getByRole('link', { name: 'contract-project', exact: true }).first();
	await expect(name).toBeVisible();
	await expect(name).toHaveAttribute('href', /tab=overview/);
	if (!isMobile) {
		for (const title of ['Tags', 'Working Directory', 'Created']) {
			await expect(page.getByRole('columnheader', { name: title, exact: true })).toHaveCount(0);
		}
		await expect(
			page.getByRole('columnheader', { name: 'Configuration source', exact: true })
		).toBeVisible();
	}
	await expect(page.getByText('category-11', { exact: true })).toHaveCount(0);
	await expect(page.getByText('automation-label-value-0', { exact: true })).toHaveCount(0);
	expect(api.mutations).toEqual([]);
	expect(api.unknownGET).toEqual([]);
});

for (const template of contractTemplates) {
	test(`actual ${template.isRemote ? 'remote' : 'local'} Template opens Overview and edits remain separate`, async ({
		page,
		isMobile
	}, testInfo) => {
		const api = await installResourcePageAPI(page, { responses: responses() });
		await page.goto(`/projects/templates/${template.id}`);
		await expect(page.getByRole('tab', { name: 'Overview', exact: true })).toHaveAttribute(
			'aria-selected',
			'true'
		);
		await expect(page.getByText(template.description, { exact: true }).first()).toBeVisible();
		await expectUsableMobileHeader(page, isMobile);
		await expect(page.getByText('Source', { exact: true }).first()).toBeVisible();
		await capture(
			page,
			testInfo,
			template.isRemote ? 'remote-template-overview' : 'local-template-overview'
		);
		expect(api.mutations).toEqual([]);
		expect(api.unknownGET).toEqual([]);
	});
}

test('actual Templates browse keeps categories bounded and registry filtering independent of source type', async ({
	page,
	isMobile
}) => {
	const api = await installResourcePageAPI(page, { responses: responses() });
	await page.goto('/projects/templates');
	await expect(page.getByText('Local contract template', { exact: true }).first()).toBeVisible();
	await expect(page.getByText('category-11', { exact: true })).toHaveCount(0);
	await page
		.getByRole('button', { name: 'Actions for Local contract template', exact: true })
		.click();
	await expect(page.getByRole('menuitem', { name: 'View Details', exact: true })).toHaveAttribute(
		'href',
		/tab=overview/
	);
	await page.keyboard.press('Escape');
	if (isMobile) await page.getByRole('button', { name: 'Filter', exact: true }).click();
	await page.getByTestId('facet-registry-trigger').filter({ visible: true }).click();
	await page.getByTestId('facet-registry-option-registry-contract').click();
	await expect
		.poll(() =>
			api.requests.some(
				(request) =>
					request.path === '/templates' &&
					request.search['registry'] === 'registry-contract' &&
					!request.search['type']
			)
		)
		.toBe(true);
	expect(api.mutations).toEqual([]);
	expect(api.unknownGET).toEqual([]);
});

test('actual task container navigation requires observation and rechecks a stopped target without leaving context', async ({
	page
}) => {
	const mocked = responses();
	mocked['/containers/container-contract'] = {
		success: true,
		data: {
			id: 'container-contract',
			labels: { 'com.docker.swarm.task.id': 'task-running' },
			state: { running: true, status: 'running' }
		}
	};
	const api = await installResourcePageAPI(page, { responses: mocked });
	await page.goto('/swarm/tasks');
	await expect(page.getByRole('button', { name: 'Verify container', exact: true })).toBeVisible();
	expect(
		api.requests.filter((request) => request.path === '/swarm/nodes/node-contract')
	).toHaveLength(0);
	expect(
		api.requests.filter((request) => request.path === '/containers/container-contract')
	).toHaveLength(0);
	await page.getByRole('button', { name: 'Verify container', exact: true }).click();
	await expect(page.getByRole('button', { name: 'View container', exact: true })).toBeVisible();
	expect(
		api.requests.filter((request) => request.path === '/containers/container-contract')
	).toHaveLength(1);
	await page.getByRole('button', { name: 'Actions for contract_web.1', exact: true }).click();
	await openRelated(page);
	mocked['/containers/container-contract'] = {
		success: true,
		data: {
			id: 'container-contract',
			labels: { 'com.docker.swarm.task.id': 'task-running' },
			state: { running: false, status: 'exited' }
		}
	};
	await page.getByRole('menuitem', { name: 'View container', exact: true }).click();
	await expect(
		page
			.getByRole('status')
			.filter({ hasText: 'Container is not running or is no longer available' })
			.filter({ visible: true })
	).toBeVisible();
	await expect(page).toHaveURL(/\/swarm\/tasks$/);
	await expect(page.getByRole('button', { name: 'View container', exact: true })).toHaveCount(0);
	expect(
		api.requests.filter((request) => request.path === '/containers/container-contract')
	).toHaveLength(2);
	expect(api.unknownGET).toEqual([]);
	expect(api.mutations).toEqual([]);
});
