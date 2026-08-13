import * as Sentry from '@sentry/sveltekit';

/** Browser Sentry init. Safe to call from hooks.client and after Infisical fills process.env. */
export function initClientSentry(dsn: string | undefined | null): void {
	const trimmed = (dsn ?? '').trim();
	if (!trimmed) return;
	if (Sentry.getClient()?.getDsn()) return;

	Sentry.init({
		dsn: trimmed,
		environment: import.meta.env.MODE,
		tracesSampleRate: import.meta.env.PROD ? 0.2 : 1.0,
		integrations: [
			Sentry.replayIntegration({
				maskAllText: true,
				blockAllMedia: true
			})
		],
		replaysSessionSampleRate: 0.1,
		replaysOnErrorSampleRate: 1.0,
		enableLogs: true
	});
}
