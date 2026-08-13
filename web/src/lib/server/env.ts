import { env } from '$env/dynamic/private';

/** httpOnly session cookie set by the BFF after login/register. */
export const SESSION_COOKIE = 'humbleshare_session';

/** Default session lifetime (30 days), matching API SESSION_MAX_AGE_SECONDS. */
export const SESSION_MAX_AGE = 60 * 60 * 24 * 30;

/** Server-side URL to the Go API. Browser never calls this directly. */
export function apiBase(): string {
	return (env.API_INTERNAL_URL ?? 'http://127.0.0.1:8080').replace(/\/$/, '');
}

export function cookieSecure(): boolean {
	return (env.COOKIE_SECURE ?? 'false').toLowerCase() === 'true';
}
