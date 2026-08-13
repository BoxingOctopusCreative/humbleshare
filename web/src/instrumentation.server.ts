import * as Sentry from '@sentry/sveltekit';
import { loadInfisicalSecrets } from '$lib/server/infisical';

// Load Infisical first so SENTRY_DSN / PUBLIC_SENTRY_DSN can come from /ui/sentry.
await loadInfisicalSecrets();

const dsn = (process.env.SENTRY_DSN ?? process.env.PUBLIC_SENTRY_DSN ?? '').trim();
const tracesSampleRate = Number(process.env.SENTRY_TRACES_SAMPLE_RATE ?? '1');

Sentry.init({
	dsn: dsn || undefined,
	environment: process.env.SENTRY_ENVIRONMENT || process.env.NODE_ENV || 'development',
	release: process.env.SENTRY_RELEASE || undefined,
	tracesSampleRate: Number.isFinite(tracesSampleRate) ? tracesSampleRate : 1,
	enableLogs: true
});
