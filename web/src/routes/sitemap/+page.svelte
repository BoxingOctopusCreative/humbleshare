<script lang="ts">
	import { base, resolve } from '$app/paths';

	type SiteLink = {
		href: '/' | '/login' | '/request-access' | '/privacy' | '/terms' | '/sitemap' | '/games' | '/settings' | '/admin';
		label: string;
		note: string;
	};

	const sitemapXmlHref = `${base}/sitemap.xml`;

	const publicPages: SiteLink[] = [
		{ href: '/', label: 'Home', note: 'Landing page' },
		{ href: '/login', label: 'Login', note: 'Sign in' },
		{ href: '/request-access', label: 'Request access', note: 'Ask for an invite' },
		{ href: '/privacy', label: 'Privacy policy', note: 'How data is handled' },
		{ href: '/terms', label: 'Terms of use', note: 'Rules for using the site' },
		{ href: '/sitemap', label: 'Sitemap', note: 'This page' }
	];

	const memberPages: SiteLink[] = [
		{ href: '/games', label: 'Games', note: 'Browse and claim spare keys' },
		{ href: '/settings', label: 'Settings', note: 'Profile, password, 2FA, OAuth' }
	];

	const adminPages: SiteLink[] = [
		{ href: '/admin', label: 'Admin', note: 'Import, inventory, users, and stats' }
	];
</script>

<svelte:head>
	<link rel="alternate" type="application/xml" title="XML sitemap" href={sitemapXmlHref} />
</svelte:head>

<article class="mx-auto max-w-2xl">
	<header class="mb-8 border-b border-[var(--color-border)] pb-6">
		<h1 class="text-3xl font-semibold tracking-tight">Sitemap</h1>
		<p class="mt-2 text-sm text-[var(--color-muted)]">
			Public pages plus signed-in routes. Crawlers can fetch
			<code class="font-mono text-xs text-[var(--color-text)]">{sitemapXmlHref}</code>.
		</p>
	</header>

	<section class="space-y-8 text-sm">
		<div>
			<h2 class="mb-3 text-base font-semibold">Public</h2>
			<ul class="divide-y divide-[var(--color-border)] rounded border border-[var(--color-border)]">
				{#each publicPages as page (page.href)}
					<li class="flex flex-wrap items-baseline justify-between gap-2 px-3 py-2.5">
						<a href={resolve(page.href)} class="font-medium">{page.label}</a>
						<span class="text-xs text-[var(--color-muted)]">{page.note}</span>
					</li>
				{/each}
				<li class="flex flex-wrap items-baseline justify-between gap-2 px-3 py-2.5">
					<span class="font-medium text-[var(--color-text)]">/u/[slug]</span>
					<span class="text-xs text-[var(--color-muted)]"
						>Public profiles when the owner opts in</span
					>
				</li>
			</ul>
		</div>

		<div>
			<h2 class="mb-3 text-base font-semibold">Signed in</h2>
			<ul class="divide-y divide-[var(--color-border)] rounded border border-[var(--color-border)]">
				{#each memberPages as page (page.href)}
					<li class="flex flex-wrap items-baseline justify-between gap-2 px-3 py-2.5">
						<a href={resolve(page.href)} class="font-medium">{page.label}</a>
						<span class="text-xs text-[var(--color-muted)]">{page.note}</span>
					</li>
				{/each}
			</ul>
		</div>

		<div>
			<h2 class="mb-3 text-base font-semibold">Admins</h2>
			<ul class="divide-y divide-[var(--color-border)] rounded border border-[var(--color-border)]">
				{#each adminPages as page (page.href)}
					<li class="flex flex-wrap items-baseline justify-between gap-2 px-3 py-2.5">
						<a href={resolve(page.href)} class="font-medium">{page.label}</a>
						<span class="text-xs text-[var(--color-muted)]">{page.note}</span>
					</li>
				{/each}
			</ul>
		</div>
	</section>
</article>
