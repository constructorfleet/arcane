import { containerListQuery } from '#lib/query/containers.js';
import { environmentStore } from '#lib/stores/environment.store.svelte.js';
import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
import { throwPageLoadError } from '#lib/utils/api.js';
import { readableEnvironments, resolveLinkedListAgent } from '#lib/utils/environments.js';
import { resolveInitialTableRequest } from '#lib/utils/tables.js';

import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, url }) => {
	const [{ queryClient }] = await Promise.all([parent(), environmentStore.ready]);
	const linkedAgent = resolveLinkedListAgent(url, 'containers:list');
	const scope = linkedAgent ?? environmentStore.shownKey;

	const containerRequestOptions = resolveInitialTableRequest(
		linkedAgent ? 'arcane-container-agent-scoped-table' : 'arcane-container-table',
		{
			pagination: { page: 1, limit: 20 },
			sort: { column: 'created', direction: 'desc' }
		} satisfies SearchPaginationSortRequest
	);

	const containers = await queryClient
		.ensureQueryData({
			...containerListQuery(scope, readableEnvironments('containers:list', linkedAgent), {
				...containerRequestOptions,
				groupByProject: false
			}),
			revalidateIfStale: true
		})
		.catch((err: unknown) => throwPageLoadError(err, 'Failed to load containers'));

	return {
		linkedAgent,
		containers,
		containerRequestOptions
	};
};
