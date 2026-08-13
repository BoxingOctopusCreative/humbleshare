import { error, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetchRaw } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';

const ALLOWED_HOSTS = new Set(['accounts.google.com', 'discord.com', 'steamcommunity.com']);

function isAllowedAuthorizeURL(raw: string): boolean {
	try {
		const u = new URL(raw);
		return u.protocol === 'https:' && ALLOWED_HOSTS.has(u.host);
	} catch {
		return false;
	}
}

export const GET: RequestHandler = async ({ params, url, cookies }) => {
	const provider = params.provider;
	const mode = url.searchParams.get('mode') || 'login';
	const sessionToken = cookies.get(SESSION_COOKIE);

	if (mode === 'link' && !sessionToken) {
		throw redirect(303, '/login?next=/settings');
	}

	const path =
		mode === 'link'
			? `/api/v1/me/oauth/${encodeURIComponent(provider)}/link`
			: `/api/v1/auth/oauth/${encodeURIComponent(provider)}`;

	const res = await apiFetchRaw(path, {
		method: 'GET',
		sessionToken: mode === 'link' ? sessionToken : undefined,
		searchParams: mode === 'login' ? { mode: 'login' } : undefined,
		redirect: 'manual'
	});

	if (res.status >= 300 && res.status < 400) {
		const location = res.headers.get('location');
		if (location && isAllowedAuthorizeURL(location)) throw redirect(302, location);
	}

	if (!res.ok) {
		const text = await res.text().catch(() => '');
		throw error(res.status, text || 'OAuth start failed');
	}

	const body = (await res.json().catch(() => null)) as { authorize_url?: string } | null;
	const authorizeURL = body?.authorize_url?.trim() ?? '';
	if (!authorizeURL || !isAllowedAuthorizeURL(authorizeURL)) {
		throw error(502, 'OAuth provider did not return a safe authorize URL');
	}
	throw redirect(302, authorizeURL);
};
