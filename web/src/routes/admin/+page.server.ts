import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import type { AccessRequest, AdminStats, CodeRow, Game, User } from '$lib/types';
import { ApiError } from '$lib/types';

export const load: PageServerLoad = async ({ cookies, url, parent }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (!sessionToken) {
		throw redirect(303, `/login?next=${encodeURIComponent('/admin')}`);
	}

	const layout = await parent();
	if (!layout.hasAdmin) {
		throw redirect(303, '/games');
	}

	const tab = url.searchParams.get('tab') || 'import';
	const status = url.searchParams.get('status') || '';
	const isSuperadmin = layout.user?.role === 'superadmin';

	try {
		const [codes, unresolved, stats, dead, accessRequests, users] = await Promise.all([
			apiFetch<CodeRow[]>('/api/v1/admin/codes', {
				sessionToken,
				searchParams: { status: status || undefined }
			}),
			apiFetch<Game[]>('/api/v1/admin/games/unresolved', { sessionToken }),
			apiFetch<AdminStats>('/api/v1/admin/stats', { sessionToken }),
			apiFetch<CodeRow[]>('/api/v1/admin/codes', {
				sessionToken,
				searchParams: { status: 'dead' }
			}),
			apiFetch<AccessRequest[]>('/api/v1/admin/access-requests', { sessionToken }).catch(
				() => [] as AccessRequest[]
			),
			apiFetch<User[]>('/api/v1/admin/users', { sessionToken }).catch(() => [] as User[])
		]);

		return {
			tab,
			status,
			isSuperadmin,
			codes: codes ?? [],
			unresolved: unresolved ?? [],
			dead: dead ?? [],
			accessRequests: (accessRequests ?? []).filter((r) => r.status === 'pending'),
			users: users ?? [],
			stats: stats ?? {
				total_games: 0,
				total_codes: 0,
				available_codes: 0,
				claimed_codes: 0,
				dead_codes: 0,
				claims_last_30_days: [],
				top_claimers: [],
				top_games: []
			}
		};
	} catch (e) {
		if (e instanceof ApiError && (e.status === 401 || e.status === 403)) {
			if (e.status === 401) cookies.delete(SESSION_COOKIE, { path: '/' });
			throw redirect(303, e.status === 401 ? '/login?next=/admin' : '/games');
		}
		throw e;
	}
};
