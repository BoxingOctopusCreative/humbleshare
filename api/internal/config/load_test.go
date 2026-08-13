package config

import (
	"os"
	"testing"

	"github.com/spf13/afero"
)

func TestLoadDefaultsAndEnv(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Setenv("AUTH_JWT_SECRET", "jwt-secret-value")
	t.Setenv("SUPERADMIN_EMAIL", "admin@example.com")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	t.Setenv("CORS_ORIGINS", "http://localhost:5173")
	_ = os.Unsetenv("HTTP_ADDR")

	cfg, err := Load(fs, &LoadOptions{InfisicalLogOut: ioDiscard{}, InfisicalErrOut: ioDiscard{}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthJWTSecret != "jwt-secret-value" || cfg.SuperadminEmail != "admin@example.com" {
		t.Fatalf("auth not loaded: %+v", cfg)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr=%q", cfg.HTTPAddr)
	}
	if cfg.ClaimRateLimitPerMinute != 10 {
		t.Fatalf("ClaimRateLimitPerMinute=%d", cfg.ClaimRateLimitPerMinute)
	}
	if len(cfg.CORSOrigins) != 1 || cfg.CORSOrigins[0] != "http://localhost:5173" {
		t.Fatalf("CORSOrigins=%v", cfg.CORSOrigins)
	}
}

func TestLoadTOMLThenEnvWins(t *testing.T) {
	fs := afero.NewMemMapFs()
	content := `
[server]
http_addr = ":9090"

[auth]
jwt_secret = "toml-jwt"
superadmin_email = "toml@example.com"

[database]
url = "postgres://toml/db"
`
	if err := afero.WriteFile(fs, "humbleshare.toml", []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AUTH_JWT_SECRET", "env-jwt")
	t.Setenv("SUPERADMIN_EMAIL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HTTP_ADDR", "")

	cfg, err := Load(fs, &LoadOptions{InfisicalLogOut: ioDiscard{}, InfisicalErrOut: ioDiscard{}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthJWTSecret != "env-jwt" {
		t.Fatalf("env should win: %q", cfg.AuthJWTSecret)
	}
	if cfg.SuperadminEmail != "toml@example.com" {
		t.Fatalf("toml superadmin expected: %q", cfg.SuperadminEmail)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("toml http addr expected: %q", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL != "postgres://toml/db" {
		t.Fatalf("toml db url expected: %q", cfg.DatabaseURL)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
