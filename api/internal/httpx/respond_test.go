package httpx

import (
	"errors"
	"testing"

	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/boxingoctopus/humbleshare/api/internal/validation"
	"github.com/gofiber/fiber/v2"
)

func TestServiceError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		defaultSt  int
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "nil",
			err:        nil,
			defaultSt:  500,
			wantStatus: 0,
		},
		{
			name:       "validation",
			err:        validation.Invalidf("bad input"),
			defaultSt:  500,
			wantStatus: fiber.StatusBadRequest,
			wantMsg:    "bad input",
		},
		{
			name:       "no available codes",
			err:        repository.ErrNoAvailableCodes,
			defaultSt:  500,
			wantStatus: fiber.StatusConflict,
			wantMsg:    "Someone just grabbed that one — try another game.",
		},
		{
			name:       "duplicate",
			err:        repository.ErrCodeDuplicate,
			defaultSt:  500,
			wantStatus: fiber.StatusConflict,
			wantMsg:    "That code is already in the inventory.",
		},
		{
			name:       "not found",
			err:        repository.ErrCodeNotFound,
			defaultSt:  500,
			wantStatus: fiber.StatusNotFound,
			wantMsg:    "Code not found.",
		},
		{
			name:       "unknown maps to generic",
			err:        errors.New("db blew up"),
			defaultSt:  fiber.StatusInternalServerError,
			wantStatus: fiber.StatusInternalServerError,
			wantMsg:    GenericClientMessage,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ServiceError(tc.defaultSt, tc.err)
			if tc.err == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			fe, ok := got.(*fiber.Error)
			if !ok {
				t.Fatalf("expected *fiber.Error, got %T", got)
			}
			if fe.Code != tc.wantStatus {
				t.Fatalf("status=%d want=%d", fe.Code, tc.wantStatus)
			}
			if fe.Message != tc.wantMsg {
				t.Fatalf("msg=%q want=%q", fe.Message, tc.wantMsg)
			}
		})
	}
}
