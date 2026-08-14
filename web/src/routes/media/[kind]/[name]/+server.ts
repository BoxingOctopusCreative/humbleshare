import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { apiBase } from '$lib/server/env';
import { env } from '$env/dynamic/private';

function refererAllowed(request: Request, pageURL: URL): boolean {
	const ref = request.headers.get('referer') || request.headers.get('origin');
	if (!ref) return true;
	try {
		return new URL(ref).host === pageURL.host;
	} catch {
		return false;
	}
}

export const GET: RequestHandler = async ({ params, request, url }) => {
	if (!refererAllowed(request, url)) {
		error(403, 'hotlink denied');
	}

	const kind = params.kind;
	const name = params.name;
	if (
		(kind !== 'avatars' && kind !== 'banners' && kind !== 'svg') ||
		!name ||
		name.includes('..') ||
		name.includes('/')
	) {
		error(400, 'invalid media path');
	}

	const headers: Record<string, string> = { Accept: '*/*' };
	const secret = (env.MEDIA_PROXY_SECRET ?? '').trim();
	if (secret) {
		headers['X-Humbleshare-Media-Proxy'] = secret;
	}

	const upstream = await fetch(
		`${apiBase()}/api/v1/media/${encodeURIComponent(kind)}/${encodeURIComponent(name)}`,
		{ headers }
	);
	if (!upstream.ok) {
		error(upstream.status === 404 ? 404 : upstream.status === 403 ? 403 : 502, 'media unavailable');
	}

	const outHeaders = new Headers();
	const ct = upstream.headers.get('content-type');
	if (ct) outHeaders.set('content-type', ct);
	const cc = upstream.headers.get('cache-control');
	outHeaders.set('cache-control', cc || 'public, max-age=86400');
	outHeaders.set('x-content-type-options', 'nosniff');

	return new Response(upstream.body, { status: 200, headers: outHeaders });
};
