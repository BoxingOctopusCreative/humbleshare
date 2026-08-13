package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

const (
	SessionCookieName = "humbleshare_session"
	LocalsUserKey     = "user"
)

// RequireSession loads the user from the session cookie or Bearer token.
func RequireSession(auth *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := extractSessionToken(c)
		if raw == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		u, err := auth.UserFromSession(c.Context(), raw)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		c.Locals(LocalsUserKey, u)
		c.Locals("session_token", raw)
		return c.Next()
	}
}

// RequireAdmin requires an authenticated admin or superadmin.
func RequireAdmin(auth *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := extractSessionToken(c)
		if raw == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		u, err := auth.UserFromSession(c.Context(), raw)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		if !u.IsAdmin() {
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		}
		c.Locals(LocalsUserKey, u)
		c.Locals("session_token", raw)
		return c.Next()
	}
}

// RequireSuperadmin requires superadmin role.
func RequireSuperadmin(auth *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := extractSessionToken(c)
		if raw == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		u, err := auth.UserFromSession(c.Context(), raw)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		if !u.IsSuperadmin() {
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		}
		c.Locals(LocalsUserKey, u)
		return c.Next()
	}
}

func UserFromLocals(c *fiber.Ctx) *models.User {
	u, _ := c.Locals(LocalsUserKey).(*models.User)
	return u
}

func extractSessionToken(c *fiber.Ctx) string {
	if tok := c.Cookies(SessionCookieName); tok != "" {
		return tok
	}
	auth := c.Get("Authorization")
	if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

// ClaimRateLimiter returns a middleware that limits claim requests per IP.
func ClaimRateLimiter(maxPerMinute int) fiber.Handler {
	sw := newSlidingWindow(maxPerMinute)
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		if !sw.allow(ip) {
			return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded — try again later")
		}
		return c.Next()
	}
}

// AuthRateLimiter limits login / request-access style endpoints.
func AuthRateLimiter(maxPerMinute int) fiber.Handler {
	return ClaimRateLimiter(maxPerMinute)
}

type slidingWindow struct {
	mu      sync.Mutex
	windows map[string][]time.Time
	max     int
}

func newSlidingWindow(max int) *slidingWindow {
	if max < 1 {
		max = 10
	}
	sw := &slidingWindow{windows: make(map[string][]time.Time), max: max}
	go func() {
		t := time.NewTicker(5 * time.Minute)
		for range t.C {
			sw.cleanup()
		}
	}()
	return sw
}

func (sw *slidingWindow) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	sw.mu.Lock()
	defer sw.mu.Unlock()
	times := sw.windows[key]
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= sw.max {
		sw.windows[key] = valid
		return false
	}
	sw.windows[key] = append(valid, now)
	return true
}

func (sw *slidingWindow) cleanup() {
	cutoff := time.Now().Add(-time.Minute)
	sw.mu.Lock()
	defer sw.mu.Unlock()
	for k, times := range sw.windows {
		valid := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(sw.windows, k)
		} else {
			sw.windows[k] = valid
		}
	}
}
