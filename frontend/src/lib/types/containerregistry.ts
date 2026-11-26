export type RegistryType = 'generic' | 'ecr';

export interface ContainerRegistryNamespace {
	name: string;
}

export interface ContainerRegistryRepository {
	name: string;
}

export interface ContainerRegistryRepositoryTag {
	created?: string;
	name: string;
	digest?: string;
	mediaType?: string;
	error?: string;
	platforms: ContainerRegistryTagPlatform[];
	size: number;
}

export interface ContainerRegistryTagPlatform {
	os: string;
	architecture: string;
	variant?: string;
	digest: string;
	size: number;
}

export interface ContainerRegistry {
	createdAt: string;
	updatedAt: string;
	description?: string;
	id: string;
	url: string;
	host: string;
	namespace: string;
	username: string;
	registryType: RegistryType;
	awsAccessKeyId?: string;
	awsRegion?: string;
	token?: string;
	awsSecretAccessKey?: string;
	repositoryNames: string[];
	insecure: boolean;
	enabled: boolean;
}

export interface ContainerRegistryPullUsageResponse {
	registries: ContainerRegistryPullUsage[];
}

export interface ContainerRegistryPullUsage {
	checkedAt: string;
	limit?: number;
	remaining?: number;
	used?: number;
	windowSeconds?: number;
	registryId: string;
	provider: string;
	registry: string;
	displayName: string;
	repository?: string;
	authMethod: 'anonymous' | 'credential' | 'unknown';
	authUsername?: string;
	source?: string;
	error?: string;
	observedPulls: number;
}

export interface ContainerRegistryCredential {
	url: string;
	username: string;
	token: string;
	enabled: boolean;
}

export interface RegistryScanFinding {
	id: string;
	package: string;
	version: string;
	fix_version?: string;
	severity: string;
	links?: string[];
}

export interface RegistryScanReport {
	digest: string;
	status: string;
	severity?: string;
	scanner?: string;
	completedAt?: string;
	platform?: string;
	findings: RegistryScanFinding[];
	counts?: Record<string, number>;
	total: number;
}
