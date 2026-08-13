package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/boxingoctopus/humbleshare/api/internal/config"
	"github.com/boxingoctopus/humbleshare/api/internal/migrate"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func main() {
	var opts config.LoadOptions
	cmd := &cobra.Command{
		Use:   "humbleshare-migrate",
		Short: "Apply HumbleShare database migrations.",
		Long: "Apply the bundled SQL migrations against DATABASE_URL.\n\n" +
			"Configuration is layered from highest to lowest precedence: environment variables, " +
			"CLI flags, the TOML config file (-c/--config or HUMBLESHARE_CONFIG, default ./humbleshare.toml), " +
			"Required (any layer): DATABASE_URL (AUTH_JWT_SECRET / SUPERADMIN_EMAIL used when Infisical/auth bootstrap runs).",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load(afero.NewOsFs(), &opts)
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			if cfg.DatabaseURL == "" {
				return fmt.Errorf("DATABASE_URL is required")
			}
			applied, err := migrate.Up(context.Background(), cfg.DatabaseURL)
			if err != nil {
				return err
			}
			if len(applied) == 0 {
				fmt.Fprintln(os.Stderr, "No new migrations (already up to date).")
				return nil
			}
			fmt.Printf("Applied: %v\n", applied)
			return nil
		},
	}
	config.RegisterFlags(cmd.Flags(), &opts)

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
