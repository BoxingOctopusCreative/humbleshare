<script lang="ts">
	import { resolve } from '$app/paths';
	import OAuthIcon from '$lib/components/OAuthIcon.svelte';
	import SplashBackdrop from '$lib/components/SplashBackdrop.svelte';

	let { data, form } = $props();

	const requires2fa = $derived(Boolean(form?.requires2fa));
	const error = $derived(form?.error ?? '');
	const email = $derived(form?.email ?? '');
	const next = $derived(form?.next ?? data.next);
	const pendingToken = $derived(form?.pendingToken ?? '');

	const providers = [
		{ id: 'google' as const, label: 'Google' },
		{ id: 'discord' as const, label: 'Discord' },
		{ id: 'steam' as const, label: 'Steam' }
	];
</script>

<SplashBackdrop />

<div class="relative z-10 mx-auto max-w-md">
	<h1 class="text-2xl font-semibold">Login</h1>
	<p class="mt-2 text-sm text-[var(--color-muted)]">
		Sign in with your HumbleShare account.
	</p>

	{#if error}
		<p class="mt-3 text-sm text-[var(--color-danger)]">{error}</p>
	{/if}

	{#if requires2fa}
		<form method="POST" action="?/login2fa" class="mt-4 space-y-3">
			<input type="hidden" name="pending_token" value={pendingToken} />
			<input type="hidden" name="next" value={next} />
			<input type="hidden" name="email" value={email} />
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Authenticator code</span>
				<input
					name="code"
					type="text"
					inputmode="numeric"
					autocomplete="one-time-code"
					required
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 font-mono text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<button
				type="submit"
				class="rounded bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white"
			>
				Verify
			</button>
		</form>
	{:else}
		<form method="POST" action="?/login" class="mt-4 space-y-3">
			<input type="hidden" name="next" value={data.next} />
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Email</span>
				<input
					name="email"
					type="email"
					required
					autocomplete="email"
					value={email}
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Password</span>
				<input
					name="password"
					type="password"
					required
					autocomplete="current-password"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<button
				type="submit"
				class="rounded bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white"
			>
				Sign in
			</button>
		</form>

		<div class="mt-6 space-y-2">
			<p class="text-xs text-[var(--color-muted)]">Or continue with</p>
			<div class="flex flex-col gap-2">
				{#each providers as provider (provider.id)}
					<a
						href={resolve(`/api/auth/oauth/${provider.id}?mode=login`)}
						class="inline-flex items-center justify-center gap-2 rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm text-[var(--color-text)] no-underline hover:bg-[var(--color-surface-2)]"
					>
						<OAuthIcon provider={provider.id} />
						{provider.label}
					</a>
				{/each}
			</div>
		</div>
	{/if}

	<p class="mt-6 text-sm text-[var(--color-muted)]">
		Need an account?
		<a href={resolve('/request-access')} class="text-[var(--color-accent)]">Request access</a>
	</p>
</div>
