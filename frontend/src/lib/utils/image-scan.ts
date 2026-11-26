export function inUseImageTargets<T extends { id: string; inUse: boolean }>(copies: T[]): T[] {
	const targets = new Map<string, T>();
	for (const copy of copies) {
		if (copy.inUse && !targets.has(copy.id)) targets.set(copy.id, copy);
	}
	return [...targets.values()];
}
