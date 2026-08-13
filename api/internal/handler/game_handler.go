package handler

import (
	"github.com/boxingoctopus/humbleshare/api/internal/httpx"
	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// GameHandler handles public game endpoints.
type GameHandler struct {
	games *service.GameService
}

func NewGameHandler(games *service.GameService) *GameHandler {
	return &GameHandler{games: games}
}

// List handles GET /api/v1/games?q=
// Returns games with at least one available code. code_value is never returned.
func (h *GameHandler) List(c *fiber.Ctx) error {
	q := c.Query("q")
	games, err := h.games.ListAvailable(c.Context(), q)
	if err != nil {
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	if games == nil {
		games = []*models.Game{}
	}
	return c.JSON(games)
}
