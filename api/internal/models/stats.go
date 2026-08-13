package models

import "github.com/google/uuid"

// Stats holds aggregated usage data returned by GET /admin/stats.
type Stats struct {
	TotalGames       int           `json:"total_games"`
	TotalCodes       int           `json:"total_codes"`
	AvailableCodes   int           `json:"available_codes"`
	ClaimedCodes     int           `json:"claimed_codes"`
	DeadCodes        int           `json:"dead_codes"`
	ClaimsLast30Days []DailyClaim  `json:"claims_last_30_days"`
	TopClaimers      []ClaimerStat `json:"top_claimers"`
	TopGames         []GameStat    `json:"top_games"`
}

type DailyClaim struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type ClaimerStat struct {
	ClaimedBy string `json:"claimed_by"`
	Count     int    `json:"count"`
}

type GameStat struct {
	GameID uuid.UUID `json:"game_id"`
	Name   string    `json:"name"`
	Count  int       `json:"count"`
}
