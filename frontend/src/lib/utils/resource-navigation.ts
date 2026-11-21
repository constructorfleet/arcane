import { page } from '$app/state';

import { contextualResourceHref } from '#lib/utils/resource-parent.js';

export { previousResourcePath, resourceTablePage } from '#lib/utils/resource-parent.js';

export function resourceDetailHref(href: string): string {
	return contextualResourceHref(href, (page.shallow?.url ?? page.url).href);
}

export function resourceTableHref(href: string, tablePage: number | undefined): string {
	const source = new URL((page.shallow?.url ?? page.url).href);
	if (tablePage && tablePage > 1) source.searchParams.set('tablePage', String(tablePage));
	else source.searchParams.delete('tablePage');
	return contextualResourceHref(href, source.href);
}
