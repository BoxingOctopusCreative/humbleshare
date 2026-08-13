package repository

import (
	"context"
	"errors"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSessionRepository struct{ pool *pgxpool.Pool }

func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*models.Session, error) {
	var s models.Session
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at`,
		userID, tokenHash, expiresAt,
	).Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresSessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.Session, error) {
	var s models.Session
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM sessions WHERE token_hash = $1 AND expires_at > NOW()`, tokenHash,
	).Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresSessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *PostgresSessionRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= NOW()`)
	return err
}

type PostgresAccessRequestRepository struct{ pool *pgxpool.Pool }

func NewPostgresAccessRequestRepository(pool *pgxpool.Pool) *PostgresAccessRequestRepository {
	return &PostgresAccessRequestRepository{pool: pool}
}

func (r *PostgresAccessRequestRepository) Create(ctx context.Context, email, note string) (*models.AccessRequest, error) {
	var ar models.AccessRequest
	err := r.pool.QueryRow(ctx, `
		INSERT INTO access_requests (email, note)
		VALUES (lower(trim($1)), $2)
		RETURNING id, email, note, status, created_at, resolved_at, resolved_by`,
		email, note,
	).Scan(&ar.ID, &ar.Email, &ar.Note, &ar.Status, &ar.CreatedAt, &ar.ResolvedAt, &ar.ResolvedBy)
	if err != nil {
		return nil, err
	}
	return &ar, nil
}

func (r *PostgresAccessRequestRepository) GetByID(ctx context.Context, id string) (*models.AccessRequest, error) {
	var ar models.AccessRequest
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, note, status, created_at, resolved_at, resolved_by
		FROM access_requests WHERE id = $1`, id,
	).Scan(&ar.ID, &ar.Email, &ar.Note, &ar.Status, &ar.CreatedAt, &ar.ResolvedAt, &ar.ResolvedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRequestNotFound
	}
	return &ar, err
}

func (r *PostgresAccessRequestRepository) ListPending(ctx context.Context) ([]models.AccessRequest, error) {
	return r.list(ctx, `WHERE status = 'pending'`)
}

func (r *PostgresAccessRequestRepository) List(ctx context.Context) ([]models.AccessRequest, error) {
	return r.list(ctx, "")
}

func (r *PostgresAccessRequestRepository) list(ctx context.Context, where string) ([]models.AccessRequest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, email, note, status, created_at, resolved_at, resolved_by
		FROM access_requests `+where+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AccessRequest
	for rows.Next() {
		var ar models.AccessRequest
		if err := rows.Scan(&ar.ID, &ar.Email, &ar.Note, &ar.Status, &ar.CreatedAt, &ar.ResolvedAt, &ar.ResolvedBy); err != nil {
			return nil, err
		}
		out = append(out, ar)
	}
	return out, rows.Err()
}

func (r *PostgresAccessRequestRepository) HasPending(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM access_requests
			WHERE lower(email) = lower(trim($1)) AND status = 'pending'
		)`, email).Scan(&exists)
	return exists, err
}

func (r *PostgresAccessRequestRepository) Resolve(ctx context.Context, id, status string, resolvedBy *string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE access_requests
		SET status = $2, resolved_at = NOW(), resolved_by = $3
		WHERE id = $1 AND status = 'pending'`, id, status, resolvedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRequestNotFound
	}
	return nil
}

type PostgresAccessInviteRepository struct{ pool *pgxpool.Pool }

func NewPostgresAccessInviteRepository(pool *pgxpool.Pool) *PostgresAccessInviteRepository {
	return &PostgresAccessInviteRepository{pool: pool}
}

func (r *PostgresAccessInviteRepository) Create(
	ctx context.Context, email, tokenHash, role string, expiresAt time.Time, createdBy, requestID *string,
) (*models.AccessInvite, error) {
	var inv models.AccessInvite
	err := r.pool.QueryRow(ctx, `
		INSERT INTO access_invites (email, token_hash, role, expires_at, created_by, request_id)
		VALUES (lower(trim($1)), $2, $3, $4, $5, $6)
		RETURNING id, email, token_hash, role, expires_at, consumed_at, created_by, request_id, created_at`,
		email, tokenHash, role, expiresAt, createdBy, requestID,
	).Scan(
		&inv.ID, &inv.Email, &inv.TokenHash, &inv.Role, &inv.ExpiresAt,
		&inv.ConsumedAt, &inv.CreatedBy, &inv.RequestID, &inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *PostgresAccessInviteRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.AccessInvite, error) {
	var inv models.AccessInvite
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, token_hash, role, expires_at, consumed_at, created_by, request_id, created_at
		FROM access_invites WHERE token_hash = $1`, tokenHash,
	).Scan(
		&inv.ID, &inv.Email, &inv.TokenHash, &inv.Role, &inv.ExpiresAt,
		&inv.ConsumedAt, &inv.CreatedBy, &inv.RequestID, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInviteNotFound
	}
	return &inv, err
}

func (r *PostgresAccessInviteRepository) Consume(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE access_invites SET consumed_at = NOW()
		WHERE id = $1 AND consumed_at IS NULL AND expires_at > NOW()`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInviteConsumed
	}
	return nil
}

func (r *PostgresAccessInviteRepository) HasOpenForEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM access_invites
			WHERE lower(email) = lower(trim($1))
			  AND consumed_at IS NULL
			  AND expires_at > NOW()
		)`, email).Scan(&exists)
	return exists, err
}

type PostgresOAuthIdentityRepository struct{ pool *pgxpool.Pool }

func NewPostgresOAuthIdentityRepository(pool *pgxpool.Pool) *PostgresOAuthIdentityRepository {
	return &PostgresOAuthIdentityRepository{pool: pool}
}

func (r *PostgresOAuthIdentityRepository) Link(
	ctx context.Context, userID, provider, providerUserID string, providerEmail *string,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO oauth_identities (user_id, provider, provider_user_id, provider_email)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, provider) DO UPDATE
		SET provider_user_id = EXCLUDED.provider_user_id,
		    provider_email = EXCLUDED.provider_email`,
		userID, provider, providerUserID, providerEmail)
	return err
}

func (r *PostgresOAuthIdentityRepository) Unlink(ctx context.Context, userID, provider string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM oauth_identities WHERE user_id = $1 AND provider = $2`, userID, provider)
	return err
}

func (r *PostgresOAuthIdentityRepository) GetByProvider(ctx context.Context, provider, providerUserID string) (*models.OAuthIdentity, error) {
	var o models.OAuthIdentity
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_user_id, provider_email, created_at
		FROM oauth_identities WHERE provider = $1 AND provider_user_id = $2`,
		provider, providerUserID,
	).Scan(&o.ID, &o.UserID, &o.Provider, &o.ProviderUserID, &o.ProviderEmail, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &o, err
}

func (r *PostgresOAuthIdentityRepository) ListByUser(ctx context.Context, userID string) ([]models.OAuthIdentity, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, provider, provider_user_id, provider_email, created_at
		FROM oauth_identities WHERE user_id = $1 ORDER BY provider`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.OAuthIdentity
	for rows.Next() {
		var o models.OAuthIdentity
		if err := rows.Scan(&o.ID, &o.UserID, &o.Provider, &o.ProviderUserID, &o.ProviderEmail, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *PostgresOAuthIdentityRepository) CountAuthMethods(ctx context.Context, userID string) (bool, int, error) {
	var hash string
	err := r.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, ErrUserNotFound
	}
	if err != nil {
		return false, 0, err
	}
	var n int
	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM oauth_identities WHERE user_id = $1`, userID).Scan(&n)
	return hash != "", n, err
}
