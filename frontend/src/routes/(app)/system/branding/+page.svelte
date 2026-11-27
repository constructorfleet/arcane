<script lang="ts">
	import { mode } from 'mode-watcher';

	import SettingsSection from '#lib/components/settings/settings-section.svelte';
	import { SettingsPageLayout } from '#lib/layouts/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import settingsStore from '#lib/stores/config-store.svelte.js';
	import { customLogoHash } from '#lib/utils/branding.js';
	import { getApplicationLogo } from '#lib/utils/docker.js';

	import BrandImageTile, { BRAND_IMAGE_MAX_SIZE_MB } from './components/brand-image-tile.svelte';

	const logoTypes = ['image/png', 'image/jpeg', 'image/webp', 'image/svg+xml'];

	const mark = $derived(customLogoHash());
	const full = $derived(customLogoHash(true));
	const faviconHash = $derived(settingsStore.current?.customFaviconHash ?? '');
	const pwaIconHash = $derived(settingsStore.current?.customPwaIconHash ?? '');

	const slots = $derived([
		{
			slot: 'mark' as const,
			label: m.icon_mark(),
			hint: m.logo_image_types(),
			previewUrl: getApplicationLogo(false, false, { custom: mark }),
			custom: !!mark,
			accept: logoTypes
		},
		{
			slot: 'full' as const,
			label: m.full_logo(),
			hint: m.logo_image_types(),
			previewUrl: getApplicationLogo(true, mode.current === 'dark', { custom: full }),
			custom: !!full,
			accept: logoTypes
		},
		{
			slot: 'favicon' as const,
			label: m.favicon(),
			hint: m.favicon_image_types(),
			previewUrl: `/api/app/images/favicon?v=${faviconHash}`,
			custom: !!faviconHash,
			accept: ['image/png', 'image/svg+xml', 'image/x-icon', 'image/vnd.microsoft.icon', '.ico']
		},
		{
			slot: 'pwa' as const,
			label: m.app_icon(),
			hint: m.app_icon_image_types(),
			previewUrl: `/api/app/images/pwa/icon-192x192.png?v=${pwaIconHash}`,
			custom: !!pwaIconHash,
			accept: ['image/png'],
			square: { min: 512, error: m.app_icon_too_small() }
		}
	]);
</script>

<SettingsPageLayout
	title={m.branding()}
	description={m.branding_description()}
	pageType="management"
	showReadOnlyTag={settingsStore.readOnly}
>
	{#snippet mainContent()}
		<SettingsSection
			title={m.logos_and_icons()}
			description={m.logos_and_icons_description({ maxSizeMb: BRAND_IMAGE_MAX_SIZE_MB })}
			variant="plain"
		>
			<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
				{#each slots as slot (slot.slot)}
					<BrandImageTile {...slot} />
				{/each}
			</div>
		</SettingsSection>
	{/snippet}
</SettingsPageLayout>
