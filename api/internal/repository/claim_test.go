package repository_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/boxingoctopus/humbleshare/api/internal/migrate"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestConcurrentClaim verifies that exactly one of two concurrent goroutines claiming the only
// available code for a game succeeds, while the other receives ErrNoAvailableCodes.
//
// Requires DATABASE_URL env var pointing at a live Postgres instance; skipped otherwise.
func TestConcurrentClaim(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping concurrent claim integration test")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// Apply migrations so the schema is ready.
	if _, err := migrate.UpWithExistingPool(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	gameRepo := repository.NewPostgresGameRepository(pool)
	codeRepo := repository.NewPostgresCodeRepository(pool)

	// Create a fresh game with a single available code.
	game, err := gameRepo.UpsertUnresolvedByName(ctx, "ConcurrentClaimTestGame_"+t.Name())
	if err != nil {
		t.Fatalf("upsert game: %v", err)
	}
	_, err = codeRepo.Create(ctx, game.ID, "TESTCODE-CONCURRENT-"+t.Name()+"-"+uuid.NewString(), "steam_key", nil, nil)
	if err != nil {
		t.Fatalf("create code: %v", err)
	}

	// Fire two goroutines simultaneously; only one should win.
	type result struct {
		err error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	for i := 0; i < 2; i++ {
		go func(i int) {
			defer wg.Done()
			claimedBy := "tester"
			_, err := codeRepo.ClaimOldestAvailable(ctx, game.ID, claimedBy, nil)
			results <- result{err: err}
		}(i)
	}

	wg.Wait()
	close(results)

	var successes, noCode int
	for r := range results {
		switch {
		case r.err == nil:
			successes++
		case errors.Is(r.err, repository.ErrNoAvailableCodes):
			noCode++
		default:
			t.Errorf("unexpected error: %v", r.err)
		}
	}

	if successes != 1 {
		t.Errorf("expected 1 success, got %d", successes)
	}
	if noCode != 1 {
		t.Errorf("expected 1 ErrNoAvailableCodes, got %d", noCode)
	}
}
