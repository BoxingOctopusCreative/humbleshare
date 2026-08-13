<script lang="ts">
	import type { Game } from '$lib/types';

	let {
		game,
		onclaim,
		ondetail
	}: { game: Game; onclaim: () => void; ondetail: () => void } = $props();
</script>

<article
	class="overflow-hidden rounded border border-[var(--color-border)] bg-[var(--color-surface)]"
>
	<button
		type="button"
		class="block w-full cursor-pointer border-0 bg-transparent p-0 text-left"
		aria-label={`Details for ${game.name}`}
		onclick={ondetail}
	>
		{#if game.header_image_url}
			<img
				src={game.header_image_url}
				alt=""
				class="aspect-[460/215] w-full object-cover bg-[var(--color-surface-2)]"
				loading="lazy"
			/>
		{:else}
			<div
				class="flex aspect-[460/215] items-center justify-center bg-[var(--color-surface-2)] text-sm text-[var(--color-muted)]"
			>
				No art yet
			</div>
		{/if}
	</button>
	<div class="flex items-start justify-between gap-3 p-3">
		<div class="min-w-0">
			<button
				type="button"
				class="block max-w-full truncate text-left font-medium leading-snug text-[var(--color-text)] hover:text-[var(--color-accent)] hover:underline"
				onclick={ondetail}
			>
				{game.name}
			</button>
			<p class="mt-0.5 text-xs text-[var(--color-muted)]">
				{game.available_count} available
			</p>
		</div>
		<button
			type="button"
			class="shrink-0 rounded bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white hover:bg-[var(--color-accent-dim)]"
			onclick={onclaim}
		>
			Claim
		</button>
	</div>
</article>
