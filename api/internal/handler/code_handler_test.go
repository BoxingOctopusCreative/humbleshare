package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/boxingoctopus/humbleshare/api/internal/middleware"
	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/boxingoctopus/humbleshare/api/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type claimStubRepo struct {
	claimFn func(ctx context.Context, gameID uuid.UUID, claimedBy string, claimedByUserID *string) (*models.ClaimResult, error)
}

func (s *claimStubRepo) ClaimOldestAvailable(ctx context.Context, gameID uuid.UUID, claimedBy string, claimedByUserID *string) (*models.ClaimResult, error) {
	return s.claimFn(ctx, gameID, claimedBy, claimedByUserID)
}
func (s *claimStubRepo) Create(context.Context, uuid.UUID, string, string, *string, *string) (*models.Code, error) {
	return nil, nil
}
func (s *claimStubRepo) List(context.Context, string) ([]*models.Code, error) { return nil, nil }
func (s *claimStubRepo) GetByID(context.Context, uuid.UUID) (*models.Code, error) {
	return nil, repository.ErrCodeNotFound
}
func (s *claimStubRepo) Update(context.Context, uuid.UUID, repository.CodePatch) (*models.Code, error) {
	return nil, repository.ErrCodeNotFound
}
func (s *claimStubRepo) ReportDead(context.Context, uuid.UUID) error { return nil }

func TestCodeHandlerClaim(t *testing.T) {
	gameID := uuid.New()
	user := &models.User{ID: uuid.NewString(), DisplayName: "Ryan", Role: models.RoleMember}

	cases := []struct {
		name       string
		body       string
		user       *models.User
		claimErr   error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "success",
			body:       `{"game_id":"` + gameID.String() + `"}`,
			user:       user,
			wantStatus: http.StatusOK,
			wantCode:   "STEAM-KEY-1",
		},
		{
			name:       "invalid game id",
			body:       `{"game_id":"not-a-uuid"}`,
			user:       user,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "conflict",
			body:       `{"game_id":"` + gameID.String() + `"}`,
			user:       user,
			claimErr:   repository.ErrNoAvailableCodes,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "unauthorized",
			body:       `{"game_id":"` + gameID.String() + `"}`,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &claimStubRepo{
				claimFn: func(_ context.Context, _ uuid.UUID, _ string, _ *string) (*models.ClaimResult, error) {
					if tc.claimErr != nil {
						return nil, tc.claimErr
					}
					return &models.ClaimResult{
						ID:        uuid.New(),
						CodeType:  "steam_key",
						CodeValue: "STEAM-KEY-1",
					}, nil
				},
			}
			h := NewCodeHandler(service.NewCodeService(repo))
			app := fiber.New(fiber.Config{
				ErrorHandler: func(c *fiber.Ctx, err error) error {
					code := fiber.StatusInternalServerError
					msg := err.Error()
					if e, ok := err.(*fiber.Error); ok {
						code = e.Code
						msg = e.Message
					}
					return c.Status(code).JSON(fiber.Map{"error": true, "msg": msg})
				},
			})
			app.Post("/claim", func(c *fiber.Ctx) error {
				if tc.user != nil {
					c.Locals(middleware.LocalsUserKey, tc.user)
				}
				return h.Claim(c)
			})

			req := httptest.NewRequest(http.MethodPost, "/claim", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tc.wantStatus {
				b, _ := io.ReadAll(res.Body)
				t.Fatalf("status=%d body=%s want %d", res.StatusCode, b, tc.wantStatus)
			}
			if tc.wantCode != "" {
				var out models.ClaimResult
				if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
					t.Fatal(err)
				}
				if out.CodeValue != tc.wantCode {
					t.Fatalf("code=%q", out.CodeValue)
				}
			}
		})
	}
}
