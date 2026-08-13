package repository

import (
	"context"

	"github.com/boxingoctopus/humbleshare/api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StatsRepository reads aggregated stats.
type StatsRepository interface {
	GetStats(ctx context.Context) (*models.Stats, error)
}

type PostgresStatsRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresStatsRepository(pool *pgxpool.Pool) *PostgresStatsRepository {
	return &PostgresStatsRepository{pool: pool}
}

func (r *PostgresStatsRepository) GetStats(ctx context.Context) (*models.Stats, error) {
	s := &models.Stats{}

	// Counts by status + total games.
	err := r.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM games) AS total_games,
			COUNT(*) AS total_codes,
			COUNT(*) FILTER (WHERE status = 'available') AS available_codes,
			COUNT(*) FILTER (WHERE status = 'claimed')   AS claimed_codes,
			COUNT(*) FILTER (WHERE status = 'dead')      AS dead_codes
		FROM codes
	`).Scan(&s.TotalGames, &s.TotalCodes, &s.AvailableCodes, &s.ClaimedCodes, &s.DeadCodes)
	if err != nil {
		return nil, err
	}

	// Claims per day for the last 30 days.
	rows, err := r.pool.Query(ctx, `
		SELECT TO_CHAR(claimed_at::date, 'YYYY-MM-DD') AS day, COUNT(*) AS cnt
		FROM codes
		WHERE status = 'claimed'
		  AND claimed_at >= NOW() - INTERVAL '30 days'
		GROUP BY day
		ORDER BY day ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d models.DailyClaim
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, err
		}
		s.ClaimsLast30Days = append(s.ClaimsLast30Days, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Top 10 claimers.
	crows, err := r.pool.Query(ctx, `
		SELECT claimed_by, COUNT(*) AS cnt
		FROM codes
		WHERE status = 'claimed' AND claimed_by IS NOT NULL
		GROUP BY claimed_by
		ORDER BY cnt DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer crows.Close()
	for crows.Next() {
		var cs models.ClaimerStat
		if err := crows.Scan(&cs.ClaimedBy, &cs.Count); err != nil {
			return nil, err
		}
		s.TopClaimers = append(s.TopClaimers, cs)
	}
	if err := crows.Err(); err != nil {
		return nil, err
	}

	// Top 10 games by claimed codes.
	grows, err := r.pool.Query(ctx, `
		SELECT g.id, g.name, COUNT(c.id) AS cnt
		FROM codes c
		JOIN games g ON g.id = c.game_id
		WHERE c.status = 'claimed'
		GROUP BY g.id, g.name
		ORDER BY cnt DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer grows.Close()
	for grows.Next() {
		var gs models.GameStat
		if err := grows.Scan(&gs.GameID, &gs.Name, &gs.Count); err != nil {
			return nil, err
		}
		s.TopGames = append(s.TopGames, gs)
	}
	if err := grows.Err(); err != nil {
		return nil, err
	}

	return s, nil
}
