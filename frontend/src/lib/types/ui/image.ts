export interface ImageAttestationRequestOptions {
	platform?: string;
	predicateType?: string;
	statement?: boolean;
}

export type ImageTransferResult = {
	success: boolean;
	imageName: string;
	error?: string;
	/** The transfer was abandoned because the user signed out. */
	cancelled?: boolean;
};

export type ImageTransferRequest = {
	imageName: string;
	environmentId: string;
	onLine?: (data: unknown) => void;
} & ({ operation: 'pull' } | { operation: 'push'; imageId: string });
