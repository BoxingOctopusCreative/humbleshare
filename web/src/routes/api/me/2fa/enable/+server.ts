import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import { ApiError } from '$lib/types';

export const POST: RequestHandler = async ({ cookies, request }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (!sessionToken) return json({ error: true, msg: 'unauthorized' }, { status: 401 });
	try {
		const body = await request.json();
		const result = await apiFetch('/api/v1/me/2fa/enable', {
			method: 'POST',
			sessionToken,
			body
		});
		return json(result ?? { ok: true });
	} catch (e) {
		if (e instanceof ApiError) return json({ error: true, msg: e.message }, { status: e.status });
		throw e;
	}
};
