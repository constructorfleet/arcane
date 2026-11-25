import type { GoroutineProfile } from '#lib/types/system.js';

import BaseAPIService from './api-service';

class DiagnosticsService extends BaseAPIService {
	async getGoroutines(): Promise<GoroutineProfile> {
		return this.handleResponse(this.api.get('/diagnostics/goroutines'));
	}
}

export const diagnosticsService = new DiagnosticsService();
