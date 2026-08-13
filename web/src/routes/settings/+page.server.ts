import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import type { OAuthIdentity, User } from '$lib/types';
import { ApiError } from '$lib/types';

export const load: PageServerLoad = async ({ cookies, parent }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	if (!sessionToken) {
		throw redirect(303, '/login?next=/settings');
	}

	const layout = await parent();
	if (!layout.user) {
		throw redirect(303, '/login?next=/settings');
	}

	try {
		const [me, oauth] = await Promise.all([
			apiFetch<User>('/api/v1/me', { sessionToken }),
			apiFetch<OAuthIdentity[]>('/api/v1/me/oauth', { sessionToken }).catch(() => [])
		]);
		return {
			user: me,
			oauth: oauth ?? []
		};
	} catch (e) {
		if (e instanceof ApiError && e.status === 401) {
			cookies.delete(SESSION_COOKIE, { path: '/' });
			throw redirect(303, '/login?next=/settings');
		}
		throw e;
	}
};
