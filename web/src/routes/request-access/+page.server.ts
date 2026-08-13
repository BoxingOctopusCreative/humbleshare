import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { apiFetchRaw } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';

function errMsg(body: Record<string, unknown>, fallback: string): string {
	if (typeof body.message === 'string') return body.message;
	if (typeof body.msg === 'string') return body.msg;
	return fallback;
}

export const load: PageServerLoad = async ({ cookies }) => {
	if (cookies.get(SESSION_COOKIE)) {
		throw redirect(303, '/games');
	}
	return {};
};

export const actions: Actions = {
	default: async ({ request }) => {
		const form = await request.formData();
		const email = String(form.get('email') ?? '').trim();
		const note = String(form.get('note') ?? '').trim();

		if (!email) {
			return fail(400, { error: 'Email required', email, note });
		}

		const res = await apiFetchRaw('/api/v1/auth/request-access', {
			method: 'POST',
			body: { email, note }
		});
		const body = (await res.json().catch(() => ({}))) as Record<string, unknown>;

		if (!res.ok) {
			return fail(res.status, { error: errMsg(body, 'Request failed'), email, note });
		}

		return { ok: true, email };
	}
};
