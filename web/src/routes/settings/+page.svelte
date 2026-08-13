<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import ImageCropModal from '$lib/components/ImageCropModal.svelte';
	import OAuthIcon from '$lib/components/OAuthIcon.svelte';

	let { data } = $props();

	let displayNameEdit = $state<string | null>(null);
	let bioEdit = $state<string | null>(null);
	let steamGamertagEdit = $state<string | null>(null);
	let discordTagEdit = $state<string | null>(null);
	let avatarUrlEdit = $state<string | null>(null);
	let bannerUrlEdit = $state<string | null>(null);
	let profileSlugEdit = $state<string | null>(null);
	let profilePublicEdit = $state<boolean | null>(null);

	const displayName = $derived(displayNameEdit ?? data.user.display_name ?? '');
	const bio = $derived(bioEdit ?? data.user.bio ?? '');
	const steamGamertag = $derived(steamGamertagEdit ?? data.user.steam_gamertag ?? '');
	const discordTag = $derived(discordTagEdit ?? data.user.discord_tag ?? '');
	const avatarUrl = $derived(avatarUrlEdit ?? data.user.avatar_url ?? '');
	const bannerUrl = $derived(bannerUrlEdit ?? data.user.banner_url ?? '');
	const profilePublic = $derived(profilePublicEdit ?? data.user.profile_public ?? false);

	let profileMsg = $state('');
	let profileErr = $state('');
	let avatarErr = $state('');
	let avatarMsg = $state('');
	let bannerErr = $state('');
	let bannerMsg = $state('');

	let cropOpen = $state(false);
	let cropKind = $state<'avatar' | 'banner'>('avatar');
	let cropFile = $state<File | null>(null);

	const cropAspect = $derived(cropKind === 'avatar' ? 1 : 3);
	const cropTitle = $derived(cropKind === 'avatar' ? 'Crop avatar' : 'Crop banner');
	const cropOutput = $derived(
		cropKind === 'avatar' ? { width: 512, height: 512 } : { width: 1500, height: 500 }
	);
	const cropMaxBytes = $derived(cropKind === 'avatar' ? 2 * 1024 * 1024 : 4 * 1024 * 1024);

	let pwCurrent = $state('');
	let pwNext = $state('');
	let pwMsg = $state('');
	let pwErr = $state('');

	let totpSecret = $state('');
	let totpUrl = $state('');
	let totpCode = $state('');
	let totpMsg = $state('');
	let totpErr = $state('');
	let busy = $state(false);

	const linked = $derived(new Set((data.oauth ?? []).map((o) => o.provider)));

	const providers = [
		{ id: 'google' as const, label: 'Google' },
		{ id: 'discord' as const, label: 'Discord' },
		{ id: 'steam' as const, label: 'Steam' }
	];

	const profileInitials = $derived.by(() => {
		const parts = displayName.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
	});

	const slugPattern = '^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$';

	function slugFromDisplayName(name: string): string {
		const raw = name
			.trim()
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 32)
			.replace(/^-+|-+$/g, '');
		return raw.length >= 3 ? raw : 'user';
	}

	const suggestedSlug = $derived(slugFromDisplayName(displayName));

	const profileSlug = $derived(
		profileSlugEdit !== null ? profileSlugEdit : data.user.profile_slug || suggestedSlug
	);

	async function saveProfile() {
		busy = true;
		profileMsg = '';
		profileErr = '';
		try {
			const res = await fetch('/api/me', {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					display_name: displayName,
					bio,
					steam_gamertag: steamGamertag,
					discord_tag: discordTag,
					profile_slug: profileSlug,
					profile_public: profileSlug.trim() ? profilePublic : false
				})
			});
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				profileErr = body.msg || body.message || 'Profile update failed';
				return;
			}
			displayNameEdit = body.display_name ?? displayName;
			bioEdit = body.bio ?? bio;
			steamGamertagEdit = body.steam_gamertag ?? steamGamertag;
			discordTagEdit = body.discord_tag ?? discordTag;
			profileSlugEdit = body.profile_slug ?? profileSlug;
			profilePublicEdit = Boolean(body.profile_public);
			profileMsg = 'Profile saved.';
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	function openCrop(kind: 'avatar' | 'banner', file: File) {
		cropKind = kind;
		cropFile = file;
		cropOpen = true;
		if (kind === 'avatar') {
			avatarErr = '';
			avatarMsg = '';
		} else {
			bannerErr = '';
			bannerMsg = '';
		}
	}

	function closeCrop() {
		cropOpen = false;
		cropFile = null;
	}

	function onAvatarSelected(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file) return;
		openCrop('avatar', file);
	}

	async function removeAvatar() {
		busy = true;
		avatarErr = '';
		avatarMsg = '';
		try {
			const res = await fetch('/api/me/avatar', { method: 'DELETE' });
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				avatarErr = body.msg || body.message || 'Could not remove avatar';
				return;
			}
			avatarUrlEdit = '';
			avatarMsg = 'Avatar removed.';
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	function onBannerSelected(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file) return;
		openCrop('banner', file);
	}

	async function uploadCropped(file: File) {
		busy = true;
		const kind = cropKind;
		try {
			const form = new FormData();
			form.set('file', file);
			const endpoint = kind === 'avatar' ? '/api/me/avatar' : '/api/me/banner';
			const res = await fetch(endpoint, { method: 'POST', body: form });
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				const msg = body.msg || body.message || 'Upload failed';
				if (kind === 'avatar') avatarErr = msg;
				else bannerErr = msg;
				return;
			}
			if (kind === 'avatar') {
				avatarUrlEdit = body.avatar_url ?? '';
				avatarMsg = 'Avatar updated.';
			} else {
				bannerUrlEdit = body.banner_url ?? '';
				bannerMsg = 'Banner updated.';
			}
			closeCrop();
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	async function removeBanner() {
		busy = true;
		bannerErr = '';
		bannerMsg = '';
		try {
			const res = await fetch('/api/me/banner', { method: 'DELETE' });
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				bannerErr = body.msg || body.message || 'Could not remove banner';
				return;
			}
			bannerUrlEdit = '';
			bannerMsg = 'Banner removed.';
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	async function changePassword() {
		busy = true;
		pwMsg = '';
		pwErr = '';
		try {
			const res = await fetch('/api/me/password', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ current_password: pwCurrent, new_password: pwNext })
			});
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				pwErr = body.msg || body.message || 'Password change failed';
				return;
			}
			pwMsg = 'Password updated.';
			pwCurrent = '';
			pwNext = '';
		} finally {
			busy = false;
		}
	}

	async function setup2fa() {
		busy = true;
		totpErr = '';
		totpMsg = '';
		try {
			const res = await fetch('/api/me/2fa/setup', { method: 'POST' });
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				totpErr = body.msg || body.message || 'Setup failed';
				return;
			}
			totpSecret = body.secret ?? '';
			totpUrl = body.otpauth_url ?? '';
		} finally {
			busy = false;
		}
	}

	async function enable2fa() {
		busy = true;
		totpErr = '';
		totpMsg = '';
		try {
			const res = await fetch('/api/me/2fa/enable', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ code: totpCode })
			});
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				totpErr = body.msg || body.message || 'Enable failed';
				return;
			}
			totpMsg = 'Two-factor authentication enabled.';
			totpCode = '';
			totpSecret = '';
			totpUrl = '';
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	async function disable2fa() {
		busy = true;
		totpErr = '';
		totpMsg = '';
		try {
			const res = await fetch('/api/me/2fa/disable', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ code: totpCode })
			});
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				totpErr = body.msg || body.message || 'Disable failed';
				return;
			}
			totpMsg = 'Two-factor authentication disabled.';
			totpCode = '';
			await invalidateAll();
		} finally {
			busy = false;
		}
	}
</script>

<div class="mx-auto max-w-5xl space-y-10">
	<div>
		<h1 class="text-2xl font-semibold">Settings</h1>
		<p class="mt-1 text-sm text-[var(--color-muted)]">
			{data.user.email} · {data.user.role}
		</p>
	</div>

	<section>
		<h2 class="font-semibold">Profile</h2>
		{#if profileErr}<p class="mt-2 text-sm text-[var(--color-danger)]">{profileErr}</p>{/if}
		{#if profileMsg}<p class="mt-2 text-sm text-[var(--color-ok)]">{profileMsg}</p>{/if}
		{#if avatarErr}<p class="mt-2 text-sm text-[var(--color-danger)]">{avatarErr}</p>{/if}
		{#if avatarMsg}<p class="mt-2 text-sm text-[var(--color-ok)]">{avatarMsg}</p>{/if}
		{#if bannerErr}<p class="mt-2 text-sm text-[var(--color-danger)]">{bannerErr}</p>{/if}
		{#if bannerMsg}<p class="mt-2 text-sm text-[var(--color-ok)]">{bannerMsg}</p>{/if}

		{#if profileSlug.trim()}
			<p class="mt-2 text-sm">
				<a
					href={resolve('/u/[slug]', { slug: profileSlug.trim().toLowerCase() })}
					class="text-[var(--color-accent)] no-underline hover:underline"
				>
					View profile
				</a>
			</p>
		{/if}

		<div class="mt-3 overflow-hidden rounded border border-[var(--color-border)] bg-[var(--color-surface-2)]">
			{#if bannerUrl}
				<img src={bannerUrl} alt="" class="aspect-[3/1] w-full object-cover" />
			{:else}
				<div class="flex aspect-[3/1] items-center justify-center text-xs text-[var(--color-muted)]">
					No banner yet
				</div>
			{/if}
		</div>
		<div class="mt-2 space-y-2 text-sm">
			<label class="inline-block cursor-pointer text-[var(--color-accent)] hover:underline">
				<input
					type="file"
					accept="image/jpeg,image/png,image/webp"
					class="sr-only"
					disabled={busy}
					onchange={onBannerSelected}
				/>
				Upload banner
			</label>
			{#if bannerUrl}
				<button
					type="button"
					class="ml-3 text-[var(--color-muted)] hover:text-[var(--color-danger)]"
					disabled={busy}
					onclick={removeBanner}
				>
					Remove banner
				</button>
			{/if}
			<p class="text-xs text-[var(--color-muted)]">
				Wide image works best. Crop and zoom before upload. JPEG, PNG, or WebP. Max 4MB.
			</p>
		</div>

		<div class="mt-4 flex items-center gap-4">
			<div
				class="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-full border border-[var(--color-border)] bg-[var(--color-surface-2)] text-sm font-semibold"
			>
				{#if avatarUrl}
					<img src={avatarUrl} alt="" class="h-full w-full object-cover" />
				{:else}
					<span aria-hidden="true">{profileInitials}</span>
				{/if}
			</div>
			<div class="space-y-2 text-sm">
				<label class="inline-block cursor-pointer text-[var(--color-accent)] hover:underline">
					<input
						type="file"
						accept="image/jpeg,image/png,image/webp"
						class="sr-only"
						disabled={busy}
						onchange={onAvatarSelected}
					/>
					Upload avatar
				</label>
				{#if avatarUrl}
					<button
						type="button"
						class="block text-[var(--color-muted)] hover:text-[var(--color-danger)]"
						disabled={busy}
						onclick={removeAvatar}
					>
						Remove avatar
					</button>
				{/if}
				<p class="text-xs text-[var(--color-muted)]">
					Crop and zoom before upload. JPEG, PNG, or WebP. Max 2MB.
				</p>
			</div>
		</div>

		<form
			class="mt-4 space-y-3"
			onsubmit={(e) => {
				e.preventDefault();
				saveProfile();
			}}
		>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Display name</span>
				<input
					type="text"
					value={displayName}
					oninput={(e) => {
						displayNameEdit = e.currentTarget.value;
					}}
					required
					maxlength="64"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Profile URL</span>
				<div class="mt-1 flex items-stretch">
					<span
						class="inline-flex shrink-0 items-center whitespace-nowrap rounded-l border border-r-0 border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 font-mono text-sm text-[var(--color-muted)]"
						>/u/</span
					>
					<input
						type="text"
						value={profileSlug}
						oninput={(e) => {
							profileSlugEdit = e.currentTarget.value.toLowerCase();
							if (!e.currentTarget.value.trim()) {
								profilePublicEdit = false;
							}
						}}
						maxlength="32"
						pattern={slugPattern}
						placeholder={suggestedSlug}
						class="min-w-0 flex-1 rounded-r border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
					/>
				</div>
				<p class="mt-1 text-xs text-[var(--color-muted)]">
					Starts from your display name. You can change it anytime. 3–32 characters;
					lowercase letters, numbers, and hyphens.
				</p>
			</label>
			<label class="flex items-start gap-2 text-sm">
				<input
					type="checkbox"
					class="mt-1"
					checked={profilePublic}
					disabled={!profileSlug.trim()}
					onchange={(e) => {
						profilePublicEdit = e.currentTarget.checked;
					}}
				/>
				<span>
					<span class="block">Public profile</span>
					<span class="text-xs text-[var(--color-muted)]">
						{#if !profileSlug.trim()}
							Set a profile URL before making your profile public.
						{:else if profilePublic}
							Anyone with the link can view your profile.
						{:else}
							Only you can open your profile while signed in.
						{/if}
					</span>
				</span>
			</label>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Bio</span>
				<textarea
					value={bio}
					oninput={(e) => {
						bioEdit = e.currentTarget.value;
					}}
					maxlength="500"
					rows="3"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				></textarea>
			</label>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Steam gamertag</span>
				<input
					type="text"
					value={steamGamertag}
					oninput={(e) => {
						steamGamertagEdit = e.currentTarget.value;
					}}
					maxlength="64"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Discord tag</span>
				<input
					type="text"
					value={discordTag}
					oninput={(e) => {
						discordTagEdit = e.currentTarget.value;
					}}
					maxlength="64"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<button
				type="submit"
				disabled={busy}
				class="rounded bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white"
			>
				Save profile
			</button>
		</form>
	</section>

	<section>
		<h2 class="font-semibold">Password</h2>
		{#if pwErr}<p class="mt-2 text-sm text-[var(--color-danger)]">{pwErr}</p>{/if}
		{#if pwMsg}<p class="mt-2 text-sm text-[var(--color-ok)]">{pwMsg}</p>{/if}
		<form
			class="mt-3 space-y-3"
			onsubmit={(e) => {
				e.preventDefault();
				changePassword();
			}}
		>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Current password</span>
				<input
					type="password"
					bind:value={pwCurrent}
					required
					autocomplete="current-password"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">New password</span>
				<input
					type="password"
					bind:value={pwNext}
					required
					minlength="8"
					autocomplete="new-password"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<button
				type="submit"
				disabled={busy}
				class="rounded bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white"
			>
				Update password
			</button>
		</form>
	</section>

	<section>
		<h2 class="font-semibold">Two-factor authentication</h2>
		<p class="mt-1 text-sm text-[var(--color-muted)]">
			Status: {data.user.two_factor_enabled ? 'enabled' : 'disabled'}
		</p>
		{#if totpErr}<p class="mt-2 text-sm text-[var(--color-danger)]">{totpErr}</p>{/if}
		{#if totpMsg}<p class="mt-2 text-sm text-[var(--color-ok)]">{totpMsg}</p>{/if}

		{#if !data.user.two_factor_enabled}
			{#if !totpSecret}
				<button
					type="button"
					class="mt-3 rounded border border-[var(--color-border)] px-3 py-1.5 text-sm"
					disabled={busy}
					onclick={setup2fa}
				>
					Set up authenticator
				</button>
			{:else}
				<div class="mt-3 space-y-2 text-sm">
					<p class="text-[var(--color-muted)]">
						Add this secret in your authenticator app, then enter a code to enable.
					</p>
					<code
						class="block break-all rounded border border-[var(--color-border)] bg-[var(--color-bg)] p-2 font-mono text-xs"
						>{totpSecret}</code
					>
					{#if totpUrl}
						<a href={totpUrl} rel="external" class="text-xs text-[var(--color-accent)]"
							>Open otpauth link</a
						>
					{/if}
					<label class="block">
						<span class="text-[var(--color-muted)]">Code</span>
						<input
							bind:value={totpCode}
							class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 font-mono text-sm"
							autocomplete="one-time-code"
						/>
					</label>
					<button
						type="button"
						class="rounded bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white"
						disabled={busy || !totpCode.trim()}
						onclick={enable2fa}
					>
						Enable 2FA
					</button>
				</div>
			{/if}
		{:else}
			<div class="mt-3 space-y-2">
				<label class="block text-sm">
					<span class="text-[var(--color-muted)]">Code to disable</span>
					<input
						bind:value={totpCode}
						class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 font-mono text-sm"
						autocomplete="one-time-code"
					/>
				</label>
				<button
					type="button"
					class="rounded border border-[var(--color-danger)] px-3 py-1.5 text-sm text-[var(--color-danger)]"
					disabled={busy || !totpCode.trim()}
					onclick={disable2fa}
				>
					Disable 2FA
				</button>
			</div>
		{/if}
	</section>

	<section>
		<h2 class="font-semibold">Linked accounts</h2>
		<p class="mt-1 text-sm text-[var(--color-muted)]">Link Google, Discord, or Steam for sign-in.</p>
		<ul class="mt-3 space-y-2">
			{#each providers as provider (provider.id)}
				<li
					class="flex items-center justify-between rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm"
				>
					<span class="inline-flex items-center gap-2">
						<OAuthIcon provider={provider.id} />
						{provider.label}
					</span>
					{#if linked.has(provider.id)}
						<span class="text-[var(--color-ok)]">Linked</span>
					{:else}
						<a
							href={resolve(`/api/auth/oauth/${provider.id}?mode=link`)}
							class="text-[var(--color-accent)] no-underline hover:underline"
						>
							Link
						</a>
					{/if}
				</li>
			{/each}
		</ul>
	</section>
</div>

{#if cropOpen}
	<ImageCropModal
		open={cropOpen}
		file={cropFile}
		aspect={cropAspect}
		title={cropTitle}
		outputSize={cropOutput}
		maxBytes={cropMaxBytes}
		{busy}
		oncancel={closeCrop}
		onconfirm={uploadCropped}
	/>
{/if}
