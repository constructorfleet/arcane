<script lang="ts">
	import { resolve } from '$app/paths';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';

	import ArcaneTable from '#lib/components/arcane-table/arcane-table.svelte';
	import DetailLinksMenu from '#lib/components/arcane-table/detail-links-menu.svelte';
	import type { ColumnSpec, MobileFieldVisibility } from '#lib/components/arcane-table/index.js';
	import { UniversalMobileCard } from '#lib/components/arcane-table/index.js';
	import RowActionsMenu from '#lib/components/arcane-table/row-actions-menu.svelte';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import {
		UsersIcon,
		EnvironmentsIcon,
		ArrowUpIcon,
		ArrowDownIcon,
		StopIcon,
		PauseIcon,
		PlayIcon,
		NetworksIcon,
		ContainersIcon,
		VolumesIcon,
		JobsIcon,
		TrashIcon,
		AgentConnectionIcon
	} from '#lib/icons/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { queryKeys } from '#lib/query/query-keys.js';
	import { swarmService } from '#lib/services/swarm-service.js';
	import { environmentStore } from '#lib/stores/environment.store.svelte.js';
	import type { SwarmNodeAgentDeployment, SwarmNode } from '#lib/types/swarm.js';
	import type { SearchPaginationSortRequest } from '#lib/types/ui/pagination.js';
	import type { SwarmNodesPaginatedResponse } from '#lib/types/ui/swarm.js';
	import { extractApiErrorMessage, handleApiResultWithCallbacks } from '#lib/utils/api.js';
	import { hasAnyPermission, hasPermission } from '#lib/utils/auth.js';
	import {
		environmentHref,
		getEnvironmentTypeLabel,
		type EnvironmentPage,
		type EnvironmentRow
	} from '#lib/utils/environments.js';
	import { capitalizeFirstLetter, formatRelativeTime } from '#lib/utils/formatting.js';
	import { resourceTableHref } from '#lib/utils/resource-navigation.js';
	import { tryCatch } from '#lib/utils/try-catch.js';

	import { getSwarmNodeAgentActionLabel, getSwarmNodeAgentLabel, getSwarmNodeAgentVariant } from '../agent-status';
	import SwarmNodeAgentDialog from './swarm-node-agent-dialog.svelte';

	let {
		nodes,
		scope,
		requestOptions = $bindable(),
		onRetry
	}: {
		nodes: NodePage;
		scope: string;
		requestOptions: SearchPaginationSortRequest;
		onRetry: () => Promise<unknown>;
	} = $props();

	const queryClient = useQueryClient();

	type NodeRow = EnvironmentRow<SwarmNode>;
	type NodePage = EnvironmentPage<SwarmNodesPaginatedResponse>;

	function canUpdateNode(node: NodeRow): boolean {
		return hasPermission('swarm-nodes:update', node.environmentId);
	}

	let isLoading = $state(false);
	let isAgentDialogOpen = $state(false);
	let selectedNodeRef = $state<NodeRow | null>(null);
	const selectedNode = $derived.by(() => {
		const current = selectedNodeRef;
		if (!current) return null;
		return nodes.data.find((node) => node.id === current.id && node.environmentId === current.environmentId) ?? current;
	});
	let agentDeploymentError = $state('');
	let agentDeployment = $state<SwarmNodeAgentDeployment | null>(null);
	let isAgentDeploymentLoading = $state(false);

	function statusVariant(state: string): 'green' | 'red' | 'amber' | 'gray' {
		if (state === 'ready') return 'green';
		if (state === 'down') return 'red';
		if (state === 'unknown') return 'amber';
		return 'gray';
	}

	function availabilityVariant(state: string): 'green' | 'amber' | 'red' | 'gray' {
		if (state === 'active') return 'green';
		if (state === 'pause') return 'amber';
		if (state === 'drain') return 'red';
		return 'gray';
	}

	function refreshNodes() {
		return queryClient.invalidateQueries({ queryKey: queryKeys.swarm.listPrefix(scope, 'nodes') });
	}

	async function loadAgentDeployment(node: NodeRow) {
		if (isAgentDeploymentLoading) return;

		isAgentDeploymentLoading = true;
		agentDeploymentError = '';
		try {
			const result = await tryCatch(swarmService.createNodeAgentDeployment(node.id, node.environmentId));
			agentDeployment = result.data ?? null;
			agentDeploymentError = result.error ? extractApiErrorMessage(result.error) : '';
			await refreshNodes();
		} finally {
			isAgentDeploymentLoading = false;
		}
	}

	function openAgentDialog(node: NodeRow) {
		selectedNodeRef = node;
		agentDeployment = null;
		agentDeploymentError = '';
		isAgentDialogOpen = true;
	}

	function provisionNodeEnvironment() {
		if (!selectedNode) return;
		void loadAgentDeployment(selectedNode);
	}

	async function attachEnvironment(environmentId: string) {
		const node = selectedNode;
		if (!node) return;
		const replaceDeployment = node.agent.bindingKind === 'dedicated';
		const attach = async () => {
			isAgentDeploymentLoading = true;
			try {
				const operationResult = await tryCatch(
					(async () => {
						await swarmService.bindNodeAgent(node.id, { environmentId, rebind: true, replaceDeployment }, node.environmentId);
						toast.success(m.swarm_node_agent_attach_success());
						await refreshNodes();
					})()
				);
				if (operationResult.error !== null) {
					const error = operationResult.error;

					agentDeploymentError = extractApiErrorMessage(error);
				}
			} finally {
				isAgentDeploymentLoading = false;
			}
		};

		if (!replaceDeployment) {
			await attach();
			return;
		}
		openConfirmDialog({
			title: m.swarm_node_agent_replace_title(),
			message: m.swarm_node_agent_replace_description(),
			confirm: { label: m.swarm_node_agent_replace_action(), destructive: true, action: attach }
		});
	}

	function detachEnvironment() {
		const node = selectedNode;
		if (!node) return;
		openConfirmDialog({
			title: m.swarm_node_agent_detach_title(),
			message: m.swarm_node_agent_detach_description(),
			confirm: {
				label: m.swarm_node_agent_detach_action(),
				destructive: true,
				action: async () => {
					await swarmService.detachNodeAgent(node.id, node.environmentId);
					await refreshNodes();
				}
			}
		});
	}

	function removeDedicatedDeployment() {
		const node = selectedNode;
		if (!node) return;
		openConfirmDialog({
			title: m.swarm_node_agent_remove_deployment_title(),
			message: m.swarm_node_agent_remove_deployment_description(),
			confirm: {
				label: m.common_remove(),
				destructive: true,
				action: async () => {
					await swarmService.removeNodeAgentDeployment(node.id, node.environmentId);
					agentDeployment = null;
					await refreshNodes();
				}
			}
		});
	}

	function generateEnrollmentToken() {
		const node = selectedNode;
		if (!node) return;

		openConfirmDialog({
			title: m.generate_enrollment_token(),
			message: m.generate_enrollment_token_dialog_message(),
			confirm: {
				label: m.generate_enrollment_token(),
				destructive: true,
				action: () => loadAgentDeployment(node)
			}
		});
	}

	function nodeDetailsHref(node: NodeRow) {
		if (!hasPermission('swarm-nodes:read', node.environmentId)) return undefined;
		return resourceTableHref(
			environmentHref(resolve('/(app)/swarm/nodes/[nodeId]', { nodeId: node.id }) + '?tab=overview', node.environmentId),
			requestOptions.pagination?.page
		);
	}

	async function mutateNode(action: () => Promise<void>, successMessage: string, failureMessage: string) {
		isLoading = true;
		await handleApiResultWithCallbacks({
			result: await tryCatch(action()),
			message: failureMessage,
			setLoadingState: (v) => (isLoading = v),
			onSuccess: async () => {
				toast.success(successMessage);
				await refreshNodes();
			}
		});
	}

	async function setAvailability(node: NodeRow, availability: 'active' | 'pause' | 'drain') {
		await mutateNode(
			() => swarmService.updateNode(node.id, { availability }, node.environmentId),
			m.common_update_success({ resource: m.swarm_node() }),
			m.swarm_node_update_failed({ name: node.hostname })
		);
	}

	async function promoteNode(node: NodeRow) {
		await mutateNode(
			() => swarmService.promoteNode(node.id, node.environmentId),
			m.swarm_node_promote_success({ name: node.hostname }),
			m.swarm_node_promote_failed({ name: node.hostname })
		);
	}

	async function demoteNode(node: NodeRow) {
		await mutateNode(
			() => swarmService.demoteNode(node.id, node.environmentId),
			m.swarm_node_demote_success({ name: node.hostname }),
			m.swarm_node_demote_failed({ name: node.hostname })
		);
	}

	function removeNode(node: NodeRow) {
		openConfirmDialog({
			title: m.common_delete_title({ resource: m.swarm_node() }),
			message: m.common_delete_confirm({ resource: m.swarm_node() }),
			confirm: {
				label: m.common_delete(),
				destructive: true,
				action: async () => {
					mutateNode(
						() => swarmService.removeNode(node.id, true, node.environmentId),
						m.swarm_node_remove_success({ name: node.hostname }),
						m.swarm_node_remove_failed({ name: node.hostname })
					);
				}
			}
		});
	}

	const columns = [
		{ accessorKey: 'id', title: m.common_id(), hidden: true },
		{ accessorKey: 'hostname', title: m.hostname(), sortable: true, cell: HostnameCell },
		{
			accessorKey: 'role',
			title: m.common_role(),
			sortable: true,
			cell: RoleCell,
			filterOptions: [
				{ label: m.manager(), value: 'manager' },
				{ label: m.worker(), value: 'worker' },
				{ label: m.leader(), value: 'leader' }
			]
		},
		{
			accessorKey: 'status',
			title: m.common_status(),
			sortable: true,
			cell: StatusCell,
			filterOptions: ['ready', 'down', 'unknown', 'disconnected'].map((value) => ({ label: value, value }))
		},
		{
			accessorKey: 'availability',
			title: m.scheduling(),
			sortable: true,
			cell: AvailabilityCell,
			filterOptions: ['active', 'drain', 'pause'].map((value) => ({ label: value, value }))
		},
		{
			id: 'agentId',
			hidden: true,
			trailing: true,
			title: m.swarm_node_agent_environment_id(),
			accessorFn: (node) => node.agent?.environmentId,
			cell: AgentIdCell
		},
		{
			id: 'agent',
			hidden: true,
			trailing: true,
			title: m.swarm_node_agent_connection(),
			accessorFn: (node) => node.agent?.state ?? 'none',
			cell: AgentConnectionCell
		},
		{ accessorKey: 'runningTasks', title: m.running_tasks(), hidden: true, sortable: true, cell: RunningTasksCell },
		{ accessorKey: 'engineVersion', title: m.swarm_engine_version(), hidden: true },
		{ accessorKey: 'platform', title: m.platform(), hidden: true },
		{ accessorKey: 'address', title: m.swarm_node_address(), hidden: true },
		{ accessorKey: 'reachability', title: m.swarm_node_reachability(), hidden: true },
		{ accessorKey: 'createdAt', id: 'created', title: m.common_created(), hidden: true, cell: DateCell, sortable: true },
		{ accessorKey: 'updatedAt', id: 'updated', title: m.common_updated(), hidden: true, cell: DateCell, sortable: true }
	] satisfies ColumnSpec<NodeRow>[];

	const mobileFields = [
		{ id: 'role', label: m.common_role(), defaultVisible: true },
		{ id: 'status', label: m.common_status(), defaultVisible: true },
		{ id: 'availability', label: m.scheduling(), defaultVisible: true },
		{ id: 'agentId', label: m.swarm_node_agent_environment_id(), defaultVisible: false },
		{ id: 'agent', label: m.swarm_node_agent_connection(), defaultVisible: false },
		{ id: 'engineVersion', label: m.swarm_engine_version(), defaultVisible: false }
	];

	let mobileFieldVisibility = $state<Record<string, boolean>>({});
	const nodeResourceViews = [
		{ id: 'networks', icon: NetworksIcon, label: m.resource_view_networks() },
		{ id: 'containers', icon: ContainersIcon, label: m.resource_view_containers() },
		{ id: 'volumes', icon: VolumesIcon, label: m.resource_view_volumes() }
	] as const;
</script>

{#snippet HostnameCell({ item }: { item: NodeRow })}
	{@const href = nodeDetailsHref(item)}
	{#if href}
		<a {href} class="font-medium hover:underline">{item.hostname}</a>
	{:else}
		{item.hostname}
	{/if}
{/snippet}

{#snippet RunningTasksCell({ value }: { value: unknown })}{typeof value === 'number' ? value : m.unavailable()}{/snippet}

{#snippet DateCell({ value }: { value: unknown })}
	{formatRelativeTime(String(value ?? ''))}
{/snippet}

{#snippet RoleCell({ item }: { item: NodeRow })}
	<div class="flex gap-1">
		<Badge variant={item.role === 'manager' ? 'purple' : 'blue'}>{capitalizeFirstLetter(item.role)}</Badge
		>{#if item.role === 'manager' && item.managerStatus === 'leader'}<Badge variant="green">{m.leader()}</Badge>{/if}
	</div>
{/snippet}

{#snippet StatusCell({ value }: { value: unknown })}
	<Badge variant={statusVariant(String(value ?? ''))} minWidth="20">{String(value ?? m.common_unknown())}</Badge>
{/snippet}

{#snippet AvailabilityCell({ value }: { value: unknown })}
	<Badge variant={availabilityVariant(String(value ?? ''))} minWidth="20">{String(value ?? m.common_unknown())}</Badge>
{/snippet}

{#snippet MobileRole(item: NodeRow)}{@render RoleCell({ item })}{/snippet}

{#snippet AgentIdCell({ value }: { value: unknown })}
	{@const agentId = typeof value === 'string' ? value : ''}
	{#if agentId}<code class="text-xs text-muted-foreground" title={agentId}
			>{agentId.slice(0, 8)}{agentId.length > 8 ? '…' : ''}</code
		>{:else}{m.common_na()}{/if}
{/snippet}

{#snippet AgentConnectionCell({ item }: { item: NodeRow })}
	{@const agentEnvironment = environmentStore.available.find((entry) => entry.id === item.agent?.environmentId)}
	<div class="flex items-center gap-2">
		<Badge variant={getSwarmNodeAgentVariant(item.agent?.state)}>{getSwarmNodeAgentLabel(item.agent?.state)}</Badge>
		{#if agentEnvironment}<span class="text-xs text-muted-foreground">{getEnvironmentTypeLabel(agentEnvironment)}</span>{/if}
	</div>
{/snippet}

{#snippet NodeMobileCardSnippet({ item, mobileFieldVisibility }: { item: NodeRow; mobileFieldVisibility: MobileFieldVisibility })}
	<UniversalMobileCard
		{item}
		icon={() => ({
			component: UsersIcon,
			variant: item.role === 'manager' ? 'purple' : 'blue'
		})}
		title={(item: NodeRow) => item.hostname}
		href={nodeDetailsHref}
		subtitle={(item: NodeRow) => ((mobileFieldVisibility['engineVersion'] ?? false) ? (item.engineVersion ?? '') : null)}
		badges={[
			(item: NodeRow) =>
				(mobileFieldVisibility['status'] ?? true) ? { variant: statusVariant(item.status), text: item.status } : null,
			(item: NodeRow) =>
				(mobileFieldVisibility['agent'] ?? false)
					? {
							variant: getSwarmNodeAgentVariant(item.agent?.state),
							text: getSwarmNodeAgentLabel(item.agent?.state)
						}
					: null
		]}
		fields={[
			{
				label: m.common_role(),
				getValue: (item: NodeRow) => item,
				type: 'component',
				component: MobileRole,
				icon: EnvironmentsIcon,
				iconVariant: 'gray' as const,
				show: mobileFieldVisibility['role'] ?? true
			},
			{
				label: m.scheduling(),
				getValue: (item: NodeRow) => capitalizeFirstLetter(item.availability),
				icon: EnvironmentsIcon,
				iconVariant: 'gray' as const,
				show: mobileFieldVisibility['availability'] ?? true
			},
			{
				label: m.swarm_node_agent_environment_id(),
				getValue: (item: NodeRow) => item.agent?.environmentId,
				icon: AgentConnectionIcon,
				iconVariant: 'gray' as const,
				show: mobileFieldVisibility['agentId'] ?? false
			}
		]}
		rowActions={RowActions}
	/>
{/snippet}

{#snippet RowActions({ item }: { item: NodeRow })}
	<RowActionsMenu name={item.hostname}>
		{@const detailsHref = nodeDetailsHref(item)}
		{#if detailsHref}
			<DetailLinksMenu
				href={detailsHref}
				tabs={[
					{ value: 'overview', label: m.common_overview() },
					{ value: 'host', label: m.swarm_node_hardware_host(), icon: EnvironmentsIcon },
					{ value: 'docker', label: m.engine(), icon: ContainersIcon },
					{ value: 'agent', label: m.agent(), icon: AgentConnectionIcon },
					{ value: 'tasks', label: m.tasks(), icon: JobsIcon, related: true },
					...nodeResourceViews.map((view) => ({
						value: view.id,
						label: view.label,
						icon: view.icon,
						related: true,
						href: detailsHref + '&tab=resources&resource=' + view.id,
						visible: !!item.agent?.environmentId && hasPermission(view.id + ':list', item.agent.environmentId)
					}))
				]}
			/>
		{/if}
		{#if canUpdateNode(item)}
			<DropdownMenu.Item onclick={() => openAgentDialog(item)}>
				<AgentConnectionIcon class="size-4" />
				{getSwarmNodeAgentActionLabel(item.agent?.state)}
			</DropdownMenu.Item>
		{/if}
		{#if hasAnyPermission(['swarm-nodes:promote', 'swarm-nodes:demote', 'swarm-nodes:update'], item.environmentId)}
			<DropdownMenu.Separator />
		{/if}
		{#if hasPermission('swarm-nodes:promote', item.environmentId)}
			<DropdownMenu.Item onclick={() => promoteNode(item)} disabled={isLoading || item.role === 'manager'}>
				<ArrowUpIcon />{m.swarm_node_promote()}
			</DropdownMenu.Item>
		{/if}
		{#if hasPermission('swarm-nodes:demote', item.environmentId)}
			<DropdownMenu.Item onclick={() => demoteNode(item)} disabled={isLoading || item.role !== 'manager'}>
				<ArrowDownIcon />{m.swarm_node_demote()}
			</DropdownMenu.Item>
		{/if}
		{#if canUpdateNode(item)}
			<DropdownMenu.Item onclick={() => setAvailability(item, 'drain')} disabled={isLoading || item.availability === 'drain'}>
				<StopIcon />{m.drain()}
			</DropdownMenu.Item>
			<DropdownMenu.Item onclick={() => setAvailability(item, 'pause')} disabled={isLoading || item.availability === 'pause'}>
				<PauseIcon />{m.common_pause()}
			</DropdownMenu.Item>
			<DropdownMenu.Item onclick={() => setAvailability(item, 'active')} disabled={isLoading || item.availability === 'active'}>
				<PlayIcon />{m.swarm_node_activate()}
			</DropdownMenu.Item>
		{/if}
		{#if hasPermission('swarm-nodes:delete', item.environmentId)}
			<DropdownMenu.Separator />
			<DropdownMenu.Item variant="destructive" onclick={() => removeNode(item)} disabled={isLoading}>
				<TrashIcon class="size-4" />
				{m.common_delete()}
			</DropdownMenu.Item>
		{/if}
	</RowActionsMenu>
{/snippet}

<ArcaneTable
	persistKey="arcane-swarm-nodes-table"
	items={nodes}
	bind:requestOptions
	bind:mobileFieldVisibility
	selectionDisabled={true}
	{onRetry}
	{columns}
	{mobileFields}
	rowActions={RowActions}
	mobileCard={NodeMobileCardSnippet}
/>

<SwarmNodeAgentDialog
	bind:open={isAgentDialogOpen}
	node={selectedNode}
	deployment={agentDeployment}
	errorMessage={agentDeploymentError}
	isLoading={isAgentDeploymentLoading}
	onRefresh={refreshNodes}
	onGenerateEnrollmentToken={generateEnrollmentToken}
	onProvision={provisionNodeEnvironment}
	onAttach={attachEnvironment}
	onDetach={detachEnvironment}
	onRemoveDeployment={removeDedicatedDeployment}
/>
