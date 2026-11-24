<script lang="ts">
	import { toast } from 'svelte-sonner';

	import LogControls from '#lib/components/logs/log-controls.svelte';
	import LogViewer from '#lib/components/logs/log-viewer.svelte';
	import { UseLogPreferences } from '#lib/hooks/use-log-preferences.svelte.js';
	import { m } from '#lib/paraglide/messages.js';
	import { containerService } from '#lib/services/container-service.js';
	import { refreshLogViewerStream, startLogViewerStream, stopLogViewerStream } from '#lib/utils/log-viewer.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	let {
		containerId,
		environmentId,
		autoScroll = $bindable(),
		onStart,
		onStop
	}: {
		containerId: string | undefined;
		environmentId: string;
		autoScroll: boolean;
		onStart?: () => void;
		onStop?: () => void;
	} = $props();

	let isStreaming = $state(false);
	let viewer = $state<ReturnType<typeof LogViewer>>();
	const preferences = new UseLogPreferences();
	let logSearchTerm = $state('');
	let hasAutoStarted = $state(false);
	function handleStart() {
		startLogViewerStream(viewer);
	}

	function handleStop() {
		stopLogViewerStream(viewer);
	}

	async function handleRefresh() {
		await refreshLogViewerStream(viewer);
	}

	async function handleDownload() {
		if (!containerId) return;
		const result = await tryCatch(containerService.downloadContainerLogs(containerId, environmentId));
		if (result.error !== null) {
			toast.error(m.common_download_error(), { description: result.error.message });
		}
	}

	function handleStreamStart() {
		isStreaming = true;
		onStart?.();
	}

	function handleStreamStop() {
		isStreaming = false;
		onStop?.();
	}

	$effect(() => {
		if (preferences.autoStartLogs && !hasAutoStarted && !isStreaming && containerId && viewer) {
			hasAutoStarted = true;
			handleStart();
		}
	});
</script>

<div class="flex h-full min-h-0 flex-col gap-3">
	<div class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl bg-table-head">
		<div class="flex h-11 shrink-0 items-center gap-2 pr-1.5 pl-3">
			<h2 class="text-sm font-medium">{m.common_logs()}</h2>
			<div class="ml-auto min-w-0">
				<LogControls
					bind:searchTerm={logSearchTerm}
					bind:autoScroll
					{preferences}
					{isStreaming}
					disabled={!containerId}
					onStart={handleStart}
					onStop={handleStop}
					onRefresh={handleRefresh}
					onDownload={handleDownload}
				/>
			</div>
		</div>
		<LogViewer
			class="min-h-0 flex-1"
			searchTerm={logSearchTerm}
			bind:this={viewer}
			bind:autoScroll
			type="container"
			{containerId}
			{environmentId}
			bind:showParsedJson={preferences.showParsedJson}
			showStreamLabels={preferences.showStreamLabels}
			tailLines={preferences.tailLines}
			maxLines={500}
			showTimestamps={preferences.showTimestamps}
			groupAdjacentLines={true}
			height="100%"
			onStart={handleStreamStart}
			onStop={handleStreamStop}
		/>
	</div>
</div>
