package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/config"
	"github.com/boxingoctopus/humbleshare/api/internal/handler"
	"github.com/boxingoctopus/humbleshare/api/internal/humble"
	"github.com/boxingoctopus/humbleshare/api/internal/mailer"
	"github.com/boxingoctopus/humbleshare/api/internal/mailgun"
	"github.com/boxingoctopus/humbleshare/api/internal/media"
	"github.com/boxingoctopus/humbleshare/api/internal/middleware"
	"github.com/boxingoctopus/humbleshare/api/internal/migrate"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/boxingoctopus/humbleshare/api/internal/sentryx"
	"github.com/boxingoctopus/humbleshare/api/internal/service"
	"github.com/boxingoctopus/humbleshare/api/internal/steam"
	"github.com/boxingoctopus/humbleshare/api/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func main() {
	var opts config.LoadOptions
	cmd := &cobra.Command{
		Use:   "humbleshare-api",
		Short: "Run the HumbleShare API HTTP server.",
		Long: "Run the HumbleShare API HTTP server.\n\n" +
			"Configuration is layered from highest to lowest precedence: environment variables, " +
			"CLI flags, the TOML config file (-c/--config or HUMBLESHARE_CONFIG, default ./humbleshare.toml), " +
			"Infisical (Machine Identity Universal Auth when bootstrap env is set), " +
			"and built-in defaults.\n\n" +
			"Required (any layer): DATABASE_URL, AUTH_JWT_SECRET, SUPERADMIN_EMAIL.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load(afero.NewOsFs(), &opts)
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			return runAPI(cfg)
		},
	}
	config.RegisterFlags(cmd.Flags(), &opts)

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runAPI(cfg config.Config) error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.AuthJWTSecret == "" {
		return fmt.Errorf("AUTH_JWT_SECRET is required")
	}
	if cfg.SuperadminEmail == "" {
		return fmt.Errorf("SUPERADMIN_EMAIL is required")
	}

	if err := sentryx.Init(cfg); err != nil {
		return fmt.Errorf("sentry: %w", err)
	}
	defer sentryx.Flush()
	if strings.TrimSpace(cfg.SentryDSN) != "" {
		logStartup("sentry", "enabled")
	} else {
		logStartup("sentry", "disabled (no DSN)")
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := connectPostgres(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pool.Close()

	applied, expected, err := migrate.StatusWithExistingPool(startupCtx, pool)
	if err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	if len(applied) < len(expected) {
		logStartup("bootstrap", "applying pending migrations")
		appliedNow, err := migrate.UpWithExistingPool(startupCtx, pool)
		if err != nil {
			return fmt.Errorf("bootstrap: %w", err)
		}
		if len(appliedNow) > 0 {
			logStartup("bootstrap", "applied: "+strings.Join(appliedNow, ", "))
		}
	} else {
		logStartup("bootstrap", "database up to date")
	}

	codeRepo := repository.NewPostgresCodeRepository(pool)
	gameRepo := repository.NewPostgresGameRepository(pool)
	statsRepo := repository.NewPostgresStatsRepository(pool)
	userRepo := repository.NewPostgresUserRepository(pool)
	sessRepo := repository.NewPostgresSessionRepository(pool)
	reqRepo := repository.NewPostgresAccessRequestRepository(pool)
	invRepo := repository.NewPostgresAccessInviteRepository(pool)
	oauthRepo := repository.NewPostgresOAuthIdentityRepository(pool)

	var mail mailer.Sender
	if smtp := mailer.NewSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom); smtp != nil {
		mail = smtp
		logStartup("mail", "smtp "+cfg.SMTPHost+":"+cfg.SMTPPort)
	} else if mg := mailgun.New(cfg.MailgunAPIKey, cfg.MailgunDomain, cfg.MailgunFrom, cfg.MailgunAPIBase); mg != nil {
		mail = mg
		logStartup("mail", "mailgun")
	} else {
		logStartup("mail", "unset (invite URLs log to stderr)")
	}

	r2Client := storage.NewR2Client(storage.R2Credentials{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
	})
	userAssetsStore := storage.NewR2(r2Client, cfg.R2UserAssetsBucket)
	switch {
	case userAssetsStore != nil && cfg.R2UserAssetsPublicBaseURL != "":
		logStartup("r2", fmt.Sprintf(
			"user-assets bucket=%s static bucket=%s",
			cfg.R2UserAssetsBucket,
			orUnset(cfg.R2Bucket),
		))
	case r2Client != nil && strings.TrimSpace(cfg.R2Bucket) != "":
		logStartup("r2", "static bucket="+cfg.R2Bucket+" (avatar upload disabled; set user_assets_bucket + user_assets_public_base_url)")
	default:
		logStartup("r2", "unset (avatar upload disabled)")
	}

	authSvc := service.NewAuthService(
		userRepo, sessRepo, reqRepo, invRepo, oauthRepo, mail, userAssetsStore,
		cfg.AuthJWTSecret, cfg.PublicWebBaseURL, cfg.SuperadminEmail, cfg.SuperadminPassword,
		cfg.R2UserAssetsPublicBaseURL, cfg.SessionMaxAge,
	)
	if err := authSvc.EnsureSuperadminInvite(startupCtx); err != nil {
		return fmt.Errorf("superadmin bootstrap: %w", err)
	}

	oauthSvc := service.NewOAuthService(
		authSvc, oauthRepo, userRepo, cfg.AuthJWTSecret, cfg.PublicWebBaseURL,
		cfg.GoogleOAuthClientID, cfg.GoogleOAuthClientSecret,
		cfg.DiscordOAuthClientID, cfg.DiscordOAuthClientSecret, cfg.SteamAPIKey,
	)

	codeSvc := service.NewCodeService(codeRepo)
	gameSvc := service.NewGameService(gameRepo)
	adminSvc := service.NewAdminService(codeRepo, gameRepo, statsRepo)
	profileSvc := service.NewProfileService(userRepo, codeRepo, cfg.PublicWebBaseURL, cfg.R2UserAssetsPublicBaseURL)

	cacheTTL := time.Duration(cfg.SteamMetadataCacheTTLSeconds) * time.Second
	steamClient := steam.NewClient(cfg.SteamRatePerSecond, cacheTTL, gameRepo)
	steamClient.StartEnrichmentWorker(context.Background(), 5*time.Minute)
	logStartup("steam", fmt.Sprintf("enrichment worker started (rate=%.1f/s, cacheTTL=%s)", cfg.SteamRatePerSecond, cacheTTL))

	humbleSvc := humble.NewService(codeRepo, gameRepo, steamClient)

	gameH := handler.NewGameHandler(gameSvc)
	codeH := handler.NewCodeHandler(codeSvc)
	adminH := handler.NewAdminHandler(adminSvc, humbleSvc, steamClient)
	authH := handler.NewAuthHandler(authSvc, oauthSvc, cfg.CookieSecure, cfg.SessionMaxAge)
	profileH := handler.NewProfileHandler(profileSvc)
	mediaHosts := media.AllowedHosts(append([]string{cfg.PublicWebBaseURL}, cfg.CORSOrigins...)...)
	mediaH := handler.NewMediaHandler(userAssetsStore, mediaHosts, cfg.MediaProxySecret)

	requireSession := middleware.RequireSession(authSvc)
	optionalSession := middleware.OptionalSession(authSvc)
	requireAdmin := middleware.RequireAdmin(authSvc)
	requireSuper := middleware.RequireSuperadmin(authSvc)
	claimLimiter := middleware.ClaimRateLimiter(cfg.ClaimRateLimitPerMinute)
	authLimiter := middleware.AuthRateLimiter(20)

	app := fiber.New(fiber.Config{
		AppName:      "HumbleShare API",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		BodyLimit:    4 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			msg := "Internal Server Error"
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				msg = e.Message
			}
			if code >= 500 {
				sentryx.CaptureHandlerError(c, err)
			}
			return c.Status(code).JSON(fiber.Map{"error": true, "msg": msg})
		},
	})

	app.Use(sentryx.Middleware())
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     joinOrigins(cfg.CORSOrigins),
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	v1 := app.Group("/api/v1")

	// Auth (public).
	v1.Post("/auth/request-access", authLimiter, authH.RequestAccess)
	v1.Get("/auth/invite/:token", authH.GetInvite)
	v1.Post("/auth/register", authLimiter, authH.Register)
	v1.Post("/auth/login", authLimiter, authH.Login)
	v1.Post("/auth/login/2fa", authLimiter, authH.Login2FA)
	v1.Get("/auth/oauth/:provider", authH.OAuthStart)
	v1.Get("/auth/oauth/:provider/callback", authH.OAuthCallback)
	v1.Get("/auth/access-requests/:id/approve", authH.ApproveAccessRequest)

	v1.Get("/profiles/:slug", optionalSession, profileH.Get)
	v1.Get("/media/:kind/:name", mediaH.Get)

	// Auth (session).
	me := v1.Group("", requireSession)
	me.Post("/auth/logout", authH.Logout)
	me.Get("/me", authH.Me)
	me.Patch("/me", authH.UpdateProfile)
	me.Post("/me/avatar", authH.UploadAvatar)
	me.Delete("/me/avatar", authH.DeleteAvatar)
	me.Post("/me/banner", authH.UploadBanner)
	me.Delete("/me/banner", authH.DeleteBanner)
	me.Post("/me/password", authH.ChangePassword)
	me.Post("/me/2fa/setup", authH.Setup2FA)
	me.Post("/me/2fa/enable", authH.Enable2FA)
	me.Post("/me/2fa/disable", authH.Disable2FA)
	me.Get("/me/oauth", authH.ListOAuth)
	me.Get("/me/oauth/:provider/link", func(c *fiber.Ctx) error {
		c.Request().URI().QueryArgs().Set("mode", "link")
		return authH.OAuthStart(c)
	})
	me.Delete("/me/oauth/:provider", authH.UnlinkOAuth)

	v1.Get("/games", requireSession, gameH.List)
	v1.Post("/codes/claim", requireSession, claimLimiter, codeH.Claim)
	v1.Post("/codes/:id/report-dead", requireSession, codeH.ReportDead)

	admin := v1.Group("/admin", requireAdmin)
	admin.Post("/import/humble", adminH.ImportHumble)
	admin.Post("/codes", adminH.AddCode)
	admin.Post("/codes/bulk", adminH.BulkAddCodes)
	admin.Get("/codes", adminH.ListCodes)
	admin.Patch("/codes/:id", adminH.UpdateCode)
	admin.Get("/games/unresolved", adminH.ListUnresolved)
	admin.Patch("/games/:id", adminH.SetGameSteamAppID)
	admin.Post("/games/:id/refresh", adminH.RefreshGameMetadata)
	admin.Get("/stats", adminH.GetStats)
	admin.Get("/access-requests", authH.ListAccessRequests)
	admin.Post("/access-requests/:id/approve", authH.ApproveAccessRequest)
	admin.Post("/access-requests/:id/deny", authH.DenyAccessRequest)
	admin.Get("/users", authH.ListUsers)
	admin.Post("/users/:id/role", requireSuper, authH.SetUserRole)

	logStartup("listen", cfg.HTTPAddr)
	if err := app.Listen(cfg.HTTPAddr); err != nil {
		return fmt.Errorf("listen %s: %w", cfg.HTTPAddr, err)
	}
	return nil
}

func connectPostgres(ctx context.Context, url string) (*pgxpool.Pool, error) {
	var last error
	for i := 0; i < 30; i++ {
		pool, err := pgxpool.New(ctx, url)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				logStartup("postgres", "connected")
				return pool, nil
			}
			pool.Close()
		}
		last = err
		log.Printf("startup: postgres: waiting (attempt %d/30): %v", i+1, err)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context done: %w", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("postgres unavailable: %w", last)
}

func logStartup(component, detail string) {
	log.Printf("startup: %s: %s", component, detail)
}

func joinOrigins(origins []string) string {
	if len(origins) == 0 {
		return "http://localhost:3000"
	}
	return strings.Join(origins, ",")
}

func orUnset(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(unset)"
	}
	return s
}
