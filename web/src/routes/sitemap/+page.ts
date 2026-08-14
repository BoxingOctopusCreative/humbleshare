import type { PageLoad } from './$types';
import { SITE_NAME } from '$lib/seo';

export const load: PageLoad = () => ({
	seo: {
		title: `Sitemap · ${SITE_NAME}`,
		description: 'Map of HumbleShare pages: public, member, and admin routes.'
	}
});
