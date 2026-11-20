import { m } from '#lib/paraglide/messages.js';
import { queryKeys } from '#lib/query/query-keys.js';
import { containerRegistryService } from '#lib/services/container-registry-service.js';
import type { Paginated } from '#lib/types/base.js';
import type { ContainerRegistryNamespace } from '#lib/types/containerregistry.js';
import type { ContainerRegistryRepositoryRow, ContainerRegistryRepositoryTagRow } from '#lib/types/ui/containerregistry.js';
import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
import { extractApiErrorMessage, throwPageLoadError } from '#lib/utils/api.js';
import { tryCatch } from '#lib/utils/try-catch.js';

import type { PageLoad } from './$types';

const catalogPageSize = 50;
const tagsPageSize = 50;

export const load: PageLoad = async ({ params, parent, url }) => {
	const { queryClient } = await parent();

	const registryResult = await tryCatch(
		queryClient.query({
			queryKey: queryKeys.containerRegistries.detail(params.id),
			queryFn: () => containerRegistryService.getRegistry(params.id)
		})
	);
	if (registryResult.error !== null) {
		throwPageLoadError(registryResult.error, m.common_load_failed({ resource: m.resource_registry() }));
	}

	const repository = url.searchParams.get('repository')?.trim() ?? '';
	const namespace = url.searchParams.get('namespace')?.trim() ?? '';
	const allRepositories = url.searchParams.get('view') === 'repositories';
	const search = url.searchParams.get('search')?.trim() ?? '';
	const page = Math.max(1, Number(url.searchParams.get('page')) || 1);

	let namespaces: Paginated<ContainerRegistryNamespace> | null = null;
	let namespaceError: string | null = null;
	let catalog: Paginated<ContainerRegistryRepositoryRow> | null = null;
	let catalogError: string | null = null;
	const catalogRequest: SearchPaginationSortRequest = {
		search,
		pagination: { page, limit: catalogPageSize },
		sort: { column: 'name', direction: 'asc' }
	};
	if (!repository && !namespace && !allRepositories) {
		const result = await tryCatch(
			queryClient.query({
				queryKey: queryKeys.containerRegistries.namespaces(params.id, catalogRequest),
				queryFn: () => containerRegistryService.getNamespaces(params.id, catalogRequest),
				staleTime: 30_000
			})
		);
		if (result.error === null) {
			namespaces = result.data;
		} else {
			namespaceError = extractApiErrorMessage(result.error);
		}
	}
	if (!repository && (!namespaces || (namespaces.pagination.grandTotalItems ?? namespaces.pagination.totalItems) === 0)) {
		const result = await tryCatch(
			queryClient.query({
				queryKey: queryKeys.containerRegistries.repositories(params.id, catalogRequest, namespace),
				queryFn: () => containerRegistryService.getRepositories(params.id, catalogRequest, namespace),
				staleTime: 30_000
			})
		);
		if (result.error === null) {
			catalog = result.data;
			namespaceError = null;
		} else {
			catalogError = extractApiErrorMessage(result.error);
		}
	}
	const tagsRequest: SearchPaginationSortRequest = {
		search,
		pagination: { page, limit: tagsPageSize },
		sort: { column: 'name', direction: 'asc' }
	};

	let tags: Paginated<ContainerRegistryRepositoryTagRow> = {
		data: [],
		pagination: { totalPages: 0, totalItems: 0, currentPage: 1, itemsPerPage: tagsPageSize }
	};
	let tagsError: string | null = null;
	if (repository) {
		const tagsResult = await tryCatch(
			queryClient.query({
				queryKey: queryKeys.containerRegistries.tags(params.id, repository, tagsRequest),
				queryFn: () => containerRegistryService.getTags(params.id, repository, tagsRequest)
			})
		);
		if (tagsResult.error === null) {
			tags = tagsResult.data;
		} else {
			tagsError = extractApiErrorMessage(tagsResult.error);
		}
	}

	return {
		registry: registryResult.data,
		namespace,
		namespaces,
		namespaceError,
		allRepositories,
		catalog,
		catalogError,
		repository,
		search,
		tags,
		tagsError
	};
};
