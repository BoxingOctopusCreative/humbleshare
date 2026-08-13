<script lang="ts">
	import type { Game } from '$lib/types';

	let { game, onclose }: { game: Game; onclose: () => void } = $props();

	const slides = $derived.by(() => {
		const urls: string[] = [];
		const seen = new Set<string>();
		const push = (url: string | null | undefined) => {
			if (!url || seen.has(url)) return;
			seen.add(url);
			urls.push(url);
		};
		push(game.header_image_url);
		for (const url of game.screenshot_urls ?? []) push(url);
		return urls;
	});

	let index = $state(0);

	const active = $derived(slides.length ? ((index % slides.length) + slides.length) % slides.length : 0);
	const current = $derived(slides.length ? slides[active] : null);
	const multi = $derived(slides.length > 1);

	function prev(e?: Event) {
		e?.stopPropagation();
		e?.preventDefault();
		if (!multi) return;
		index = (active - 1 + slides.length) % slides.length;
	}

	function next(e?: Event) {
		e?.stopPropagation();
		e?.preventDefault();
		if (!multi) return;
		index = (active + 1) % slides.length;
	}

	function goTo(i: number, e?: Event) {
		e?.stopPropagation();
		e?.preventDefault();
		index = i;
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onclose();
			return;
		}
		if (e.key === 'ArrowLeft') prev();
		if (e.key === 'ArrowRight') next();
	}
</script>

<svelte:window onkeydown={onKeydown} />

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
	role="dialog"
	aria-modal="true"
	aria-labelledby="game-detail-title"
>
	<button
		type="button"
		class="absolute inset-0 cursor-default"
		aria-label="Close"
		onclick={onclose}
	></button>
	<div
		class="relative z-10 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded border border-[var(--color-border)] bg-[var(--color-surface)] shadow-xl"
	>
		{#if current}
			<div class="relative isolate bg-black">
				<img
					src={current}
					alt=""
					class="aspect-[16/9] w-full object-contain"
				/>

				{#if multi}
					<button
						type="button"
						class="group absolute inset-y-0 left-0 z-20 flex w-14 items-center justify-start bg-gradient-to-r from-black/35 to-transparent pl-2 sm:w-16 sm:pl-3"
						aria-label="Previous screenshot"
						onclick={prev}
					>
						<span
							class="flex size-10 items-center justify-center rounded-full bg-black/35 text-white opacity-55 shadow-md transition group-hover:bg-black/55 group-hover:opacity-95 sm:size-11"
						>
							<svg
								class="size-6"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2.5"
								stroke-linecap="round"
								stroke-linejoin="round"
								aria-hidden="true"
							>
								<path d="M15 18l-6-6 6-6" />
							</svg>
						</span>
					</button>

					<button
						type="button"
						class="group absolute inset-y-0 right-0 z-20 flex w-14 items-center justify-end bg-gradient-to-l from-black/35 to-transparent pr-2 sm:w-16 sm:pr-3"
						aria-label="Next screenshot"
						onclick={next}
					>
						<span
							class="flex size-10 items-center justify-center rounded-full bg-black/35 text-white opacity-55 shadow-md transition group-hover:bg-black/55 group-hover:opacity-95 sm:size-11"
						>
							<svg
								class="size-6"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2.5"
								stroke-linecap="round"
								stroke-linejoin="round"
								aria-hidden="true"
							>
								<path d="M9 18l6-6-6-6" />
							</svg>
						</span>
					</button>

					<div
						class="absolute bottom-2 left-0 right-0 z-20 flex justify-center gap-1.5"
						role="tablist"
						aria-label="Screenshots"
					>
						{#each slides as _, i (i)}
							<button
								type="button"
								role="tab"
								aria-selected={i === active}
								aria-label={`Screenshot ${i + 1}`}
								class="size-2.5 rounded-full border-0 {i === active
									? 'bg-white opacity-95'
									: 'bg-white/40 opacity-70 hover:opacity-100'}"
								onclick={(e) => goTo(i, e)}
							></button>
						{/each}
					</div>

					<p
						class="pointer-events-none absolute right-3 top-2 z-20 rounded bg-black/50 px-2 py-0.5 text-xs text-white/90"
					>
						{active + 1} / {slides.length}
					</p>
				{/if}
			</div>
		{:else}
			<div
				class="flex aspect-[16/9] items-center justify-center bg-[var(--color-surface-2)] text-sm text-[var(--color-muted)]"
			>
				No art yet
			</div>
		{/if}

		<div class="p-5">
			<h2 id="game-detail-title" class="text-lg font-semibold leading-snug">{game.name}</h2>

			{#if game.short_description}
				<p class="mt-3 text-sm leading-relaxed text-[var(--color-muted)]">
					{game.short_description}
				</p>
			{:else}
				<p class="mt-3 text-sm text-[var(--color-muted)]">No description yet.</p>
			{/if}

			<div class="mt-4 flex flex-wrap items-center justify-between gap-2">
				{#if game.store_url}
					<a
						href={game.store_url}
						target="_blank"
						rel="noopener noreferrer external"
						class="text-sm no-underline hover:underline"
					>
						View on Steam
					</a>
				{:else}
					<span></span>
				{/if}
				<button
					type="button"
					class="rounded border border-[var(--color-border)] px-3 py-1.5 text-sm text-[var(--color-text)]"
					onclick={onclose}
				>
					Close
				</button>
			</div>
		</div>
	</div>
</div>
