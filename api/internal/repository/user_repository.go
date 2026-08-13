package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserEmailTaken  = errors.New("email already registered")
	ErrSessionNotFound = errors.New("session not found")
	ErrInviteNotFound  = errors.New("invite not found")
	ErrInviteConsumed  = errors.New("invite already used")
	ErrInviteExpired   = errors.New("invite expired")
	ErrRequestNotFound = errors.New("access request not found")
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash, displayName, role string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	List(ctx context.Context) ([]models.User, error)
	ListAdminEmails(ctx context.Context) ([]string, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	UpdateProfile(ctx context.Context, id, displayName, bio, steamGamertag, discordTag string) error
	UpdateAvatarURL(ctx context.Context, id, avatarURL string) error
	UpdateRole(ctx context.Context, id, role string) error
	SetTwoFactorSecret(ctx context.Context, id, secret string) error
	EnableTwoFactor(ctx context.Context, id string) error
	DisableTwoFactor(ctx context.Context, id string) error
	GetTwoFactorSecret(ctx context.Context, id string) (string, error)
}

type SessionRepository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*models.Session, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteExpired(ctx context.Context) error
}

type AccessRequestRepository interface {
	Create(ctx context.Context, email, note string) (*models.AccessRequest, error)
	GetByID(ctx context.Context, id string) (*models.AccessRequest, error)
	ListPending(ctx context.Context) ([]models.AccessRequest, error)
	List(ctx context.Context) ([]models.AccessRequest, error)
	HasPending(ctx context.Context, email string) (bool, error)
	Resolve(ctx context.Context, id, status string, resolvedBy *string) error
}

type AccessInviteRepository interface {
	Create(ctx context.Context, email, tokenHash, role string, expiresAt time.Time, createdBy, requestID *string) (*models.AccessInvite, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.AccessInvite, error)
	Consume(ctx context.Context, id string) error
	HasOpenForEmail(ctx context.Context, email string) (bool, error)
}

type OAuthIdentityRepository interface {
	Link(ctx context.Context, userID, provider, providerUserID string, providerEmail *string) error
	Unlink(ctx context.Context, userID, provider string) error
	GetByProvider(ctx context.Context, provider, providerUserID string) (*models.OAuthIdentity, error)
	ListByUser(ctx context.Context, userID string) ([]models.OAuthIdentity, error)
	CountAuthMethods(ctx context.Context, userID string) (password bool, oauthCount int, err error)
}

type PostgresUserRepository struct{ pool *pgxpool.Pool }

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
		&u.Bio, &u.SteamGamertag, &u.DiscordTag, &u.AvatarURL,
		&u.Role, &u.TwoFactorEnabled, &u.TwoFactorSecret, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

const userCols = `id, email, password_hash, display_name, bio, steam_gamertag, discord_tag, avatar_url, role, two_factor_enabled, two_factor_secret, created_at, updated_at`

func (r *PostgresUserRepository) Create(ctx context.Context, email, passwordHash, displayName, role string) (*models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING `+userCols, email, passwordHash, displayName, role)
	u, err := scanUser(row)
	if err != nil && strings.Contains(err.Error(), "idx_users_email_lower") {
		return nil, ErrUserEmailTaken
	}
	return u, err
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE lower(email) = $1`, email))
}

func (r *PostgresUserRepository) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+userCols+` FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
			&u.Bio, &u.SteamGamertag, &u.DiscordTag, &u.AvatarURL,
			&u.Role, &u.TwoFactorEnabled, &u.TwoFactorSecret, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PostgresUserRepository) ListAdminEmails(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT email FROM users WHERE role IN ('admin', 'superadmin') ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresUserRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, id, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) UpdateProfile(ctx context.Context, id, displayName, bio, steamGamertag, discordTag string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET
			display_name = $2,
			bio = $3,
			steam_gamertag = $4,
			discord_tag = $5,
			updated_at = NOW()
		WHERE id = $1`, id, displayName, bio, steamGamertag, discordTag)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) UpdateAvatarURL(ctx context.Context, id, avatarURL string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET avatar_url = $2, updated_at = NOW() WHERE id = $1`, id, avatarURL)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) UpdateRole(ctx context.Context, id, role string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET role = $2, updated_at = NOW() WHERE id = $1`, id, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) SetTwoFactorSecret(ctx context.Context, id, secret string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET two_factor_secret = $2, two_factor_enabled = FALSE, updated_at = NOW() WHERE id = $1`, id, secret)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) EnableTwoFactor(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET two_factor_enabled = TRUE, updated_at = NOW()
		WHERE id = $1 AND two_factor_secret IS NOT NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) DisableTwoFactor(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET two_factor_enabled = FALSE, two_factor_secret = NULL, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) GetTwoFactorSecret(ctx context.Context, id string) (string, error) {
	var secret *string
	err := r.pool.QueryRow(ctx, `SELECT two_factor_secret FROM users WHERE id = $1`, id).Scan(&secret)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", err
	}
	if secret == nil || *secret == "" {
		return "", errors.New("two-factor secret not set")
	}
	return *secret, nil
}
