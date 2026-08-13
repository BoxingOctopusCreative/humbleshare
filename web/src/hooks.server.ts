import * as Sentry from '@sentry/sveltekit';
import type { Handle, HandleServerError } from '@sveltejs/kit';

// Infisical + Sentry.init live in instrumentation.server.ts (runs once at process start).

export const handleError: HandleServerError = Sentry.handleErrorWithSentry();

export const handle: Handle = Sentry.sentryHandle();
