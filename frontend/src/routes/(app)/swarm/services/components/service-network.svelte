<script lang="ts">
	import { resolve } from '$app/paths';

	import RelatedResourceTable from '#lib/components/resource-detail/related-resource-table.svelte';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { GlobeIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { SwarmServiceNetworkDetail, SwarmServicePort } from '#lib/types/swarm.js';
	import type { ServiceNetworkAttachment, ServiceVirtualIP } from '#lib/types/ui/swarm.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { environmentHref } from '#lib/utils/environments.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';

	interface Props {
		ports: SwarmServicePort[];
		environmentId: string;
		networks: ServiceNetworkAttachment[];
		virtualIPs: ServiceVirtualIP[];
		networkDetails: Record<string, SwarmServiceNetworkDetail>;
	}

	let { ports, networks, virtualIPs, networkDetails, environmentId }: Props = $props();
	const canReadNetwork = $derived(hasPermission('networks:read', environmentId));

	function formatPort(port: SwarmServicePort): string {
		const protocol = port.protocol || 'tcp';
		const target = port.targetPort || 0;
		const published = port.publishedPort || 0;
		const mode = port.publishMode || '';
		if (published) {
			return `${published}:${target}/${protocol}${mode ? ` (${mode})` : ''}`;
		}
		return `${target}/${protocol}`;
	}

	const rows = $derived.by(() => {
		const attachments = new Map(networks.map((network) => [network.target, network.aliases]));
		const ips = new Map(virtualIPs.flatMap((vip) => (vip.networkID && vip.addr ? [[vip.networkID, vip.addr] as const] : [])));
		const ids = new Set([...attachments.keys(), ...ips.keys()]);
		return [...ids].map((id) => {
			const info = networkDetails[id];
			const aliases = attachments.get(id) ?? [];
			const config = info?.configNetwork;
			const addresses = [...(config?.ipv4Configs ?? []), ...(config?.ipv6Configs ?? [])].map(({ subnet, gateway, ipRange }) =>
				[subnet, gateway, ipRange].filter(Boolean).join(' / ')
			);
			const flags = [
				info?.internal && m.internal(),
				info?.attachable && m.attachable(),
				info?.ingress && m.ingress(),
				info?.configOnly && m.config_only(),
				config?.name ?? info?.configFrom,
				config?.driver,
				config?.scope,
				config?.options?.['parent'],
				...addresses
			]
				.filter(Boolean)
				.join(' · ');
			return {
				id,
				name: info?.name ?? aliases[0] ?? id.slice(0, 12),
				values: [info?.driver, info?.scope, ips.get(id), aliases.join(', '), flags],
				href:
					info?.id && canReadNetwork
						? resourceDetailHref(environmentHref(resolve('/(app)/networks/[networkId]', { networkId: info.id }), environmentId))
						: undefined
			};
		});
	});
</script>

<div class="space-y-6">
	<Card.Root>
		<Card.Header icon={GlobeIcon}>
			<div class="flex flex-col space-y-1.5">
				<Card.Title>
					<h2>{m.common_port_mappings()}</h2>
				</Card.Title>
			</div>
		</Card.Header>
		<Card.Content>
			{#if ports.length > 0}
				<div class="flex flex-wrap gap-2">
					{#each ports as port (`${port.publishedPort ?? 'internal'}:${port.targetPort}/${port.protocol}:${port.publishMode ?? ''}`)}
						<Badge variant="gray">{formatPort(port)}</Badge>
					{/each}
				</div>
			{:else}
				<div class="rounded-lg border border-dashed py-12 text-center text-muted-foreground">
					<div class="text-sm">{m.containers_no_ports()}</div>
				</div>
			{/if}
		</Card.Content>
	</Card.Root>

	<RelatedResourceTable
		columns={[
			m.common_driver(),
			m.common_scope(),
			m.networks_service_vip_label(),
			m.containers_aliases(),
			m.common_configuration()
		]}
		{rows}
		empty={m.containers_no_networks_connected()}
	/>
</div>
