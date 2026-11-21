import { untrack } from 'svelte';

import { environmentStore } from '#lib/stores/environment.store.svelte.js';

export class ResourceListPageState<TReq> {
	requestOptions = $state() as TReq;
	selectedIds = $state<string[]>([]);
	isCreateDialogOpen = $state(false);
	scope = $derived(this.linkedAgent?.() ?? environmentStore.shownKey);

	private linkedAgent?: () => string | undefined;

	constructor(requestOptions: TReq, linkedAgent?: () => string | undefined, initialRequest?: () => TReq) {
		this.requestOptions = requestOptions;
		this.linkedAgent = linkedAgent;
		if (linkedAgent && initialRequest) {
			let previous = untrack(linkedAgent);
			$effect.pre(() => {
				const next = linkedAgent();
				if (next === previous) return;
				previous = next;
				this.requestOptions = untrack(initialRequest);
				this.selectedIds = [];
				this.isCreateDialogOpen = false;
			});
		}
	}
}
