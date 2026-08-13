export type User = {
	id: string;
	email: string;
	display_name: string;
	bio: string;
	steam_gamertag: string;
	discord_tag: string;
	avatar_url: string;
	banner_url: string;
	profile_slug: string;
	profile_public: boolean;
	role: string;
	two_factor_enabled: boolean;
};

export type PublicClaim = {
	game_id: string;
	game_name: string;
	header_image_url?: string | null;
	code_type: string;
	claimed_at: string;
};

export type PublicProfile = {
	profile_slug: string;
	display_name: string;
	bio: string;
	steam_gamertag: string;
	discord_tag: string;
	avatar_url: string;
	banner_url: string;
	profile_public: boolean;
	recent_claims: PublicClaim[];
	is_owner: boolean;
};

export type AccessRequest = {
	id: string;
	email: string;
	note: string;
	status: string;
	created_at: string;
	resolved_at?: string | null;
	resolved_by?: string | null;
};

export type OAuthIdentity = {
	id: string;
	user_id: string;
	provider: string;
	provider_user_id: string;
	provider_email?: string | null;
	created_at: string;
};

export type Game = {
	id: string;
	steam_appid: number | null;
	name: string;
	header_image_url?: string | null;
	capsule_image_url?: string | null;
	short_description?: string | null;
	store_url?: string | null;
	screenshot_urls?: string[] | null;
	available_count: number;
	created_at?: string;
};

export type ClaimResult = {
	id: string;
	game_id: string;
	code_type: string;
	code_value: string;
	claimed_by: string;
	claimed_at: string;
};

export type CodeRow = {
	id: string;
	game_id: string;
	game_name?: string;
	code_type: string;
	code_value: string;
	source?: string | null;
	source_order_id?: string | null;
	status: string;
	claimed_by?: string | null;
	claimed_at?: string | null;
	notes?: string | null;
	created_at: string;
};

export type AdminStats = {
	total_games: number;
	total_codes: number;
	available_codes: number;
	claimed_codes: number;
	dead_codes: number;
	claims_last_30_days: { date: string; count: number }[];
	top_claimers: { claimed_by: string; count: number }[];
	top_games: { game_id: string; name: string; count: number }[];
};

export type BulkLineResult = {
	line: number;
	title: string;
	code?: string;
	success: boolean;
	error?: string;
};

export type HumbleImportResult = {
	order_id: string;
	imported: number;
	skipped_duplicate: number;
	skipped_non_steam: number;
	skipped_unrevealed: number;
	failed: number;
	needs_title?: boolean;
	messages?: string[];
};

export class ApiError extends Error {
	status: number;

	constructor(status: number, message: string) {
		super(message);
		this.status = status;
		this.name = 'ApiError';
	}
}

export async function readApiError(res: Response): Promise<string> {
	const text = await res.text();
	try {
		const j = JSON.parse(text) as Record<string, unknown>;
		if (typeof j.message === 'string') return j.message;
		if (typeof j.msg === 'string') return j.msg;
		if (typeof j.error === 'string' && j.error !== 'true') return j.error;
	} catch {
		/* plain text */
	}
	return text.trim() || res.statusText || 'Request failed';
}
