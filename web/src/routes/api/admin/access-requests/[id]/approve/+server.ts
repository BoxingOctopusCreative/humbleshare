import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import { ApiError } from '$lib/types';

export const POST: RequestHandler = async ({ cookies, params }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (!sessionToken) return json({ error: true, msg: 'unauthorized' }, { status: 401 });
	try {
		const result = await apiFetch(`/api/v1/admin/access-requests/${params.id}/approve`, {
			method: 'POST',
			sessionToken
		});
		return json(result ?? { ok: true });
	} catch (e) {
		if (e instanceof ApiError) return json({ error: true, msg: e.message }, { status: e.status });
		throw e;
	}
};
