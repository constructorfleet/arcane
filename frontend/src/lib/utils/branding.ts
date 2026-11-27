import settingsStore from '#lib/stores/config-store.svelte.js';

export function customLogoHash(full = false): string {
	return (full ? settingsStore.current?.customLogoFullHash : settingsStore.current?.customLogoHash) ?? '';
}
