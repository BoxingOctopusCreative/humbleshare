import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiFetchRaw } from '$lib/server/api';

export const POST: RequestHandler = async ({ request }) => {
	const body = await request.json().catch(() => ({}));
	const res = await apiFetchRaw('/api/v1/auth/request-access', {
		method: 'POST',
		body
	});
	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		return json(
			{
				error: true,
				msg:
					(data as { message?: string; msg?: string }).message ||
					(data as { msg?: string }).msg ||
					'Request failed'
			},
			{ status: res.status }
		);
	}
	return json(data, { status: 201 });
};
