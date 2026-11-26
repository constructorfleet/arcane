import { describe, expect, it } from 'vite-plus/test';

import type { ImageSummary } from '#lib/types/image.js';
import type { EnvironmentRow } from '#lib/utils/environments.js';

import { imageActionTargets } from './image-table.helpers.js';

function image(): EnvironmentRow<ImageSummary> {
	return {
		id: 'sha256:representative',
		repo: 'example/app',
		tag: 'latest',
		repoTags: ['example/app:latest'],
		repoDigests: ['example/app@sha256:representative'],
		created: 0,
		size: 0,
		virtualSize: 0,
		labels: {},
		inUse: false,
		environmentId: 'representative-agent',
		environmentName: 'Representative',
		environmentTagged: true
	};
}

describe('image action targets', () => {
	it('preserves a node-local image target', () => {
		const row = image();
		expect(imageActionTargets(row)).toEqual([{ ...row, repositoryGroup: undefined }]);
	});

	it('keeps each copy on its owning agent with its own references', () => {
		const row = image();
		row.repositoryGroup = {
			copies: [
				{
					id: 'sha256:same',
					environmentId: 'agent-a',
					environmentName: 'Node A',
					tags: ['example/app:v1'],
					repoDigests: ['example/app@sha256:a'],
					pinnedReferences: ['example/app:v1@sha256:a'],
					inUse: false
				},
				{ id: 'sha256:same', environmentId: 'agent-b', tags: null, repoDigests: [], pinnedReferences: [], inUse: false }
			],
			digests: ['sha256:a'],
			minSize: 0,
			maxSize: 0,
			variants: 1,
			scannedVariants: 0
		};
		const targets = imageActionTargets(row);
		expect(targets).toHaveLength(2);
		expect(targets[0]).toMatchObject({
			id: 'sha256:same',
			environmentId: 'agent-a',
			environmentName: 'Node A',
			repoTags: ['example/app:v1'],
			repoDigests: ['example/app@sha256:a'],
			pinnedReferences: ['example/app:v1@sha256:a'],
			environmentTagged: true
		});
		expect(targets[1]).toMatchObject({
			id: 'sha256:same',
			environmentId: 'agent-b',
			environmentName: 'agent-b',
			repoTags: [],
			repoDigests: [],
			pinnedReferences: []
		});
		expect(targets.every((target) => target.repositoryGroup === undefined)).toBe(true);
	});
});
