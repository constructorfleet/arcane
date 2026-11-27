import { m } from '#lib/paraglide/messages.js';
import { ENVIRONMENT_FILTER } from '#lib/utils/environments.js';

import type { ColumnSpec } from './arcane-table.types.svelte';
import EnvironmentCell from './cells/environment-cell.svelte';

export function environmentColumns<T extends { environmentId?: string }>(): ColumnSpec<T>[] {
	return [
		{
			id: ENVIRONMENT_FILTER,
			accessorFn: (row) => row.environmentId,
			title: m.swarm_node(),
			sortable: true,
			hidden: true,
			trailing: true,
			width: 'min',
			cellComponent: EnvironmentCell
		}
	];
}
