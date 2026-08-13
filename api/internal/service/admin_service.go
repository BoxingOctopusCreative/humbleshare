package service

import (
	"context"
	"errors"
	"strings"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/boxingoctopus/humbleshare/api/internal/repository"
	"github.com/boxingoctopus/humbleshare/api/internal/validation"
	"github.com/google/uuid"
)

// BulkImportResult holds per-line outcome for POST /admin/codes/bulk.
type BulkImportResult struct {
	Line    int    `json:"line"`
	Title   string `json:"title"`
	Code    string `json:"code"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// AdminService handles privileged operations.
type AdminService struct {
	codes repository.CodeRepository
	games repository.GameRepository
	stats repository.StatsRepository
}

func NewAdminService(
	codes repository.CodeRepository,
	games repository.GameRepository,
	stats repository.StatsRepository,
) *AdminService {
	return &AdminService{codes: codes, games: games, stats: stats}
}

// AddCode looks up or creates a game by title, then inserts a code.
func (s *AdminService) AddCode(ctx context.Context, title, codeValue, codeType string, source *string) (*models.Code, error) {
	title = strings.TrimSpace(title)
	codeValue = strings.TrimSpace(codeValue)
	if title == "" {
		return nil, validation.Invalidf("title is required")
	}
	if codeValue == "" {
		return nil, validation.Invalidf("code_value is required")
	}
	if codeType != models.CodeTypeSteamKey && codeType != models.CodeTypeGiftLink {
		return nil, validation.Invalidf("code_type must be steam_key or gift_link")
	}

	game, err := s.games.UpsertUnresolvedByName(ctx, title)
	if err != nil {
		return nil, err
	}
	return s.codes.Create(ctx, game.ID, codeValue, codeType, source, nil)
}

// BulkAdd parses a blob of Tab-separated or comma-separated lines (Title<sep>Code).
// Returns per-line results; never aborts early.
func (s *AdminService) BulkAdd(ctx context.Context, blob string, codeType string) []BulkImportResult {
	if codeType == "" {
		codeType = models.CodeTypeSteamKey
	}
	lines := strings.Split(strings.ReplaceAll(blob, "\r\n", "\n"), "\n")
	results := make([]BulkImportResult, 0, len(lines))

	for i, raw := range lines {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		lineNum := i + 1

		var title, code string
		if idx := strings.IndexByte(raw, '\t'); idx >= 0 {
			title = strings.TrimSpace(raw[:idx])
			code = strings.TrimSpace(raw[idx+1:])
		} else if idx := strings.IndexByte(raw, ','); idx >= 0 {
			title = strings.TrimSpace(raw[:idx])
			code = strings.TrimSpace(raw[idx+1:])
		} else {
			results = append(results, BulkImportResult{
				Line:  lineNum,
				Error: "cannot parse line: expected Title<tab>Code or Title,Code",
			})
			continue
		}

		if title == "" || code == "" {
			results = append(results, BulkImportResult{
				Line:  lineNum,
				Title: title,
				Code:  code,
				Error: "title or code is empty",
			})
			continue
		}

		_, err := s.AddCode(ctx, title, code, codeType, nil)
		if errors.Is(err, repository.ErrCodeDuplicate) {
			results = append(results, BulkImportResult{
				Line:  lineNum,
				Title: title,
				Code:  code,
				Error: "duplicate code",
			})
			continue
		}
		if err != nil {
			results = append(results, BulkImportResult{
				Line:  lineNum,
				Title: title,
				Code:  code,
				Error: err.Error(),
			})
			continue
		}
		results = append(results, BulkImportResult{
			Line:    lineNum,
			Title:   title,
			Code:    code,
			Success: true,
		})
	}
	return results
}

// ListCodes returns all codes for admin inventory (includes code_value).
func (s *AdminService) ListCodes(ctx context.Context, status string) ([]*models.Code, error) {
	return s.codes.List(ctx, status)
}

// UpdateCode patches a code's status and/or notes.
func (s *AdminService) UpdateCode(ctx context.Context, id uuid.UUID, patch repository.CodePatch) (*models.Code, error) {
	return s.codes.Update(ctx, id, patch)
}

// ListUnresolved returns games without a steam_appid.
func (s *AdminService) ListUnresolved(ctx context.Context) ([]*models.Game, error) {
	return s.games.ListUnresolved(ctx)
}

// SetSteamAppID sets a game's steam_appid.
func (s *AdminService) SetSteamAppID(ctx context.Context, gameID uuid.UUID, steamAppID int) error {
	return s.games.SetSteamAppID(ctx, gameID, steamAppID)
}

// UpdateGameName renames a game.
func (s *AdminService) UpdateGameName(ctx context.Context, gameID uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return validation.Invalidf("name is required")
	}
	if len(name) > 256 {
		return validation.Invalidf("name is too long")
	}
	return s.games.UpdateName(ctx, gameID, name)
}

// GetStats returns aggregated usage statistics.
func (s *AdminService) GetStats(ctx context.Context) (*models.Stats, error) {
	return s.stats.GetStats(ctx)
}

// GetGame fetches a game by ID.
func (s *AdminService) GetGame(ctx context.Context, id uuid.UUID) (*models.Game, error) {
	return s.games.GetByID(ctx, id)
}

// UpdateGameMetadata patches a game's Steam metadata fields.
func (s *AdminService) UpdateGameMetadata(ctx context.Context, id uuid.UUID, patch repository.GameMetadataPatch) error {
	return s.games.UpdateMetadata(ctx, id, patch)
}
