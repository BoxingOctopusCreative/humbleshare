<script lang="ts">
	import type { ClaimResult, Game } from '$lib/types';

	let {
		game,
		result,
		error,
		busy,
		onclose,
		onsubmit
	}: {
		game: Game;
		result: ClaimResult | null;
		error: string;
		busy: boolean;
		onclose: () => void;
		onsubmit: () => void | Promise<void>;
	} = $props();

	let copied = $state(false);
	let reporting = $state(false);
	let reported = $state(false);
	let reportError = $state('');

	async function copy() {
		if (!result) return;
		await navigator.clipboard.writeText(result.code_value);
		copied = true;
		setTimeout(() => (copied = false), 1500);
	}

	async function reportBroken() {
		if (!result || reporting || reported) return;
		const ok = confirm(
			"Report this as broken? Use this if the gift link expired, the page didn't load, or the Steam key didn't redeem."
		);
		if (!ok) return;
		reporting = true;
		reportError = '';
		try {
			const res = await fetch(`/api/codes/${result.id}/report-dead`, { method: 'POST' });
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				reportError = body.msg || body.message || 'Could not report this key.';
				return;
			}
			reported = true;
		} finally {
			reporting = false;
		}
	}

	const redeemUrl = $derived(
		result?.code_type === 'steam_key'
			? `https://store.steampowered.com/account/registerkey?key=${encodeURIComponent(result.code_value)}`
			: result?.code_value
	);
</script>

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
	role="dialog"
	aria-modal="true"
	aria-labelledby="claim-title"
>
	<button
		type="button"
		class="absolute inset-0 cursor-default"
		aria-label="Close"
		onclick={onclose}
	></button>
	<div
		class="relative w-full max-w-md rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-5 shadow-xl"
	>
		<h2 id="claim-title" class="text-lg font-semibold">{game.name}</h2>

		{#if result}
			<p class="mt-2 text-sm text-[var(--color-danger)]">
				This is the only time you'll see this code. Copy it now.
			</p>
			<div
				class="mt-3 break-all rounded border border-[var(--color-border)] bg-[var(--color-bg)] p-3 font-mono text-sm"
			>
				{result.code_value}
			</div>
			<div class="mt-3 flex flex-wrap gap-2">
				<button
					type="button"
					class="rounded bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white"
					onclick={copy}
				>
					{copied ? 'Copied' : 'Copy'}
				</button>
				{#if redeemUrl}
					<a
						class="rounded border border-[var(--color-border)] px-3 py-1.5 text-sm no-underline"
						href={redeemUrl}
						target="_blank"
						rel="noopener noreferrer external"
					>
						{result.code_type === 'steam_key' ? 'Redeem on Steam' : 'Open gift link'}
					</a>
				{/if}
				<button
					type="button"
					class="rounded px-3 py-1.5 text-sm text-[var(--color-muted)]"
					onclick={onclose}
				>
					Done
				</button>
			</div>
			{#if reported}
				<p class="mt-3 text-sm text-[var(--color-muted)]">
					Thanks. We marked this as broken so it won't count as a working key.
				</p>
			{:else}
				<p class="mt-4 text-xs text-[var(--color-muted)]">
					Gift link expired or Steam key didn't redeem?
				</p>
				<button
					type="button"
					class="mt-1 text-sm text-[var(--color-danger)]"
					disabled={reporting}
					onclick={reportBroken}
				>
					{reporting ? 'Reporting…' : 'Report Game'}
				</button>
				{#if reportError}
					<p class="mt-1 text-sm text-[var(--color-danger)]">{reportError}</p>
				{/if}
			{/if}
		{:else}
			<p class="mt-2 text-sm text-[var(--color-muted)]">
				Claim this key under your account. You'll only see it once.
			</p>
			{#if error}
				<p class="mt-2 text-sm text-[var(--color-danger)]">{error}</p>
			{/if}
			<div class="mt-4 flex justify-end gap-2">
				<button type="button" class="px-3 py-1.5 text-sm text-[var(--color-muted)]" onclick={onclose}>
					Cancel
				</button>
				<button
					type="button"
					class="rounded bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white"
					disabled={busy}
					onclick={() => onsubmit()}
				>
					{busy ? 'Claiming…' : 'Claim key'}
				</button>
			</div>
		{/if}
	</div>
</div>
