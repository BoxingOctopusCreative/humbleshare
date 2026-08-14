# HumbleShare

Self-hosted spare Steam key / gift-link share for a small group of friends. Replaces the Humble Bundle spreadsheet.

**Stack:** Go (Fiber) + Postgres 16 + SvelteKit 2 (Svelte 5) + Tailwind 4.

## Local setup

```bash
cp .env.example .env
# set SUPERADMIN_EMAIL and AUTH_JWT_SECRET

docker compose -f docker-compose.dev.yml up --build
```

Dev compose puts **Traefik** in front (same host / path split idea as production):

- App: http://humbleshare.localhost
- Games catalog: http://humbleshare.localhost/games
- Health: http://humbleshare.localhost/health
- Login: http://humbleshare.localhost/login
- Request access: http://humbleshare.localhost/request-access
- Traefik dashboard: http://localhost:8081
- Mailpit (caught email): http://localhost:8025

`*.localhost` resolves to `127.0.0.1` on current browsers / macOS. Traefik routes `/api/v1`, `/health`, and `/webhooks` to the Go API; everything else (including the SvelteKit BFF under `/api/...`) goes to the web container.

On first boot the API creates a **superadmin** for `SUPERADMIN_EMAIL`. If `SUPERADMIN_PASSWORD` is set (Infisical `/api/auth` key `superadmin_password`), the account is created immediately. Otherwise an invite is emailed (Mailpit in dev).

Or run pieces on the host (no Traefik; hit Vite/API ports directly):

```bash
docker compose -f docker-compose.dev.yml up -d postgres mailpit

cd api && cp ../.env.example .env
make dev

cd web && npm i && npm run dev
```

## Environment variables

Config precedence: **env > CLI > optional TOML > Infisical > defaults**.

When Infisical Machine Identity bootstrap env is set, secrets are loaded on startup
(API: `/api` recursively; web: `/ui` recursively). Failures abort startup. Process env
still overrides Infisical for operational tweaks.

### Infisical paths (folder → key → env)

| Path | Key | Env |
| --- | --- | --- |
| `/api/db` | `url` | `DATABASE_URL` |
| `/api/db` | `ca_cert` | PEM; written to a temp file and `PGSSLROOTCERT` is set if unset |
| `/api/auth` | `jwt_secret` | `AUTH_JWT_SECRET` |
| `/api/auth` | `superadmin_email` | `SUPERADMIN_EMAIL` |
| `/api/auth` | `superadmin_password` | `SUPERADMIN_PASSWORD` |
| `/api/auth/superadmin_password` | `password` | `SUPERADMIN_PASSWORD` |
| `/api/auth` | `cookie_secure` | `COOKIE_SECURE` |
| `/api/auth` | `session_max_age_seconds` | `SESSION_MAX_AGE_SECONDS` |
| `/api/auth` | `public_web_base_url` | `PUBLIC_WEB_BASE_URL` |
| `/api/mailgun` | `api_key`, `domain`, `from`, `api_base` | `MAILGUN_*` |
| `/api/auth/google` | `client_id`, `client_secret` | `GOOGLE_OAUTH_*` |
| `/api/auth/discord` | `client_id`, `client_secret` | `DISCORD_OAUTH_*` |
| `/api/auth/steam` | `api_key` | `STEAM_API_KEY` |
| `/api/auth/turnstile` | `site_key`, `secret_key` | `TURNSTILE_SITE_KEY` / `TURNSTILE_SECRET_KEY` |
| `/api/r2` | `account_id`, `access_key_id`, `secret_access_key`, `bucket`, `public_base_url`, `user_assets_bucket`, `user_assets_public_base_url` | `R2_*` (static + user-assets buckets) |
| `/api/steam` | `claim_rate_limit_per_minute` / `metadata_cache_ttl_seconds` / `rate_per_second` | claim / Steam knobs |
| `/api/sentry` | `dsn`, `environment`, `release`, `traces_sample_rate` | `SENTRY_*` (API) |
| `/ui/api` | `internal_url` | `API_INTERNAL_URL` |
| `/ui/auth` | `cookie_secure` | `COOKIE_SECURE` |
| `/ui/sentry` | `dsn`, `server_dsn`, `environment`, `release`, `traces_sample_rate` | `PUBLIC_SENTRY_DSN` / `SENTRY_*` (web) |

Operational knobs such as `HTTP_ADDR` and `CORS_ORIGINS` stay on env/TOML.

### Bootstrap + common env

| Variable | Service | Description |
| --- | --- | --- |
| `DATABASE_URL` | API | Postgres connection string |
| `AUTH_JWT_SECRET` | API | Signs 2FA pending / invite approve / OAuth state JWTs |
| `SUPERADMIN_EMAIL` | API | First superadmin email |
| `SUPERADMIN_PASSWORD` | API | Optional; creates that account on boot (skips invite) |
| `MAILGUN_API_KEY` / `MAILGUN_DOMAIN` | API | Production email (ignored when `SMTP_HOST` is set) |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_FROM` | API | Local SMTP (Mailpit in `docker-compose.dev.yml`) |
| `PUBLIC_WEB_BASE_URL` | API | Browser origin for invite/OAuth links |
| `API_INTERNAL_URL` | Web | Go API base URL for the BFF |
| `SENTRY_DSN` | API (+ optional web server) | Sentry DSN; empty disables the SDK |
| `PUBLIC_SENTRY_DSN` | Web | Browser Sentry DSN (same project DSN is fine) |
| `SENTRY_ENVIRONMENT` / `SENTRY_RELEASE` | API + Web | Environment and release tags |
| `COOKIE_SECURE` | API + Web | `true` in production HTTPS |
| `SESSION_MAX_AGE_SECONDS` | API | Session cookie max-age (default 30d) |
| `R2_ACCOUNT_ID` / `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` | API | Shared R2 credentials |
| `R2_BUCKET` / `R2_PUBLIC_BASE_URL` | API | Static assets bucket + public CDN (`https://assets.humbleshare.com`) |
| `R2_USER_ASSETS_BUCKET` / `R2_USER_ASSETS_PUBLIC_BASE_URL` | API | User uploads (avatars, banners) + public CDN (`https://user-assets.humbleshare.com`) |

## Auth

1. Visitors submit **Request access** with an email.
2. Admins (and `SUPERADMIN_EMAIL`) get a Mailgun notification; approve in Admin or via the email link.
3. The requester gets an invite email, sets a password, and signs in.
4. Optional: enable TOTP 2FA and link Google / Discord / Steam from Settings.
5. Superadmin can promote other users to `admin` in the portal.

Sessions use an opaque `humbleshare_session` cookie (hashed at rest). The SvelteKit BFF proxies `/api/*` so the browser never talks to Go with secrets in client JS.

## Humble session cookie

For **Admin → Import**:

1. Log into [humblebundle.com](https://www.humblebundle.com) in a browser.
2. DevTools → Application/Storage → Cookies → copy `_simpleauth_sess`.
3. Paste into the import form: Humble **gift links** (`gift?key=…`) and/or **order gamekeys** (`downloads?key=…`). Steam CD keys go in Bulk paste.
4. The cookie is sent only for that request and is never written to the database or logs.

## Production

Root `docker-compose.yml` is the Traefik/Infisical production stack (GHCR images on an external `shared-network`). Build and push `humbleshare-api` / `humbleshare-web` via CI; prefer Infisical for secrets (process env remains a valid override).

Dev Traefik mirrors production router names (`humbleshare-api` / `humbleshare-web`) and priorities. Local uses plain HTTP on `:80` and `PathPrefix(/api/v1)` so OAuth callbacks hit Go while the BFF keeps `/api/*`. Production labels still use the broader `PathPrefix(/api)` (TLS + Let's Encrypt on the external Traefik).

## Design notes

- Claiming is a single conditional `UPDATE … WHERE status = 'available'` (see `ClaimOldestAvailable`). Concurrent losers get **409**.
- Public game list never includes `code_value`. The key appears only after a successful claim (and on admin inventory).
- Game identity comes from Humble titles or manual entry; Steam metadata is resolved from the **title**, never from the opaque key.
- Claims store `claimed_by_user_id` plus a denormalized display name.
