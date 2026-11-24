<script lang="ts">
	import { DetailSection, KeyValueCard, KeyValueGrid } from '#lib/components/resource-detail/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as CodeBlock from '#lib/components/ui/code-block/index.js';
	import { useHighlightedCode } from '#lib/hooks/use-highlighted-code.svelte.js';
	import { InfoIcon, TagIcon, TerminalIcon } from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { PodInspect } from '#lib/types/podman.js';
	import { formatDateTime } from '#lib/utils/formatting.js';
	import { podmanStateVariant } from '#lib/utils/podman.js';

	interface Props {
		pod: PodInspect;
	}

	let { pod }: Props = $props();

	const metaItems = $derived([
		{ label: m.common_created(), value: formatDateTime(pod.created) || pod.created },
		{ label: m.containers(), value: String(pod.numContainers) },
		{ label: m.podman_pod_infra(), value: pod.createInfra ? m.common_yes() : m.common_no() },
		...(pod.restartPolicy ? [{ label: m.common_restart_policy(), value: pod.restartPolicy }] : [])
	]);

	const details = $derived(
		[
			{ label: m.common_id(), value: pod.id },
			{ label: m.hostname(), value: pod.hostname },
			{ label: m.podman_pod_namespace(), value: pod.namespace },
			{ label: m.podman_pod_exit_policy(), value: pod.exitPolicy },
			{ label: m.podman_pod_cgroup_parent(), value: pod.cgroupParent },
			{ label: m.podman_pod_cgroup_path(), value: pod.cgroupPath },
			{ label: m.podman_pod_shared_namespaces(), value: pod.sharedNamespaces?.join(', ') },
			{ label: m.podman_pod_pid_namespace(), value: pod.infraConfig?.pidNs },
			{ label: m.podman_pod_user_namespace(), value: pod.infraConfig?.userNs },
			{ label: m.podman_pod_uts_namespace(), value: pod.infraConfig?.utsNs },
			{ label: m.podman_pod_infra_container_id(), value: pod.infraContainerId }
		].flatMap((entry) => (entry.value ? [{ label: entry.label, value: entry.value }] : []))
	);

	const labels = $derived(Object.entries(pod.labels ?? {}));
	const createCommand = $derived(pod.createCommand?.join(' ') ?? '');
	const createCommandLines = useHighlightedCode(
		() => createCommand,
		() => 'shell'
	);
</script>

<div class="space-y-6">
	<DetailSection title={m.common_overview()} icon={InfoIcon} class="border-t-0 pt-0">
		<KeyValueGrid>
			<KeyValueCard label={m.common_status()} valueFormat="custom">
				<Badge variant={podmanStateVariant(pod.state)}>{pod.state}</Badge>
			</KeyValueCard>
			{#each metaItems as entry (entry.label)}
				<KeyValueCard label={entry.label} valueFormat="text">{entry.value}</KeyValueCard>
			{/each}
			{#each details as entry (entry.label)}
				<KeyValueCard label={entry.label} valueTitle={m.common_click_to_select()}>{entry.value}</KeyValueCard>
			{/each}
		</KeyValueGrid>
	</DetailSection>

	{#if labels.length > 0}
		<DetailSection title={m.common_labels()} icon={TagIcon}>
			<KeyValueGrid>
				{#each labels as [key, value] (key)}
					<KeyValueCard label={key} valueTitle={m.common_click_to_select()}>{value}</KeyValueCard>
				{/each}
			</KeyValueGrid>
		</DetailSection>
	{/if}

	{#if createCommand}
		<DetailSection title={m.podman_pod_create_command()} icon={TerminalIcon}>
			<CodeBlock.Root size="sm">
				<CodeBlock.Content wrap>
					{#each createCommandLines.current as line, index (index)}
						<CodeBlock.Line code={line} />
					{/each}
				</CodeBlock.Content>
			</CodeBlock.Root>
		</DetailSection>
	{/if}
</div>
