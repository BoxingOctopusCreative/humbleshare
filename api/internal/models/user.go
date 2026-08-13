package models

import "time"

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
	Role             string    `json:"role"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
	TwoFactorSecret  *string   `json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
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
