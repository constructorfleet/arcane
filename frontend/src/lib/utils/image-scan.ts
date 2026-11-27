export function inUseImageTargets<T extends { id: string; inUse: boolean; artifactKey?: string }>(copies: T[]): T[] {
	const targets = new Map<string, T>();
	for (const copy of copies) {
		const key = copy.artifactKey ?? copy.id;
		if (copy.inUse && !targets.has(key)) targets.set(key, copy);
	}
	return [...targets.values()];
}
