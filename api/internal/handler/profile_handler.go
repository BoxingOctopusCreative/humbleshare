package handler

import (
	"errors"

	"github.com/boxingoctopus/humbleshare/api/internal/httpx"
	"github.com/boxingoctopus/humbleshare/api/internal/middleware"
	"github.com/boxingoctopus/humbleshare/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

type ProfileHandler struct {
	profiles *service.ProfileService
}

func NewProfileHandler(profiles *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profiles: profiles}
}

func (h *ProfileHandler) Get(c *fiber.Ctx) error {
	slug := c.Params("slug")
	viewer := middleware.UserFromLocals(c)
	profile, err := h.profiles.GetBySlug(c.Context(), slug, viewer)
	if err != nil {
		if errors.Is(err, service.ErrProfileNotFound) {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	return c.JSON(profile)
}
