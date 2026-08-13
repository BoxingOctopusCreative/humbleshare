package service

import (
	"context"
	"strings"
	"testing"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/boxingoctopus/humbleshare/api/internal/validation"
	"github.com/google/uuid"
)

type stubCodeRepo struct {
	claimFn      func(ctx context.Context, gameID uuid.UUID, claimedBy string) (*models.ClaimResult, error)
	createFn     func(ctx context.Context, gameID uuid.UUID, codeValue, codeType string, source, sourceOrderID *string) (*models.Code, error)
	reportDeadFn func(ctx context.Context, id uuid.UUID) error
}

func (s *stubCodeRepo) ClaimOldestAvailable(ctx context.Context, gameID uuid.UUID, claimedBy string, claimedByUserID *string) (*models.ClaimResult, error) {
	if s.claimFn != nil {
		return s.claimFn(ctx, gameID, claimedBy)
	}
	return nil, repository.ErrNoAvailableCodes
}
func (s *stubCodeRepo) Create(ctx context.Context, gameID uuid.UUID, codeValue, codeType string, source, sourceOrderID *string) (*models.Code, error) {
	if s.createFn != nil {
		return s.createFn(ctx, gameID, codeValue, codeType, source, sourceOrderID)
	}
	return &models.Code{ID: uuid.New(), GameID: gameID, CodeValue: codeValue, CodeType: codeType}, nil
}
func (s *stubCodeRepo) List(context.Context, string) ([]*models.Code, error) { return nil, nil }
func (s *stubCodeRepo) ListRecentClaimsByUserID(context.Context, string, int) ([]models.PublicClaim, error) {
	return nil, nil
}
func (s *stubCodeRepo) GetByID(context.Context, uuid.UUID) (*models.Code, error) {
	return nil, repository.ErrCodeNotFound
}
func (s *stubCodeRepo) Update(context.Context, uuid.UUID, repository.CodePatch) (*models.Code, error) {
	return nil, repository.ErrCodeNotFound
}
func (s *stubCodeRepo) ReportDead(ctx context.Context, id uuid.UUID) error {
	if s.reportDeadFn != nil {
		return s.reportDeadFn(ctx, id)
	}
	return nil
}

type stubGameRepo struct {
	upsertFn func(ctx context.Context, name string) (*models.Game, error)
}

func (s *stubGameRepo) ListAvailable(context.Context, string) ([]*models.Game, error) { return nil, nil }
func (s *stubGameRepo) GetByID(context.Context, uuid.UUID) (*models.Game, error) {
	return nil, repository.ErrGameNotFound
}
func (s *stubGameRepo) UpsertBySteamAppID(context.Context, int, string) (*models.Game, error) {
	return nil, nil
}
func (s *stubGameRepo) UpsertUnresolvedByName(ctx context.Context, name string) (*models.Game, error) {
	if s.upsertFn != nil {
		return s.upsertFn(ctx, name)
	}
	return &models.Game{ID: uuid.New(), Name: name}, nil
}
func (s *stubGameRepo) ListUnresolved(context.Context) ([]*models.Game, error) { return nil, nil }
func (s *stubGameRepo) ListMissingScreenshots(context.Context) ([]*models.Game, error) {
	return nil, nil
}
func (s *stubGameRepo) SetSteamAppID(context.Context, uuid.UUID, int) error    { return nil }
func (s *stubGameRepo) UpdateName(context.Context, uuid.UUID, string) error    { return nil }
func (s *stubGameRepo) UpdateMetadata(context.Context, uuid.UUID, repository.GameMetadataPatch) error {
	return nil
}

func asInvalid(err error) *validation.InvalidInputError {
	inv, _ := err.(*validation.InvalidInputError)
	return inv
}

func TestCodeServiceClaim(t *testing.T) {
	gameID := uuid.New()

	t.Run("requires claimed_by", func(t *testing.T) {
		svc := NewCodeService(&stubCodeRepo{
			claimFn: func(context.Context, uuid.UUID, string) (*models.ClaimResult, error) {
				t.Fatal("should not call repo")
				return nil, nil
			},
		})
		_, err := svc.Claim(context.Background(), gameID, "  ", "user-1")
		if asInvalid(err) == nil {
			t.Fatalf("want InvalidInputError, got %v", err)
		}
	})

	t.Run("success trims name", func(t *testing.T) {
		svc := NewCodeService(&stubCodeRepo{
			claimFn: func(_ context.Context, id uuid.UUID, by string) (*models.ClaimResult, error) {
				if id != gameID || by != "Ryan" {
					t.Fatalf("unexpected args id=%s by=%q", id, by)
				}
				return &models.ClaimResult{GameID: id, ClaimedBy: by, CodeValue: "ABC"}, nil
			},
		})
		got, err := svc.Claim(context.Background(), gameID, " Ryan ", "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if got.CodeValue != "ABC" {
			t.Fatalf("unexpected: %+v", got)
		}
	})

	t.Run("propagates conflict", func(t *testing.T) {
		svc := NewCodeService(&stubCodeRepo{
			claimFn: func(context.Context, uuid.UUID, string) (*models.ClaimResult, error) {
				return nil, repository.ErrNoAvailableCodes
			},
		})
		_, err := svc.Claim(context.Background(), gameID, "Ryan", "user-1")
		if err != repository.ErrNoAvailableCodes {
			t.Fatalf("got %v", err)
		}
	})
}

func TestAdminServiceAddCodeValidation(t *testing.T) {
	svc := NewAdminService(&stubCodeRepo{}, &stubGameRepo{}, nil)

	cases := []struct {
		name      string
		title     string
		code      string
		codeType  string
		wantError string
	}{
		{name: "empty title", title: "", code: "X", codeType: models.CodeTypeSteamKey, wantError: "title is required"},
		{name: "empty code", title: "Game", code: "", codeType: models.CodeTypeSteamKey, wantError: "code_value is required"},
		{name: "bad type", title: "Game", code: "X", codeType: "origin", wantError: "code_type must be steam_key or gift_link"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.AddCode(context.Background(), tc.title, tc.code, tc.codeType, nil)
			inv := asInvalid(err)
			if inv == nil {
				t.Fatalf("want InvalidInputError, got %v", err)
			}
			if inv.Message != tc.wantError {
				t.Fatalf("msg=%q want=%q", inv.Message, tc.wantError)
			}
		})
	}
}

func TestAdminServiceBulkAdd(t *testing.T) {
	seen := map[string]bool{}
	codes := &stubCodeRepo{
		createFn: func(_ context.Context, _ uuid.UUID, codeValue, _ string, _, _ *string) (*models.Code, error) {
			if seen[codeValue] {
				return nil, repository.ErrCodeDuplicate
			}
			seen[codeValue] = true
			return &models.Code{ID: uuid.New(), CodeValue: codeValue}, nil
		},
	}
	svc := NewAdminService(codes, &stubGameRepo{}, nil)

	blob := "Good Game\tAAAA-BBBB\n" +
		"BadLineWithoutSep\n" +
		"Another,CCCC-DDDD\n" +
		"\n" +
		",ONLYCODE\n" +
		"Dup Game\tAAAA-BBBB\n"

	results := svc.BulkAdd(context.Background(), blob, "")
	if len(results) != 5 {
		t.Fatalf("got %d results want 5: %+v", len(results), results)
	}

	byLine := map[int]BulkImportResult{}
	for _, r := range results {
		byLine[r.Line] = r
	}

	if !byLine[1].Success || !byLine[3].Success {
		t.Fatalf("expected lines 1 and 3 success: %+v", results)
	}
	if byLine[2].Success || !strings.Contains(byLine[2].Error, "cannot parse line") {
		t.Fatalf("line 2: %+v", byLine[2])
	}
	if byLine[5].Success || !strings.Contains(byLine[5].Error, "title or code is empty") {
		t.Fatalf("line 5: %+v", byLine[5])
	}
	if byLine[6].Success || byLine[6].Error != "duplicate code" {
		t.Fatalf("line 6: %+v", byLine[6])
	}
}
