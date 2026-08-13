/**
 * Maps Infisical (secretPath, secretKey) under /ui → process.env names used by the SvelteKit app.
 * Paths are normalized with a leading slash and no trailing slash.
 */

export const DEFAULT_INFISICAL_SECRET_PATH = '/ui';

const INFISICAL_ENV_BY_PATH_KEY: Record<string, Record<string, string>> = {
	'/ui/api': {
		internal_url: 'API_INTERNAL_URL'
	},
	'/ui/auth': {
		cookie_secure: 'COOKIE_SECURE'
	},
	'/ui/sentry': {
		dsn: 'PUBLIC_SENTRY_DSN',
		server_dsn: 'SENTRY_DSN',
		environment: 'SENTRY_ENVIRONMENT',
		release: 'SENTRY_RELEASE',
		traces_sample_rate: 'SENTRY_TRACES_SAMPLE_RATE'
	}
};

/** Canonicalize Infisical folder paths for map lookup. */
export function normalizeInfisicalPath(p: string): string {
	let path = p.trim();
	if (!path) return '/';
	if (!path.startsWith('/')) path = `/${path}`;
	while (path.length > 1 && path.endsWith('/')) {
		path = path.slice(0, -1);
	}
	return path;
}

/** Returns the HumbleShare env var name for a path+key, or "" if unmapped. */
export function mapInfisicalSecret(secretPath: string, secretKey: string): string {
	const pathMap = INFISICAL_ENV_BY_PATH_KEY[normalizeInfisicalPath(secretPath)];
	if (!pathMap) return '';
	return pathMap[secretKey.trim()] ?? '';
}

/** Bootstrap path used as the recursive Infisical list root. */
export function infisicalListPaths(secretPath: string): string[] {
	return [normalizeInfisicalPath(secretPath)];
}

export type InfisicalSecret = {
	path: string;
	key: string;
	value: string;
};

/** Map Infisical secrets to env-var keys (first mapped value wins per env key). */
export function secretsToEnvMap(secrets: InfisicalSecret[]): Record<string, string> {
	const out: Record<string, string> = {};
	for (const s of secrets) {
		const envKey = mapInfisicalSecret(s.path, s.key);
		if (!envKey) continue;
		const v = s.value.trim();
		if (!v) continue;
		if (out[envKey] === undefined) {
			out[envKey] = v;
		}
	}
	return out;
}

/**
 * Apply Infisical values as a base layer: only set process.env keys that are unset/blank.
 * Existing process env wins (same precedence idea as the API).
 */
export function applyInfisicalEnv(
	envMap: Record<string, string>,
	env: NodeJS.ProcessEnv = process.env
): string[] {
	const applied: string[] = [];
	for (const [key, value] of Object.entries(envMap)) {
		const existing = env[key]?.trim();
		if (existing) continue;
		env[key] = value;
		applied.push(key);
	}
	return applied;
}
