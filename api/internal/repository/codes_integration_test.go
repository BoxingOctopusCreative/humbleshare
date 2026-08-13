package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/boxingoctopus/humbleshare/api/internal/migrate"
	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping repository integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := migrate.UpWithExistingPool(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func TestCreateDuplicateAndClaimFlow(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	gameRepo := repository.NewPostgresGameRepository(pool)
	codeRepo := repository.NewPostgresCodeRepository(pool)

	game, err := gameRepo.UpsertUnresolvedByName(ctx, "DupFlowGame_"+t.Name())
	if err != nil {
		t.Fatal(err)
	}

	codeVal := "DUP-FLOW-" + t.Name() + "-" + uuid.NewString()
	_, err = codeRepo.Create(ctx, game.ID, codeVal, models.CodeTypeSteamKey, nil, nil)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err = codeRepo.Create(ctx, game.ID, codeVal, models.CodeTypeSteamKey, nil, nil)
	if !errors.Is(err, repository.ErrCodeDuplicate) {
		t.Fatalf("second create want ErrCodeDuplicate, got %v", err)
	}

	claim, err := codeRepo.ClaimOldestAvailable(ctx, game.ID, "tester", nil)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claim.CodeValue != codeVal {
		t.Fatalf("claimed %q want %q", claim.CodeValue, codeVal)
	}

	_, err = codeRepo.ClaimOldestAvailable(ctx, game.ID, "other", nil)
	if !errors.Is(err, repository.ErrNoAvailableCodes) {
		t.Fatalf("second claim want ErrNoAvailableCodes, got %v", err)
	}

	available, err := gameRepo.ListAvailable(ctx, game.Name)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range available {
		if g.ID == game.ID {
			t.Fatal("claimed-out game should not appear in ListAvailable")
		}
	}
}
