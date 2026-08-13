import { describe, expect, it } from 'vitest';
import {
	applyInfisicalEnv,
	infisicalListPaths,
	mapInfisicalSecret,
	normalizeInfisicalPath,
	secretsToEnvMap
} from '$lib/server/infisical-map';

describe('normalizeInfisicalPath', () => {
	it('canonicalizes trailing slashes and missing leading slash', () => {
		expect(normalizeInfisicalPath('/ui/api/')).toBe('/ui/api');
		expect(normalizeInfisicalPath('ui/auth')).toBe('/ui/auth');
		expect(normalizeInfisicalPath('')).toBe('/');
	});
});

describe('infisicalListPaths', () => {
	it('returns the bootstrap path only', () => {
		expect(infisicalListPaths('/ui')).toEqual(['/ui']);
		expect(infisicalListPaths('/api')).toEqual(['/api']);
		expect(infisicalListPaths('/')).toEqual(['/']);
		expect(infisicalListPaths('/shared')).toEqual(['/shared']);
	});
});

describe('mapInfisicalSecret', () => {
	it('maps documented /ui path keys', () => {
		expect(mapInfisicalSecret('/ui/api', 'internal_url')).toBe('API_INTERNAL_URL');
		expect(mapInfisicalSecret('/ui/auth', 'cookie_secure')).toBe('COOKIE_SECURE');
		expect(mapInfisicalSecret('/ui/sentry', 'dsn')).toBe('PUBLIC_SENTRY_DSN');
		expect(mapInfisicalSecret('/ui/sentry', 'server_dsn')).toBe('SENTRY_DSN');
		expect(mapInfisicalSecret('/ui/sentry', 'environment')).toBe('SENTRY_ENVIRONMENT');
		expect(mapInfisicalSecret('/shared/auth', 'access_token')).toBe('');
		expect(mapInfisicalSecret('/ui/api', 'unknown')).toBe('');
		expect(mapInfisicalSecret('/api/db', 'url')).toBe('');
	});
});

describe('secretsToEnvMap + applyInfisicalEnv', () => {
	it('maps secrets and only fills unset env keys', () => {
		const mapped = secretsToEnvMap([
			{ path: '/ui/api', key: 'internal_url', value: ' http://infisical-api:8080 ' },
			{ path: '/ui/auth', key: 'cookie_secure', value: 'true' },
			{ path: '/shared/auth', key: 'access_token', value: 'tok' }
		]);
		expect(mapped).toEqual({
			API_INTERNAL_URL: 'http://infisical-api:8080',
			COOKIE_SECURE: 'true'
		});

		const env: NodeJS.ProcessEnv = {
			COOKIE_SECURE: 'already-set'
		};
		const applied = applyInfisicalEnv(mapped, env);
		expect(applied).toEqual(['API_INTERNAL_URL']);
		expect(env.COOKIE_SECURE).toBe('already-set');
		expect(env.API_INTERNAL_URL).toBe('http://infisical-api:8080');
	});
});
