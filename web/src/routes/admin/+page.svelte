<script lang="ts">
	import type { AccessRequest, BulkLineResult, CodeRow, HumbleImportResult, User } from '$lib/types';
	import { maskCode, parseBulkPreview } from '$lib/bulkParse';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import InventoryEditModal from '$lib/components/InventoryEditModal.svelte';

	let { data } = $props();

	const tab = $derived(data.tab);
	const statusFilter = $derived(data.status);
	let inventoryQuery = $state('');
	let revealed = $state<Record<string, boolean>>({});

	let humbleCookie = $state('');
	let humbleMode = $state<'gifts' | 'orders'>('gifts');
	let orderIds = $state('');
	let humbleLog = $state('');
	let humbleGiftTitle = $state('');
	let humbleGiftUrl = $state('');
	let bulkBlob = $state('');
	let bulkPreview = $state<{ title: string; code: string }[]>([]);
	let bulkResult = $state<BulkLineResult[] | null>(null);
	let busy = $state(false);
	let lookupBusy = $state<Record<string, boolean>>({});
	let lookupNote = $state('');
	let editingCode = $state<CodeRow | null>(null);
	let editBusy = $state(false);
	let editError = $state('');

	const tabs = ['import', 'inventory', 'review', 'stats', 'requests', 'users'];

	const filteredCodes = $derived.by(() => {
		const q = inventoryQuery.trim().toLowerCase();
		if (!q) return data.codes;
		return data.codes.filter(
			(c) =>
				(c.game_name ?? '').toLowerCase().includes(q) ||
				c.code_value.toLowerCase().includes(q) ||
				(c.claimed_by ?? '').toLowerCase().includes(q)
		);
	});

	function refreshBulkPreview() {
		bulkPreview = parseBulkPreview(bulkBlob);
	}

	function tabLabel(t: string) {
		return t[0].toUpperCase() + t.slice(1);
	}

	async function runHumbleImport(ids: string[], titles?: Record<string, string>) {
		busy = true;
		humbleLog = titles ? 'Importing with titles…\n' : 'Importing…\n';
		try {
			const payload: {
				session_cookie: string;
				order_ids: string[];
				titles?: Record<string, string>;
			} = {
				session_cookie: humbleCookie,
				order_ids: ids
			};
			if (titles && Object.keys(titles).length > 0) {
				payload.titles = titles;
			}
			const res = await fetch('/api/admin/import/humble', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});
			const body = await res.json();
			if (!res.ok) {
				humbleLog += `Error: ${body.msg || 'failed'}\n`;
				return;
			}
			const rows = (Array.isArray(body) ? body : [body]) as HumbleImportResult[];
			for (const row of rows) {
				const need = row.needs_title ? ' needs_title' : '';
				humbleLog += `[${row.order_id}] imported=${row.imported} duplicates=${row.skipped_duplicate} non-steam=${row.skipped_non_steam} unrevealed=${row.skipped_unrevealed} failed=${row.failed}${need}\n`;
				for (const m of row.messages ?? []) humbleLog += `  ${m}\n`;
			}
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	async function runHumbleFromPaste() {
		if (humbleMode === 'gifts') {
			const giftURL = humbleGiftUrl.trim();
			const title = humbleGiftTitle.trim();
			if (!giftURL || !title) {
				humbleLog = 'Enter a game title and one gift URL.\n';
				return;
			}
			if (parseHumbleLines(giftURL).length !== 1) {
				humbleLog = 'Gift link mode accepts one URL at a time.\n';
				return;
			}
			if (!isHumbleGiftURL(giftURL)) {
				humbleLog = 'Gift link mode only accepts a gift?key=… URL.\n';
				return;
			}
			await runHumbleImport([giftURL], { [giftURL]: title });
			return;
		}
		const ids = parseHumbleLines(orderIds);
		if (ids.length === 0) return;
		if (ids.some(isHumbleGiftURL)) {
			humbleLog = 'Order gamekey mode only accepts downloads?key=… URLs or order gamekeys.\n';
			return;
		}
		await runHumbleImport(ids);
	}

	function parseHumbleLines(text: string): string[] {
		return text
			.split(/[\s,]+/)
			.map((s) => s.trim())
			.filter(Boolean);
	}

	function isHumbleGiftURL(line: string): boolean {
		try {
			const u = new URL(line);
			const host = u.hostname.toLowerCase();
			if (host !== 'humblebundle.com' && !host.endsWith('.humblebundle.com')) return false;
			const path = u.pathname.replace(/\/+$/, '').toLowerCase();
			return (path === '/gift' || path.startsWith('/gift/')) && Boolean(u.searchParams.get('key'));
		} catch {
			return false;
		}
	}

	async function runBulk() {
		busy = true;
		bulkResult = null;
		try {
			const res = await fetch('/api/admin/codes/bulk', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ blob: bulkBlob })
			});
			const body = await res.json();
			if (!res.ok) {
				bulkResult = [{ line: 0, title: '', success: false, error: body.msg || 'failed' }];
				return;
			}
			bulkResult = (Array.isArray(body) ? body : body.results) as BulkLineResult[];
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	async function patchStatus(code: CodeRow, status: string) {
		await fetch(`/api/admin/codes/${code.id}`, {
			method: 'PATCH',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ status })
		});
		await invalidateAll();
	}

	async function resolveGame(id: string, steamAppid: number) {
		await fetch(`/api/admin/games/${id}`, {
			method: 'PATCH',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ steam_appid: steamAppid })
		});
		await invalidateAll();
	}

	async function lookupSteam(gameId: string, gameName?: string) {
		if (lookupBusy[gameId]) return;
		lookupBusy[gameId] = true;
		lookupNote = '';
		try {
			const res = await fetch(`/api/admin/games/${gameId}/refresh`, { method: 'POST' });
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				lookupNote = `${gameName ?? gameId}: ${body.msg || 'Steam lookup failed'}`;
				return;
			}
			const appid = body.steam_appid;
			lookupNote =
				appid != null
					? `${body.name ?? gameName ?? gameId}: linked Steam app ${appid}`
					: `${gameName ?? gameId}: metadata refreshed`;
			await invalidateAll();
		} finally {
			lookupBusy[gameId] = false;
		}
	}

	function openEdit(code: CodeRow) {
		editingCode = code;
		editError = '';
	}

	function closeEdit() {
		if (editBusy) return;
		editingCode = null;
		editError = '';
	}

	async function saveEdit(patch: {
		game_name: string;
		status: string;
		code_value: string;
		notes: string;
	}) {
		if (!editingCode) return;
		editBusy = true;
		editError = '';
		try {
			const currentName = (editingCode.game_name ?? '').trim();
			if (patch.game_name && patch.game_name !== currentName) {
				const gameRes = await fetch(`/api/admin/games/${editingCode.game_id}`, {
					method: 'PATCH',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ name: patch.game_name })
				});
				const gameBody = await gameRes.json().catch(() => ({}));
				if (!gameRes.ok) {
					editError = gameBody.msg || 'Failed to update game title';
					return;
				}
			}

			const res = await fetch(`/api/admin/codes/${editingCode.id}`, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					status: patch.status,
					code_value: patch.code_value,
					notes: patch.notes
				})
			});
			const body = await res.json().catch(() => ({}));
			if (!res.ok) {
				editError = body.msg || 'Save failed';
				return;
			}
			editingCode = null;
			await invalidateAll();
		} finally {
			editBusy = false;
		}
	}

	async function approveRequest(req: AccessRequest) {
		busy = true;
		try {
			await fetch(`/api/admin/access-requests/${req.id}/approve`, { method: 'POST' });
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	async function denyRequest(req: AccessRequest) {
		busy = true;
		try {
			await fetch(`/api/admin/access-requests/${req.id}/deny`, { method: 'POST' });
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	async function setRole(user: User, role: string) {
		busy = true;
		try {
			await fetch(`/api/admin/users/${user.id}/role`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ role })
			});
			await invalidateAll();
		} finally {
			busy = false;
		}
	}

	function mask(value: string, id: string) {
		return maskCode(value, Boolean(revealed[id]));
	}
</script>

<div class="mb-4 rounded border border-[var(--color-admin)]/40 bg-[var(--color-admin)]/10 px-3 py-2 text-sm text-[var(--color-admin)]">
	Admin · full inventory · codes masked until revealed
</div>

<div class="mb-6 flex flex-wrap gap-2 border-b border-[var(--color-border)] pb-2 text-sm">
	{#each tabs as t (t)}
		<a
			href={resolve(`/admin?tab=${t}`)}
			class="rounded px-3 py-1.5 no-underline {tab === t
				? 'bg-[var(--color-admin)] text-black'
				: 'text-[var(--color-muted)] hover:text-[var(--color-text)]'}"
		>
			{tabLabel(t)}
		</a>
	{/each}
</div>

{#if tab === 'import'}
	<section class="grid gap-8 lg:grid-cols-2">
		<div>
			<h2 class="font-semibold text-[var(--color-admin)]">Humble import</h2>
			<div class="mt-3 flex flex-wrap gap-2 text-sm" role="group" aria-label="Humble import type">
				<button
					type="button"
					class="rounded px-3 py-1.5 {humbleMode === 'gifts'
						? 'bg-[var(--color-admin)] text-black'
						: 'text-[var(--color-muted)] hover:text-[var(--color-text)]'}"
					aria-pressed={humbleMode === 'gifts'}
					onclick={() => (humbleMode = 'gifts')}
				>
					Gift links
				</button>
				<button
					type="button"
					class="rounded px-3 py-1.5 {humbleMode === 'orders'
						? 'bg-[var(--color-admin)] text-black'
						: 'text-[var(--color-muted)] hover:text-[var(--color-text)]'}"
					aria-pressed={humbleMode === 'orders'}
					onclick={() => (humbleMode = 'orders')}
				>
					Order gamekeys
				</button>
			</div>
			{#if humbleMode === 'gifts'}
				<p class="mt-3 text-sm text-[var(--color-muted)]">
					One gift link at a time. Enter the game title and a
					<code class="font-mono text-xs">gift?key=…</code> URL. Steam CD keys go in Bulk paste.
				</p>
				<label class="mt-3 block text-sm">
					<span class="text-[var(--color-muted)]">Game title</span>
					<input
						class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm"
						bind:value={humbleGiftTitle}
						placeholder="Game title"
						disabled={busy}
					/>
				</label>
				<label class="mt-3 block text-sm">
					<span class="text-[var(--color-muted)]">Gift URL</span>
					<input
						class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 font-mono text-xs"
						bind:value={humbleGiftUrl}
						placeholder="https://www.humblebundle.com/gift?key=…"
						disabled={busy}
					/>
				</label>
			{:else}
				<p class="mt-3 text-sm text-[var(--color-muted)]">
					Paste <code class="font-mono text-xs">downloads?key=…</code> URLs or order gamekeys,
					one per line. Titles come from Humble. The cookie stays in memory for this request.
					Steam CD keys go in Bulk paste.
				</p>
				<label class="mt-3 block text-sm">
					<span class="text-[var(--color-muted)]">_simpleauth_sess cookie</span>
					<input
						class="mt-1 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 font-mono text-xs"
						bind:value={humbleCookie}
						autocomplete="off"
					/>
				</label>
				<label class="mt-3 block text-sm">
					<span class="text-[var(--color-muted)]">Downloads URLs or order gamekeys (one per line)</span>
					<textarea
						class="mt-1 h-28 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 font-mono text-xs"
						bind:value={orderIds}
						placeholder="https://www.humblebundle.com/downloads?key=…"
					></textarea>
				</label>
			{/if}
			<button
				type="button"
				class="mt-3 rounded bg-[var(--color-admin)] px-3 py-1.5 text-sm font-medium text-black"
				disabled={busy ||
					(humbleMode === 'gifts'
						? !humbleGiftTitle.trim() || !humbleGiftUrl.trim()
						: !orderIds.trim() || !humbleCookie.trim())}
				onclick={runHumbleFromPaste}
			>
				Import
			</button>
			<pre
				class="mt-3 max-h-48 overflow-auto rounded border border-[var(--color-border)] bg-[var(--color-bg)] p-3 text-xs text-[var(--color-muted)]"
			>{humbleLog || 'Result log…'}</pre>
		</div>

		<div>
			<h2 class="font-semibold text-[var(--color-admin)]">Bulk paste</h2>
			<p class="mt-1 text-sm text-[var(--color-muted)]">Title[TAB]Code or Title,Code - one per line.</p>
			<textarea
				class="mt-3 h-40 w-full rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 font-mono text-xs"
				bind:value={bulkBlob}
				oninput={refreshBulkPreview}
			></textarea>
			{#if bulkPreview.length}
				<div class="mt-2 max-h-40 overflow-auto text-xs">
					<table class="w-full text-left">
						<thead class="text-[var(--color-muted)]">
							<tr><th class="py-1">Title</th><th>Code</th></tr>
						</thead>
						<tbody>
							{#each bulkPreview as row, i (i)}
								<tr class="border-t border-[var(--color-border)]">
									<td class="py-1 pr-2">{row.title}</td>
									<td class="font-mono">{row.code.slice(0, 12)}…</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
			<button
				type="button"
				class="mt-3 rounded bg-[var(--color-admin)] px-3 py-1.5 text-sm font-medium text-black"
				disabled={busy || !bulkBlob.trim()}
				onclick={runBulk}
			>
				Commit bulk
			</button>
			{#if bulkResult}
				<ul class="mt-3 space-y-1 text-xs">
					{#each bulkResult as r, i (i)}
						<li class={r.success ? 'text-[var(--color-ok)]' : 'text-[var(--color-danger)]'}>
							L{r.line} {r.title}: {r.success ? 'ok' : r.error}
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</section>
{:else if tab === 'inventory'}
	<section>
		<div class="mb-3 flex flex-wrap gap-2">
			<input
				class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-1.5 text-sm"
				placeholder="Search title or code…"
				bind:value={inventoryQuery}
			/>
			{#each ['', 'available', 'claimed', 'dead'] as s (s)}
				{#if s === ''}
					<a
						href={resolve('/admin?tab=inventory')}
						class="rounded px-2 py-1 text-xs no-underline {statusFilter === s
							? 'bg-[var(--color-surface-2)] text-[var(--color-text)]'
							: 'text-[var(--color-muted)]'}"
					>
						all
					</a>
				{:else}
					<a
						href={resolve(`/admin?tab=inventory&status=${s}`)}
						class="rounded px-2 py-1 text-xs no-underline {statusFilter === s
							? 'bg-[var(--color-surface-2)] text-[var(--color-text)]'
							: 'text-[var(--color-muted)]'}"
					>
						{s}
					</a>
				{/if}
			{/each}
		</div>
		{#if lookupNote}
			<p class="mb-2 text-xs text-[var(--color-muted)]">{lookupNote}</p>
		{/if}
		<div class="overflow-x-auto rounded border border-[var(--color-border)]">
			<table class="w-full min-w-[960px] text-left text-xs">
				<thead class="bg-[var(--color-surface)] text-[var(--color-muted)]">
					<tr>
						<th class="px-2 py-2">Game</th>
						<th class="px-2 py-2">Type</th>
						<th class="px-2 py-2">Source</th>
						<th class="px-2 py-2">Status</th>
						<th class="px-2 py-2">Code</th>
						<th class="px-2 py-2">Claimed by</th>
						<th class="px-2 py-2">Claimed at</th>
						<th class="px-2 py-2">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each filteredCodes as code (code.id)}
						<tr class="border-t border-[var(--color-border)] hover:bg-[var(--color-surface)]/60">
							<td class="px-2 py-1.5">{code.game_name ?? code.game_id.slice(0, 8)}</td>
							<td class="px-2 py-1.5">{code.code_type}</td>
							<td class="px-2 py-1.5">{code.source ?? '-'}</td>
							<td class="px-2 py-1.5">
								<select
									class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-1 py-0.5"
									value={code.status}
									onchange={(e) => patchStatus(code, e.currentTarget.value)}
								>
									<option value="available">available</option>
									<option value="claimed">claimed</option>
									<option value="dead">dead</option>
								</select>
							</td>
							<td class="px-2 py-1.5 font-mono">
								<button
									type="button"
									class="text-left hover:text-[var(--color-accent)]"
									onclick={() => (revealed[code.id] = !revealed[code.id])}
								>
									{mask(code.code_value, code.id)}
								</button>
							</td>
							<td class="px-2 py-1.5">{code.claimed_by ?? '-'}</td>
							<td class="px-2 py-1.5 whitespace-nowrap">
								{code.claimed_at ? new Date(code.claimed_at).toLocaleString() : '-'}
							</td>
							<td class="px-2 py-1.5">
								<div class="flex flex-wrap gap-1">
									<button
										type="button"
										class="rounded border border-[var(--color-border)] px-2 py-0.5 text-[var(--color-text)] hover:bg-[var(--color-surface-2)]"
										onclick={() => openEdit(code)}
									>
										Edit
									</button>
									<button
										type="button"
										class="rounded border border-[var(--color-border)] px-2 py-0.5 text-[var(--color-text)] hover:bg-[var(--color-surface-2)] disabled:opacity-50"
										disabled={lookupBusy[code.game_id]}
										onclick={() => lookupSteam(code.game_id, code.game_name)}
									>
										{lookupBusy[code.game_id] ? 'Looking up…' : 'Lookup'}
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		{#if editingCode}
			{#key editingCode.id}
				<InventoryEditModal
					code={editingCode}
					busy={editBusy}
					error={editError}
					onclose={closeEdit}
					onsave={saveEdit}
				/>
			{/key}
		{/if}
	</section>
{:else if tab === 'review'}
	<section class="grid gap-8 lg:grid-cols-2">
		<div>
			<h2 class="font-semibold">Unresolved games</h2>
			<p class="mt-1 text-sm text-[var(--color-muted)]">Set a Steam appid to fetch metadata.</p>
			<ul class="mt-3 space-y-2">
				{#each data.unresolved as g (g.id)}
					<li class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-3 text-sm">
						<div class="font-medium">{g.name}</div>
						<form
							class="mt-2 flex gap-2"
							onsubmit={(e) => {
								e.preventDefault();
								const fd = new FormData(e.currentTarget);
								const id = Number(fd.get('appid'));
								if (id > 0) resolveGame(g.id, id);
							}}
						>
							<input
								name="appid"
								type="number"
								placeholder="Steam appid"
								class="w-36 rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-2 py-1 text-xs"
								required
							/>
							<button type="submit" class="rounded bg-[var(--color-admin)] px-2 py-1 text-xs text-black">
								Set + refresh
							</button>
						</form>
					</li>
				{:else}
					<li class="text-sm text-[var(--color-muted)]">Queue empty.</li>
				{/each}
			</ul>
		</div>
		<div>
			<h2 class="font-semibold">Dead codes</h2>
			<ul class="mt-3 space-y-2 text-sm">
				{#each data.dead as c (c.id)}
					<li class="flex items-center justify-between gap-2 rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2">
						<span>{c.game_name ?? c.id.slice(0, 8)}</span>
						<button
							type="button"
							class="text-xs text-[var(--color-ok)]"
							onclick={() => patchStatus(c, 'available')}
						>
							Restore available
						</button>
					</li>
				{:else}
					<li class="text-[var(--color-muted)]">No dead codes.</li>
				{/each}
			</ul>
		</div>
	</section>
{:else if tab === 'requests'}
	<section>
		<h2 class="font-semibold">Access requests</h2>
		<p class="mt-1 text-sm text-[var(--color-muted)]">Pending requests. Approve sends an invite email.</p>
		<ul class="mt-4 space-y-2">
			{#each data.accessRequests as req (req.id)}
				<li
					class="flex flex-col gap-2 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-3 sm:flex-row sm:items-center sm:justify-between"
				>
					<div class="text-sm">
						<div class="font-medium">{req.email}</div>
						{#if req.note}
							<p class="mt-1 text-[var(--color-muted)]">{req.note}</p>
						{/if}
						<p class="mt-1 text-xs text-[var(--color-muted)]">
							{new Date(req.created_at).toLocaleString()}
						</p>
					</div>
					<div class="flex gap-2">
						<button
							type="button"
							class="rounded bg-[var(--color-ok)] px-3 py-1.5 text-xs font-medium text-black"
							disabled={busy}
							onclick={() => approveRequest(req)}
						>
							Approve
						</button>
						<button
							type="button"
							class="rounded border border-[var(--color-danger)] px-3 py-1.5 text-xs text-[var(--color-danger)]"
							disabled={busy}
							onclick={() => denyRequest(req)}
						>
							Deny
						</button>
					</div>
				</li>
			{:else}
				<li class="text-sm text-[var(--color-muted)]">No pending requests.</li>
			{/each}
		</ul>
	</section>
{:else if tab === 'users'}
	<section>
		<h2 class="font-semibold">Users</h2>
		{#if !data.isSuperadmin}
			<p class="mt-1 text-sm text-[var(--color-muted)]">Role changes require superadmin.</p>
		{/if}
		<div class="mt-4 overflow-x-auto rounded border border-[var(--color-border)]">
			<table class="w-full min-w-[640px] text-left text-sm">
				<thead class="bg-[var(--color-surface)] text-[var(--color-muted)]">
					<tr>
						<th class="px-3 py-2">Name</th>
						<th class="px-3 py-2">Email</th>
						<th class="px-3 py-2">Role</th>
						<th class="px-3 py-2">2FA</th>
					</tr>
				</thead>
				<tbody>
					{#each data.users as user (user.id)}
						<tr class="border-t border-[var(--color-border)]">
							<td class="px-3 py-2">{user.display_name}</td>
							<td class="px-3 py-2">{user.email}</td>
							<td class="px-3 py-2">
								{#if data.isSuperadmin && user.role !== 'superadmin'}
									<select
										class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-2 py-1 text-xs"
										value={user.role}
										disabled={busy}
										onchange={(e) => setRole(user, e.currentTarget.value)}
									>
										<option value="member">member</option>
										<option value="admin">admin</option>
									</select>
								{:else}
									{user.role}
								{/if}
							</td>
							<td class="px-3 py-2">{user.two_factor_enabled ? 'on' : 'off'}</td>
						</tr>
					{:else}
						<tr>
							<td colspan="4" class="px-3 py-4 text-[var(--color-muted)]">No users.</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</section>
{:else}
	<section class="grid gap-6 sm:grid-cols-2">
		<div class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
			<h2 class="text-sm font-semibold text-[var(--color-muted)]">By status</h2>
			<ul class="mt-2 space-y-1 text-sm">
				<li class="flex justify-between"><span>available</span><span class="font-mono">{data.stats.available_codes}</span></li>
				<li class="flex justify-between"><span>claimed</span><span class="font-mono">{data.stats.claimed_codes}</span></li>
				<li class="flex justify-between"><span>dead</span><span class="font-mono">{data.stats.dead_codes}</span></li>
				<li class="flex justify-between"><span>games</span><span class="font-mono">{data.stats.total_games}</span></li>
			</ul>
		</div>
		<div class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
			<h2 class="text-sm font-semibold text-[var(--color-muted)]">Claims · 30 days</h2>
			<ul class="mt-2 max-h-48 space-y-1 overflow-auto text-sm">
				{#each data.stats.claims_last_30_days as row (row.date)}
					<li class="flex justify-between font-mono text-xs">
						<span>{row.date}</span><span>{row.count}</span>
					</li>
				{:else}
					<li class="text-[var(--color-muted)]">No claims yet.</li>
				{/each}
			</ul>
		</div>
		<div class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
			<h2 class="text-sm font-semibold text-[var(--color-muted)]">Top claimers</h2>
			<ul class="mt-2 space-y-1 text-sm">
				{#each data.stats.top_claimers as row (row.claimed_by)}
					<li class="flex justify-between"><span>{row.claimed_by}</span><span class="font-mono">{row.count}</span></li>
				{:else}
					<li class="text-[var(--color-muted)]">-</li>
				{/each}
			</ul>
		</div>
		<div class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
			<h2 class="text-sm font-semibold text-[var(--color-muted)]">Top games</h2>
			<ul class="mt-2 space-y-1 text-sm">
				{#each data.stats.top_games as row (row.game_id)}
					<li class="flex justify-between gap-2">
						<span class="truncate">{row.name}</span>
						<span class="font-mono">{row.count}</span>
					</li>
				{:else}
					<li class="text-[var(--color-muted)]">-</li>
				{/each}
			</ul>
		</div>
	</section>
{/if}
