import { ApiError, readApiError } from '$lib/types';
import { apiBase, cookieSecure, SESSION_COOKIE, SESSION_MAX_AGE } from '$lib/server/env';
import type { Cookies } from '@sveltejs/kit';

type FetchOpts = {
	method?: string;
	body?: unknown;
	sessionToken?: string;
	cookie?: string;
	searchParams?: Record<string, string | undefined>;
	redirect?: RequestRedirect;
};

function buildUrl(path: string, searchParams?: Record<string, string | undefined>): URL {
	const url = new URL(path.startsWith('http') ? path : `${apiBase()}${path.startsWith('/') ? path : `/${path}`}`);
	if (searchParams) {
		for (const [k, v] of Object.entries(searchParams)) {
			if (v != null && v !== '') url.searchParams.set(k, v);
		}
	}
	return url;
}

function buildHeaders(opts: FetchOpts): Record<string, string> {
	const headers: Record<string, string> = {
		Accept: 'application/json'
	};
	if (opts.body !== undefined && !(opts.body instanceof FormData)) {
		headers['Content-Type'] = 'application/json';
	}
	if (opts.sessionToken) {
		headers.Authorization = `Bearer ${opts.sessionToken}`;
	}
	if (opts.cookie) {
		headers.Cookie = opts.cookie;
	}
	return headers;
}

/**
 * Server-only proxy to the Go API. Returns the raw Response (for Set-Cookie / redirects).
 */
export async function apiFetchRaw(path: string, opts: FetchOpts = {}): Promise<Response> {
	const url = buildUrl(path, opts.searchParams);
	let body: BodyInit | undefined;
	if (opts.body !== undefined) {
		body = opts.body instanceof FormData ? opts.body : JSON.stringify(opts.body);
	}
	return fetch(url, {
		method: opts.method ?? (opts.body !== undefined ? 'POST' : 'GET'),
		headers: buildHeaders(opts),
		body,
		redirect: opts.redirect ?? 'follow'
	});
}

/**
 * Server-only proxy to the Go API. Session token stays on the BFF.
 */
export async function apiFetch<T = unknown>(path: string, opts: FetchOpts = {}): Promise<T> {
	const res = await apiFetchRaw(path, opts);

	if (!res.ok) {
		throw new ApiError(res.status, await readApiError(res));
	}

	if (res.status === 204) {
		return undefined as T;
	}

	const text = await res.text();
	if (!text) return undefined as T;
	return JSON.parse(text) as T;
}

/** Extract humbleshare_session value from an API Set-Cookie header. */
export function extractSessionFromSetCookie(res: Response): string | null {
	const headers = res.headers as Headers & { getSetCookie?: () => string[] };
	const list =
		typeof headers.getSetCookie === 'function'
			? headers.getSetCookie()
			: [res.headers.get('set-cookie')].filter((v): v is string => Boolean(v));

	for (const raw of list) {
		// Prefer the first cookie segment that starts with our name (avoid comma-joined cookies).
		for (const part of raw.split(/,(?=\s*[^;=]+=)/)) {
			const trimmed = part.trim();
			if (trimmed.toLowerCase().startsWith('humbleshare_session=')) {
				const value = trimmed.slice('humbleshare_session='.length).split(';')[0];
				if (value) return decodeURIComponent(value.trim());
			}
		}
	}
	return null;
}

export function setSessionCookie(cookies: Cookies, value: string): void {
	cookies.set(SESSION_COOKIE, value, {
		path: '/',
		httpOnly: true,
		sameSite: 'lax',
		secure: cookieSecure(),
		maxAge: SESSION_MAX_AGE
	});
}

export function clearSessionCookie(cookies: Cookies): void {
	cookies.delete(SESSION_COOKIE, { path: '/' });
}

/** Apply session cookie from an API login/register response if present. */
export function applySessionFromApiResponse(cookies: Cookies, res: Response): boolean {
	const value = extractSessionFromSetCookie(res);
	if (!value) return false;
	setSessionCookie(cookies, value);
	return true;
}
