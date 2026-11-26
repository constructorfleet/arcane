import type { RegistryScanFinding } from '#lib/types/containerregistry.js';

export function compareRegistryFindings(before: RegistryScanFinding[], after: RegistryScanFinding[]) {
	const findingKeys = (findings: RegistryScanFinding[]) =>
		new Set(findings.map(({ id, package: packageName }) => JSON.stringify([id, packageName])));
	const previous = findingKeys(before);
	const candidate = findingKeys(after);
	const persisting = [...previous].filter((key) => candidate.has(key)).length;
	return { removed: previous.size - persisting, persisting, added: candidate.size - persisting };
}
