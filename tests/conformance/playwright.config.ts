import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
	testDir: '.',
	testMatch: '*.spec.ts',
	fullyParallel: true,
	forbidOnly: !!process.env.CI,
	retries: 0,
	failOnFlakyTests: !!process.env.CI,
	reporter: [['list'], ['html', { outputFolder: '../.report/conformance-html', open: 'never' }]],
	outputDir: '../.report/conformance-results',
	use: {
		baseURL: 'http://127.0.0.1:4175',
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure'
	},
	projects: ['light', 'dark'].flatMap((theme) => [
		{
			name: `desktop-${theme}`,
			use: {
				...devices['Desktop Chrome'],
				viewport: { width: 1280, height: 800 },
				colorScheme: theme as 'light' | 'dark'
			}
		},
		{
			name: `mobile-${theme}`,
			use: {
				...devices['iPhone 13'],
				defaultBrowserType: 'chromium' as const,
				colorScheme: theme as 'light' | 'dark'
			}
		}
	]),
	webServer: [
		{
			command: 'vp -C ../../frontend dev --config conformance/vite.config.ts --strictPort',
			url: 'http://127.0.0.1:4175',
			reuseExistingServer: !process.env.CI
		},
		{
			command: 'vp -C ../../frontend preview --port 4176 --host 127.0.0.1 --strictPort',
			url: 'http://127.0.0.1:4176',
			reuseExistingServer: !process.env.CI
		}
	]
});
