import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetchRaw, extractSessionFromSetCookie, setSessionCookie } from '$lib/server/api';

export const POST: RequestHandler = async ({ request, cookies }) => {
	const body = await request.json().catch(() => ({}));
	const res = await apiFetchRaw('/api/v1/auth/login', {
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
					'Login failed'
			},
			{ status: res.status }
		);
	}
	if ((data as { requires_2fa?: boolean }).requires_2fa) {
		return json(data);
	}
	if (sessionToken) setSessionCookie(cookies, sessionToken);
	return json(data);
};
