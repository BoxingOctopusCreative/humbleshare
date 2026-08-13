package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleMember     = "member"
	RoleAdmin      = "admin"
	RoleSuperadmin = "superadmin"
)

type User struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	PasswordHash     string    `json:"-"`
	DisplayName      string    `json:"display_name"`
	Bio              string    `json:"bio"`
	SteamGamertag    string    `json:"steam_gamertag"`
	DiscordTag       string    `json:"discord_tag"`
	AvatarURL        string    `json:"avatar_url"`
	BannerURL        string    `json:"banner_url"`
	ProfileSlug      string    `json:"profile_slug"`
	ProfilePublic    bool      `json:"profile_public"`
	Role             string    `json:"role"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	TwoFactorSecret  *string   `json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// PublicProfile is the safe view of a user for /u/[slug]. No email, role, or keys.
type PublicProfile struct {
	ProfileSlug   string        `json:"profile_slug"`
	DisplayName   string        `json:"display_name"`
	Bio           string        `json:"bio"`
	SteamGamertag string        `json:"steam_gamertag"`
	DiscordTag    string        `json:"discord_tag"`
	AvatarURL     string        `json:"avatar_url"`
	BannerURL     string        `json:"banner_url"`
	ProfilePublic bool          `json:"profile_public"`
	RecentClaims  []PublicClaim `json:"recent_claims"`
	IsOwner       bool          `json:"is_owner"`
}

// PublicClaim is a claimed game shown on a profile. Never includes code_value.
type PublicClaim struct {
	GameID         uuid.UUID `json:"game_id"`
	GameName       string    `json:"game_name"`
	HeaderImageURL *string   `json:"header_image_url,omitempty"`
	CodeType       string    `json:"code_type"`
	ClaimedAt      time.Time `json:"claimed_at"`
}

func (u *User) IsAdmin() bool {
	return u != nil && (u.Role == RoleAdmin || u.Role == RoleSuperadmin)
}

func (u *User) IsSuperadmin() bool {
	return u != nil && u.Role == RoleSuperadmin
}

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AccessRequest struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	Note       string     `json:"note"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy *string    `json:"resolved_by,omitempty"`
}

type AccessInvite struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	TokenHash  string     `json:"-"`
	Role       string     `json:"role"`
	ExpiresAt  time.Time  `json:"expires_at"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`
	CreatedBy  *string    `json:"created_by,omitempty"`
	RequestID  *string    `json:"request_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type OAuthIdentity struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	ProviderEmail  *string   `json:"provider_email,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
