<script lang="ts">
	import { resolve } from '$app/paths';
	import SplashBackdrop from '$lib/components/SplashBackdrop.svelte';

	const logoVert = 'https://assets.humbleshare.com/SVG/humbleshare_vert.svg';

	let { data } = $props();
</script>

<svelte:head>
	<title>HumbleShare</title>
	<meta
		name="description"
		content="Spare Steam keys and gift links for a small group of friends."
	/>
</svelte:head>

<SplashBackdrop />

<section
	class="relative z-10 flex min-h-[calc(100vh-8rem)] flex-col items-center justify-center px-2 py-10 text-center"
>
	<h1 class="m-0">
		<img
			src={logoVert}
			alt="HumbleShare"
			width="288"
			height="350"
			fetchpriority="high"
			class="mx-auto h-auto w-56 drop-shadow-[0_2px_0_rgba(0,0,0,0.45)] drop-shadow-[0_8px_24px_rgba(0,0,0,0.45)] sm:w-72"
		/>
	</h1>

	<p class="mt-4 max-w-md text-base text-[var(--color-muted)] sm:text-lg">
		Spare Steam keys for friends. Claim once, copy once.
	</p>

	<div class="mt-8 flex flex-wrap items-center justify-center gap-3">
		{#if data.user}
			<a
				href={resolve('/games')}
				class="rounded bg-[var(--color-accent)] px-5 py-2.5 text-sm font-medium text-white no-underline hover:bg-[var(--color-accent-dim)] hover:no-underline"
			>
				Browse games
			</a>
			<a
				href={resolve('/settings')}
				class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-5 py-2.5 text-sm text-[var(--color-text)] no-underline hover:bg-[var(--color-surface-2)] hover:no-underline"
			>
				Settings
			</a>
		{:else}
			<a
				href={resolve('/login?next=/games')}
				class="rounded bg-[var(--color-accent)] px-5 py-2.5 text-sm font-medium text-white no-underline hover:bg-[var(--color-accent-dim)] hover:no-underline"
			>
				Login
			</a>
			<a
				href={resolve('/request-access')}
				class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-5 py-2.5 text-sm text-[var(--color-text)] no-underline hover:bg-[var(--color-surface-2)] hover:no-underline"
			>
				Request access
			</a>
		{/if}
	</div>
</section>
