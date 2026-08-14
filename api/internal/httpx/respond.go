package httpx

import (
	"errors"
	"log"

	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/boxingoctopus/humbleshare/api/internal/validation"
	"github.com/gofiber/fiber/v2"
)

// GenericClientMessage is returned for unexpected server errors so clients never see internals.
const GenericClientMessage = "Something went wrong. Please try again."

// ServiceError maps a service-layer error to an HTTP response. Known validation and business-rule
// errors return a safe client message; everything else is logged and returns GenericClientMessage.
func ServiceError(defaultStatus int, err error) error {
	if err == nil {
		return nil
	}
	if msg, code, ok := clientError(err); ok {
		return fiber.NewError(code, msg)
	}
	log.Printf("internal error: %v", err)
	return fiber.NewError(defaultStatus, GenericClientMessage)
}

func clientError(err error) (message string, status int, ok bool) {
	var inv *validation.InvalidInputError
	if errors.As(err, &inv) {
		return inv.Message, fiber.StatusBadRequest, true
	}
	if errors.Is(err, repository.ErrNoAvailableCodes) {
		return "Someone just grabbed that one — try another game.", fiber.StatusConflict, true
	}
	if errors.Is(err, repository.ErrCodeNotFound) {
		return "Code not found.", fiber.StatusNotFound, true
	}
	if errors.Is(err, repository.ErrGameNotFound) {
		return "Game not found.", fiber.StatusNotFound, true
	}
	if errors.Is(err, repository.ErrCodeDuplicate) {
		return "That code is already in the inventory.", fiber.StatusConflict, true
	}
	if errors.Is(err, repository.ErrCodeAlreadyTaken) {
		return "Code is already claimed.", fiber.StatusConflict, true
	}
	if errors.Is(err, repository.ErrNotClaimant) {
		return "You can only report a key you claimed.", fiber.StatusForbidden, true
	}
	return "", 0, false
}
