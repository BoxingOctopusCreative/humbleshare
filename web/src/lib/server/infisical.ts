import { InfisicalSDK } from '@infisical/sdk';
import {
	DEFAULT_INFISICAL_SECRET_PATH,
	applyInfisicalEnv,
	infisicalListPaths,
	mapInfisicalSecret,
	normalizeInfisicalPath,
	secretsToEnvMap,
	type InfisicalSecret
} from '$lib/server/infisical-map';

export type InfisicalBootstrap = {
	clientId: string;
	clientSecret: string;
	projectId: string;
	environment: string;
	siteUrl: string;
	secretPath: string;
};

export type SecretsFetcher = (boot: InfisicalBootstrap) => Promise<InfisicalSecret[]>;

type LineWriter = { write(s: string): void };

const DEFAULT_SITE_URL = 'https://app.infisical.com';
const FETCH_TIMEOUT_MS = 30_000;

export function readInfisicalBootstrap(
	env: NodeJS.ProcessEnv = process.env
): InfisicalBootstrap | null {
	const clientId = env.INFISICAL_UNIVERSAL_AUTH_CLIENT_ID?.trim() ?? '';
	const clientSecret = env.INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET?.trim() ?? '';
	const projectId = env.INFISICAL_PROJECT_ID?.trim() ?? '';
	const environment = env.INFISICAL_ENVIRONMENT?.trim() ?? '';
	if (!clientId || !clientSecret || !projectId || !environment) {
		return null;
	}
	const siteUrl = env.INFISICAL_SITE_URL?.trim() || DEFAULT_SITE_URL;
	const secretPath = normalizeInfisicalPath(
		env.INFISICAL_SECRET_PATH?.trim() || DEFAULT_INFISICAL_SECRET_PATH
	);
	return { clientId, clientSecret, projectId, environment, siteUrl, secretPath };
}

function appendListedSecrets(
	out: InfisicalSecret[],
	listed: {
		secrets: Array<{ secretPath?: string; secretKey: string; secretValue: string }>;
		imports?: Array<{
			secretPath: string;
			secrets: Array<{ secretPath?: string; secretKey: string; secretValue: string }>;
		}>;
	},
	fallbackPath: string
): void {
	for (const s of listed.secrets) {
		out.push({
			path: s.secretPath ?? fallbackPath,
			key: s.secretKey,
			value: s.secretValue
		});
	}
	for (const imp of listed.imports ?? []) {
		for (const s of imp.secrets) {
			out.push({
				path: s.secretPath ?? imp.secretPath ?? fallbackPath,
				key: s.secretKey,
				value: s.secretValue
			});
		}
	}
}

async function fetchInfisicalSecrets(boot: InfisicalBootstrap): Promise<InfisicalSecret[]> {
	const client = new InfisicalSDK({ siteUrl: boot.siteUrl });
	await client.auth().universalAuth.login({
		clientId: boot.clientId,
		clientSecret: boot.clientSecret
	});

	const out: InfisicalSecret[] = [];
	for (const path of infisicalListPaths(boot.secretPath)) {
		const listed = await client.secrets().listSecrets({
			projectId: boot.projectId,
			environment: boot.environment,
			secretPath: path,
			recursive: true,
			expandSecretReferences: true,
			includeImports: true,
			viewSecretValue: true
		});
		appendListedSecrets(out, listed, path);
	}
	return out;
}

function infisicalLog(logOut: LineWriter, message: string): void {
	logOut.write(`${message}\n`);
}

function logInfisicalError(
	message: string,
	boot: InfisicalBootstrap,
	cause: unknown,
	errOut: LineWriter
): void {
	const detail = cause instanceof Error ? cause.message : String(cause);
	errOut.write(
		`humbleshare-web: infisical: ${message} (site=${boot.siteUrl} project=${boot.projectId} env=${boot.environment} path=${boot.secretPath}): ${detail}\n`
	);
}

function logSeenSecrets(secrets: InfisicalSecret[], logOut: LineWriter): void {
	for (const s of secrets) {
		const path = normalizeInfisicalPath(s.path);
		const key = s.key.trim();
		const envKey = mapInfisicalSecret(s.path, s.key);
		const empty = s.value.trim() === '';
		if (!envKey) {
			infisicalLog(
				logOut,
				`humbleshare-web: infisical: seen path=${path} key=${key} mapped=(none) empty=${empty}`
			);
		} else {
			infisicalLog(
				logOut,
				`humbleshare-web: infisical: seen path=${path} key=${key} mapped=${envKey} empty=${empty}`
			);
		}
	}
}

/**
 * Load Infisical secrets for the web app when Machine Identity bootstrap env is set.
 * Applies values only for unset process.env keys. Fail-closed when enabled and fetch fails.
 */
export async function loadInfisicalSecrets(options?: {
	fetch?: SecretsFetcher;
	env?: NodeJS.ProcessEnv;
	errOut?: LineWriter;
	logOut?: LineWriter;
}): Promise<void> {
	const env = options?.env ?? process.env;
	const errOut = options?.errOut ?? process.stderr;
	const logOut = options?.logOut ?? process.stderr;
	const boot = readInfisicalBootstrap(env);
	if (!boot) {
		infisicalLog(
			logOut,
			'humbleshare-web: infisical: skipped (bootstrap env not set; using process env / defaults)'
		);
		return;
	}

	const fetch = options?.fetch ?? fetchInfisicalSecrets;
	const paths = infisicalListPaths(boot.secretPath);
	infisicalLog(
		logOut,
		`humbleshare-web: infisical: connecting (site=${boot.siteUrl} project=${boot.projectId} env=${boot.environment} paths=${paths.join(',')})`
	);

	try {
		const secrets = await Promise.race([
			fetch(boot),
			new Promise<never>((_, reject) => {
				setTimeout(() => reject(new Error('timed out')), FETCH_TIMEOUT_MS);
			})
		]);
		logSeenSecrets(secrets, logOut);
		const mapped = secretsToEnvMap(secrets);
		const keys = Object.keys(mapped).sort();
		applyInfisicalEnv(mapped, env);
		infisicalLog(logOut, `humbleshare-web: infisical: ok mapped=${keys.length}`);
		for (const k of keys) {
			infisicalLog(logOut, `humbleshare-web: infisical: mapped ${k}`);
		}
	} catch (err) {
		logInfisicalError('failed to load secrets', boot, err, errOut);
		throw new Error(`infisical: ${err instanceof Error ? err.message : String(err)}`);
	}
}
