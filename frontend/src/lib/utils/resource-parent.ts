function isLocalPath(value: string | null): value is string {
	return !!value && value.startsWith('/') && !value.startsWith('//') && !value.startsWith('/\\');
}

const previousParam = 'prev';
const maximumPreviousLength = 2048;

export function previousResourcePath(current: string, fallback: string): string {
	const url = new URL(current);
	const previous = url.searchParams.get(previousParam);
	if (!isLocalPath(previous) || previous.length > maximumPreviousLength) return fallback;
	const parent = new URL(previous, current);
	const href = parent.pathname + parent.search + parent.hash;
	if (parent.origin !== url.origin || parent.pathname === url.pathname || !isLocalPath(href)) return fallback;
	return href;
}

export function contextualResourceHref(href: string, from: string): string {
	const source = new URL(from);
	const target = new URL(href, from);
	if (target.origin !== source.origin || !isLocalPath(target.pathname)) return href;
	const previous =
		target.pathname === source.pathname ? previousResourcePath(from, '') : source.pathname + source.search + source.hash;
	if (previous && previous.length <= maximumPreviousLength) target.searchParams.set(previousParam, previous);
	return target.pathname + target.search + target.hash;
}

export function resourceTablePage(value: string | null): number {
	const pageNumber = Number(value);
	return Number.isSafeInteger(pageNumber) && pageNumber > 0 && pageNumber <= 1_000_000 ? pageNumber : 1;
}
