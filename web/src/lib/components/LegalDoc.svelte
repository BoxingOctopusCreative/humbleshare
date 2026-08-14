<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		updated,
		children
	}: {
		title: string;
		updated: string;
		children: Snippet;
	} = $props();
</script>

<article class="mx-auto max-w-2xl">
	<header class="mb-8 border-b border-[var(--color-border)] pb-6">
		<h1 class="text-3xl font-semibold tracking-tight">{title}</h1>
		<p class="mt-2 text-sm text-[var(--color-muted)]">Last updated {updated}</p>
	</header>
	<div class="legal-body space-y-6 text-sm leading-relaxed text-[var(--color-text)]">
		{@render children()}
	</div>
</article>

<style>
	:global(.legal-body h2) {
		margin-top: 1.75rem;
		font-size: 1.05rem;
		font-weight: 700;
		color: var(--color-text);
	}
	:global(.legal-body p),
	:global(.legal-body li) {
		color: var(--color-muted);
	}
	:global(.legal-body ul) {
		list-style: disc;
		padding-left: 1.25rem;
	}
	:global(.legal-body ul li + li) {
		margin-top: 0.35rem;
	}
	:global(.legal-body strong) {
		color: var(--color-text);
		font-weight: 700;
	}
</style>
