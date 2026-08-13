package repository

import (
	"context"
	"errors"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GameMetadataPatch carries optional Steam metadata fields for UpdateMetadata.
// ScreenshotURLs is applied when non-nil (empty slice clears stored screenshots).
type GameMetadataPatch struct {
	HeaderImageURL   *string
	CapsuleImageURL  *string
	ShortDescription *string
	StoreURL         *string
	ScreenshotURLs   *[]string
}

// GameRepository manages game records.
type GameRepository interface {
	// ListAvailable returns games with at least one available code. Never returns code_value.
	ListAvailable(ctx context.Context, q string) ([]*models.Game, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Game, error)
	UpsertBySteamAppID(ctx context.Context, steamAppID int, name string) (*models.Game, error)
	UpsertUnresolvedByName(ctx context.Context, name string) (*models.Game, error)
	ListUnresolved(ctx context.Context) ([]*models.Game, error)
	ListMissingScreenshots(ctx context.Context) ([]*models.Game, error)
	SetSteamAppID(ctx context.Context, id uuid.UUID, steamAppID int) error
	UpdateName(ctx context.Context, id uuid.UUID, name string) error
	UpdateMetadata(ctx context.Context, id uuid.UUID, patch GameMetadataPatch) error
}

type PostgresGameRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresGameRepository(pool *pgxpool.Pool) *PostgresGameRepository {
	return &PostgresGameRepository{pool: pool}
}

const gameSelectCols = `
		g.id, g.steam_appid, g.name,
		g.header_image_url, g.capsule_image_url, g.short_description,
		g.store_url, g.screenshot_urls, g.metadata_fetched_at, g.created_at`

const gameReturningCols = `
		id, steam_appid, name, header_image_url, capsule_image_url,
		short_description, store_url, screenshot_urls, metadata_fetched_at, created_at`

func scanGameRow(g *models.Game, available *int) []any {
	dest := []any{
		&g.ID, &g.SteamAppID, &g.Name,
		&g.HeaderImageURL, &g.CapsuleImageURL, &g.ShortDescription,
		&g.StoreURL, &g.ScreenshotURLs, &g.MetadataFetchedAt, &g.CreatedAt,
	}
	if available != nil {
		dest = append(dest, available)
	}
	return dest
}

// ListAvailable returns games that have at least one available code.
// When q is non-empty it filters by case-insensitive name prefix/substring.
// available_count is populated; code_value is never returned.
func (r *PostgresGameRepository) ListAvailable(ctx context.Context, q string) ([]*models.Game, error) {
	args := []any{}
	filter := ""
	if q != "" {
		filter = " AND g.name ILIKE $1"
		args = append(args, "%"+q+"%")
	}

	query := `
		SELECT` + gameSelectCols + `,
		       COUNT(c.id) FILTER (WHERE c.status = 'available') AS available_count
		FROM games g
		LEFT JOIN codes c ON c.game_id = g.id
		WHERE TRUE` + filter + `
		GROUP BY g.id
		HAVING COUNT(c.id) FILTER (WHERE c.status = 'available') > 0
		ORDER BY g.name ASC
	`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows, true)
}

func (r *PostgresGameRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Game, error) {
	var g models.Game
	err := r.pool.QueryRow(ctx, `
		SELECT`+gameSelectCols+`,
		       COUNT(c.id) FILTER (WHERE c.status = 'available') AS available_count
		FROM games g
		LEFT JOIN codes c ON c.game_id = g.id
		WHERE g.id = $1
		GROUP BY g.id
	`, id).Scan(scanGameRow(&g, &g.AvailableCount)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGameNotFound
	}
	return &g, err
}

// UpsertBySteamAppID inserts or updates a game keyed by steam_appid.
func (r *PostgresGameRepository) UpsertBySteamAppID(ctx context.Context, steamAppID int, name string) (*models.Game, error) {
	var g models.Game
	err := r.pool.QueryRow(ctx, `
		INSERT INTO games (steam_appid, name)
		VALUES ($1, $2)
		ON CONFLICT (steam_appid) DO UPDATE SET name = EXCLUDED.name
		RETURNING`+gameReturningCols+`
	`, steamAppID, name).Scan(scanGameRow(&g, nil)...)
	return &g, err
}

// UpsertUnresolvedByName inserts a game by name when no steam_appid is known yet.
// If a game with that exact name already exists it is returned without modification.
func (r *PostgresGameRepository) UpsertUnresolvedByName(ctx context.Context, name string) (*models.Game, error) {
	var g models.Game
	err := r.pool.QueryRow(ctx, `
		INSERT INTO games (name)
		VALUES ($1)
		ON CONFLICT DO NOTHING
		RETURNING`+gameReturningCols+`
	`, name).Scan(scanGameRow(&g, nil)...)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already exists — fetch by name.
		return r.getByName(ctx, name)
	}
	return &g, err
}

func (r *PostgresGameRepository) getByName(ctx context.Context, name string) (*models.Game, error) {
	var g models.Game
	err := r.pool.QueryRow(ctx, `
		SELECT`+gameReturningCols+`
		FROM games WHERE name = $1 LIMIT 1
	`, name).Scan(scanGameRow(&g, nil)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGameNotFound
	}
	return &g, err
}

// ListUnresolved returns games without a steam_appid.
func (r *PostgresGameRepository) ListUnresolved(ctx context.Context) ([]*models.Game, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT`+gameReturningCols+`
		FROM games
		WHERE steam_appid IS NULL
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows, false)
}

// ListMissingScreenshots returns resolved games that still need screenshot backfill.
func (r *PostgresGameRepository) ListMissingScreenshots(ctx context.Context) ([]*models.Game, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT`+gameReturningCols+`
		FROM games
		WHERE steam_appid IS NOT NULL
		  AND cardinality(screenshot_urls) = 0
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows, false)
}

func (r *PostgresGameRepository) SetSteamAppID(ctx context.Context, id uuid.UUID, steamAppID int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE games SET steam_appid = $2 WHERE id = $1
	`, id, steamAppID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGameNotFound
	}
	return nil
}

func (r *PostgresGameRepository) UpdateName(ctx context.Context, id uuid.UUID, name string) error {
	// Clear Steam linkage when the title changes so metadata matches the new name.
	tag, err := r.pool.Exec(ctx, `
		UPDATE games
		SET name = $2,
		    steam_appid = CASE WHEN name IS DISTINCT FROM $2 THEN NULL ELSE steam_appid END,
		    header_image_url = CASE WHEN name IS DISTINCT FROM $2 THEN NULL ELSE header_image_url END,
		    capsule_image_url = CASE WHEN name IS DISTINCT FROM $2 THEN NULL ELSE capsule_image_url END,
		    short_description = CASE WHEN name IS DISTINCT FROM $2 THEN NULL ELSE short_description END,
		    store_url = CASE WHEN name IS DISTINCT FROM $2 THEN NULL ELSE store_url END,
		    screenshot_urls = CASE WHEN name IS DISTINCT FROM $2 THEN '{}'::text[] ELSE screenshot_urls END,
		    metadata_fetched_at = CASE WHEN name IS DISTINCT FROM $2 THEN NULL ELSE metadata_fetched_at END
		WHERE id = $1
	`, id, name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGameNotFound
	}
	return nil
}

func (r *PostgresGameRepository) UpdateMetadata(ctx context.Context, id uuid.UUID, patch GameMetadataPatch) error {
	var screenshots any
	if patch.ScreenshotURLs != nil {
		screenshots = *patch.ScreenshotURLs
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE games
		SET header_image_url   = COALESCE($2, header_image_url),
		    capsule_image_url  = COALESCE($3, capsule_image_url),
		    short_description  = COALESCE($4, short_description),
		    store_url          = COALESCE($5, store_url),
		    screenshot_urls    = COALESCE($6, screenshot_urls),
		    metadata_fetched_at = NOW()
		WHERE id = $1
	`, id, patch.HeaderImageURL, patch.CapsuleImageURL, patch.ShortDescription, patch.StoreURL, screenshots)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGameNotFound
	}
	return nil
}

func scanGames(rows pgx.Rows, withAvailable bool) ([]*models.Game, error) {
	var games []*models.Game
	for rows.Next() {
		var g models.Game
		var err error
		if withAvailable {
			err = rows.Scan(scanGameRow(&g, &g.AvailableCount)...)
		} else {
			err = rows.Scan(scanGameRow(&g, nil)...)
		}
		if err != nil {
			return nil, err
		}
		games = append(games, &g)
	}
	return games, rows.Err()
}
