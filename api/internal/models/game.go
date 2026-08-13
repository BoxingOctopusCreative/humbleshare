package models

import (
	"time"

	"github.com/google/uuid"
)

// Game is a title that one or more codes unlock. Identity comes from Humble import
// or a manually typed name — never from the opaque Steam key itself.
type Game struct {
	ID                 uuid.UUID  `json:"id"`
	SteamAppID         *int       `json:"steam_appid"`
	Name               string     `json:"name"`
	HeaderImageURL     *string    `json:"header_image_url,omitempty"`
	CapsuleImageURL    *string    `json:"capsule_image_url,omitempty"`
	ShortDescription   *string    `json:"short_description,omitempty"`
	StoreURL           *string    `json:"store_url,omitempty"`
	ScreenshotURLs     []string   `json:"screenshot_urls,omitempty"`
	MetadataFetchedAt  *time.Time `json:"metadata_fetched_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	AvailableCount     int        `json:"available_count,omitempty"`
}
