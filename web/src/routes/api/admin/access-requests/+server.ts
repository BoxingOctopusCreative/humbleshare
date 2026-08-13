import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import type { AccessRequest } from '$lib/types';
import { ApiError } from '$lib/types';

export const GET: RequestHandler = async ({ cookies }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (!sessionToken) return json({ error: true, msg: 'unauthorized' }, { status: 401 });
	try {
		const list = await apiFetch<AccessRequest[]>('/api/v1/admin/access-requests', {
			sessionToken
		});
		return json(list ?? []);
	} catch (e) {
		if (e instanceof ApiError) return json({ error: true, msg: e.message }, { status: e.status });
		throw e;
	}
};
