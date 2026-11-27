import { imageLibraryQuery, imageUsageCountsQuery } from '#lib/query/images.js';
import { environmentStore } from '#lib/stores/environment.store.svelte.js';
import { featureStore } from '#lib/stores/features.store.svelte.js';
import { throwPageLoadError } from '#lib/utils/api.js';
import { readableEnvironments, resolveLinkedListAgent } from '#lib/utils/environments.js';
import { resourceTablePage } from '#lib/utils/resource-navigation.js';
import { resolveInitialListPageRequest } from '#lib/utils/tables.js';
import { tryCatch } from '#lib/utils/try-catch.js';

import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, url }) => {
	const [{ queryClient }] = await Promise.all([parent(), environmentStore.ready]);
	const linkedAgent = resolveLinkedListAgent(url, 'images:list');
	const scope = linkedAgent ?? environmentStore.shownKey;
	const environments = readableEnvironments('images:list', linkedAgent);
	const imageRequestOptions = resolveInitialListPageRequest(
		linkedAgent ? 'arcane-image-agent-scoped-table' : 'arcane-image-table',
		{
			column: 'created',
			direction: 'desc'
		}
	);

	if (imageRequestOptions.pagination) imageRequestOptions.pagination.page = resourceTablePage(url.searchParams.get('tablePage'));

	const listQuery = imageLibraryQuery;
	const result = await tryCatch(
		Promise.all([
			queryClient.ensureQueryData({ ...listQuery(scope, environments, imageRequestOptions), revalidateIfStale: true }),
			queryClient.ensureQueryData({ ...imageUsageCountsQuery(scope, environments), revalidateIfStale: true }),
			featureStore.load(environments.map((env) => env.id))
		])
	);
	if (result.error !== null) throwPageLoadError(result.error, 'Failed to load images');
	const [images, imageUsageCounts] = result.data;

	return { linkedAgent, scope, images, imageRequestOptions, imageUsageCounts };
};
