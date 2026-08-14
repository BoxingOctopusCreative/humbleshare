<script lang="ts">
	let { data } = $props();

	const profile = $derived(data.profile);

	const initials = $derived.by(() => {
		const parts = profile.display_name.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
	});

	function formatClaimedAt(iso: string): string {
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return '';
		return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
	}

	function codeTypeLabel(t: string): string {
		if (t === 'gift_link') return 'Gift link';
		if (t === 'steam_key') return 'Steam key';
		return t;
	}
</script>

{#if profile.is_owner && !profile.profile_public}
	<div
		class="mb-4 rounded border border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 py-2 text-sm text-[var(--color-muted)]"
	>
		Private profile. Only you can see this page while signed in.
	</div>
{/if}

<article class="mx-auto max-w-3xl">
	<div class="overflow-hidden rounded border border-[var(--color-border)] bg-[var(--color-surface)]">
		<div class="relative">
			{#if profile.banner_url}
				<img
					src={profile.banner_url}
					alt=""
					class="aspect-[3/1] w-full object-cover bg-[var(--color-surface-2)]"
				/>
			{:else}
				<div
					class="aspect-[3/1] w-full bg-gradient-to-br from-[var(--color-surface-2)] via-[var(--color-surface)] to-[var(--color-bg)]"
					aria-hidden="true"
				></div>
			{/if}
			<div
				class="absolute -bottom-10 left-4 flex h-20 w-20 items-center justify-center overflow-hidden rounded-full border-4 border-[var(--color-surface)] bg-[var(--color-surface-2)] text-lg font-semibold"
			>
				{#if profile.avatar_url}
					<img src={profile.avatar_url} alt="" class="h-full w-full object-cover" />
				{:else}
					<span aria-hidden="true">{initials}</span>
				{/if}
			</div>
		</div>

		<div class="space-y-3 px-4 pb-6 pt-14">
			<div>
				<h1 class="text-2xl font-semibold tracking-tight">{profile.display_name}</h1>
				<p class="text-sm text-[var(--color-muted)]">/u/{profile.profile_slug}</p>
			</div>

			{#if profile.bio.trim()}
				<p class="whitespace-pre-wrap text-sm leading-relaxed text-[var(--color-text)]">
					{profile.bio}
				</p>
			{/if}

			{#if profile.steam_gamertag || profile.discord_tag}
				<ul class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-[var(--color-muted)]">
					{#if profile.steam_gamertag}
						<li>
							<span class="text-[var(--color-text)]">Steam:</span>
							{profile.steam_gamertag}
						</li>
					{/if}
					{#if profile.discord_tag}
						<li>
							<span class="text-[var(--color-text)]">Discord:</span>
							{profile.discord_tag}
						</li>
					{/if}
				</ul>
			{/if}
		</div>
	</div>

	<section class="mt-8">
		<h2 class="text-lg font-semibold">Recent claims</h2>
		{#if profile.recent_claims.length === 0}
			<p class="mt-3 text-sm text-[var(--color-muted)]">No claims yet.</p>
		{:else}
			<ul class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
				{#each profile.recent_claims as claim (claim.game_id + claim.claimed_at)}
					<li
						class="overflow-hidden rounded border border-[var(--color-border)] bg-[var(--color-surface)]"
					>
						{#if claim.header_image_url}
							<img
								src={claim.header_image_url}
								alt=""
								class="aspect-[460/215] w-full object-cover bg-[var(--color-surface-2)]"
								loading="lazy"
							/>
						{:else}
							<div
								class="flex aspect-[460/215] items-center justify-center bg-[var(--color-surface-2)] text-xs text-[var(--color-muted)]"
							>
								No art yet
							</div>
						{/if}
						<div class="space-y-1 p-3">
							<p class="truncate font-medium">{claim.game_name}</p>
							<p class="text-xs text-[var(--color-muted)]">
								{codeTypeLabel(claim.code_type)} · {formatClaimedAt(claim.claimed_at)}
							</p>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</article>
