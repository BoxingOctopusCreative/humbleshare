<script lang="ts">
	import { onMount } from 'svelte';
	import { initClientSentry } from '$lib/sentry-client';

	let { dsn }: { dsn: string } = $props();

	onMount(() => {
		initClientSentry(dsn);
	});
</script>
