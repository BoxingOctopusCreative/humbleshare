import type { RequestHandler } from './$types';

/** Public URLs listed for crawlers. Auth-gated routes are omitted. */
const PATHS = ['/', '/login', '/request-access', '/privacy', '/terms', '/sitemap'] as const;

export const GET: RequestHandler = async ({ url }) => {
	const origin = url.origin.replace(/\/$/, '');
	const lastmod = new Date().toISOString().slice(0, 10);
	const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${PATHS.map(
	(path) => `  <url>
    <loc>${origin}${path === '/' ? '/' : path}</loc>
    <lastmod>${lastmod}</lastmod>
  </url>`
).join('\n')}
</urlset>
`;

	return new Response(body, {
		headers: {
			'Content-Type': 'application/xml; charset=utf-8',
			'Cache-Control': 'public, max-age=3600'
		}
	});
};
