import type { ImageSummary } from '#lib/types/image.js';
import type { EnvironmentRow } from '#lib/utils/environments.js';

export type ImageActionTarget = Pick<
	EnvironmentRow<ImageSummary>,
	'id' | 'repo' | 'tag' | 'repoTags' | 'environmentId' | 'environmentName' | 'environmentTagged' | 'inUse'
> & {
	repoDigests?: string[];
	pinnedReferences?: string[];
	repositoryGroup?: undefined;
	hardened?: ImageSummary['hardened'];
};

export function imageActionTargets(image: EnvironmentRow<ImageSummary>): ImageActionTarget[] {
	if (!image.repositoryGroup) return [{ ...image, repositoryGroup: undefined }];
	return image.repositoryGroup.copies.map((copy) => ({
		id: copy.id,
		inUse: copy.inUse,
		repo: image.repo,
		tag: image.tag,
		repoTags: copy.tags ?? [],
		repoDigests: copy.repoDigests,
		pinnedReferences: copy.pinnedReferences,
		hardened: copy.hardened,
		environmentId: copy.environmentId,
		environmentName: copy.environmentName ?? copy.environmentId,
		environmentTagged: true
	}));
}
