import { m } from '#lib/paraglide/messages.js';
import { swarmService } from '#lib/services/swarm-service.js';
import { throwPageLoadError } from '#lib/utils/api.js';
import { resolveDetailEnvironmentId } from '#lib/utils/environments.js';

import type { PageLoad } from './$types';
export const load: PageLoad = async ({ params, url }) => {
	const envId = await resolveDetailEnvironmentId(url);
	try {
		return { envId, item: await swarmService.getKvItem('secrets', params.id, envId) };
	} catch (err) {
		throwPageLoadError(err, m.common_load_failed({ resource: m.swarm_secret() }));
	}
};
