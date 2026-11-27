<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';
	import { onMount, untrack } from 'svelte';
	import { toast } from 'svelte-sonner';

	import { ActionButtonGroup } from '#lib/components/action-button-group/index.js';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import EditorTabStrip from '#lib/components/editor-tab-strip.svelte';
	import ResizableSplit from '#lib/components/resizable-split.svelte';
	import { KeyValueCard } from '#lib/components/resource-detail/index.js';
	import SwarmKvScopedPanel from '#lib/components/swarm/swarm-kv-scoped-panel.svelte';
	import { type TabItem } from '#lib/components/tab-bar/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as CodeBlock from '#lib/components/ui/code-block/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import WorkspaceFileTreePanel from '#lib/components/workspace-file-tree-panel.svelte';
	import { useHighlightedCode } from '#lib/hooks/use-highlighted-code.svelte.js';
	import { useUrlTab } from '#lib/hooks/use-url-tab.svelte.js';
	import {
		ContainersIcon,
		JobsIcon,
		LayersIcon,
		TrashIcon,
		EditIcon,
		FileTextIcon,
		StopIcon,
		PlayIcon,
		NetworksIcon,
		VolumesIcon,
		InspectIcon
	} from '#lib/icons/index.js';
	import type { ActionButton } from '#lib/layouts/index.js';
	import TabbedPageLayout from '#lib/layouts/tabbed-page-layout.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { swarmStackQuery, swarmStackServiceListQuery, swarmStackTaskListQuery } from '#lib/query/swarm.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import type { SwarmStackSource } from '#lib/types/swarm.js';
	import { handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { hasPermission } from '#lib/utils/auth.js';
	import { formatRelativeTime } from '#lib/utils/formatting.js';
	import { resourceDetailHref } from '#lib/utils/resource-navigation.js';
	import { declaredExternalResources } from '#lib/utils/swarm-resource-relations.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	import SwarmServicesTable from '../../services/components/services-table.svelte';
	import SwarmTasksTable from '../../tasks/components/tasks-table.svelte';
	import StackRelatedResources from '../components/stack-related-resources.svelte';
	import { bringDownStack, bringUpStack } from '../stack-actions.js';

	let { data } = $props();

	let source = $derived<SwarmStackSource | null>(data.source);
	let sourceState = $derived<'loading' | 'available' | 'missing' | 'forbidden' | 'error'>(data.sourceState);
	let servicesRequestOptions = $state(untrack(() => data.servicesRequestOptions));
	let tasksRequestOptions = $state(untrack(() => data.tasksRequestOptions));

	const stackQuery = createQuery(() => ({
		...swarmStackQuery(data.envId, data.stackName),
		placeholderData: (previous, query) => (query?.queryKey[1] === data.envId ? previous : undefined)
	}));
	const stack = $derived(stackQuery.data ?? data.stack);

	let selectedSourceFile = $state('compose');
	let openSourceTabs = $state<string[]>(['compose']);
	let sourceTreeWidth = $state<number | null>(null);
	const sourceOpenTabs = $derived(openSourceTabs.length > 0 ? openSourceTabs : ['compose']);
	const activeSourceTab = $derived(
		sourceOpenTabs.includes(selectedSourceFile) ? selectedSourceFile : (sourceOpenTabs[0] ?? 'compose')
	);
	const sourceTabs = $derived(
		sourceOpenTabs.map((key) => ({
			key,
			label: key === 'compose' ? 'compose.yaml' : '.env',
			title: key === 'compose' ? 'compose.yaml' : '.env',
			iconClass: key === 'compose' ? 'text-info' : 'text-success',
			pending: false
		}))
	);
	const sourceLines = useHighlightedCode(
		() => (activeSourceTab === 'compose' ? (source?.composeContent ?? '') : (source?.envContent ?? '')),
		() => (activeSourceTab === 'compose' ? 'yaml' : 'env')
	);
	const sourceWorkspaceLeadingRows = [
		{ key: 'compose', label: 'compose.yaml', iconClass: 'text-info', locked: true },
		{ key: 'env', label: '.env', iconClass: 'text-success', locked: true }
	];

	function openSourceTab(key: string) {
		if (!openSourceTabs.includes(key)) {
			openSourceTabs = [...openSourceTabs, key];
		}
		selectedSourceFile = key;
	}

	function closeSourceTab(key: string) {
		const index = sourceOpenTabs.indexOf(key);
		const remaining = sourceOpenTabs.filter((tab) => tab !== key);
		openSourceTabs = openSourceTabs.filter((tab) => tab !== key);
		if (selectedSourceFile === key) {
			selectedSourceFile = remaining[Math.min(Math.max(index - 1, 0), remaining.length - 1)] ?? 'compose';
		}
	}
	type StackTab = 'overview' | 'services' | 'tasks' | 'source' | 'networks' | 'volumes' | 'secrets' | 'configs';
	const isLoading = $state({ refresh: false, remove: false, down: false, up: false });
	const isMutating = $derived(isLoading.remove || isLoading.down || isLoading.up);

	const stackName = $derived(stack?.name ?? data.stackName);
	const hasLiveStack = $derived((stack?.services ?? 0) > 0);
	const envId = $derived(data.envId);
	const canReadStackSource = $derived(hasPermission('swarm-stacks:read', envId));
	const canViewSource = $derived(canReadStackSource && sourceState !== 'forbidden');
	const tabItems = $derived<TabItem[]>([
		{ value: 'overview', label: m.common_overview(), icon: InspectIcon },
		{ value: 'services', label: m.services(), icon: ContainersIcon },
		{ value: 'tasks', label: m.tasks(), icon: JobsIcon },
		...(hasPermission('networks:list', envId)
			? [{ value: 'networks', label: m.resource_networks_cap(), icon: NetworksIcon }]
			: []),
		...(hasPermission('volumes:list', envId) ? [{ value: 'volumes', label: m.resource_volumes_cap(), icon: VolumesIcon }] : []),
		...(hasPermission('swarm-secrets:list', envId)
			? [{ value: 'secrets', label: m.swarm_secrets_title(), icon: FileTextIcon }]
			: []),
		...(hasPermission('swarm-configs:list', envId)
			? [{ value: 'configs', label: m.swarm_configs_title(), icon: FileTextIcon }]
			: []),
		...(canViewSource ? [{ value: 'source', label: m.source(), icon: FileTextIcon }] : [])
	]);
	const urlTab = useUrlTab<StackTab>({
		history: true,
		validTabs: () => tabItems.map((tab) => tab.value as StackTab),
		defaultTab: () => 'overview'
	});
	const selectedTab = $derived(urlTab.value);
	const servicesQuery = createQuery(() => ({
		...swarmStackServiceListQuery(data.environment, data.stackName, servicesRequestOptions),
		enabled: selectedTab === 'services',
		placeholderData: (previous, query) => (query?.queryKey[1] === data.envId ? previous : undefined)
	}));
	const tasksQuery = createQuery(() => ({
		...swarmStackTaskListQuery(data.environment, data.stackName, tasksRequestOptions),
		enabled: selectedTab === 'tasks',
		placeholderData: (previous, query) => (query?.queryKey[1] === data.envId ? previous : undefined)
	}));
	const services = $derived(servicesQuery.data ?? data.services);
	const tasks = $derived(tasksQuery.data ?? data.tasks);
	const relationServices = createQuery(() => ({
		queryKey: ['swarm', data.envId, 'stack-service-references', data.stackName],
		enabled: ['networks', 'volumes', 'secrets', 'configs'].includes(selectedTab),
		queryFn: () => swarmService.getStackServices(data.stackName, { pagination: { page: 1, limit: -1 } }, data.envId)
	}));

	async function refreshSource(showErrorToast = false) {
		if (!canReadStackSource) {
			source = null;
			sourceState = 'forbidden';
			return;
		}
		const operationResult = await tryCatch(
			(async () => {
				source = await swarmService.getStackSource(stackName, envId);
				sourceState = 'available';
			})()
		);
		if (operationResult.error !== null) {
			const err = operationResult.error;

			if ('status' in err && err.status === 404) {
				source = null;
				sourceState = 'missing';
				return;
			}
			if ('status' in err && err.status === 403) {
				source = null;
				sourceState = 'forbidden';
				return;
			}

			source = null;
			sourceState = 'error';
			if (showErrorToast) {
				toast.error(m.common_refresh_failed({ resource: `saved source (${stackName})` }));
			}
		}
	}

	async function refresh() {
		isLoading.refresh = true;
		try {
			const [stackResult, servicesResult, tasksResult] = await Promise.all([
				stackQuery.refetch(),
				servicesQuery.refetch(),
				tasksQuery.refetch()
			]);

			if (stackResult.isError) {
				toast.error(m.common_refresh_failed({ resource: `${m.swarm_stack()} "${stackName}"` }));
			}

			if (servicesResult.isError) {
				toast.error(m.common_refresh_failed({ resource: `${m.services()} (${stackName})` }));
			}

			if (tasksResult.isError) {
				toast.error(m.common_refresh_failed({ resource: `${m.tasks()} (${stackName})` }));
			}
			if (canReadStackSource) {
				await refreshSource(true);
			}
		} finally {
			isLoading.refresh = false;
		}
	}

	onMount(() => {
		if (canReadStackSource) {
			void refreshSource();
		}
	});

	function handleDelete() {
		openConfirmDialog({
			title: m.common_delete_title({ resource: m.swarm_stack() }),
			message: m.common_delete_confirm({ resource: m.swarm_stack() }),
			confirm: {
				label: m.common_delete(),
				destructive: true,
				action: async () => {
					isLoading.remove = true;
					await handleApiResultWithCallbacks({
						result: await tryCatch(swarmService.removeStack(stackName, envId)),
						message: m.common_delete_failed({ resource: `${m.swarm_stack()} "${stackName}"` }),
						setLoadingState: (v) => (isLoading.remove = v),
						onSuccess: async () => {
							toast.success(m.common_delete_success({ resource: `${m.swarm_stack()} "${stackName}"` }));
							goto(resolve('/(app)/swarm/stacks'));
						}
					});
				}
			}
		});
	}

	const actionButtons: ActionButton[] = $derived([
		...(hasPermission('swarm-stacks:update', envId)
			? [
					{
						id: 'edit',
						action: 'base' as const,
						placement: 'primary' as const,
						label: m.common_edit(),
						icon: EditIcon,
						onclick: () =>
							goto(
								resolve(`/(app)/swarm/stacks/new?fromStack=${encodeURIComponent(stackName)}&env=${encodeURIComponent(envId)}`)
							),
						disabled: isMutating
					}
				]
			: []),
		...(sourceState === 'available' && hasPermission('swarm-stacks:update', envId)
			? [
					{
						id: 'up',
						action: 'start' as const,
						label: m.deploy_saved_source(),
						icon: PlayIcon,
						onclick: () =>
							bringUpStack({
								name: stackName,
								environmentId: envId,
								setLoading: (loading) => (isLoading.up = loading),
								onSuccess: refresh
							}),
						loading: isLoading.up,
						disabled: isMutating
					}
				]
			: []),
		...(hasLiveStack && hasPermission('swarm-stacks:update', envId)
			? [
					{
						id: 'down',
						action: 'stop' as const,
						label: m.common_down(),
						icon: StopIcon,
						onclick: () =>
							bringDownStack({
								name: stackName,
								environmentId: envId,
								setLoading: (loading) => (isLoading.down = loading),
								onSuccess: refresh
							}),
						loading: isLoading.down,
						disabled: isMutating
					}
				]
			: []),
		...(hasPermission('swarm-stacks:delete', envId)
			? [
					{
						id: 'remove',
						action: 'remove' as const,
						group: 'danger' as const,
						destructive: true,
						label: m.common_delete(),
						icon: TrashIcon,
						onclick: handleDelete,
						loading: isLoading.remove,
						disabled: isMutating
					}
				]
			: []),
		{
			id: 'refresh',
			action: 'refresh',
			placement: 'secondary',
			iconOnly: true,
			label: m.common_refresh(),
			onclick: refresh,
			loading: isLoading.refresh,
			disabled: isLoading.refresh
		}
	]);
</script>

<TabbedPageLayout backUrl="/swarm/stacks" {tabItems} {selectedTab} onTabChange={urlTab.select}>
	{#snippet headerInfo()}
		<div class="flex min-h-9 min-w-0 items-center gap-2">
			<LayersIcon class="size-5 shrink-0 text-primary" />
			<h1 class="min-w-0 truncate font-semibold" title={stackName}>{stackName}</h1>
		</div>
	{/snippet}
	{#snippet headerActions()}
		<ActionButtonGroup buttons={actionButtons} />
	{/snippet}
	{#snippet tabContent()}
		<div class="flex min-h-0 flex-1 flex-col gap-4">
			{#if !hasLiveStack && canViewSource}
				<Card.Root>
					<Card.Content>
						<p class="text-sm">{m.swarm_stacks_not_deployed_files_found()}</p>
					</Card.Content>
				</Card.Root>
			{/if}

			<Tabs.Content value="services" layout="table">
				{#if services}
					<SwarmServicesTable
						{services}
						invalidateKey={queryKeys.swarm.stack(envId, data.stackName)}
						bind:requestOptions={servicesRequestOptions}
						onRetry={() => servicesQuery.refetch()}
						persistKey={`arcane-swarm-stack-services-table-${stackName}`}
						showEnvironment={false}
						showStack={false}
					/>
				{:else}<p role="status">{servicesQuery.error?.message ?? m.common_loading()}</p>{/if}
			</Tabs.Content>
			<Tabs.Content value="overview">
				<Card.Root
					><Card.Header><Card.Title>{m.details()}</Card.Title></Card.Header><Card.Content
						><div class="grid gap-4 sm:grid-cols-2">
							<KeyValueCard label={m.common_name()}>{stackName}</KeyValueCard>
							<KeyValueCard label={m.namespace()}>{stack?.namespace ?? stackName}</KeyValueCard>
							<KeyValueCard label={m.services()} valueFormat="custom"
								><a
									class="text-primary-text hover:underline"
									href={resourceDetailHref(
										resolve(`/(app)/swarm/stacks/[name]?env=${encodeURIComponent(envId)}&tab=services`, { name: stackName })
									)}>{stack?.services ?? m.unavailable()}</a
								></KeyValueCard
							>
							<KeyValueCard label={m.tasks()} valueFormat="custom"
								><a
									class="text-primary-text hover:underline"
									href={resourceDetailHref(
										resolve(`/(app)/swarm/stacks/[name]?env=${encodeURIComponent(envId)}&tab=tasks`, { name: stackName })
									)}>{stack?.runningTasks ?? m.unavailable()} / {stack?.desiredTasks ?? m.unavailable()}</a
								></KeyValueCard
							>
							<KeyValueCard label={m.source()}
								>{sourceState === 'available'
									? m.saved_compose()
									: sourceState === 'missing'
										? m.not_imported()
										: m.unavailable()}</KeyValueCard
							>
							<KeyValueCard label={m.common_created()}
								>{formatRelativeTime(stack?.createdAt ?? '') || m.unavailable()}</KeyValueCard
							>
							<KeyValueCard label={m.common_updated()}
								>{formatRelativeTime(stack?.updatedAt ?? '') || m.unavailable()}</KeyValueCard
							>
						</div></Card.Content
					></Card.Root
				>
			</Tabs.Content>
			{#each ['networks', 'volumes', 'secrets', 'configs'] as kind (kind)}
				{@const externalNames =
					sourceState === 'available' &&
					source &&
					(kind === 'networks' || kind === 'volumes' || kind === 'secrets' || kind === 'configs')
						? declaredExternalResources({ composeContent: source.composeContent, overrideContent: source.overrideContent, kind })
						: new Set<string>()}
				<Tabs.Content value={kind} layout="table"
					>{#if selectedTab === kind}
						{#if relationServices.data}
							{#if kind === 'secrets' || kind === 'configs'}<SwarmKvScopedPanel
									{kind}
									environmentId={envId}
									{stackName}
									{externalNames}
									references={relationServices.data.data.flatMap((service) => service[kind] ?? [])}
									incomplete={relationServices.data.data.some((service) => service[kind] === undefined)}
								/>
							{:else if kind === 'networks' || kind === 'volumes'}<StackRelatedResources
									{kind}
									environmentId={envId}
									{stackName}
									{externalNames}
									services={relationServices.data.data}
								/>{/if}
						{:else}<p role="status">{relationServices.error?.message ?? m.common_loading()}</p>{/if}
					{/if}</Tabs.Content
				>
			{/each}
			<Tabs.Content value="tasks" layout="table">
				{#if tasks}
					<SwarmTasksTable
						{tasks}
						bind:requestOptions={tasksRequestOptions}
						onRetry={() => tasksQuery.refetch()}
						persistKey={`arcane-swarm-stack-tasks-table-${stackName}`}
						showEnvironment={false}
					/>
				{:else}<p role="status">{tasksQuery.error?.message ?? m.common_loading()}</p>{/if}
			</Tabs.Content>
			<Tabs.Content value="source" class="flex min-h-96 flex-1 flex-col">
				{#if sourceState === 'available' && source}
					{@const stackSource = source}
					<div class="flex min-h-0 flex-1 flex-col">
						{#key `arcane.swarm.split:${stackName}:source`}
							<ResizableSplit
								class="min-h-0 flex-1"
								variant="gap"
								handleSize={12}
								firstClass="flex min-h-0 flex-col"
								secondClass="flex min-h-0 flex-col"
								bind:size={sourceTreeWidth}
								minSize={200}
								maxSize={480}
								minSecondSize={360}
								defaultRatio={0.2}
								stackBelow={1024}
								ariaLabel={m.compose_editor_resize_files_panel()}
								persistKey={`arcane.swarm.split:${stackName}:source`}
							>
								{#snippet first()}
									<WorkspaceFileTreePanel
										leadingRows={sourceWorkspaceLeadingRows}
										entries={[]}
										selectedFile={selectedSourceFile}
										onSelect={openSourceTab}
									/>
								{/snippet}

								{#snippet second()}
									<EditorTabStrip tabs={sourceTabs} activeKey={activeSourceTab} onSelect={openSourceTab} onClose={closeSourceTab}>
										<div class="relative min-h-0 flex-1">
											{#if activeSourceTab === 'compose' || stackSource.envContent?.length}
												<CodeBlock.Root variant="ghost" class="absolute inset-0">
													<CodeBlock.Content lineNumbers class="flex-1">
														{#each sourceLines.current as line, index (index)}
															<CodeBlock.Line code={line} />
														{/each}
													</CodeBlock.Content>
												</CodeBlock.Root>
											{:else}
												<div class="flex h-full items-center justify-center p-6 text-center text-sm text-muted-foreground">
													No saved `.env` file was stored for this stack.
												</div>
											{/if}
										</div>
									</EditorTabStrip>
								{/snippet}
							</ResizableSplit>
						{/key}
					</div>
				{:else if sourceState === 'loading'}
					<Card.Root>
						<Card.Content>
							<p class="text-center text-sm text-muted-foreground">{m.swarm_stack_source_loading()}</p>
						</Card.Content>
					</Card.Root>
				{:else if sourceState === 'missing'}
					<Card.Root>
						<Card.Content>
							<div class="space-y-2 text-sm">
								<p class="font-medium">{m.common_not_found_title({ resource: 'Saved source' })}</p>
								<p class="text-muted-foreground">
									{m.common_not_found_description({ resource: 'saved source' })}
								</p>
							</div>
						</Card.Content>
					</Card.Root>
				{:else if sourceState === 'error'}
					<Card.Root>
						<Card.Content>
							<div class="space-y-2 text-sm">
								<p class="font-medium">{m.common_action_failed()}</p>
								<p class="text-muted-foreground">{m.swarm_stack_source_load_error()}</p>
							</div>
						</Card.Content>
					</Card.Root>
				{/if}
			</Tabs.Content>
		</div>
	{/snippet}
</TabbedPageLayout>
