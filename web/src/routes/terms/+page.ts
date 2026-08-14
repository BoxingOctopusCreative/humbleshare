import type { PageLoad } from './$types';
import { SITE_NAME } from '$lib/seo';

export const load: PageLoad = () => ({
	seo: {
		title: `Terms of use · ${SITE_NAME}`,
		description: 'Rules for using HumbleShare: invite-only access, keys, and acceptable use.'
	}
});
