import { sentrySvelteKit } from '@sentry/sveltekit';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		// sentrySvelteKit must run before sveltekit() for source map upload.
		sentrySvelteKit({
			org: process.env['SENTRY_ORG'],
			project: process.env['SENTRY_PROJECT'] ?? 'humbleshare-ui',
			authToken: process.env['SENTRY_AUTH_TOKEN'],
			telemetry: false
		}),
		tailwindcss(),
		sveltekit()
	],
	server: {
		host: '0.0.0.0',
		port: 5173,
		// When behind Traefik (docker-compose.dev.yml), point the HMR client at the proxy.
		hmr: process.env['HMR_HOST']
			? {
					host: process.env['HMR_HOST'],
					clientPort: Number(process.env['HMR_CLIENT_PORT'] || '80'),
					protocol: 'ws'
				}
			: undefined,
		watch: {
			usePolling: process.env['CHOKIDAR_USEPOLLING'] === '1'
		}
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}'],
		environment: 'node'
	}
});
