package service

import (
	"context"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
)

// GameService handles public game queries.
type GameService struct {
	games repository.GameRepository
}

func NewGameService(games repository.GameRepository) *GameService {
	return &GameService{games: games}
}

// ListAvailable returns games that have at least one available code. q is an optional search term.
func (s *GameService) ListAvailable(ctx context.Context, q string) ([]*models.Game, error) {
	return s.games.ListAvailable(ctx, q)
}
