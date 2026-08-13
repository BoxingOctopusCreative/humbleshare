import { json, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetchRaw, clearSessionCookie } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';

export const POST: RequestHandler = async ({ cookies, request, url }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (sessionToken) {
		await apiFetchRaw('/api/v1/auth/logout', {
			method: 'POST',
			sessionToken
		}).catch(() => undefined);
	}
	clearSessionCookie(cookies);

	const ct = request.headers.get('content-type') ?? '';
	if (!ct.includes('application/json')) {
		const next = url.searchParams.get('next') || '/login';
		throw redirect(303, next.startsWith('/') ? next : '/login');
	}
	return json({ ok: true });
};
