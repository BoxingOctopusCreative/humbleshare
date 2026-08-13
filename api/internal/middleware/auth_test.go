package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestClaimRateLimiter(t *testing.T) {
	app := fiber.New()
	app.Get("/x", ClaimRateLimiter(2), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("request %d status=%d", i, res.StatusCode)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status=%d want 429", res.StatusCode)
	}
}
