import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { apiFetchRaw, extractSessionFromSetCookie, setSessionCookie } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';

function errMsg(body: Record<string, unknown>, fallback: string): string {
	if (typeof body.message === 'string') return body.message;
	if (typeof body.msg === 'string') return body.msg;
	if (typeof body.error === 'string' && body.error !== 'true') return body.error;
	return fallback;
}

export const load: PageServerLoad = async ({ cookies, url }) => {
	if (cookies.get(SESSION_COOKIE)) {
		const next = url.searchParams.get('next') || '/games';
		throw redirect(303, next.startsWith('/') ? next : '/games');
	}
	return {
		next: url.searchParams.get('next') || '/games'
	};
};

export const actions: Actions = {
	login: async ({ request, cookies }) => {
		const form = await request.formData();
		const email = String(form.get('email') ?? '').trim();
		const password = String(form.get('password') ?? '');
		const next = String(form.get('next') ?? '/games') || '/games';

		if (!email || !password) {
			return fail(400, { error: 'Email and password required', email });
		}

		const res = await apiFetchRaw('/api/v1/auth/login', {
			method: 'POST',
			body: { email, password }
		});
		const sessionToken = extractSessionFromSetCookie(res);
		const body = (await res.json().catch(() => ({}))) as Record<string, unknown>;

		if (!res.ok) {
			return fail(res.status, { error: errMsg(body, 'Login failed'), email });
		}

		if (body.requires_2fa) {
			return {
				requires2fa: true,
				pendingToken: String(body.pending_token ?? ''),
				next,
				email
			};
		}

		if (!sessionToken) {
			return fail(500, { error: 'Login succeeded but no session cookie returned', email });
		}
		setSessionCookie(cookies, sessionToken);
		throw redirect(303, next.startsWith('/') ? next : '/games');
	},

	login2fa: async ({ request, cookies }) => {
		const form = await request.formData();
		const pendingToken = String(form.get('pending_token') ?? '').trim();
		const code = String(form.get('code') ?? '').trim();
		const next = String(form.get('next') ?? '/games') || '/games';
		const email = String(form.get('email') ?? '');

		if (!pendingToken || !code) {
			return fail(400, {
				error: 'Authentication code required',
				requires2fa: true,
				pendingToken,
				next,
				email
			});
		}

		const res = await apiFetchRaw('/api/v1/auth/login/2fa', {
			method: 'POST',
			body: { pending_token: pendingToken, code }
		});
		const sessionToken = extractSessionFromSetCookie(res);
		const body = (await res.json().catch(() => ({}))) as Record<string, unknown>;

		if (!res.ok) {
			return fail(res.status, {
				error: errMsg(body, 'Invalid code'),
				requires2fa: true,
				pendingToken,
				next,
				email
			});
		}

		if (!sessionToken) {
			return fail(500, {
				error: 'Login succeeded but no session cookie returned',
				requires2fa: true,
				pendingToken,
				next,
				email
			});
		}
		setSessionCookie(cookies, sessionToken);
		throw redirect(303, next.startsWith('/') ? next : '/games');
	}
};
