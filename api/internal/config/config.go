package config

// Config holds runtime settings loaded from env, optional TOML file, Infisical, and defaults.
type Config struct {
	HTTPAddr    string
	DatabaseURL string
	CORSOrigins []string

	// AuthJWTSecret signs short-lived JWTs (2FA pending, invite approve, OAuth state).
	AuthJWTSecret    string
	SuperadminEmail  string
	SuperadminPassword string
	CookieSecure     bool
	SessionMaxAge    int // seconds
	PublicWebBaseURL string

	// Mailgun (production). Ignored when SMTP_HOST is set.
	MailgunAPIKey  string
	MailgunDomain  string
	MailgunFrom    string
	MailgunAPIBase string

	// SMTP (local Mailpit / catcher). When SMTPHost is set, used instead of Mailgun.
	SMTPHost string
	SMTPPort string
	SMTPFrom string

	// OAuth (optional; link-only after account exists).
	GoogleOAuthClientID      string
	GoogleOAuthClientSecret  string
	DiscordOAuthClientID     string
	DiscordOAuthClientSecret string
	SteamAPIKey              string // Steam Web API key for OpenID link

	// Cloudflare Turnstile (optional bot protection).
	TurnstileSiteKey   string
	TurnstileSecretKey string

	// Cloudflare R2 (optional; avatar upload needs user-assets bucket + public URL).
	// Shared credentials; two buckets like Kurator: static assets vs user assets.
	R2AccountID                 string
	R2AccessKeyID               string
	R2SecretAccessKey           string
	R2Bucket                    string // static assets (brand, openapi, etc.)
	R2PublicBaseURL             string // public CDN for static bucket
	R2UserAssetsBucket          string // user uploads (avatars)
	R2UserAssetsPublicBaseURL   string // public CDN for user-assets bucket

	ClaimRateLimitPerMinute int

	SteamMetadataCacheTTLSeconds int
	SteamRatePerSecond           float64

	// Sentry (optional; empty DSN disables the SDK).
	SentryDSN              string
	SentryEnvironment      string
	SentryRelease          string
	SentryTracesSampleRate float64
}
