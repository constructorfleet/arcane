import { describe, expect, it } from 'vite-plus/test';

import { applyColumnVisibilityPatch, encodeColumnVisibility } from './arcane-table.types.svelte';

describe('resource table column preferences', () => {
	it('keeps legacy hidden columns and leaves new default-hidden columns hidden', () => {
		const visibility = { environment: false, status: true };
		applyColumnVisibilityPatch(visibility, ['status'], []);
		expect(visibility).toEqual({ environment: false, status: false });
	});

	it('restores a default-hidden Agent column after the user enables it', () => {
		const defaults = { environment: false };
		const saved = encodeColumnVisibility({ environment: true, name: true, status: false }, defaults);
		const restored = { ...defaults };
		applyColumnVisibilityPatch(restored, saved.v, saved.visibleColumns);
		expect(restored).toEqual({ environment: true, status: false });
	});

	it('persists hiding the Agent column again', () => {
		const defaults = { environment: false };
		const saved = encodeColumnVisibility(defaults, defaults);
		expect(saved).toEqual({ v: ['environment'], visibleColumns: [] });
	});
});
