<script lang="ts">
	import { resolve } from '$app/paths';
	import SplashBackdrop from '$lib/components/SplashBackdrop.svelte';

	let { form } = $props();

	const error = $derived(form?.error ?? '');
	const ok = $derived(Boolean(form?.ok));
</script>

<SplashBackdrop />

<div class="relative z-10 mx-auto max-w-md">
	<h1 class="text-2xl font-semibold">Request access</h1>
	<p class="mt-2 text-sm text-[var(--color-muted)]">
		Send a request. An admin will email you an invite if approved.
	</p>

	{#if ok}
		<p class="mt-4 text-sm text-[var(--color-ok)]">
			Request submitted for {form?.email}. Check your inbox when an invite arrives.
		</p>
		<p class="mt-3 text-sm">
			<a href={resolve('/login')} class="text-[var(--color-accent)]">Back to login</a>
		</p>
	{:else}
		{#if error}
			<p class="mt-3 text-sm text-[var(--color-danger)]">{error}</p>
		{/if}

		<form method="POST" class="mt-4 space-y-3">
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Email</span>
				<input
					name="email"
					type="email"
					required
					autocomplete="email"
					value={form?.email ?? ''}
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				/>
			</label>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Note (optional)</span>
				<textarea
					name="note"
					rows="3"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm outline-none focus:border-[var(--color-accent)]"
				>{form?.note ?? ''}</textarea>
			</label>
			<button
				type="submit"
				class="rounded bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white"
			>
				Submit request
			</button>
		</form>

		<p class="mt-6 text-sm text-[var(--color-muted)]">
			Already invited?
			<a href={resolve('/login')} class="text-[var(--color-accent)]">Login</a>
		</p>
	{/if}
</div>
