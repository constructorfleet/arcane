import { z } from 'zod/v4';

import type { KeyValueRow } from '#lib/components/form/key-value-editor.svelte';
import type { NetworkAttachmentRow } from '#lib/components/form/network-attachment-editor.svelte';
import type { PortMappingRow } from '#lib/components/form/port-mapping-editor.svelte';
import type { VolumeMountRow } from '#lib/components/form/volume-mount-editor.svelte';
import { m } from '#lib/paraglide/messages.js';
import type {
	ContainerEditConfig,
	ContainerEdit,
	ContainerHealthcheckCreate,
	ContainerHealthcheckUpdate,
	ContainerHostConfigEdit,
	ContainerEndpointSettingsCreate,
	ContainerHostConfigCreate,
	ContainerMountCreate,
	ContainerPortBindingCreate
} from '#lib/types/container.js';
import type { ContainerCreate } from '#lib/types/ui/container.js';

export const LINUX_CAPABILITIES = [
	'AUDIT_CONTROL',
	'AUDIT_READ',
	'AUDIT_WRITE',
	'BLOCK_SUSPEND',
	'BPF',
	'CHECKPOINT_RESTORE',
	'CHOWN',
	'DAC_OVERRIDE',
	'DAC_READ_SEARCH',
	'FOWNER',
	'FSETID',
	'IPC_LOCK',
	'IPC_OWNER',
	'KILL',
	'LEASE',
	'LINUX_IMMUTABLE',
	'MAC_ADMIN',
	'MAC_OVERRIDE',
	'MKNOD',
	'NET_ADMIN',
	'NET_BIND_SERVICE',
	'NET_BROADCAST',
	'NET_RAW',
	'PERFMON',
	'SETFCAP',
	'SETGID',
	'SETPCAP',
	'SETUID',
	'SYSLOG',
	'SYS_ADMIN',
	'SYS_BOOT',
	'SYS_CHROOT',
	'SYS_MODULE',
	'SYS_NICE',
	'SYS_PACCT',
	'SYS_PTRACE',
	'SYS_RAWIO',
	'SYS_RESOURCE',
	'SYS_TIME',
	'SYS_TTY_CONFIG',
	'WAKE_ALARM'
] as const;

export const containerFormSchema = z.object({
	name: z.string().min(1, m.container_name_required()),
	image: z.string().min(1, m.image_is_required()),
	workingDir: z.string().optional().default(''),
	user: z.string().optional().default(''),
	hostname: z.string().optional().default(''),
	restartPolicy: z.string().default('no'),
	restartMaxRetries: z.coerce.number().min(0).default(0),
	memoryMb: z.coerce.number().min(0).default(0),
	memorySwapMb: z.coerce
		.number()
		.refine((value) => value === -1 || value >= 0, m.memory_swap_limit_invalid())
		.default(0),
	cpus: z.coerce.number().min(0).default(0),
	cpuShares: z.coerce.number().min(0).default(0),
	privileged: z.boolean().default(false),
	readonlyRootfs: z.boolean().default(false),
	autoRemove: z.boolean().default(false),
	healthMode: z.enum(['inherit', 'custom', 'disable']).default('inherit'),
	healthTest: z.string().optional().default(''),
	healthInterval: z.coerce.number().min(0).default(0),
	healthTimeout: z.coerce.number().min(0).default(0),
	healthStartPeriod: z.coerce.number().min(0).default(0),
	healthRetries: z.coerce.number().min(0).default(0)
});

export type ContainerFormSchema = typeof containerFormSchema;
export type ContainerFormValues = z.infer<ContainerFormSchema>;

export type ContainerFormRows = {
	command: { value: string }[];
	entrypoint: { value: string }[];
	env: KeyValueRow[];
	labels: KeyValueRow[];
	ports: PortMappingRow[];
	volumes: VolumeMountRow[];
	networks: NetworkAttachmentRow[];
	capAdd: string[];
	capDrop: string[];
};

function joinShellWords(words: string[] | undefined): string {
	if (!words?.length) return '';
	return words.map((word) => (/\s/.test(word) ? `"${word}"` : word)).join(' ');
}

export function emptyContainerFormValues(): ContainerFormValues {
	return {
		name: '',
		image: '',
		workingDir: '',
		user: '',
		hostname: '',
		restartPolicy: 'no',
		restartMaxRetries: 0,
		memoryMb: 0,
		memorySwapMb: 0,
		cpus: 0,
		cpuShares: 0,
		privileged: false,
		readonlyRootfs: false,
		autoRemove: false,
		healthMode: 'inherit',
		healthTest: '',
		healthInterval: 0,
		healthTimeout: 0,
		healthStartPeriod: 0,
		healthRetries: 0
	};
}

export function emptyContainerFormRows(): ContainerFormRows {
	return { command: [], entrypoint: [], env: [], labels: [], ports: [], volumes: [], networks: [], capAdd: [], capDrop: [] };
}

function healthModeFromConfig(hc: ContainerHealthcheckCreate | undefined): ContainerFormValues['healthMode'] {
	if (!hc || !hc.test?.length) return 'inherit';
	if (hc.test[0] === 'NONE') return 'disable';
	return 'custom';
}

function healthTestToString(test: string[] | undefined): string {
	if (!test?.length) return '';
	if (test[0] === 'CMD-SHELL') return test.slice(1).join(' ');
	if (test[0] === 'CMD') return joinShellWords(test.slice(1));
	return joinShellWords(test);
}

export function formValuesFromEditConfig(cfg: ContainerEditConfig): ContainerFormValues {
	const hc = cfg.hostConfig;
	return {
		...emptyContainerFormValues(),
		name: cfg.name,
		image: cfg.image,
		workingDir: cfg.workingDir ?? '',
		user: cfg.user ?? '',
		hostname: cfg.hostname ?? '',
		restartPolicy: hc.restartPolicy?.name || 'no',
		restartMaxRetries: hc.restartPolicy?.maximumRetryCount ?? 0,
		memoryMb: (hc.memory ?? 0) / (1024 * 1024),
		memorySwapMb: hc.memorySwap === -1 ? -1 : (hc.memorySwap ?? 0) / (1024 * 1024),
		cpus: hc.nanoCpus ? hc.nanoCpus / 1e9 : 0,
		cpuShares: hc.cpuShares ?? 0,
		privileged: !!hc.privileged,
		readonlyRootfs: !!hc.readonlyRootfs,
		autoRemove: !!hc.autoRemove,
		healthMode: healthModeFromConfig(cfg.healthcheck),
		healthTest: cfg.healthcheck?.test?.[0] === 'NONE' ? '' : healthTestToString(cfg.healthcheck?.test),
		healthInterval: cfg.healthcheck?.interval ?? 0,
		healthTimeout: cfg.healthcheck?.timeout ?? 0,
		healthStartPeriod: cfg.healthcheck?.startPeriod ?? 0,
		healthRetries: cfg.healthcheck?.retries ?? 0
	};
}

function parseBindString(bind: string): VolumeMountRow {
	const parts = bind.split(':');
	const source = parts[0] ?? '';
	const target = parts[1] ?? '';
	const optionTokens = (parts[2] ?? '').split(',').filter(Boolean);
	const readOnly = optionTokens.includes('ro');
	const rawOptions = optionTokens.filter((token) => token !== 'ro' && token !== 'rw').join(',');
	const kind = source.startsWith('/') || source.startsWith('.') || source.startsWith('~') ? 'bind' : 'volume';
	return { kind, source, target, readOnly, rawOptions: rawOptions || undefined };
}

function serializeBindRow(row: VolumeMountRow): string {
	const options = row.rawOptions ? row.rawOptions.split(',').filter(Boolean) : [];
	if (row.readOnly) options.push('ro');
	const suffix = options.length > 0 ? `:${options.join(',')}` : '';
	return `${row.source.trim()}:${row.target.trim()}${suffix}`;
}

function portRowsFromEditConfig(cfg: ContainerEditConfig): PortMappingRow[] {
	const ports: PortMappingRow[] = [];
	for (const [portSpec, bindings] of Object.entries(cfg.hostConfig.portBindings ?? {})) {
		const [portPart, protocolPart] = portSpec.split('/');
		const protocol = protocolPart === 'udp' || protocolPart === 'sctp' ? protocolPart : 'tcp';
		for (const binding of bindings) {
			ports.push({
				hostIp: binding.hostIp ?? '',
				hostPort: binding.hostPort ?? '',
				containerPort: portPart ?? '',
				protocol
			});
		}
	}

	return ports;
}

function volumeRowsFromEditConfig(cfg: ContainerEditConfig): VolumeMountRow[] {
	const volumes: VolumeMountRow[] = [];

	for (const bind of cfg.hostConfig.binds ?? []) {
		volumes.push(parseBindString(bind));
	}
	for (const mount of cfg.hostConfig.mounts ?? []) {
		volumes.push({
			kind: 'mount',
			mountType: mount.type,
			source: mount.source,
			target: mount.target,
			readOnly: !!mount.readOnly
		});
	}

	return volumes;
}

export function rowsFromEditConfig(cfg: ContainerEditConfig): ContainerFormRows {
	const rows = emptyContainerFormRows();
	rows.command = (cfg.command ?? []).map((value) => ({ value }));
	rows.entrypoint = (cfg.entrypoint ?? []).map((value) => ({ value }));

	for (const entry of cfg.environment ?? []) {
		const eq = entry.indexOf('=');
		rows.env.push(eq === -1 ? { key: entry, value: '' } : { key: entry.slice(0, eq), value: entry.slice(eq + 1) });
	}

	for (const [key, value] of Object.entries(cfg.labels ?? {})) {
		rows.labels.push({ key, value });
	}

	rows.ports = portRowsFromEditConfig(cfg);
	rows.volumes = volumeRowsFromEditConfig(cfg);

	for (const [network, endpoint] of Object.entries(cfg.networks ?? {})) {
		rows.networks.push({
			network,
			aliases: (endpoint.aliases ?? []).join(', '),
			ipv4Address: endpoint.ipv4Address ?? '',
			ipv6Address: endpoint.ipv6Address ?? '',
			linkLocalAddresses: (endpoint.linkLocalIPs ?? []).join(', ')
		});
	}

	rows.capAdd = [...(cfg.hostConfig.capAdd ?? [])];
	rows.capDrop = [...(cfg.hostConfig.capDrop ?? [])];

	return rows;
}

function buildHealthcheck(values: ContainerFormValues): ContainerHealthcheckCreate | undefined {
	if (values.healthMode === 'disable') return { test: ['NONE'] };
	if (values.healthMode !== 'custom' || !values.healthTest.trim()) return undefined;
	return {
		test: ['CMD-SHELL', values.healthTest.trim()],
		interval: values.healthInterval || undefined,
		timeout: values.healthTimeout || undefined,
		startPeriod: values.healthStartPeriod || undefined,
		retries: values.healthRetries || undefined
	};
}

function buildEnv(rows: ContainerFormRows): string[] {
	return rows.env.filter((row) => row.key.trim()).map((row) => `${row.key.trim()}=${row.value}`);
}

function buildLabels(rows: ContainerFormRows): Record<string, string> {
	const labels: Record<string, string> = {};
	for (const row of rows.labels) {
		if (row.key.trim()) labels[row.key.trim()] = row.value;
	}
	return labels;
}

function buildPortBindings(rows: ContainerFormRows): Record<string, ContainerPortBindingCreate[]> {
	const bindings: Record<string, ContainerPortBindingCreate[]> = {};
	for (const row of rows.ports) {
		if (!row.containerPort.trim()) continue;
		const key = `${row.containerPort.trim()}/${row.protocol}`;
		const binding: ContainerPortBindingCreate = { hostPort: row.hostPort.trim() };
		if (row.hostIp.trim()) binding.hostIp = row.hostIp.trim();
		bindings[key] = [...(bindings[key] ?? []), binding];
	}
	return bindings;
}

function buildBinds(rows: ContainerFormRows): string[] {
	return rows.volumes.filter((row) => row.kind !== 'mount' && row.source.trim() && row.target.trim()).map(serializeBindRow);
}

function buildMounts(rows: ContainerFormRows): ContainerMountCreate[] {
	return rows.volumes
		.filter((row) => row.kind === 'mount')
		.map((row) => ({
			type: row.mountType ?? 'volume',
			source: row.source,
			target: row.target,
			readOnly: row.readOnly
		}));
}

function buildEndpoints(rows: ContainerFormRows): Record<string, ContainerEndpointSettingsCreate> {
	const endpoints: Record<string, ContainerEndpointSettingsCreate> = {};
	for (const row of rows.networks) {
		if (!row.network.trim()) continue;
		const endpoint: ContainerEndpointSettingsCreate = {};
		const aliases = row.aliases
			.split(',')
			.map((alias) => alias.trim())
			.filter(Boolean);
		if (aliases.length > 0) endpoint.aliases = aliases;
		if (row.ipv4Address.trim()) endpoint.ipv4Address = row.ipv4Address.trim();
		if (row.ipv6Address.trim()) endpoint.ipv6Address = row.ipv6Address.trim();
		const linkLocalIPs = row.linkLocalAddresses
			.split(',')
			.map((address) => address.trim())
			.filter(Boolean);
		if (linkLocalIPs.length > 0) endpoint.linkLocalIPs = linkLocalIPs;
		endpoints[row.network.trim()] = endpoint;
	}
	return endpoints;
}

function buildRestartPolicy(values: ContainerFormValues) {
	return {
		name: values.restartPolicy as 'no' | 'always' | 'on-failure' | 'unless-stopped',
		maximumRetryCount: values.restartPolicy === 'on-failure' ? values.restartMaxRetries : undefined
	};
}

export function toCreateRequest(values: ContainerFormValues, rows: ContainerFormRows): ContainerCreate {
	const env = buildEnv(rows);
	const labels = buildLabels(rows);
	const portBindings = buildPortBindings(rows);
	const binds = buildBinds(rows);
	const endpoints = buildEndpoints(rows);
	let memorySwap: number | undefined;
	if (values.memorySwapMb === -1) {
		memorySwap = -1;
	} else if (values.memorySwapMb > 0) {
		memorySwap = Math.round(values.memorySwapMb * 1024 * 1024);
	}

	const hostConfig: ContainerHostConfigCreate = {
		binds: binds.length > 0 ? binds : undefined,
		portBindings: Object.keys(portBindings).length > 0 ? portBindings : undefined,
		restartPolicy: values.restartPolicy !== 'no' ? buildRestartPolicy(values) : undefined,
		privileged: values.privileged || undefined,
		readonlyRootfs: values.readonlyRootfs || undefined,
		autoRemove: values.autoRemove || undefined,
		memory: values.memoryMb > 0 ? Math.round(values.memoryMb * 1024 * 1024) : undefined,
		memorySwap,
		nanoCpus: values.cpus > 0 ? Math.round(values.cpus * 1e9) : undefined,
		cpuShares: values.cpuShares > 0 ? values.cpuShares : undefined,
		capAdd: rows.capAdd.length > 0 ? [...rows.capAdd] : undefined,
		capDrop: rows.capDrop.length > 0 ? [...rows.capDrop] : undefined
	};

	return {
		name: values.name.trim(),
		image: values.image.trim(),
		cmd: rows.command.length > 0 ? rows.command.map((argument) => argument.value) : undefined,
		entrypoint: rows.entrypoint.length > 0 ? rows.entrypoint.map((argument) => argument.value) : undefined,
		workingDir: values.workingDir.trim() || undefined,
		user: values.user.trim() || undefined,
		hostname: values.hostname.trim() || undefined,
		env: env.length > 0 ? env : undefined,
		labels: Object.keys(labels).length > 0 ? labels : undefined,
		healthcheck: buildHealthcheck(values),
		hostConfig,
		networkingConfig: Object.keys(endpoints).length > 0 ? { endpointsConfig: endpoints } : undefined
	};
}

// Execution settings are omitted when unchanged so the backend preserves them.
// Other form-owned sections always include empty values to allow clearing.
export function toEditRequest(
	values: ContainerFormValues,
	rows: ContainerFormRows,
	original: ContainerEditConfig
): ContainerEdit {
	const initial = formValuesFromEditConfig(original);
	const commandValues = rows.command.map((argument) => argument.value);
	let command: string[] | undefined;
	if (JSON.stringify(commandValues) !== JSON.stringify(original.command ?? [])) {
		command = commandValues;
	}
	const entrypointValues = rows.entrypoint.map((argument) => argument.value);
	let entrypoint: string[] | undefined;
	if (JSON.stringify(entrypointValues) !== JSON.stringify(original.entrypoint ?? [])) {
		entrypoint = entrypointValues;
	}

	const healthFields = ['healthMode', 'healthInterval', 'healthTimeout', 'healthStartPeriod', 'healthRetries'] as const;
	const healthTestChanged = values.healthTest.trim() !== initial.healthTest.trim();
	let healthcheck: ContainerHealthcheckCreate | undefined;
	let healthcheckUpdate: ContainerHealthcheckUpdate | undefined;
	let clearHealthcheck = false;
	if (healthTestChanged || healthFields.some((field) => values[field] !== initial[field])) {
		clearHealthcheck = values.healthMode === 'inherit';
		if (values.healthMode === 'custom' && initial.healthMode === 'custom' && values.healthTest.trim()) {
			const update: ContainerHealthcheckUpdate = {};
			if (healthTestChanged) update.test = ['CMD-SHELL', values.healthTest.trim()];
			for (const [field, target] of [
				['healthInterval', 'interval'],
				['healthTimeout', 'timeout'],
				['healthStartPeriod', 'startPeriod'],
				['healthRetries', 'retries']
			] as const) {
				if (values[field] !== initial[field]) update[target] = values[field];
			}
			if (Object.keys(update).length > 0) healthcheckUpdate = update;
		} else {
			healthcheck = buildHealthcheck(values);
		}
	}

	let memorySwap: number | undefined;
	if (values.memorySwapMb !== initial.memorySwapMb) {
		memorySwap = values.memorySwapMb === -1 ? -1 : Math.round(values.memorySwapMb * 1024 * 1024);
	}
	const hostConfig: ContainerHostConfigEdit = {
		binds: buildBinds(rows),
		mounts: buildMounts(rows),
		portBindings: buildPortBindings(rows),
		restartPolicy: buildRestartPolicy(values),
		privileged: values.privileged,
		readonlyRootfs: values.readonlyRootfs,
		autoRemove: values.autoRemove,
		memory: values.memoryMb === initial.memoryMb ? undefined : Math.round(values.memoryMb * 1024 * 1024),
		memorySwap,
		nanoCpus: Math.round(values.cpus * 1e9),
		cpuShares: values.cpuShares,
		capAdd: [...rows.capAdd],
		capDrop: [...rows.capDrop]
	};

	return {
		name: values.name.trim(),
		image: values.image.trim(),
		workingDir: values.workingDir.trim(),
		user: values.user.trim(),
		hostname: values.hostname.trim(),
		command,
		entrypoint,
		environment: buildEnv(rows),
		labels: buildLabels(rows),
		healthcheck,
		healthcheckUpdate,
		clearHealthcheck,
		hostConfig,
		networkingConfig:
			JSON.stringify(rows.networks) === JSON.stringify(rowsFromEditConfig(original).networks)
				? undefined
				: { endpointsConfig: buildEndpoints(rows) }
	};
}
