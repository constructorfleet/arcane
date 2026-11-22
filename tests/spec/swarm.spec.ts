import { test, expect } from '../fixtures/test.fixture';

test.describe('Swarm UI', () => {
	test('cluster panel renders correct lifecycle controls for the environment swarm state', async ({
		page
	}) => {
		await page.goto('/swarm/cluster');
		await page.waitForLoadState('load');
		await expect(page.getByRole('heading', { name: 'Cluster', level: 1 })).toBeVisible();

		const environmentResponse = await page.request.get('/api/agents/0');
		expect(environmentResponse.ok()).toBeTruthy();
		const { data: environment }: { data: { name: string } } = await environmentResponse.json();
		const administration = page.locator('[data-slot="card"]').filter({
			has: page.locator('[data-slot="card-title"]').filter({ hasText: /^Cluster administration$/ })
		});
		await expect(administration.getByText(environment.name, { exact: true })).toBeVisible();

		const swarmInfoResponsePromise = page.waitForResponse('**/api/agents/0/swarm/info');
		await administration.getByRole('button', { name: 'Inspect cluster', exact: true }).click();
		const swarmInfoResponse = await swarmInfoResponsePromise;
		const clusterDialog = page.getByRole('dialog', { name: 'Inspect cluster', exact: true });
		await expect(clusterDialog).toBeVisible();

		const initializeCard = clusterDialog
			.locator('[data-slot="card-title"]')
			.filter({ hasText: 'Initialize Cluster' });
		if (swarmInfoResponse.ok()) {
			await expect(
				clusterDialog.getByRole('button', { name: 'Actions', exact: true })
			).toBeVisible();
			await expect(clusterDialog.getByText('Initialize Cluster')).toHaveCount(0);
			await expect(clusterDialog.getByText('Join Existing Cluster')).toHaveCount(0);
			await clusterDialog.getByRole('button', { name: 'Actions', exact: true }).click();
			const lifecycleDialog = page.getByRole('dialog', { name: 'Unlock / Leave', exact: true });
			await expect(
				lifecycleDialog.getByRole('button', { name: 'Leave Cluster', exact: true })
			).toBeVisible();
		} else {
			await expect(initializeCard).toBeVisible();
			await expect(clusterDialog.getByText('Join Existing Cluster')).toBeVisible();
			await expect(clusterDialog.getByText('Choose how to set up this engine')).toBeVisible();
			await expect(
				clusterDialog.locator('[data-slot="card-title"]').filter({ hasText: 'Cluster Status' })
			).toHaveCount(0);
			await expect(
				clusterDialog.locator('[data-slot="card-title"]').filter({ hasText: 'Join Tokens' })
			).toHaveCount(0);
			await expect(
				clusterDialog.locator('[data-slot="card-title"]').filter({ hasText: 'Update Swarm Spec' })
			).toHaveCount(0);

			await expect(
				clusterDialog.getByPlaceholder('Remote manager addrs (comma separated)')
			).toBeVisible();
			await expect(
				clusterDialog.getByPlaceholder('Listen address (optional)').first()
			).toBeHidden();
			await clusterDialog.getByText('Advanced settings').first().click();
			await expect(
				clusterDialog.getByPlaceholder('Listen address (optional)').first()
			).toBeVisible();
			await expect(
				clusterDialog.getByRole('button', { name: 'Initialize', exact: true })
			).toBeVisible();
			await expect(clusterDialog.getByRole('button', { name: 'Join', exact: true })).toBeVisible();
		}
	});
});
