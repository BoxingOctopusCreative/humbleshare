package repository

import (
	"context"
	"errors"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCodeNotFound     = errors.New("code not found")
	ErrCodeAlreadyTaken = errors.New("code already claimed or unavailable")
	ErrCodeDuplicate    = errors.New("code already exists")
	ErrNoAvailableCodes = errors.New("no available codes for game")
	ErrGameNotFound     = errors.New("game not found")
)

// CodePatch carries optional fields for PATCH /admin/codes/:id.
type CodePatch struct {
	Status    *string
	Notes     *string
	CodeValue *string
}

// CodeRepository persists redeemable codes and atomic claims.
type CodeRepository interface {
	// ClaimOldestAvailable claims the oldest available code for gameID in a single
	// conditional UPDATE. Returns ErrNoAvailableCodes when none remain (or a race lost).
	ClaimOldestAvailable(ctx context.Context, gameID uuid.UUID, claimedBy string, claimedByUserID *string) (*models.ClaimResult, error)
	Create(ctx context.Context, gameID uuid.UUID, codeValue, codeType string, source, sourceOrderID *string) (*models.Code, error)
	List(ctx context.Context, status string) ([]*models.Code, error)
	ListRecentClaimsByUserID(ctx context.Context, userID string, limit int) ([]models.PublicClaim, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Code, error)
	Update(ctx context.Context, id uuid.UUID, patch CodePatch) (*models.Code, error)
	ReportDead(ctx context.Context, id uuid.UUID) error
}

type PostgresCodeRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCodeRepository(pool *pgxpool.Pool) *PostgresCodeRepository {
	return &PostgresCodeRepository{pool: pool}
}

// ClaimOldestAvailable picks the oldest available code for the game and claims it
// atomically. Zero rows → ErrNoAvailableCodes (map to 409 at the handler).
func (r *PostgresCodeRepository) ClaimOldestAvailable(ctx context.Context, gameID uuid.UUID, claimedBy string, claimedByUserID *string) (*models.ClaimResult, error) {
	var (
		id        uuid.UUID
		codeType  string
		codeValue string
		claimedAt time.Time
	)

	// CTE locks the candidate row; UPDATE only succeeds while status is still available.
	// SKIP LOCKED lets a concurrent claimer take the next oldest instead of waiting.
	err := r.pool.QueryRow(ctx, `
		WITH next_code AS (
			SELECT id
			FROM codes
			WHERE game_id = $1 AND status = 'available'
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE codes c
		SET status = 'claimed',
		    claimed_by = $2,
		    claimed_by_user_id = $3,
		    claimed_at = NOW()
		FROM next_code
		WHERE c.id = next_code.id AND c.status = 'available'
		RETURNING c.id, c.code_type, c.code_value, c.claimed_at
	`, gameID, claimedBy, claimedByUserID).Scan(&id, &codeType, &codeValue, &claimedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoAvailableCodes
	}
	if err != nil {
		return nil, err
	}

	return &models.ClaimResult{
		ID:        id,
		GameID:    gameID,
		CodeType:  codeType,
		CodeValue: codeValue,
		ClaimedBy: claimedBy,
		ClaimedAt: claimedAt,
	}, nil
}

func (r *PostgresCodeRepository) Create(ctx context.Context, gameID uuid.UUID, codeValue, codeType string, source, sourceOrderID *string) (*models.Code, error) {
	var c models.Code
	err := r.pool.QueryRow(ctx, `
		INSERT INTO codes (game_id, code_value, code_type, source, source_order_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (code_value) DO NOTHING
		RETURNING id, game_id, code_type, code_value, source, source_order_id, status, claimed_by, claimed_at, notes, created_at
	`, gameID, codeValue, codeType, source, sourceOrderID).Scan(
		&c.ID, &c.GameID, &c.CodeType, &c.CodeValue, &c.Source, &c.SourceOrderID,
		&c.Status, &c.ClaimedBy, &c.ClaimedAt, &c.Notes, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCodeDuplicate
	}
	return &c, err
}

func (r *PostgresCodeRepository) List(ctx context.Context, status string) ([]*models.Code, error) {
	query := `
		SELECT c.id, c.game_id, g.name, c.code_type, c.code_value, c.source, c.source_order_id,
		       c.status, c.claimed_by, c.claimed_at, c.notes, c.created_at
		FROM codes c
		JOIN games g ON g.id = c.game_id
	`
	args := []any{}
	if status != "" {
		query += " WHERE c.status = $1"
		args = append(args, status)
	}
	query += " ORDER BY c.created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []*models.Code
	for rows.Next() {
		var c models.Code
		if err := rows.Scan(
			&c.ID, &c.GameID, &c.GameName, &c.CodeType, &c.CodeValue, &c.Source, &c.SourceOrderID,
			&c.Status, &c.ClaimedBy, &c.ClaimedAt, &c.Notes, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		codes = append(codes, &c)
	}
	return codes, rows.Err()
}

func (r *PostgresCodeRepository) ListRecentClaimsByUserID(ctx context.Context, userID string, limit int) ([]models.PublicClaim, error) {
	if limit < 1 {
		limit = 12
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.game_id, g.name, g.header_image_url, c.code_type, c.claimed_at
		FROM codes c
		JOIN games g ON g.id = c.game_id
		WHERE c.claimed_by_user_id = $1 AND c.status = 'claimed' AND c.claimed_at IS NOT NULL
		ORDER BY c.claimed_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.PublicClaim, 0)
	for rows.Next() {
		var c models.PublicClaim
		if err := rows.Scan(&c.GameID, &c.GameName, &c.HeaderImageURL, &c.CodeType, &c.ClaimedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresCodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Code, error) {
	var c models.Code
	err := r.pool.QueryRow(ctx, `
		SELECT id, game_id, code_type, code_value, source, source_order_id, status, claimed_by, claimed_at, notes, created_at
		FROM codes WHERE id = $1
	`, id).Scan(
		&c.ID, &c.GameID, &c.CodeType, &c.CodeValue, &c.Source, &c.SourceOrderID,
		&c.Status, &c.ClaimedBy, &c.ClaimedAt, &c.Notes, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCodeNotFound
	}
	return &c, err
}

func (r *PostgresCodeRepository) Update(ctx context.Context, id uuid.UUID, patch CodePatch) (*models.Code, error) {
	var c models.Code
	err := r.pool.QueryRow(ctx, `
		UPDATE codes
		SET status     = COALESCE($2, status),
		    notes      = COALESCE($3, notes),
		    code_value = COALESCE($4, code_value)
		WHERE id = $1
		RETURNING id, game_id, code_type, code_value, source, source_order_id, status, claimed_by, claimed_at, notes, created_at
	`, id, patch.Status, patch.Notes, patch.CodeValue).Scan(
		&c.ID, &c.GameID, &c.CodeType, &c.CodeValue, &c.Source, &c.SourceOrderID,
		&c.Status, &c.ClaimedBy, &c.ClaimedAt, &c.Notes, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCodeNotFound
	}
	return &c, err
}

func (r *PostgresCodeRepository) ReportDead(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE codes SET status = 'dead' WHERE id = $1 AND status != 'dead'
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Either not found or already dead — verify existence.
		var exists bool
		_ = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM codes WHERE id = $1)`, id).Scan(&exists)
		if !exists {
			return ErrCodeNotFound
		}
	}
	return nil
}
