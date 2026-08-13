package config

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func clearHumbleShareEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"AUTH_JWT_SECRET", "SUPERADMIN_EMAIL", "SUPERADMIN_PASSWORD", "DATABASE_URL", "HTTP_ADDR", "CORS_ORIGINS",
		"COOKIE_SECURE", "SESSION_MAX_AGE_SECONDS", "PUBLIC_WEB_BASE_URL",
		"CLAIM_RATE_LIMIT_PER_MINUTE", "STEAM_METADATA_CACHE_TTL_SECONDS", "STEAM_RATE_PER_SECOND",
		"SENTRY_DSN", "SENTRY_ENVIRONMENT", "SENTRY_RELEASE", "SENTRY_TRACES_SAMPLE_RATE",
		"MAILGUN_API_KEY", "MAILGUN_DOMAIN", "MAILGUN_FROM", "MAILGUN_API_BASE",
		"GOOGLE_OAUTH_CLIENT_ID", "GOOGLE_OAUTH_CLIENT_SECRET",
		"DISCORD_OAUTH_CLIENT_ID", "DISCORD_OAUTH_CLIENT_SECRET", "STEAM_API_KEY",
		"HUMBLESHARE_CONFIG",
		"INFISICAL_UNIVERSAL_AUTH_CLIENT_ID", "INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET",
		"INFISICAL_PROJECT_ID", "INFISICAL_ENVIRONMENT", "INFISICAL_SITE_URL", "INFISICAL_SECRET_PATH",
		pgSSLRootCertEnv,
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func setInfisicalBootstrap(t *testing.T) {
	t.Helper()
	t.Setenv("INFISICAL_UNIVERSAL_AUTH_CLIENT_ID", "client-id")
	t.Setenv("INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("INFISICAL_PROJECT_ID", "proj-123")
	t.Setenv("INFISICAL_ENVIRONMENT", "dev")
}

func TestMapInfisicalSecret(t *testing.T) {
	cases := []struct {
		path, key, want string
	}{
		{"/api/db", "url", "DATABASE_URL"},
		{"/api/db/", "url", "DATABASE_URL"},
		{"/api/db", "ca_cert", infisicalPGCACertPEMKey},
		{"/api/auth", "jwt_secret", "AUTH_JWT_SECRET"},
		{"/api/auth", "superadmin_email", "SUPERADMIN_EMAIL"},
		{"/api/auth", "superadmin_password", "SUPERADMIN_PASSWORD"},
		{"/api/mailgun", "api_key", "MAILGUN_API_KEY"},
		{"/api/auth/google", "client_id", "GOOGLE_OAUTH_CLIENT_ID"},
		{"/api/auth/google", "client_secret", "GOOGLE_OAUTH_CLIENT_SECRET"},
		{"/api/auth/discord", "client_id", "DISCORD_OAUTH_CLIENT_ID"},
		{"/api/auth/steam", "api_key", "STEAM_API_KEY"},
		{"/api/auth/superadmin_password", "password", "SUPERADMIN_PASSWORD"},
		{"/api/auth/turnstile", "site_key", "TURNSTILE_SITE_KEY"},
		{"/api/auth/turnstile", "secret_key", "TURNSTILE_SECRET_KEY"},
		{"/api/r2", "account_id", "R2_ACCOUNT_ID"},
		{"/api/r2", "access_key_id", "R2_ACCESS_KEY_ID"},
		{"/api/r2", "secret_access_key", "R2_SECRET_ACCESS_KEY"},
		{"/api/r2", "bucket", "R2_BUCKET"},
		{"/api/r2", "public_base_url", "R2_PUBLIC_BASE_URL"},
		{"/api/r2", "user_assets_bucket", "R2_USER_ASSETS_BUCKET"},
		{"/api/r2", "user_assets_public_base_url", "R2_USER_ASSETS_PUBLIC_BASE_URL"},
		{"/api/oauth", "google_client_id", ""},
		{"/api/steam", "rate_per_second", "STEAM_RATE_PER_SECOND"},
		{"/api/sentry", "dsn", "SENTRY_DSN"},
		{"/api/sentry", "environment", "SENTRY_ENVIRONMENT"},
		{"/api/sentry", "release", "SENTRY_RELEASE"},
		{"/api/sentry", "traces_sample_rate", "SENTRY_TRACES_SAMPLE_RATE"},
		{"/shared/auth", "access_token", ""},
		{"/api/db", "unknown", ""},
	}
	for _, tc := range cases {
		got := mapInfisicalSecret(tc.path, tc.key)
		if got != tc.want {
			t.Errorf("mapInfisicalSecret(%q, %q) = %q, want %q", tc.path, tc.key, got, tc.want)
		}
	}
}

func TestInfisicalListPaths(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/api", []string{"/api"}},
		{"/ui", []string{"/ui"}},
		{"/", []string{"/"}},
		{"/shared", []string{"/shared"}},
		{"/shared/auth", []string{"/shared/auth"}},
	}
	for _, tc := range cases {
		got := infisicalListPaths(tc.in)
		if len(got) != len(tc.want) {
			t.Fatalf("infisicalListPaths(%q) = %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("infisicalListPaths(%q) = %v, want %v", tc.in, got, tc.want)
			}
		}
	}
}

func TestLoadUsesInfisicalAsBaseLayer(t *testing.T) {
	clearHumbleShareEnv(t)
	setInfisicalBootstrap(t)

	var logBuf bytes.Buffer
	cfg, err := Load(afero.NewMemMapFs(), &LoadOptions{
		InfisicalLogOut: &logBuf,
		InfisicalErrOut: io.Discard,
		InfisicalFetcher: func(ctx context.Context, boot InfisicalBootstrap) ([]InfisicalSecret, error) {
			if boot.ProjectID != "proj-123" || boot.Environment != "dev" {
				t.Fatalf("unexpected bootstrap: %+v", boot)
			}
			return []InfisicalSecret{
				{Path: "/api/db", Key: "url", Value: "postgres://infisical/db"},
				{Path: "/api/auth", Key: "jwt_secret", Value: "from-infisical-jwt"},
				{Path: "/api/auth", Key: "superadmin_email", Value: "admin@infisical.example"},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseURL != "postgres://infisical/db" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.AuthJWTSecret != "from-infisical-jwt" {
		t.Errorf("AuthJWTSecret = %q", cfg.AuthJWTSecret)
	}
	if cfg.SuperadminEmail != "admin@infisical.example" {
		t.Errorf("SuperadminEmail = %q", cfg.SuperadminEmail)
	}
	msg := logBuf.String()
	if !strings.Contains(msg, "humbleshare: infisical: connecting") {
		t.Fatalf("missing connecting log: %q", msg)
	}
	if !strings.Contains(msg, "humbleshare: infisical: ok mapped=") {
		t.Fatalf("missing ok log: %q", msg)
	}
	if !strings.Contains(msg, "humbleshare: infisical: mapped AUTH_JWT_SECRET") ||
		!strings.Contains(msg, "humbleshare: infisical: mapped DATABASE_URL") {
		t.Fatalf("per-key mapped lines missing from status: %q", msg)
	}
	if strings.Contains(msg, "from-infisical") || strings.Contains(msg, "postgres://infisical") {
		t.Fatalf("status log leaked secret values: %q", msg)
	}
}

func TestLoadTOMLBeatsInfisical(t *testing.T) {
	clearHumbleShareEnv(t)
	setInfisicalBootstrap(t)

	mem := afero.NewMemMapFs()
	content := `
[database]
url = "postgres://toml/db"
`
	if err := afero.WriteFile(mem, "humbleshare.toml", []byte(content), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg, err := Load(mem, &LoadOptions{
		InfisicalLogOut: io.Discard,
		InfisicalErrOut: io.Discard,
		InfisicalFetcher: func(context.Context, InfisicalBootstrap) ([]InfisicalSecret, error) {
			return []InfisicalSecret{
				{Path: "/api/db", Key: "url", Value: "postgres://infisical/db"},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseURL != "postgres://toml/db" {
		t.Errorf("DatabaseURL = %q, want TOML value", cfg.DatabaseURL)
	}
}

func TestLoadEnvBeatsInfisical(t *testing.T) {
	clearHumbleShareEnv(t)
	setInfisicalBootstrap(t)
	t.Setenv("DATABASE_URL", "postgres://env/db")

	cfg, err := Load(afero.NewMemMapFs(), &LoadOptions{
		InfisicalLogOut: io.Discard,
		InfisicalErrOut: io.Discard,
		InfisicalFetcher: func(context.Context, InfisicalBootstrap) ([]InfisicalSecret, error) {
			return []InfisicalSecret{
				{Path: "/api/db", Key: "url", Value: "postgres://infisical/db"},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseURL != "postgres://env/db" {
		t.Errorf("DatabaseURL = %q, want env", cfg.DatabaseURL)
	}
}

func TestLoadSkipsInfisicalWithoutBootstrap(t *testing.T) {
	clearHumbleShareEnv(t)
	called := false
	var logBuf bytes.Buffer
	cfg, err := Load(afero.NewMemMapFs(), &LoadOptions{
		InfisicalLogOut: &logBuf,
		InfisicalErrOut: io.Discard,
		InfisicalFetcher: func(context.Context, InfisicalBootstrap) ([]InfisicalSecret, error) {
			called = true
			return nil, errors.New("should not call")
		},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if called {
		t.Fatal("fetcher should not run when bootstrap env is unset")
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if !strings.Contains(logBuf.String(), "humbleshare: infisical: skipped") {
		t.Fatalf("expected skip log, got %q", logBuf.String())
	}
}

func TestLoadInfisicalMaterializesPostgresCACert(t *testing.T) {
	clearHumbleShareEnv(t)
	setInfisicalBootstrap(t)
	t.Setenv(pgSSLRootCertEnv, "")

	pem := "-----BEGIN CERTIFICATE-----\nMIIBtest\n-----END CERTIFICATE-----"
	_, err := Load(afero.NewMemMapFs(), &LoadOptions{
		InfisicalLogOut: io.Discard,
		InfisicalErrOut: io.Discard,
		InfisicalFetcher: func(context.Context, InfisicalBootstrap) ([]InfisicalSecret, error) {
			return []InfisicalSecret{
				{Path: "/api/db", Key: "ca_cert", Value: pem},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := os.Getenv(pgSSLRootCertEnv)
	if got == "" {
		t.Fatal("PGSSLROOTCERT unset after Infisical ca_cert")
	}
	b, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("read CA file: %v", err)
	}
	if !strings.Contains(string(b), "BEGIN CERTIFICATE") {
		t.Fatalf("CA file contents unexpected: %q", b)
	}
	_ = os.Remove(got)
	t.Setenv(pgSSLRootCertEnv, "")
}

func TestLoadInfisicalFailureWritesStderr(t *testing.T) {
	clearHumbleShareEnv(t)
	setInfisicalBootstrap(t)

	var buf bytes.Buffer
	_, err := Load(afero.NewMemMapFs(), &LoadOptions{
		InfisicalLogOut: io.Discard,
		InfisicalErrOut: &buf,
		InfisicalFetcher: func(context.Context, InfisicalBootstrap) ([]InfisicalSecret, error) {
			return nil, errors.New("connection refused")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := buf.String()
	if !strings.Contains(msg, "humbleshare: infisical:") {
		t.Fatalf("stderr missing prefix: %q", msg)
	}
	if !strings.Contains(msg, "project=proj-123") || !strings.Contains(msg, "env=dev") {
		t.Fatalf("stderr missing context: %q", msg)
	}
	if !strings.Contains(msg, "connection refused") {
		t.Fatalf("stderr missing cause: %q", msg)
	}
	if strings.Contains(msg, "client-secret") {
		t.Fatalf("stderr leaked secret: %q", msg)
	}
}
