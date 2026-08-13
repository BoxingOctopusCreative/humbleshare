package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/google/uuid"
)

type profileUserStub struct {
	bySlug map[string]*models.User
}

func (s *profileUserStub) Create(context.Context, string, string, string, string, string) (*models.User, error) {
	return nil, errors.New("unused")
}
func (s *profileUserStub) GetByID(context.Context, string) (*models.User, error) {
	return nil, repository.ErrUserNotFound
}
func (s *profileUserStub) GetByEmail(context.Context, string) (*models.User, error) {
	return nil, repository.ErrUserNotFound
}
func (s *profileUserStub) GetByProfileSlug(_ context.Context, slug string) (*models.User, error) {
	u, ok := s.bySlug[slug]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return u, nil
}
func (s *profileUserStub) List(context.Context) ([]models.User, error) { return nil, nil }
func (s *profileUserStub) ListAdminEmails(context.Context) ([]string, error) {
	return nil, nil
}
func (s *profileUserStub) UpdatePassword(context.Context, string, string) error { return nil }
func (s *profileUserStub) UpdateProfile(context.Context, string, string, string, string, string, string, bool) error {
	return nil
}
func (s *profileUserStub) UpdateAvatarURL(context.Context, string, string) error { return nil }
func (s *profileUserStub) UpdateBannerURL(context.Context, string, string) error { return nil }
func (s *profileUserStub) UpdateRole(context.Context, string, string) error      { return nil }
func (s *profileUserStub) SetTwoFactorSecret(context.Context, string, string) error {
	return nil
}
func (s *profileUserStub) EnableTwoFactor(context.Context, string) error  { return nil }
func (s *profileUserStub) DisableTwoFactor(context.Context, string) error { return nil }
func (s *profileUserStub) GetTwoFactorSecret(context.Context, string) (string, error) {
	return "", errors.New("unused")
}

type profileCodeStub struct {
	claims []models.PublicClaim
}

func (s *profileCodeStub) ClaimOldestAvailable(context.Context, uuid.UUID, string, *string) (*models.ClaimResult, error) {
	return nil, repository.ErrNoAvailableCodes
}
func (s *profileCodeStub) Create(context.Context, uuid.UUID, string, string, *string, *string) (*models.Code, error) {
	return nil, nil
}
func (s *profileCodeStub) List(context.Context, string) ([]*models.Code, error) { return nil, nil }
func (s *profileCodeStub) ListRecentClaimsByUserID(context.Context, string, int) ([]models.PublicClaim, error) {
	return s.claims, nil
}
func (s *profileCodeStub) GetByID(context.Context, uuid.UUID) (*models.Code, error) {
	return nil, repository.ErrCodeNotFound
}
func (s *profileCodeStub) Update(context.Context, uuid.UUID, repository.CodePatch) (*models.Code, error) {
	return nil, repository.ErrCodeNotFound
}
func (s *profileCodeStub) ReportDead(context.Context, uuid.UUID) error { return nil }

func TestProfileServiceGetBySlug(t *testing.T) {
	ownerID := uuid.NewString()
	claim := models.PublicClaim{
		GameID:    uuid.New(),
		GameName:  "Portal",
		CodeType:  models.CodeTypeSteamKey,
		ClaimedAt: time.Now().UTC(),
	}
	users := &profileUserStub{bySlug: map[string]*models.User{
		"ryan": {
			ID:            ownerID,
			DisplayName:   "Ryan",
			Bio:           "hi",
			ProfileSlug:   "ryan",
			ProfilePublic: true,
			Email:         "secret@example.com",
			Role:          models.RoleMember,
		},
		"private-user": {
			ID:            uuid.NewString(),
			DisplayName:   "Hidden",
			ProfileSlug:   "private-user",
			ProfilePublic: false,
		},
	}}
	codes := &profileCodeStub{claims: []models.PublicClaim{claim}}
	svc := NewProfileService(users, codes, "https://app.example", "https://cdn.example")

	t.Run("public visible to anon", func(t *testing.T) {
		p, err := svc.GetBySlug(context.Background(), "ryan", nil)
		if err != nil {
			t.Fatal(err)
		}
		if p.DisplayName != "Ryan" || !p.ProfilePublic || p.IsOwner {
			t.Fatalf("unexpected profile: %+v", p)
		}
		if len(p.RecentClaims) != 1 || p.RecentClaims[0].GameName != "Portal" {
			t.Fatalf("claims: %+v", p.RecentClaims)
		}
	})

	t.Run("private hidden from stranger", func(t *testing.T) {
		_, err := svc.GetBySlug(context.Background(), "private-user", &models.User{ID: "other"})
		if !errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("private visible to owner", func(t *testing.T) {
		u := users.bySlug["private-user"]
		p, err := svc.GetBySlug(context.Background(), "private-user", &models.User{ID: u.ID})
		if err != nil {
			t.Fatal(err)
		}
		if !p.IsOwner || p.ProfilePublic {
			t.Fatalf("unexpected: %+v", p)
		}
	})

	t.Run("missing slug", func(t *testing.T) {
		_, err := svc.GetBySlug(context.Background(), "nobody", nil)
		if !errors.Is(err, ErrProfileNotFound) {
			t.Fatalf("got %v", err)
		}
	})
}
