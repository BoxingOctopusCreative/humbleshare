<script lang="ts">
	import { resolve } from '$app/paths';

	let { data, form } = $props();

	const error = $derived(form?.error || data.error);
</script>

<div class="mx-auto max-w-md">
	<h1 class="text-2xl font-semibold">Accept invite</h1>

	{#if !data.invite}
		<p class="mt-3 text-sm text-[var(--color-danger)]">{error || 'Invalid invite'}</p>
		<p class="mt-4 text-sm">
			<a href={resolve('/request-access')} class="text-[var(--color-accent)]">Request access</a>
		</p>
	{:else}
		<p class="mt-2 text-sm text-[var(--color-muted)]">
			Create your account for <span class="text-[var(--color-text)]">{data.invite.email}</span>
			({data.invite.role}).
		</p>

		{#if form?.error}
			<p class="mt-3 text-sm text-[var(--color-danger)]">{form.error}</p>
		{/if}

		<form method="POST" class="mt-4 space-y-3">
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Display name</span>
				<input
					name="display_name"
					type="text"
					required
					maxlength="64"
					autocomplete="nickname"
					value={form?.displayName ?? ''}
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Password</span>
				<input
					name="password"
					type="password"
					required
					minlength="8"
					autocomplete="new-password"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<button
				type="submit"
				class="rounded bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white"
			>
				Create account
			</button>
		</form>
	{/if}
</div>
