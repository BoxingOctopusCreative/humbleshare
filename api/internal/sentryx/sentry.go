package sentryx

import (
	"strings"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/config"
	"github.com/getsentry/sentry-go"
	sentryfiber "github.com/getsentry/sentry-go/fiber"
	"github.com/gofiber/fiber/v2"
)

// Init configures the global Sentry client. An empty DSN disables the SDK.
func Init(cfg config.Config) error {
	dsn := strings.TrimSpace(cfg.SentryDSN)
	if dsn == "" {
		return nil
	}

	return sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      strings.TrimSpace(cfg.SentryEnvironment),
		Release:          strings.TrimSpace(cfg.SentryRelease),
		AttachStacktrace: true,
		EnableTracing:    true,
		TracesSampleRate: cfg.SentryTracesSampleRate,
		BeforeSend: func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			if event.Request != nil {
				event.Request.Cookies = ""
				if event.Request.Headers != nil {
					delete(event.Request.Headers, "Cookie")
					delete(event.Request.Headers, "Authorization")
				}
			}
			return event
		},
	})
}

// Middleware returns Fiber middleware that captures panics and ties requests to a hub.
// Safe to register when Sentry is disabled (no-op hub behavior).
// Repanic is true so Fiber's Recover middleware can still write the HTTP response.
func Middleware() fiber.Handler {
	return sentryfiber.New(sentryfiber.Options{
		Repanic:         true,
		WaitForDelivery: true,
	})
}

// CaptureHandlerError reports 5xx handler errors when a request hub is present.
func CaptureHandlerError(c *fiber.Ctx, err error) {
	if err == nil {
		return
	}
	hub := sentryfiber.GetHubFromContext(c)
	if hub == nil {
		return
	}
	hub.CaptureException(err)
}

// Flush waits briefly for queued events (call on shutdown).
func Flush() {
	sentry.Flush(2 * time.Second)
}
