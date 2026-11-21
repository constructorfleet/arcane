import { describe, expect, it } from 'vite-plus/test';

import { contextualResourceHref, previousResourcePath, resourceTablePage } from './resource-parent';

const origin = 'https://arcane.example';

describe('resource return navigation', () => {
	it('carries nested parent tabs and keeps target agent scope', () => {
		const stack = '/swarm/stacks/metrics?env=manager&tab=services&tablePage=2';
		const service = contextualResourceHref('/swarm/services/storage?env=manager&tab=tasks', origin + stack);
		const container = contextualResourceHref('/containers/storage.1?env=worker', origin + service);
		expect(new URL(container, origin).searchParams.get('env')).toBe('worker');
		expect(previousResourcePath(origin + container, '/containers')).toBe(service);
		expect(previousResourcePath(origin + service, '/swarm/services')).toBe(stack);
	});

	it('keeps the original parent when navigating within the same detail page', () => {
		const current = contextualResourceHref('/swarm/services/one?tab=tasks', origin + '/swarm/stacks/stack');
		const next = contextualResourceHref('/swarm/services/one?tab=overview', origin + current);
		expect(previousResourcePath(origin + next, '/swarm/services')).toBe('/swarm/stacks/stack');
	});

	it('falls back for external, normalized external, and self-referencing return paths', () => {
		for (const destination of ['//example.org', '/x/..//example.org', '/\\example.org', '/containers/one']) {
			const params = new URLSearchParams({ prev: destination });
			expect(previousResourcePath(`${origin}/containers/one?${params}`, '/containers')).toBe('/containers');
		}
	});

	it('accepts only bounded positive page numbers', () => {
		expect(resourceTablePage('2')).toBe(2);
		for (const value of [null, '0', '-1', '1.5', 'bad', '1000001']) expect(resourceTablePage(value)).toBe(1);
	});
});
