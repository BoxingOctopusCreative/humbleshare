import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { apiFetch, apiFetchRaw, extractSessionFromSetCookie, setSessionCookie } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import { ApiError } from '$lib/types';

function errMsg(body: Record<string, unknown>, fallback: string): string {
	if (typeof body.message === 'string') return body.message;
	if (typeof body.msg === 'string') return body.msg;
	return fallback;
}

export const load: PageServerLoad = async ({ cookies, params }) => {
	if (cookies.get(SESSION_COOKIE)) {
		throw redirect(303, '/games');
	}

	const token = params.token;
	try {
		const invite = await apiFetch<{ email: string; role: string; expires_at: string }>(
			`/api/v1/auth/invite/${encodeURIComponent(token)}`
		);
		return { token, invite, error: '' };
	} catch (e) {
		if (e instanceof ApiError) {
			return { token, invite: null, error: e.message || 'Invalid or expired invite' };
		}
		throw e;
	}
};

export const actions: Actions = {
	default: async ({ request, cookies, params }) => {
		const form = await request.formData();
		const password = String(form.get('password') ?? '');
		const displayName = String(form.get('display_name') ?? '').trim();
		const token = params.token;

		if (!password || !displayName) {
			return fail(400, { error: 'Display name and password required', displayName });
		}

		const res = await apiFetchRaw('/api/v1/auth/register', {
			method: 'POST',
			body: { token, password, display_name: displayName }
		});
		const sessionToken = extractSessionFromSetCookie(res);
		const body = (await res.json().catch(() => ({}))) as Record<string, unknown>;

		if (!res.ok) {
			return fail(res.status, { error: errMsg(body, 'Registration failed'), displayName });
		}

		if (!sessionToken) {
			return fail(500, {
				error: 'Registered but no session cookie returned',
				displayName
			});
		}
		setSessionCookie(cookies, sessionToken);
		throw redirect(303, '/games');
	}
};
