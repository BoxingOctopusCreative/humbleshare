package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	CodeTypeSteamKey = "steam_key"
	CodeTypeGiftLink = "gift_link"

	CodeStatusAvailable = "available"
	CodeStatusClaimed   = "claimed"
	CodeStatusDead      = "dead"
)

// Code is a redeemable Steam key or gift link. code_value must only appear in
// responses after a successful claim (or on admin inventory).
type Code struct {
	ID            uuid.UUID  `json:"id"`
	GameID        uuid.UUID  `json:"game_id"`
	GameName      *string    `json:"game_name,omitempty"`
	CodeType      string     `json:"code_type"`
	CodeValue     string     `json:"code_value,omitempty"`
	Source        *string    `json:"source,omitempty"`
	SourceOrderID *string    `json:"source_order_id,omitempty"`
	Status        string     `json:"status"`
	ClaimedBy     *string    `json:"claimed_by,omitempty"`
	ClaimedAt     *time.Time `json:"claimed_at,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ClaimResult is returned from an atomic claim. CodeValue is set only here for public flows.
type ClaimResult struct {
	ID        uuid.UUID `json:"id"`
	GameID    uuid.UUID `json:"game_id"`
	CodeType  string    `json:"code_type"`
	CodeValue string    `json:"code_value"`
	ClaimedBy string    `json:"claimed_by"`
	ClaimedAt time.Time `json:"claimed_at"`
}
