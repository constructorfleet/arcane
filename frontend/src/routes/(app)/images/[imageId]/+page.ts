import { error } from '@sveltejs/kit';

import { m } from '#lib/paraglide/messages.js';
import { imageDetailQuery, imageLibraryQuery } from '#lib/query/images.js';
import { environmentStore } from '#lib/stores/environment.store.svelte.js';
import { featureStore } from '#lib/stores/features.store.svelte.js';
import type { ImageDetailSummary, ImageRepositoryCopy } from '#lib/types/image.js';
import { hasPermission } from '#lib/utils/auth.js';
import { readableEnvironments, resolveDetailEnvironmentId } from '#lib/utils/environments.js';
import { tryCatch } from '#lib/utils/try-catch.js';

import type { PageLoad } from './$types';

type ImageDetailData = {
	envId: string;
	imageId: string;
	image: ImageDetailSummary & { repo: string; tag: string };
	repository: string | null;
	repositoryCopies: ImageRepositoryCopy[];
};

export const load: PageLoad = async ({ params, parent, url }): Promise<ImageDetailData> => {
	const [{ queryClient }, envId] = await Promise.all([parent(), resolveDetailEnvironmentId(url)]);

	const operationResult = await tryCatch(
		Promise.all([
			queryClient.ensureQueryData({ ...imageDetailQuery(envId, params.imageId), revalidateIfStale: true }),
			featureStore.load([envId])
		])
	);
	if (operationResult.error !== null) {
		const err = operationResult.error;

		console.error('Failed to load image:', err);
		if ('status' in err && err.status === 404) {
			throw err;
		}
		throw error(500, err.message || m.failed_to_load_image_details());
	}

	const [image] = operationResult.data;
	if (!image) throw error(404, 'Image not found');

	const repository = url.searchParams.get('repo');
	const repositoryResult = repository
		? await tryCatch(
				queryClient.ensureQueryData(
					imageLibraryQuery(
						environmentStore.shownKey,
						readableEnvironments('images:list'),
						{
							pagination: { page: 1, limit: 1 }
						},
						repository
					)
				)
			)
		: null;
	const repositoryCopies = (repositoryResult?.data?.data[0]?.repositoryGroup?.copies ?? []).filter((copy) =>
		hasPermission('images:read', copy.environmentId)
	);
	return { envId, imageId: params.imageId, image, repository, repositoryCopies };
};
