import { expect, it } from 'vite-plus/test';

import { inUseImageTargets } from './image-scan.js';

it('scans each running image once across nodes and excludes cached copies', () => {
	const copies = [
		{ id: 'image-a', environmentId: 'node-1', inUse: false },
		{ id: 'image-a', environmentId: 'node-2', inUse: true },
		{ id: 'image-a', environmentId: 'node-3', inUse: true },
		{ id: 'image-b', environmentId: 'node-1', inUse: true },
		{ id: 'old-image', environmentId: 'node-1', inUse: false }
	];
	expect(inUseImageTargets(copies)).toEqual([copies[1], copies[3]]);
});
