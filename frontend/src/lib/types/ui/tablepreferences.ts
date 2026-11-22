import type { ColumnFiltersState } from '@tanstack/table-core';

import type { FilterMap } from './pagination';

export type PersistedPreferencesSnapshot = {
	hiddenColumns: string[];
	visibleColumns: string[];
	restoredFilters: ColumnFiltersState;
	filtersMap: FilterMap;
	search: string;
	sort?: { column: string; direction: 'asc' | 'desc' };
	limit: number;
	mobileVisibility?: string[];
	viewMode?: 'list' | 'grid';
	customSettings?: Record<string, unknown>;
};
