import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import type { Game } from '$lib/types';
import { ApiError } from '$lib/types';

export const load: PageServerLoad = async ({ cookies, url }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (!sessionToken) {
		throw redirect(303, `/login?next=${encodeURIComponent(url.pathname + url.search)}`);
	}

	try {
		const games = await apiFetch<Game[]>('/api/v1/games', { sessionToken });
		return { games: games ?? [] };
	} catch (e) {
		if (e instanceof ApiError && e.status === 401) {
			cookies.delete(SESSION_COOKIE, { path: '/' });
			throw redirect(303, '/login');
		}
		throw e;
	}
};
