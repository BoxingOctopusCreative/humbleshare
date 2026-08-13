package handler

import (
	"github.com/boxingoctopus/humbleshare/api/internal/httpx"
	"github.com/boxingoctopus/humbleshare/api/internal/middleware"
	"github.com/boxingoctopus/humbleshare/api/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CodeHandler handles public code endpoints.
type CodeHandler struct {
	codes *service.CodeService
}

func NewCodeHandler(codes *service.CodeService) *CodeHandler {
	return &CodeHandler{codes: codes}
}

// Claim handles POST /api/v1/codes/claim {game_id}. Claimant comes from the session.
func (h *CodeHandler) Claim(c *fiber.Ctx) error {
	u := middleware.UserFromLocals(c)
	if u == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var body struct {
		GameID string `json:"game_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	gameID, err := uuid.Parse(body.GameID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid game_id")
	}

	result, err := h.codes.Claim(c.Context(), gameID, u.DisplayName, u.ID)
	if err != nil {
		return httpx.ServiceError(fiber.StatusConflict, err)
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

// ReportDead handles POST /api/v1/codes/:id/report-dead
func (h *CodeHandler) ReportDead(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid code id")
	}

	if err := h.codes.ReportDead(c.Context(), id); err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
