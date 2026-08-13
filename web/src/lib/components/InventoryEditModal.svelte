<script lang="ts">
	import type { CodeRow } from '$lib/types';

	let {
		code,
		busy = false,
		error = '',
		onclose,
		onsave
	}: {
		code: CodeRow;
		busy?: boolean;
		error?: string;
		onclose: () => void;
		onsave: (patch: {
			game_name: string;
			status: string;
			code_value: string;
			notes: string;
		}) => void | Promise<void>;
	} = $props();

	async function submit(e: Event) {
		e.preventDefault();
		const form = e.currentTarget as HTMLFormElement;
		const fd = new FormData(form);
		await onsave({
			game_name: String(fd.get('game_name') ?? '').trim(),
			status: String(fd.get('status') ?? ''),
			code_value: String(fd.get('code_value') ?? '').trim(),
			notes: String(fd.get('notes') ?? '').trim()
		});
	}
</script>

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
	role="dialog"
	aria-modal="true"
	aria-labelledby="edit-code-title"
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
		<h2 id="edit-code-title" class="text-lg font-semibold">Edit inventory item</h2>
		<p class="mt-1 text-xs text-[var(--color-muted)]">{code.code_type}</p>

		<form class="mt-4 space-y-3" onsubmit={submit}>
			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Game title</span>
				<input
					name="game_name"
					type="text"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-2 py-1.5"
					value={code.game_name ?? ''}
					disabled={busy}
					required
				/>
			</label>

			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Status</span>
				<select
					name="status"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-2 py-1.5"
					value={code.status}
					disabled={busy}
				>
					<option value="available">available</option>
					<option value="claimed">claimed</option>
					<option value="dead">dead</option>
				</select>
			</label>

			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Code value</span>
				<textarea
					name="code_value"
					class="mt-1 w-full break-all rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-2 py-1.5 font-mono text-xs"
					rows="3"
					disabled={busy}
					required
				>{code.code_value}</textarea>
			</label>

			<label class="block text-sm">
				<span class="text-[var(--color-muted)]">Notes</span>
				<textarea
					name="notes"
					class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-2 py-1.5 text-sm"
					rows="2"
					disabled={busy}
					placeholder="Optional"
				>{code.notes ?? ''}</textarea>
			</label>

			{#if error}
				<p class="text-sm text-[var(--color-danger)]">{error}</p>
			{/if}

			<div class="flex justify-end gap-2 pt-1">
				<button
					type="button"
					class="px-3 py-1.5 text-sm text-[var(--color-muted)]"
					onclick={onclose}
					disabled={busy}
				>
					Cancel
				</button>
				<button
					type="submit"
					class="rounded bg-[var(--color-admin)] px-3 py-1.5 text-sm font-medium text-black disabled:opacity-50"
					disabled={busy}
				>
					{busy ? 'Saving…' : 'Save'}
				</button>
			</div>
		</form>
	</div>
</div>
