import type { LayoutServerLoad } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import type { User } from '$lib/types';
import { ApiError } from '$lib/types';

export const load: LayoutServerLoad = async ({ cookies }) => {
	const sentryDsn = (process.env.PUBLIC_SENTRY_DSN ?? '').trim();
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (!sessionToken) {
		return { user: null, hasAccess: false, hasAdmin: false, sentryDsn };
	}

	try {
		const user = await apiFetch<User>('/api/v1/me', { sessionToken });
		const role = user?.role ?? '';
		const hasAdmin = role === 'admin' || role === 'superadmin';
		return {
			user: user
				? {
						id: user.id,
						email: user.email,
						display_name: user.display_name,
						bio: user.bio ?? '',
						steam_gamertag: user.steam_gamertag ?? '',
						discord_tag: user.discord_tag ?? '',
						avatar_url: user.avatar_url ?? '',
						role: user.role,
						two_factor_enabled: user.two_factor_enabled
					}
				: null,
			hasAccess: Boolean(user),
			hasAdmin,
			sentryDsn
		};
	} catch (e) {
		if (e instanceof ApiError && (e.status === 401 || e.status === 403)) {
			cookies.delete(SESSION_COOKIE, { path: '/' });
			return { user: null, hasAccess: false, hasAdmin: false, sentryDsn };
		}
		throw e;
	}
};
