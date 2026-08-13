import { afterEach, describe, expect, it, vi } from 'vitest';

describe('loadInfisicalSecrets', () => {
	afterEach(() => {
		vi.resetModules();
		vi.unstubAllEnvs();
	});

	it('skips when bootstrap env is absent', async () => {
		vi.stubEnv('INFISICAL_UNIVERSAL_AUTH_CLIENT_ID', '');
		vi.stubEnv('INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET', '');
		vi.stubEnv('INFISICAL_PROJECT_ID', '');
		vi.stubEnv('INFISICAL_ENVIRONMENT', '');

		const chunks: string[] = [];
		const logOut = { write: (s: string) => chunks.push(s) };

		const { loadInfisicalSecrets } = await import('$lib/server/infisical');
		const fetch = vi.fn();
		await loadInfisicalSecrets({ fetch, logOut });
		expect(fetch).not.toHaveBeenCalled();
		expect(chunks.join('')).toContain('humbleshare-web: infisical: skipped');
	});

	it('applies mapped secrets, does not overwrite existing env, and logs status', async () => {
		const chunks: string[] = [];
		const logOut = { write: (s: string) => chunks.push(s) };

		const { loadInfisicalSecrets } = await import('$lib/server/infisical');
		const env: NodeJS.ProcessEnv = {
			INFISICAL_UNIVERSAL_AUTH_CLIENT_ID: 'cid',
			INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET: 'csecret',
			INFISICAL_PROJECT_ID: 'proj',
			INFISICAL_ENVIRONMENT: 'dev',
			COOKIE_SECURE: 'already'
		};

		await loadInfisicalSecrets({
			env,
			logOut,
			fetch: async () => [
				{ path: '/ui/auth', key: 'cookie_secure', value: 'from-infisical' },
				{ path: '/ui/api', key: 'internal_url', value: 'http://infisical-api:8080' },
				{ path: '/ui/unknown', key: 'noise', value: 'x' }
			]
		});

		expect(env.COOKIE_SECURE).toBe('already');
		expect(env.API_INTERNAL_URL).toBe('http://infisical-api:8080');

		const msg = chunks.join('');
		expect(msg).toContain('humbleshare-web: infisical: connecting');
		expect(msg).toContain('paths=/ui');
		expect(msg).toContain(
			'seen path=/ui/auth key=cookie_secure mapped=COOKIE_SECURE empty=false'
		);
		expect(msg).toContain(
			'seen path=/ui/api key=internal_url mapped=API_INTERNAL_URL empty=false'
		);
		expect(msg).toContain('seen path=/ui/unknown key=noise mapped=(none) empty=false');
		expect(msg).toContain('ok mapped=2');
		expect(msg).toContain('mapped API_INTERNAL_URL');
		expect(msg).toContain('mapped COOKIE_SECURE');
		expect(msg).not.toContain('csecret');
	});

	it('writes stderr and throws when Infisical fetch fails', async () => {
		const chunks: string[] = [];
		const errOut = { write: (s: string) => chunks.push(s) };

		const { loadInfisicalSecrets } = await import('$lib/server/infisical');
		const env: NodeJS.ProcessEnv = {
			INFISICAL_UNIVERSAL_AUTH_CLIENT_ID: 'cid',
			INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET: 'csecret',
			INFISICAL_PROJECT_ID: 'proj',
			INFISICAL_ENVIRONMENT: 'dev'
		};

		await expect(
			loadInfisicalSecrets({
				env,
				errOut,
				logOut: { write: () => {} },
				fetch: async () => {
					throw new Error('connection refused');
				}
			})
		).rejects.toThrow(/infisical/);

		const msg = chunks.join('');
		expect(msg).toContain('humbleshare-web: infisical:');
		expect(msg).toContain('project=proj');
		expect(msg).toContain('connection refused');
		expect(msg).not.toContain('csecret');
	});
});
