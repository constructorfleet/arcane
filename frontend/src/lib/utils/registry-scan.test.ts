import { expect, it } from 'vite-plus/test';

import { compareRegistryFindings } from './registry-scan.js';

it('compares advisory and package together across installed version changes', () => {
	const finding = { id: 'CVE-test', package: 'lib', version: '1', severity: 'Medium' };
	expect(
		compareRegistryFindings(
			[finding, { ...finding, package: 'old' }],
			[
				{ ...finding, version: '2' },
				{ ...finding, version: '2' },
				{ ...finding, package: 'new' }
			]
		)
	).toEqual({ removed: 1, persisting: 1, added: 1 });
});
