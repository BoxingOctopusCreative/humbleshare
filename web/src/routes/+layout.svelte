<script lang="ts">
	import '../app.css';
	import { resolve } from '$app/paths';
	import SentryClientInit from '$lib/components/SentryClientInit.svelte';
	import UserMenu from '$lib/components/UserMenu.svelte';

	const favicon = 'https://assets.humbleshare.com/SVG/favicon.svg';
	const logoWide = 'https://assets.humbleshare.com/SVG/humbleshare_wide.svg';

	let { children, data } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} type="image/svg+xml" />
	<title>HumbleShare</title>
</svelte:head>

<SentryClientInit dsn={data.sentryDsn} />

<div class="flex h-dvh flex-col overflow-hidden">
	<header
		class="sticky top-0 z-20 shrink-0 border-b border-[var(--color-border)] bg-[var(--color-surface)]/80 shadow-[0_4px_16px_rgba(0,0,0,0.35)] backdrop-blur"
	>
		<div class="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-3">
			<a href={resolve('/')} class="flex items-center no-underline hover:no-underline">
				<img
					src={logoWide}
					alt="HumbleShare"
					width="141"
					height="32"
					class="h-8 w-auto drop-shadow-[0_2px_0_rgba(0,0,0,0.45)] drop-shadow-[0_4px_12px_rgba(0,0,0,0.35)]"
				/>
			</a>
			<nav class="flex flex-wrap items-center gap-3 text-sm text-[var(--color-muted)]">
				{#if data.user}
					<a href={resolve('/games')} class="no-underline hover:underline">Games</a>
					{#if data.hasAdmin}
						<a href={resolve('/admin')} class="text-[var(--color-admin)] no-underline hover:underline"
							>Admin</a
						>
					{/if}
					<UserMenu user={data.user} />
				{:else}
					<a href={resolve('/login?next=/games')} class="no-underline hover:underline">Login</a>
					<a href={resolve('/request-access')} class="no-underline hover:underline">Request access</a>
				{/if}
			</nav>
		</div>
	</header>
	<main class="relative z-10 min-h-0 flex-1 overflow-y-auto">
		<div class="mx-auto w-full max-w-6xl px-4 py-6">
			{@render children()}
		</div>
	</main>
	<footer
		class="sticky bottom-0 z-20 shrink-0 border-t border-[var(--color-border)] bg-[var(--color-surface)]/80 shadow-[0_-4px_16px_rgba(0,0,0,0.25)] backdrop-blur"
	>
		<div
			class="mx-auto flex max-w-6xl flex-col gap-2 px-4 py-3 text-xs text-[var(--color-muted)] sm:py-4"
		>
			<div class="flex flex-wrap items-center justify-between gap-3">
				<span>
					© {new Date().getFullYear()}
					<a
						href="https://boxingoctop.us"
						target="_blank"
						rel="noopener noreferrer"
						class="no-underline hover:underline"
					>
						Boxing Octopus Creative
					</a>
				</span>
				<nav class="flex flex-wrap gap-3">
					<a href={resolve('/privacy')} class="no-underline hover:underline">Privacy</a>
					<a href={resolve('/terms')} class="no-underline hover:underline">Terms</a>
					<a href={resolve('/sitemap')} class="no-underline hover:underline">Sitemap</a>
				</nav>
			</div>
			<p class="m-0 max-w-3xl leading-snug">
				HumbleShare is not affiliated with, endorsed by, or sponsored by Humble Bundle Inc., IGN
				Entertainment Inc., or Ziff Davis, LLC.
			</p>
		</div>
	</footer>
</div>
