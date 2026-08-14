import type { PageLoad } from './$types';
import { SITE_NAME } from '$lib/seo';

export const load: PageLoad = () => ({
	seo: {
		title: `Privacy policy · ${SITE_NAME}`,
		description: 'How HumbleShare collects, uses, and stores account and usage data.'
	}
});
