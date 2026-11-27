<script lang="ts">
	import SwarmKvDetail from '#lib/components/swarm/swarm-kv-detail.svelte';
	import { swarmKvResources } from '#lib/utils/swarm-kv.js';
	let { data } = $props();
</script>

<SwarmKvDetail resource={swarmKvResources.configs} item={data.item} environmentId={data.envId} />
