<script lang="ts">
	import type { ClaimResult, Game } from '$lib/types';
	import ClaimModal from '$lib/components/ClaimModal.svelte';
	import GameCard from '$lib/components/GameCard.svelte';
	import GameDetailModal from '$lib/components/GameDetailModal.svelte';

	let { data } = $props();

	let query = $state('');
	let localGames = $state<Game[] | null>(null);
	let claiming = $state<Game | null>(null);
	let detailing = $state<Game | null>(null);
	let claimResult = $state<ClaimResult | null>(null);
	let claimError = $state('');
	let busy = $state(false);

	const games = $derived(localGames ?? data.games);

	const filtered = $derived(
		query.trim()
			? games.filter((g) => g.name.toLowerCase().includes(query.trim().toLowerCase()))
			: games
	);

	async function refresh() {
		const res = await fetch('/api/games');
		if (res.ok) {
			localGames = (await res.json()) as Game[];
		}
	}

	async function onClaim() {
		if (!claiming) return;
		busy = true;
		claimError = '';
		claimResult = null;
		try {
			const res = await fetch('/api/codes/claim', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ game_id: claiming.id })
			});
			const body = await res.json().catch(() => ({}));
			if (res.status === 409) {
				claimError = body.msg || body.message || 'Someone just grabbed that one.';
				await refresh();
				return;
			}
			if (!res.ok) {
				claimError = body.msg || body.message || 'Claim failed';
				return;
			}
			claimResult = body as ClaimResult;
			await refresh();
		} finally {
			busy = false;
		}
	}

	function openClaim(game: Game) {
		claiming = game;
		claimResult = null;
		claimError = '';
	}

	function closeClaim() {
		claiming = null;
		claimResult = null;
		claimError = '';
	}
</script>

<div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Available games</h1>
		<p class="mt-1 text-sm text-[var(--color-muted)]">
			{filtered.length} with keys ready · claim once, copy once
		</p>
	</div>
	<label class="block w-full sm:w-72">
		<span class="sr-only">Search</span>
		<input
			class="w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
			placeholder="Filter by title…"
			bind:value={query}
		/>
	</label>
</div>

{#if filtered.length === 0}
	<p class="text-[var(--color-muted)]">Nothing available right now.</p>
{:else}
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each filtered as game (game.id)}
			<GameCard
				{game}
				onclaim={() => openClaim(game)}
				ondetail={() => (detailing = game)}
			/>
		{/each}
	</div>
{/if}

{#if detailing}
	<GameDetailModal game={detailing} onclose={() => (detailing = null)} />
{/if}

{#if claiming}
	<ClaimModal
		game={claiming}
		result={claimResult}
		error={claimError}
		{busy}
		onclose={closeClaim}
		onsubmit={onClaim}
	/>
{/if}
