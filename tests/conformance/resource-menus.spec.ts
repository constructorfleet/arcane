import AxeBuilder from '@axe-core/playwright';
import { expect, test, type Locator, type Page } from '@playwright/test';

async function openActions(page: Page, scenario = 'service') {
	await page.goto(`/?scenario=${scenario}`);
	const actions = page.getByRole('button', { name: 'Actions for observability_prometheus' });
	await expect(actions.locator('.sr-only')).toHaveCSS('position', 'absolute');
	await expect(actions.locator('.sr-only')).toHaveCSS('width', '1px');
	await actions.click();
	const rootFontSize = await page
		.locator('html')
		.evaluate((element) => parseFloat(getComputedStyle(element).fontSize));
	const menuPadding = await page
		.getByRole('menu')
		.first()
		.evaluate((element) => parseFloat(getComputedStyle(element).paddingTop));
	expect(menuPadding).toBeCloseTo(rootFontSize * 0.25);
	await expect(page.getByRole('menu').first()).toHaveCSS('border-top-width', '1px');
	await expect(page.getByRole('menuitem').first()).toHaveCSS('display', 'flex');
	const itemFontSize = await page
		.getByRole('menuitem')
		.first()
		.evaluate((element) => parseFloat(getComputedStyle(element).fontSize));
	expect(itemFontSize).toBeCloseTo(rootFontSize * 0.875);
	expect(
		await page
			.getByRole('menu')
			.first()
			.evaluate((element) => getComputedStyle(element).backgroundColor)
	).not.toBe('rgba(0, 0, 0, 0)');
	expect(
		await page.evaluate(
			() =>
				document.documentElement.classList.contains('dark') ===
				matchMedia('(prefers-color-scheme: dark)').matches
		)
	).toBe(true);
}

async function openSubmenu(page: Page, name: string) {
	const trigger = page.getByRole('menuitem', { name, exact: true });
	await trigger.hover();
	await trigger.focus();
	await page.keyboard.press('ArrowRight');
}

async function expectInsideViewport(page: Page, locator: Locator) {
	const box = await locator.boundingBox();
	expect(box).not.toBeNull();
	const viewport = page.viewportSize();
	expect(viewport).not.toBeNull();
	expect(box!.x).toBeGreaterThanOrEqual(0);
	expect(box!.y).toBeGreaterThanOrEqual(0);
	expect(box!.x + box!.width).toBeLessThanOrEqual(viewport!.width + 1);
	expect(box!.y + box!.height).toBeLessThanOrEqual(viewport!.height + 1);
	const receivesPointer = await locator.evaluate((element) => {
		const bounds = element.getBoundingClientRect();
		const hit = document.elementFromPoint(
			bounds.x + bounds.width / 2,
			bounds.y + bounds.height / 2
		);
		return hit !== null && element.contains(hit);
	});
	expect(receivesPointer, 'menu surface is visible above surrounding content, not clipped').toBe(
		true
	);
}

test('Overview is the single top-level destination; details and relationships are separate', async ({
	page
}, testInfo) => {
	await openActions(page);
	await expect(page.getByRole('menuitem', { name: 'View Details', exact: true })).toHaveCount(1);
	await expect(page.getByRole('menuitem', { name: 'Overview', exact: true })).toHaveCount(0);
	await expect(page.getByRole('menuitem', { name: 'Details', exact: true })).toBeVisible();
	await expect(
		page.getByRole('menuitem', { name: 'Related resources', exact: true })
	).toBeVisible();
	await openSubmenu(page, 'Details');
	await expect(page.getByRole('menuitem', { name: 'Configuration', exact: true })).toBeVisible();
	await expect(page.getByRole('menuitem', { name: 'Logs', exact: true })).toBeVisible();
	await expect(page.getByRole('menuitem', { name: 'Tasks', exact: true })).toHaveCount(0);
	await expectInsideViewport(page, page.getByRole('menu').last());
	await testInfo.attach('details-menu', {
		body: await page.screenshot({ animations: 'disabled' }),
		contentType: 'image/png'
	});
});

test('scoped related destinations preserve identity, return context and fragment', async ({
	page
}, testInfo) => {
	await openActions(page);
	const overview = page.getByRole('menuitem', { name: 'View Details', exact: true });
	await expect(overview).toHaveAttribute(
		'href',
		'/swarm/services/service-7?env=node-2&cluster=cluster-9&agents=node-2%2Cnode-3&prev=%2Fswarm%2Fstacks%2Fobservability%3Fenv%3Dmanager%26tab%3Dservices%26tablePage%3D2&tablePage=3&tab=overview#resource'
	);
	await openSubmenu(page, 'Related resources');
	const tasks = page.getByRole('menuitem', { name: 'Tasks', exact: true });
	await expect(tasks).toHaveAttribute(
		'href',
		'/swarm/services/service-7?env=node-2&cluster=cluster-9&agents=node-2%2Cnode-3&prev=%2Fswarm%2Fstacks%2Fobservability%3Fenv%3Dmanager%26tab%3Dservices%26tablePage%3D2&tablePage=3&tab=tasks#resource'
	);
	await expect(tasks.locator('svg')).toHaveCount(1);
	const disabled = page.getByRole('menuitem', { name: 'Volumes', exact: true });
	await expect(disabled).not.toHaveAttribute('href');
	await expect(page.getByRole('menuitem', { name: 'Configuration', exact: true })).toHaveCount(0);
	await expect(page.getByRole('menuitem', { name: 'Volumes', exact: true })).toHaveAttribute(
		'aria-disabled',
		'true'
	);
	await expectInsideViewport(page, page.getByRole('menu').last());
	await testInfo.attach('related-menu', {
		body: await page.screenshot({ animations: 'disabled' }),
		contentType: 'image/png'
	});
	await tasks.click();
	await expect(page).toHaveURL(/tab=tasks#resource$/);
});

test('empty groups are omitted and permission-hidden tabs do not appear', async ({ page }) => {
	await openActions(page, 'overview-only');
	await expect(page.getByRole('menuitem')).toHaveCount(1);
	await page.keyboard.press('Escape');
	await openActions(page, 'restricted');
	await openSubmenu(page, 'Details');
	await expect(page.getByRole('menuitem', { name: 'Logs', exact: true })).toHaveCount(0);
	await expect(page.getByRole('menuitem', { name: 'Configuration', exact: true })).toBeVisible();
});

test('unavailable resources disable all navigation groups', async ({ page }) => {
	await openActions(page, 'unavailable');
	for (const name of ['View Details', 'Details', 'Related resources']) {
		await expect(page.getByRole('menuitem', { name, exact: true })).toHaveAttribute(
			'aria-disabled',
			'true'
		);
	}
	await page.keyboard.press('Escape');
	await expect(
		page.getByRole('button', { name: 'Actions for observability_prometheus' })
	).toBeFocused();
});

test('duplicate destinations do not leak into multiple submenus', async ({ page }) => {
	await openActions(page, 'duplicates');
	await expect(page.getByRole('menuitem', { name: 'View Details', exact: true })).toHaveCount(1);
	await openSubmenu(page, 'Details');
	await expect(
		page.getByRole('menuitem', { name: 'Duplicate configuration', exact: true })
	).toHaveCount(0);
	await expect(page.getByRole('menuitem', { name: 'Configuration', exact: true })).toHaveCount(1);
	await page.keyboard.press('ArrowLeft');
	await openSubmenu(page, 'Related resources');
	await expect(
		page.getByRole('menuitem', { name: 'Duplicate configuration', exact: true })
	).toHaveCount(0);
});

test('open menus meet automated accessibility checks including contrast', async ({ page }) => {
	await openActions(page);
	await openSubmenu(page, 'Related resources');
	const results = await new AxeBuilder({ page })
		.withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
		.analyze();
	expect(results.violations).toEqual([]);
});

test('a disabled destination cannot navigate even when activated directly', async ({ page }) => {
	await openActions(page);
	await openSubmenu(page, 'Related resources');
	const currentURL = page.url();
	await page
		.getByRole('menuitem', { name: 'Volumes', exact: true })
		.evaluate((element) => (element as HTMLElement).click());
	await expect(page).toHaveURL(currentURL);
});

test('resources without an Overview keep their other destinations', async ({ page }) => {
	await openActions(page, 'no-overview');
	await expect(page.getByRole('menuitem', { name: 'View Details', exact: true })).toHaveCount(0);
	await openSubmenu(page, 'Details');
	await expect(page.getByRole('menuitem', { name: 'Configuration', exact: true })).toBeVisible();
});

test('resource-specific details actions remain actionable in the Details group', async ({
	page
}) => {
	await openActions(page, 'image');
	await openSubmenu(page, 'Details');
	const releaseNotes = page.getByRole('menuitem', { name: 'Release notes', exact: true });
	await expect(releaseNotes.locator('svg')).toHaveCount(1);
	await releaseNotes.click();
	await expect(page.getByRole('status')).toHaveText('Release notes requested');
});
