import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { apiFetch } from '$lib/server/api';
import { SESSION_COOKIE } from '$lib/server/env';
import type { PublicProfile } from '$lib/types';
import { ApiError } from '$lib/types';

export const load: PageServerLoad = async ({ cookies, params }) => {
	const sessionToken = cookies.get(SESSION_COOKIE);
	try {
		const profile = await apiFetch<PublicProfile>(`/api/v1/profiles/${encodeURIComponent(params.slug)}`, {
			sessionToken
		});
		const bio = profile.bio?.trim();
		return {
			profile,
			seo: {
				title: `${profile.display_name} · HumbleShare`,
				description: bio
					? bio.slice(0, 160)
					: `${profile.display_name} on HumbleShare`
			}
		};
	} catch (e) {
		if (e instanceof ApiError && e.status === 404) {
			error(404, 'Profile not found');
		}
		throw e;
	}
};
