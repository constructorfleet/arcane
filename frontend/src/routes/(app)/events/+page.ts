import { resolve } from '$app/paths';

import { eventListQuery, eventStatsQuery } from '#lib/query/events.js';
import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
import { throwPageLoadError } from '#lib/utils/api.js';
import { previousResourcePath } from '#lib/utils/resource-parent.js';
import { resolveInitialTableRequest } from '#lib/utils/tables.js';
import { tryCatch } from '#lib/utils/try-catch.js';

import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, url }) => {
	const { queryClient } = await parent();

	const imageUpdateRef = url.searchParams.get('imageUpdateRef')?.trim() || '';
	const defaults = {
		pagination: {
			page: 1,
			limit: 20
		},
		sort: {
			column: 'timestamp',
			direction: 'desc'
		}
	} satisfies SearchPaginationSortRequest;
	const eventRequestOptions = imageUpdateRef
		? { ...defaults, filters: { imageUpdateRef, severity: 'error' } }
		: resolveInitialTableRequest('arcane-events-table', defaults);

	let events;
	let eventStats;
	const operationResult = await tryCatch(
		(async () => Promise.all([queryClient.query(eventListQuery(eventRequestOptions)), queryClient.query(eventStatsQuery())]))()
	);
	if (operationResult.error !== null) {
		const err = operationResult.error;

		throwPageLoadError(err, 'Failed to load events');
	} else {
		[events, eventStats] = operationResult.data;
	}

	const imagesPath = previousResourcePath(url.href, '');
	const source = imagesPath && new URL(imagesPath, url).pathname === resolve('/(app)/images') ? imagesPath : '';
	return { events, eventStats, eventRequestOptions, imageUpdateRef, source };
};
