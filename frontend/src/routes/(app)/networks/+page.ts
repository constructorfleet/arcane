import { networkListQuery } from '#lib/query/networks.js';
import { environmentStore } from '#lib/stores/environment.store.svelte.js';
import { throwPageLoadError } from '#lib/utils/api.js';
import { readableEnvironments, resolveLinkedListAgent } from '#lib/utils/environments.js';
import { resolveInitialListPageRequest } from '#lib/utils/tables.js';

import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, url }) => {
	const [{ queryClient }] = await Promise.all([parent(), environmentStore.ready]);
	const linkedAgent = resolveLinkedListAgent(url, 'networks:list');
	const scope = linkedAgent ?? environmentStore.shownKey;
	const networkRequestOptions = resolveInitialListPageRequest(
		linkedAgent ? 'arcane-network-agent-scoped-table' : 'arcane-networks-table',
		{
			column: 'name',
			direction: 'asc'
		}
	);

	const networks = await queryClient
		.ensureQueryData({
			...networkListQuery(scope, readableEnvironments('networks:list', linkedAgent), networkRequestOptions),
			revalidateIfStale: true
		})
		.catch((err: unknown) => throwPageLoadError(err, 'Failed to load networks'));

	return {
		linkedAgent,
		scope,
		networks,
		networkRequestOptions
	};
};
