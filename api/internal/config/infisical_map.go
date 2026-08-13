package config

import "strings"

// infisicalEnvByPathKey maps Infisical (secretPath, secretKey) → process env var name used by merge.
var infisicalEnvByPathKey = map[string]map[string]string{
	"/api/db": {
		"url":     "DATABASE_URL",
		"ca_cert": infisicalPGCACertPEMKey,
	},
	"/api/auth": {
		"jwt_secret":              "AUTH_JWT_SECRET",
		"superadmin_email":        "SUPERADMIN_EMAIL",
		"superadmin_password":     "SUPERADMIN_PASSWORD",
		"cookie_secure":           "COOKIE_SECURE",
		"session_max_age_seconds": "SESSION_MAX_AGE_SECONDS",
		"public_web_base_url":     "PUBLIC_WEB_BASE_URL",
	},
	"/api/mailgun": {
		"api_key":  "MAILGUN_API_KEY",
		"domain":   "MAILGUN_DOMAIN",
		"from":     "MAILGUN_FROM",
		"api_base": "MAILGUN_API_BASE",
	},
	"/api/auth/google": {
		"client_id":     "GOOGLE_OAUTH_CLIENT_ID",
		"client_secret": "GOOGLE_OAUTH_CLIENT_SECRET",
	},
	"/api/auth/discord": {
		"client_id":     "DISCORD_OAUTH_CLIENT_ID",
		"client_secret": "DISCORD_OAUTH_CLIENT_SECRET",
	},
	"/api/auth/steam": {
		"api_key": "STEAM_API_KEY",
	},
	"/api/auth/superadmin_password": {
		"password": "SUPERADMIN_PASSWORD",
	},
	"/api/auth/turnstile": {
		"site_key":   "TURNSTILE_SITE_KEY",
		"secret_key": "TURNSTILE_SECRET_KEY",
	},
	"/api/r2": {
		"account_id":                  "R2_ACCOUNT_ID",
		"access_key_id":               "R2_ACCESS_KEY_ID",
		"secret_access_key":           "R2_SECRET_ACCESS_KEY",
		"bucket":                      "R2_BUCKET",
		"public_base_url":             "R2_PUBLIC_BASE_URL",
		"user_assets_bucket":          "R2_USER_ASSETS_BUCKET",
		"user_assets_public_base_url": "R2_USER_ASSETS_PUBLIC_BASE_URL",
	},
	"/api/steam": {
		"claim_rate_limit_per_minute": "CLAIM_RATE_LIMIT_PER_MINUTE",
		"metadata_cache_ttl_seconds":  "STEAM_METADATA_CACHE_TTL_SECONDS",
		"rate_per_second":             "STEAM_RATE_PER_SECOND",
	},
	"/api/sentry": {
		"dsn":                "SENTRY_DSN",
		"environment":        "SENTRY_ENVIRONMENT",
		"release":            "SENTRY_RELEASE",
		"traces_sample_rate": "SENTRY_TRACES_SAMPLE_RATE",
	},
}

// normalizeInfisicalPath canonicalizes Infisical folder paths for map lookup.
func normalizeInfisicalPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	for strings.HasSuffix(p, "/") && p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

// mapInfisicalSecret returns the HumbleShare env var name for a path+key, or "" if unmapped.
func mapInfisicalSecret(secretPath, secretKey string) string {
	pathMap := infisicalEnvByPathKey[normalizeInfisicalPath(secretPath)]
	if pathMap == nil {
		return ""
	}
	return pathMap[strings.TrimSpace(secretKey)]
}
