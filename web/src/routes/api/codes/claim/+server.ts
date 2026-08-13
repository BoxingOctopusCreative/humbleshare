import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import { ApiError } from '$lib/types';
import type { ClaimResult } from '$lib/types';

export const POST: RequestHandler = async ({ cookies, request }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (!sessionToken) return json({ error: true, msg: 'unauthorized' }, { status: 401 });

	const body = await request.json();
	try {
		const result = await apiFetch<ClaimResult>('/api/v1/codes/claim', {
			method: 'POST',
			sessionToken,
			body: { game_id: body.game_id }
		});
		return json(result);
	} catch (e) {
		if (e instanceof ApiError) return json({ error: true, msg: e.message }, { status: e.status });
		throw e;
	}
};
