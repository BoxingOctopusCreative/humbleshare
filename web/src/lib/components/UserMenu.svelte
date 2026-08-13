<script lang="ts">
	import { resolve } from '$app/paths';

	type MenuUser = {
		display_name: string;
		avatar_url?: string;
	};

	let { user }: { user: MenuUser } = $props();

	let open = $state(false);
	let rootEl: HTMLElement | null = null;

	const initials = $derived.by(() => {
		const parts = user.display_name.trim().split(/\s+/).filter(Boolean);
		if (parts.length === 0) return '?';
		if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
		return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
	});

	const menuRoot = (node: HTMLElement) => {
		rootEl = node;
		return () => {
			rootEl = null;
		};
	};

	function toggle() {
		open = !open;
	}

	function close() {
		open = false;
	}

	function onDocClick(e: MouseEvent) {
		if (!open || !rootEl) return;
		if (!rootEl.contains(e.target as Node)) close();
	}

	function onKey(e: KeyboardEvent) {
		if (e.key === 'Escape') close();
	}
</script>

<svelte:window onclick={onDocClick} onkeydown={onKey} />

<div class="relative" {@attach menuRoot}>
	<button
		type="button"
		class="flex h-9 w-9 items-center justify-center overflow-hidden rounded-full border border-[var(--color-border)] bg-[var(--color-surface-2)] text-xs font-semibold text-[var(--color-text)] outline-none ring-[var(--color-accent)] hover:border-[var(--color-muted)] focus-visible:ring-2"
		aria-haspopup="menu"
		aria-expanded={open}
		aria-label="Account menu"
		onclick={toggle}
	>
		{#if user.avatar_url}
			<img src={user.avatar_url} alt="" class="h-full w-full object-cover" />
		{:else}
			<span aria-hidden="true">{initials}</span>
		{/if}
	</button>

	{#if open}
		<div
			role="menu"
			class="absolute right-0 z-30 mt-2 min-w-40 overflow-hidden rounded border border-[var(--color-border)] bg-[var(--color-surface)] py-1 shadow-lg"
		>
			<a
				role="menuitem"
				href={resolve('/settings')}
				class="block px-3 py-2 text-sm text-[var(--color-text)] no-underline hover:bg-[var(--color-surface-2)] hover:no-underline"
				onclick={close}
			>
				Settings
			</a>
			<form method="POST" action="/api/auth/logout">
				<button
					role="menuitem"
					type="submit"
					class="block w-full px-3 py-2 text-left text-sm text-[var(--color-muted)] hover:bg-[var(--color-surface-2)] hover:text-[var(--color-text)]"
				>
					Logout
				</button>
			</form>
		</div>
	{/if}
</div>
