import type { PageServerLoad } from './$types';

/** Public splash; catalog lives at /games. */
export const load: PageServerLoad = async ({ parent }) => {
	const { user } = await parent();
	return { user };
};
