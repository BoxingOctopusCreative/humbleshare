import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetchRaw, extractSessionFromSetCookie, setSessionCookie } from '$lib/server/api';

export const POST: RequestHandler = async ({ request, cookies }) => {
	const body = await request.json().catch(() => ({}));
	const res = await apiFetchRaw('/api/v1/auth/login/2fa', {
		method: 'POST',
		body
	});
	const sessionToken = extractSessionFromSetCookie(res);
	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		return json(
			{
				error: true,
				msg:
					(data as { message?: string; msg?: string }).message ||
					(data as { msg?: string }).msg ||
					'Invalid code'
			},
			{ status: res.status }
		);
	}
	if (sessionToken) setSessionCookie(cookies, sessionToken);
	return json(data);
};
