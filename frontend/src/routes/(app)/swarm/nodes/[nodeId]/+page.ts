import { error } from '@sveltejs/kit';

import { m } from '#lib/paraglide/messages.js';
import { swarmService } from '#lib/services/swarm-service.js';
import { throwPageLoadError } from '#lib/utils/api.js';
import { hasPermission } from '#lib/utils/auth.js';
import { resolveDetailEnvironmentId } from '#lib/utils/environments.js';

import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, url }) => {
	const envId = await resolveDetailEnvironmentId(url);
	if (!hasPermission('swarm-nodes:read', envId)) error(403, m.common_access_denied());

	try {
		return { envId, node: await swarmService.getNode(params.nodeId, envId) };
	} catch (cause) {
		throwPageLoadError(cause, m.swarm_node_details_load_failed());
	}
};
