import { toast } from 'svelte-sonner';

import { openConfirmDialog } from '#lib/components/confirm-dialog/index.js';
import { m } from '#lib/paraglide/messages.js';
import { swarmService } from '#lib/services/swarm-service.js';
import { handleApiResultWithCallbacks } from '#lib/utils/api.js';
import { tryCatch } from '#lib/utils/try-catch.js';

export function confirmForceUpdateService({
	serviceId,
	serviceName,
	environmentId,
	refresh
}: {
	serviceId: string;
	serviceName: string;
	environmentId: string;
	refresh: () => Promise<unknown>;
}) {
	openConfirmDialog({
		title: m.swarm_service_force_update(),
		message: m.swarm_service_force_update_confirm({ name: serviceName }),
		confirm: {
			label: m.swarm_service_force_update_short(),
			destructive: false,
			action: async () => {
				const result = await tryCatch(swarmService.forceUpdateService(serviceId, environmentId));
				await handleApiResultWithCallbacks({
					result,
					message: m.swarm_service_force_update_failed({ name: serviceName }),
					onSuccess: async () => {
						toast.success(m.swarm_service_force_update_started({ name: serviceName }));
						for (const warning of result.data?.warnings ?? []) toast.warning(warning);
						await refresh();
					}
				});
			}
		}
	});
}
