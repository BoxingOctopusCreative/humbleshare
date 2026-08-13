package handler

import (
	"context"
	"strings"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/httpx"
	"github.com/boxingoctopus/humbleshare/api/internal/humble"
	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/boxingoctopus/humbleshare/api/internal/service"
	"github.com/boxingoctopus/humbleshare/api/internal/steam"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AdminHandler handles all privileged admin endpoints.
type AdminHandler struct {
	admin       *service.AdminService
	humbleSvc   *humble.Service
	steamClient *steam.Client
}

func NewAdminHandler(admin *service.AdminService, humbleSvc *humble.Service, steamClient *steam.Client) *AdminHandler {
	return &AdminHandler{
		admin:       admin,
		humbleSvc:   humbleSvc,
		steamClient: steamClient,
	}
}

// ImportHumble handles POST /api/v1/admin/import/humble
// Body: {session_cookie, order_ids: [], titles?: { [gift_url]: "Game Title" }}
func (h *AdminHandler) ImportHumble(c *fiber.Ctx) error {
	var body struct {
		SessionCookie string            `json:"session_cookie"`
		OrderIDs      []string          `json:"order_ids"`
		Titles        map[string]string `json:"titles"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.SessionCookie == "" {
		return fiber.NewError(fiber.StatusBadRequest, "session_cookie is required")
	}
	if len(body.OrderIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "order_ids must not be empty")
	}

	results, err := h.humbleSvc.ImportOrders(c.Context(), body.SessionCookie, body.OrderIDs, body.Titles)
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(results)
}

// AddCode handles POST /api/v1/admin/codes
// Body: {title, code_value, code_type, source?}
func (h *AdminHandler) AddCode(c *fiber.Ctx) error {
	var body struct {
		Title     string  `json:"title"`
		CodeValue string  `json:"code_value"`
		CodeType  string  `json:"code_type"`
		Source    *string `json:"source"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	code, err := h.admin.AddCode(c.Context(), body.Title, body.CodeValue, body.CodeType, body.Source)
	if err != nil {
		return httpx.ServiceError(fiber.StatusBadRequest, err)
	}
	return c.Status(fiber.StatusCreated).JSON(code)
}

// BulkAddCodes handles POST /api/v1/admin/codes/bulk
// Body: {blob, code_type?}
func (h *AdminHandler) BulkAddCodes(c *fiber.Ctx) error {
	var body struct {
		Blob     string `json:"blob"`
		CodeType string `json:"code_type"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.Blob == "" {
		return fiber.NewError(fiber.StatusBadRequest, "blob is required")
	}

	results := h.admin.BulkAdd(c.Context(), body.Blob, body.CodeType)
	return c.JSON(results)
}

// ListCodes handles GET /api/v1/admin/codes?status=
func (h *AdminHandler) ListCodes(c *fiber.Ctx) error {
	status := c.Query("status")
	codes, err := h.admin.ListCodes(c.Context(), status)
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	if codes == nil {
		codes = []*models.Code{}
	}
	return c.JSON(codes)
}

// UpdateCode handles PATCH /api/v1/admin/codes/:id
func (h *AdminHandler) UpdateCode(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid code id")
	}

	var body struct {
		Status    *string `json:"status"`
		Notes     *string `json:"notes"`
		CodeValue *string `json:"code_value"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	code, err := h.admin.UpdateCode(c.Context(), id, repository.CodePatch{
		Status:    body.Status,
		Notes:     body.Notes,
		CodeValue: body.CodeValue,
	})
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(code)
}

// ListUnresolved handles GET /api/v1/admin/games/unresolved
func (h *AdminHandler) ListUnresolved(c *fiber.Ctx) error {
	games, err := h.admin.ListUnresolved(c.Context())
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	if games == nil {
		games = []*models.Game{}
	}
	return c.JSON(games)
}

// SetGameSteamAppID handles PATCH /api/v1/admin/games/:id
// Body may include steam_appid and/or name.
func (h *AdminHandler) SetGameSteamAppID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid game id")
	}

	var body struct {
		SteamAppID *int    `json:"steam_appid"`
		Name       *string `json:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.Name == nil && body.SteamAppID == nil {
		return fiber.NewError(fiber.StatusBadRequest, "name or steam_appid is required")
	}

	if body.Name != nil {
		existing, err := h.admin.GetGame(c.Context(), id)
		if err != nil {
			return httpx.ServiceError(fiber.StatusInternalServerError, err)
		}
		newName := strings.TrimSpace(*body.Name)
		if err := h.admin.UpdateGameName(c.Context(), id, newName); err != nil {
			return httpx.ServiceError(fiber.StatusBadRequest, err)
		}
		// Re-resolve Steam metadata when the title changes (unless appid is set in this request).
		if body.SteamAppID == nil && existing.Name != newName {
			if err := h.steamClient.EnrichGame(c.Context(), id, newName); err != nil {
				// Non-fatal; title is saved and admin can retry Lookup.
				_ = err
			}
		}
	}

	if body.SteamAppID != nil {
		if *body.SteamAppID <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "steam_appid must be a positive integer")
		}
		if err := h.admin.SetSteamAppID(c.Context(), id, *body.SteamAppID); err != nil {
			return httpx.ServiceError(fiber.StatusInternalServerError, err)
		}

		// Refresh metadata in background to not block the response.
		appID := *body.SteamAppID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.steamClient.RefreshMetadata(ctx, id, appID); err != nil {
				// Non-fatal; admin can retry via /refresh endpoint.
				_ = err
			}
		}()
	}

	game, err := h.admin.GetGame(c.Context(), id)
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(game)
}

// RefreshGameMetadata handles POST /api/v1/admin/games/:id/refresh.
// With a steam_appid, re-fetches store metadata. Without one, searches Steam by title
// and sets appid + metadata on an exact name match.
func (h *AdminHandler) RefreshGameMetadata(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid game id")
	}

	game, err := h.admin.GetGame(c.Context(), id)
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}

	if game.SteamAppID == nil {
		if err := h.steamClient.EnrichGame(c.Context(), id, game.Name); err != nil {
			return httpx.ServiceError(fiber.StatusBadGateway, err)
		}
		updated, err := h.admin.GetGame(c.Context(), id)
		if err != nil {
			return httpx.ServiceError(fiber.StatusInternalServerError, err)
		}
		if updated.SteamAppID == nil {
			return fiber.NewError(fiber.StatusNotFound, "no exact Steam store match for that title")
		}
		return c.JSON(updated)
	}

	if err := h.steamClient.RefreshMetadata(c.Context(), id, *game.SteamAppID); err != nil {
		return httpx.ServiceError(fiber.StatusBadGateway, err)
	}

	updated, err := h.admin.GetGame(c.Context(), id)
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(updated)
}

// GetStats handles GET /api/v1/admin/stats
func (h *AdminHandler) GetStats(c *fiber.Ctx) error {
	stats, err := h.admin.GetStats(c.Context())
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(stats)
}
