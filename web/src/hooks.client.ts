import * as Sentry from '@sentry/sveltekit';
import { env } from '$env/dynamic/public';
import { initClientSentry } from '$lib/sentry-client';

// Build-time / Vite .env PUBLIC_SENTRY_DSN (Docker ARG). Infisical-only DSNs arrive via layout.
initClientSentry(env.PUBLIC_SENTRY_DSN);

export const handleError = Sentry.handleErrorWithSentry();
