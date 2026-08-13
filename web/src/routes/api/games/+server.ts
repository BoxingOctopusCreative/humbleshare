import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import { ApiError } from '$lib/types';
import type { Game } from '$lib/types';

export const GET: RequestHandler = async ({ cookies, url }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (!sessionToken) return json({ error: true, msg: 'unauthorized' }, { status: 401 });

	try {
		const games = await apiFetch<Game[]>('/api/v1/games', {
			sessionToken,
			searchParams: { q: url.searchParams.get('q') ?? undefined }
		});
		return json(games ?? []);
	} catch (e) {
		if (e instanceof ApiError) return json({ error: true, msg: e.message }, { status: e.status });
		throw e;
	}
};
