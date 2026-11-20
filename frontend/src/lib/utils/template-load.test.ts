import { describe, expect, it, vi } from 'vite-plus/test';

import type { GlobalVariable } from '#lib/types/env.js';

import { globalVariablesToMap } from './template-load.js';

vi.mock('#lib/paraglide/messages.js', () => ({ m: {} }));
vi.mock('#lib/services/template-service.js', () => ({ templateService: {} }));
vi.mock('#lib/services/variable-service.js', () => ({ variableService: {} }));
vi.mock('#lib/utils/api.js', () => ({ extractApiErrorMessage: vi.fn() }));
vi.mock('#lib/utils/auth.js', () => ({ userHasPermission: vi.fn() }));

function variable(key: string, value: string, options: Partial<GlobalVariable> = {}): GlobalVariable {
	return {
		id: key,
		key,
		value,
		isSecret: false,
		allEnvironments: true,
		injectIntoContainers: false,
		environmentIds: [],
		createdAt: '',
		...options
	};
}

describe('shared stack variable completion', () => {
	it('uses destination scope and scoped overrides independent of list order', () => {
		const variables = [
			variable('SHARED', 'node-a', { allEnvironments: false, environmentIds: ['a'] }),
			variable('SHARED', 'default'),
			variable('ONLY_B', 'other-node', { allEnvironments: false, environmentIds: ['b'] }),
			variable('PUBLIC', 'public-value')
		];
		expect(globalVariablesToMap(variables, 'a')).toEqual({ SHARED: 'node-a', PUBLIC: 'public-value' });
		expect(globalVariablesToMap(variables, 'b')).toEqual({ SHARED: 'default', PUBLIC: 'public-value', ONLY_B: 'other-node' });
	});

	it('offers secret keys without exposing their values', () => {
		expect(globalVariablesToMap([variable('SECRET', 'owned-secret', { isSecret: true })], 'a')).toEqual({ SECRET: '' });
		expect(globalVariablesToMap(undefined, 'a')).toEqual({});
	});
});
