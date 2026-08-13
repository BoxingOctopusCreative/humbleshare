package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	infisical "github.com/infisical/go-sdk"
)

const (
	defaultInfisicalSiteURL    = "https://app.infisical.com"
	defaultInfisicalSecretPath = "/api"
	infisicalFetchTimeout      = 30 * time.Second

	// infisicalPGCACertPEMKey is an internal map key for /api/db ca_cert (PEM contents).
	// It is not a real merge env; loadInfisicalEnvMap writes the PEM to a file and sets PGSSLROOTCERT.
	infisicalPGCACertPEMKey = "__HUMBLESHARE_PG_CA_CERT_PEM__"
	pgSSLRootCertEnv        = "PGSSLROOTCERT"
)

// InfisicalBootstrap holds chicken/egg credentials read from the process environment only.
type InfisicalBootstrap struct {
	ClientID     string
	ClientSecret string
	ProjectID    string
	Environment  string
	SiteURL      string
	SecretPath   string
}

// InfisicalSecret is a single secret returned by a SecretsFetcher.
type InfisicalSecret struct {
	Path  string
	Key   string
	Value string
}

// SecretsFetcher lists secrets for an Infisical project/environment. Tests inject a fake.
type SecretsFetcher func(ctx context.Context, boot InfisicalBootstrap) ([]InfisicalSecret, error)

// readInfisicalBootstrap returns bootstrap config when Infisical is enabled (all four required
// vars present). When disabled, ok is false.
func readInfisicalBootstrap() (boot InfisicalBootstrap, ok bool) {
	boot = InfisicalBootstrap{
		ClientID:     strings.TrimSpace(os.Getenv("INFISICAL_UNIVERSAL_AUTH_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET")),
		ProjectID:    strings.TrimSpace(os.Getenv("INFISICAL_PROJECT_ID")),
		Environment:  strings.TrimSpace(os.Getenv("INFISICAL_ENVIRONMENT")),
		SiteURL:      strings.TrimSpace(os.Getenv("INFISICAL_SITE_URL")),
		SecretPath:   strings.TrimSpace(os.Getenv("INFISICAL_SECRET_PATH")),
	}
	if boot.ClientID == "" || boot.ClientSecret == "" || boot.ProjectID == "" || boot.Environment == "" {
		return InfisicalBootstrap{}, false
	}
	if boot.SiteURL == "" {
		boot.SiteURL = defaultInfisicalSiteURL
	}
	if boot.SecretPath == "" {
		boot.SecretPath = defaultInfisicalSecretPath
	}
	boot.SecretPath = normalizeInfisicalPath(boot.SecretPath)
	return boot, true
}

func infisicalLog(w io.Writer, format string, args ...any) {
	if w == nil {
		w = os.Stderr
	}
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

// loadInfisicalEnvMap fetches and maps Infisical secrets to env-var keys. When Infisical is not
// configured, returns (nil, nil). On failure, writes a concise message to errOut (default stderr)
// and returns an error. Status lines (skip/success) go to logOut (default stderr).
func loadInfisicalEnvMap(ctx context.Context, fetch SecretsFetcher, errOut, logOut io.Writer) (map[string]string, error) {
	boot, ok := readInfisicalBootstrap()
	if !ok {
		infisicalLog(logOut, "humbleshare: infisical: skipped (bootstrap env not set; using env/CLI/TOML/defaults)")
		return nil, nil
	}
	if fetch == nil {
		fetch = fetchInfisicalSecrets
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	if logOut == nil {
		logOut = os.Stderr
	}

	paths := infisicalListPaths(boot.SecretPath)
	infisicalLog(logOut,
		"humbleshare: infisical: connecting (site=%s project=%s env=%s paths=%s)",
		boot.SiteURL, boot.ProjectID, boot.Environment, strings.Join(paths, ","),
	)

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, infisicalFetchTimeout)
	defer cancel()

	secrets, err := fetch(ctx, boot)
	if err != nil {
		msg := fmt.Sprintf(
			"humbleshare: infisical: failed to load secrets (site=%s project=%s env=%s path=%s): %v\n",
			boot.SiteURL, boot.ProjectID, boot.Environment, boot.SecretPath, err,
		)
		_, _ = io.WriteString(errOut, msg)
		return nil, fmt.Errorf("infisical: %w", err)
	}

	out := make(map[string]string)
	for _, s := range secrets {
		path := normalizeInfisicalPath(s.Path)
		key := strings.TrimSpace(s.Key)
		envKey := mapInfisicalSecret(s.Path, s.Key)
		empty := strings.TrimSpace(s.Value) == ""
		switch {
		case envKey == "":
			infisicalLog(logOut, "humbleshare: infisical: seen path=%s key=%s mapped=(none) empty=%t", path, key, empty)
		case envKey == infisicalPGCACertPEMKey:
			infisicalLog(logOut, "humbleshare: infisical: seen path=%s key=%s mapped=(postgres-ca) empty=%t", path, key, empty)
		default:
			infisicalLog(logOut, "humbleshare: infisical: seen path=%s key=%s mapped=%s empty=%t", path, key, envKey, empty)
		}
		if envKey == "" || empty {
			continue
		}
		out[envKey] = strings.TrimSpace(s.Value)
	}

	keys := make([]string, 0, len(out))
	for k := range out {
		if k == infisicalPGCACertPEMKey {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	caNote := ""
	if pem := out[infisicalPGCACertPEMKey]; pem != "" {
		delete(out, infisicalPGCACertPEMKey)
		alreadySet := strings.TrimSpace(os.Getenv(pgSSLRootCertEnv)) != ""
		if err := materializePGSSLRootCert(pem); err != nil {
			msg := fmt.Sprintf("humbleshare: infisical: failed to write Postgres CA cert: %v\n", err)
			_, _ = io.WriteString(errOut, msg)
			return nil, fmt.Errorf("infisical: postgres CA cert: %w", err)
		}
		if alreadySet {
			caNote = "; postgres CA cert skipped (PGSSLROOTCERT already set)"
		} else {
			caNote = "; postgres CA cert materialized (PGSSLROOTCERT)"
		}
	}

	infisicalLog(logOut, "humbleshare: infisical: ok mapped=%d%s", len(keys), caNote)
	for _, k := range keys {
		infisicalLog(logOut, "humbleshare: infisical: mapped %s", k)
	}

	return out, nil
}

// materializePGSSLRootCert writes PEM to a temp file and sets PGSSLROOTCERT when that env is unset.
// Process env already pointing at a CA file wins (same precedence as other Infisical values).
func materializePGSSLRootCert(pem string) error {
	if strings.TrimSpace(os.Getenv(pgSSLRootCertEnv)) != "" {
		return nil
	}
	if !strings.Contains(pem, "BEGIN") {
		return fmt.Errorf("value does not look like a PEM certificate")
	}
	if !strings.HasSuffix(pem, "\n") {
		pem += "\n"
	}
	path := filepath.Join(os.TempDir(), "humbleshare-pg-ca.pem")
	if err := os.WriteFile(path, []byte(pem), 0o600); err != nil {
		return err
	}
	return os.Setenv(pgSSLRootCertEnv, path)
}

// infisicalListPaths returns the bootstrap secret path (recursive list root).
func infisicalListPaths(secretPath string) []string {
	return []string{normalizeInfisicalPath(secretPath)}
}

func fetchInfisicalSecrets(ctx context.Context, boot InfisicalBootstrap) ([]InfisicalSecret, error) {
	client := infisical.NewInfisicalClient(ctx, infisical.Config{
		SiteUrl: boot.SiteURL,
	})

	if _, err := client.Auth().UniversalAuthLogin(boot.ClientID, boot.ClientSecret); err != nil {
		return nil, fmt.Errorf("machine identity universal auth: %w", err)
	}

	var out []InfisicalSecret
	for _, path := range infisicalListPaths(boot.SecretPath) {
		listed, err := client.Secrets().List(infisical.ListSecretsOptions{
			ProjectID:              boot.ProjectID,
			Environment:            boot.Environment,
			SecretPath:             path,
			Recursive:              true,
			ExpandSecretReferences: true,
			IncludeImports:         true,
			SkipUniqueValidation:   true,
		})
		if err != nil {
			return nil, fmt.Errorf("list secrets path=%s: %w", path, err)
		}
		for _, s := range listed {
			out = append(out, InfisicalSecret{
				Path:  s.SecretPath,
				Key:   s.SecretKey,
				Value: s.SecretValue,
			})
		}
	}
	return out, nil
}
