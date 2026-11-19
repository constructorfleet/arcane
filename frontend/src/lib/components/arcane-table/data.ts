import {
	GlobeIcon,
	FolderOpenIcon,
	VerifiedCheckIcon,
	AlertIcon,
	InfoIcon,
	CloseIcon,
	CheckIcon,
	UpdateIcon,
	PlayIcon,
	StopIcon,
	ShieldCheckIcon,
	ShieldAlertIcon,
	ShieldXIcon
} from '#lib/icons/index.js';
import { m } from '#lib/paraglide/messages.js';

import type { FilterOption } from './arcane-table.types.svelte';

export const usageFilters: FilterOption[] = [
	{
		value: true,
		get label() {
			return m.common_in_use();
		},
		icon: VerifiedCheckIcon
	},
	{
		value: false,
		get label() {
			return m.common_unused();
		},
		icon: AlertIcon
	}
];

export const imageUpdateFilters: FilterOption[] = [
	{
		value: 'has_update',
		get label() {
			return m.images_has_updates();
		},
		icon: UpdateIcon
	},
	{
		value: 'up_to_date',
		get label() {
			return m.images_no_updates();
		},
		icon: VerifiedCheckIcon
	},
	{
		value: 'error',
		get label() {
			return m.common_error();
		},
		icon: CloseIcon
	},
	{
		value: 'unknown',
		get label() {
			return m.common_unknown();
		},
		icon: InfoIcon
	}
];

export const hardenedImageFilters: FilterOption[] = [
	{
		value: true,
		get label() {
			return m.hardened();
		},
		icon: ShieldCheckIcon
	},
	{
		value: 'ending_soon',
		get label() {
			return m.ending_soon();
		},
		icon: ShieldAlertIcon
	},
	{
		value: 'end_of_life',
		get label() {
			return m.end_of_life();
		},
		icon: ShieldXIcon
	},
	{
		value: false,
		get label() {
			return m.not_hardened();
		},
		icon: InfoIcon
	}
];

export const severityFilters: FilterOption[] = [
	{
		value: 'info',
		get label() {
			return m.info();
		},
		icon: InfoIcon
	},
	{
		value: 'success',
		get label() {
			return m.common_success();
		},
		icon: CheckIcon
	},
	{
		value: 'warning',
		get label() {
			return m.warning();
		},
		icon: AlertIcon
	},
	{
		value: 'error',
		get label() {
			return m.common_error();
		},
		icon: CloseIcon
	}
];

export const vulnerabilitySeverityFilters: FilterOption[] = [
	{
		value: 'CRITICAL',
		get label() {
			return m.vuln_severity_critical();
		},
		dotClass: 'bg-destructive'
	},
	{
		value: 'HIGH',
		get label() {
			return m.vuln_severity_high();
		},
		dotClass: 'bg-orange'
	},
	{
		value: 'MEDIUM',
		get label() {
			return m.vuln_severity_medium();
		},
		dotClass: 'bg-warning'
	},
	{
		value: 'LOW',
		get label() {
			return m.vuln_severity_low();
		},
		dotClass: 'bg-success'
	},
	{
		value: 'UNKNOWN',
		get label() {
			return m.common_unknown();
		},
		dotClass: 'bg-muted-foreground'
	}
];

export const templateTypeFilters: FilterOption[] = [
	{
		value: 'false',
		get label() {
			return m.local();
		},
		icon: FolderOpenIcon
	},
	{
		value: 'true',
		get label() {
			return m.templates_remote();
		},
		icon: GlobeIcon
	}
];

export const projectStatusFilters: FilterOption[] = [
	{
		value: 'running',
		get label() {
			return m.common_running();
		},
		icon: PlayIcon
	},
	{
		value: 'stopped',
		get label() {
			return m.common_stopped();
		},
		icon: StopIcon
	},
	{
		value: 'partially running',
		get label() {
			return m.projects_status_partial();
		},
		icon: AlertIcon
	},
	{
		value: 'unknown',
		get label() {
			return m.common_unknown();
		},
		icon: InfoIcon
	}
];
