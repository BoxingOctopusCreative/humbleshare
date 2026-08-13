package config

import (
	"io"

	"github.com/spf13/pflag"
)

// LoadOptions holds CLI overrides. Empty strings mean "not set on CLI".
//
// InfisicalFetcher and InfisicalErrOut are test hooks (not CLI flags). When Infisical bootstrap
// env is set and InfisicalFetcher is nil, the real Infisical SDK is used.
type LoadOptions struct {
	ConfigFile  string
	HTTPAddr    string
	DatabaseURL string

	// InfisicalFetcher overrides the SDK list call (tests). Nil = real SDK when Infisical enabled.
	InfisicalFetcher SecretsFetcher
	// InfisicalErrOut receives concise Infisical failure messages. Nil = os.Stderr.
	InfisicalErrOut io.Writer
	// InfisicalLogOut receives Infisical/config source status lines. Nil = os.Stderr.
	InfisicalLogOut io.Writer
}

// RegisterFlags binds common settings to a pflag.FlagSet.
func RegisterFlags(fs *pflag.FlagSet, o *LoadOptions) {
	fs.StringVarP(
		&o.ConfigFile,
		"config",
		"c",
		"",
		"path to TOML config file (overrides HUMBLESHARE_CONFIG; default: ./humbleshare.toml if present)",
	)
	fs.StringVarP(&o.HTTPAddr, "http-addr", "a", "", "HTTP listen address (env: HTTP_ADDR)")
	fs.StringVarP(&o.DatabaseURL, "database-url", "d", "", "Postgres URL (env: DATABASE_URL)")
}
