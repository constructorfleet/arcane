import { volumeListQuery } from '#lib/query/volumes.js';
import { environmentStore } from '#lib/stores/environment.store.svelte.js';
import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
import { readableEnvironments, resolveLinkedListAgent } from '#lib/utils/environments.js';
import { resolveInitialTableRequest } from '#lib/utils/tables.js';

import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, url }) => {
	const [{ queryClient }] = await Promise.all([parent(), environmentStore.ready]);
	const linkedAgent = resolveLinkedListAgent(url, 'volumes:list');
	const scope = linkedAgent ?? environmentStore.shownKey;

	const volumeRequestOptions = resolveInitialTableRequest(
		linkedAgent ? 'arcane-volume-agent-scoped-table' : 'arcane-volumes-table',
		{
			pagination: {
				page: 1,
				limit: 20
			},
			sort: {
				column: 'name',
				direction: 'asc'
			}
		} satisfies SearchPaginationSortRequest,
		{ deferredSortColumns: ['size'] }
	);

	const volumes = await queryClient.ensureQueryData({
		...volumeListQuery(scope, readableEnvironments('volumes:list', linkedAgent), volumeRequestOptions),
		revalidateIfStale: true
	});

	return {
		linkedAgent,
		scope,
		volumes,
		volumeRequestOptions
	};
};
