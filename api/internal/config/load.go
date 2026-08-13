package config

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/pelletier/go-toml"
	"github.com/spf13/afero"
)

// Load merges configuration from environment variables (highest precedence), then CLI
// (LoadOptions), then an optional TOML file, then Infisical (when bootstrap env is set), then
// built-in defaults.
//
// Required at runtime (from any layer): DATABASE_URL, AUTH_JWT_SECRET, SUPERADMIN_EMAIL.
func Load(filesystem afero.Fs, opts *LoadOptions) (Config, error) {
	if filesystem == nil {
		filesystem = afero.NewOsFs()
	}
	if opts == nil {
		opts = &LoadOptions{}
	}

	path, err := resolveConfigPath(filesystem, opts.ConfigFile)
	if err != nil {
		return Config{}, err
	}

	var fc fileConfig
	if path != "" {
		b, err := afero.ReadFile(filesystem, path)
		if err != nil {
			return Config{}, fmt.Errorf("config file %q: %w", path, err)
		}
		if err := toml.Unmarshal(b, &fc); err != nil {
			return Config{}, fmt.Errorf("parse config file %q: %w", path, err)
		}
	}

	logOut := opts.InfisicalLogOut
	if logOut == nil {
		logOut = os.Stderr
	}
	infisicalLog(logOut, "humbleshare: config: precedence env > cli > toml > infisical > defaults")
	if path != "" {
		infisicalLog(logOut, "humbleshare: config: toml=%s", path)
	} else {
		infisicalLog(logOut, "humbleshare: config: toml=(none)")
	}

	inf, err := loadInfisicalEnvMap(context.Background(), opts.InfisicalFetcher, opts.InfisicalErrOut, logOut)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr: mergeString(
			"HTTP_ADDR",
			opts.HTTPAddr,
			fc.Server.HTTPAddr,
			inf,
			":8080",
		),
		DatabaseURL: mergeString(
			"DATABASE_URL",
			opts.DatabaseURL,
			fc.Database.URL,
			inf,
			"",
		),
		CORSOrigins: mergeStringSlice(
			"CORS_ORIGINS",
			"",
			fc.CORS.Origins,
			inf,
			[]string{"http://localhost:5173", "http://localhost:3000"},
		),
		AuthJWTSecret:   mergeString("AUTH_JWT_SECRET", "", fc.Auth.JWTSecret, inf, ""),
		SuperadminEmail: mergeString("SUPERADMIN_EMAIL", "", fc.Auth.SuperadminEmail, inf, ""),
		SuperadminPassword: mergeString(
			"SUPERADMIN_PASSWORD", "", fc.Auth.SuperadminPassword, inf, "",
		),
		CookieSecure:    mergeBool("COOKIE_SECURE", "", fc.Auth.CookieSecure, inf, false),
		SessionMaxAge: mergePositiveInt(
			"SESSION_MAX_AGE_SECONDS",
			fc.Auth.SessionMaxAge,
			inf,
			30*24*3600,
		),
		PublicWebBaseURL: mergeString("PUBLIC_WEB_BASE_URL", "", fc.Auth.PublicWebBaseURL, inf, ""),
		MailgunAPIKey:  mergeString("MAILGUN_API_KEY", "", fc.Mailgun.APIKey, inf, ""),
		MailgunDomain:  mergeString("MAILGUN_DOMAIN", "", fc.Mailgun.Domain, inf, ""),
		MailgunFrom:    mergeString("MAILGUN_FROM", "", fc.Mailgun.From, inf, ""),
		MailgunAPIBase: mergeString("MAILGUN_API_BASE", "", fc.Mailgun.APIBase, inf, ""),
		SMTPHost:       mergeString("SMTP_HOST", "", fc.SMTP.Host, inf, ""),
		SMTPPort:       mergeString("SMTP_PORT", "", fc.SMTP.Port, inf, "1025"),
		SMTPFrom:       mergeString("SMTP_FROM", "", fc.SMTP.From, inf, ""),
		GoogleOAuthClientID: mergeString(
			"GOOGLE_OAUTH_CLIENT_ID", "", fc.OAuth.GoogleClientID, inf, "",
		),
		GoogleOAuthClientSecret: mergeString(
			"GOOGLE_OAUTH_CLIENT_SECRET", "", fc.OAuth.GoogleClientSecret, inf, "",
		),
		DiscordOAuthClientID: mergeString(
			"DISCORD_OAUTH_CLIENT_ID", "", fc.OAuth.DiscordClientID, inf, "",
		),
		DiscordOAuthClientSecret: mergeString(
			"DISCORD_OAUTH_CLIENT_SECRET", "", fc.OAuth.DiscordClientSecret, inf, "",
		),
		SteamAPIKey: mergeString("STEAM_API_KEY", "", fc.OAuth.SteamAPIKey, inf, ""),
		TurnstileSiteKey: mergeString(
			"TURNSTILE_SITE_KEY", "", fc.Turnstile.SiteKey, inf, "",
		),
		TurnstileSecretKey: mergeString(
			"TURNSTILE_SECRET_KEY", "", fc.Turnstile.SecretKey, inf, "",
		),
		R2AccountID: mergeString("R2_ACCOUNT_ID", "", fc.R2.AccountID, inf, ""),
		R2AccessKeyID: mergeString(
			"R2_ACCESS_KEY_ID", "", fc.R2.AccessKeyID, inf, "",
		),
		R2SecretAccessKey: mergeString(
			"R2_SECRET_ACCESS_KEY", "", fc.R2.SecretAccessKey, inf, "",
		),
		R2Bucket:        mergeString("R2_BUCKET", "", fc.R2.Bucket, inf, ""),
		R2PublicBaseURL: mergeString("R2_PUBLIC_BASE_URL", "", fc.R2.PublicBaseURL, inf, ""),
		R2UserAssetsBucket: mergeString(
			"R2_USER_ASSETS_BUCKET", "", fc.R2.UserAssetsBucket, inf, "",
		),
		R2UserAssetsPublicBaseURL: mergeString(
			"R2_USER_ASSETS_PUBLIC_BASE_URL", "", fc.R2.UserAssetsPublicBaseURL, inf, "",
		),
		MediaProxySecret: mergeString("MEDIA_PROXY_SECRET", "", "", inf, ""),
		ClaimRateLimitPerMinute: mergePositiveInt(
			"CLAIM_RATE_LIMIT_PER_MINUTE",
			fc.Steam.ClaimRateLimitPerMinute,
			inf,
			10,
		),
		SteamMetadataCacheTTLSeconds: mergePositiveInt(
			"STEAM_METADATA_CACHE_TTL_SECONDS",
			fc.Steam.MetadataCacheTTLSeconds,
			inf,
			7*24*3600,
		),
		SteamRatePerSecond: mergePositiveFloat(
			"STEAM_RATE_PER_SECOND",
			fc.Steam.RatePerSecond,
			inf,
			1.0,
		),
		SentryDSN:         mergeString("SENTRY_DSN", "", fc.Sentry.DSN, inf, ""),
		SentryEnvironment: mergeString("SENTRY_ENVIRONMENT", "", fc.Sentry.Environment, inf, ""),
		SentryRelease:     mergeString("SENTRY_RELEASE", "", fc.Sentry.Release, inf, ""),
		SentryTracesSampleRate: mergePositiveFloat(
			"SENTRY_TRACES_SAMPLE_RATE",
			fc.Sentry.TracesSampleRate,
			inf,
			1.0,
		),
	}

	return cfg, nil
}

func resolveConfigPath(filesystem afero.Fs, cliPath string) (string, error) {
	if p := strings.TrimSpace(cliPath); p != "" {
		return p, nil
	}
	if p := strings.TrimSpace(os.Getenv("HUMBLESHARE_CONFIG")); p != "" {
		return p, nil
	}
	if _, err := filesystem.Stat("humbleshare.toml"); err == nil {
		return "humbleshare.toml", nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("humbleshare.toml: %w", err)
	}
	return "", nil
}
