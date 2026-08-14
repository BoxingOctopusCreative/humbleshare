package handler

import (
	"io"
	"strconv"
	"strings"

	"github.com/boxingoctopus/humbleshare/api/internal/httpx"
	"github.com/boxingoctopus/humbleshare/api/internal/media"
	"github.com/boxingoctopus/humbleshare/api/internal/storage"
	"github.com/gofiber/fiber/v2"
)

type MediaHandler struct {
	userAssets   storage.ObjectStore
	staticAssets storage.ObjectStore
	allowedHosts map[string]struct{}
	proxySecret  string
}

func NewMediaHandler(userAssets, staticAssets storage.ObjectStore, allowedHosts map[string]struct{}, proxySecret string) *MediaHandler {
	return &MediaHandler{
		userAssets:   userAssets,
		staticAssets: staticAssets,
		allowedHosts: allowedHosts,
		proxySecret:  strings.TrimSpace(proxySecret),
	}
}

func (h *MediaHandler) Get(c *fiber.Ctx) error {
	if !h.allowRequest(c) {
		return fiber.NewError(fiber.StatusForbidden, "hotlink denied")
	}
	kind := c.Params("kind")
	key, ok := media.ParseObjectKey(kind, c.Params("name"))
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid media path")
	}
	store := h.userAssets
	if kind == "svg" {
		store = h.staticAssets
	}
	if store == nil {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	}
	body, contentType, size, err := store.Get(c.Context(), key)
	if err != nil {
		if isObjectNotFound(err) {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return httpx.ServiceError(fiber.StatusInternalServerError, err)
	}
	defer body.Close()

	c.Set("Content-Type", contentType)
	if kind == "svg" {
		c.Set("Content-Type", "image/svg+xml")
	}
	c.Set("Cache-Control", "public, max-age=86400")
	c.Set("X-Content-Type-Options", "nosniff")
	if size > 0 {
		c.Set("Content-Length", strconv.FormatInt(size, 10))
	}
	_, err = io.Copy(c.Response().BodyWriter(), body)
	return err
}

func (h *MediaHandler) allowRequest(c *fiber.Ctx) bool {
	if h.proxySecret != "" && c.Get("X-Humbleshare-Media-Proxy") == h.proxySecret {
		return true
	}
	return media.RefererAllowed(c.Get("Referer"), c.Get("Origin"), h.allowedHosts)
}

func isObjectNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "nosuchkey") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "statuscode: 404")
}
