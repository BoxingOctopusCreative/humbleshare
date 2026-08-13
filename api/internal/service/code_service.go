package service

import (
	"context"
	"strings"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/boxingoctopus/humbleshare/api/internal/validation"
	"github.com/google/uuid"
)

// CodeService handles code claiming and lifecycle management.
type CodeService struct {
	codes repository.CodeRepository
}

func NewCodeService(codes repository.CodeRepository) *CodeService {
	return &CodeService{codes: codes}
}

// Claim atomically claims a code for gameID for the signed-in user.
func (s *CodeService) Claim(ctx context.Context, gameID uuid.UUID, claimedBy string, claimedByUserID string) (*models.ClaimResult, error) {
	if strings.TrimSpace(claimedBy) == "" {
		return nil, validation.Invalidf("claimed_by is required")
	}
	uid := strings.TrimSpace(claimedByUserID)
	var uidPtr *string
	if uid != "" {
		uidPtr = &uid
	}
	return s.codes.ClaimOldestAvailable(ctx, gameID, strings.TrimSpace(claimedBy), uidPtr)
}

// ReportDead marks a code as dead (e.g. already redeemed before we got it).
func (s *CodeService) ReportDead(ctx context.Context, id uuid.UUID) error {
	return s.codes.ReportDead(ctx, id)
}
