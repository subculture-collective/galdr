package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageSnapshotRecord represents one metered usage value for one org/day.
type UsageSnapshotRecord struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	Metric       string    `json:"metric"`
	Value        int       `json:"value"`
	RecordedAt   time.Time `json:"recorded_at"`
	RecordedDate time.Time `json:"recorded_date"`
}

// UsageSnapshotRepository persists usage analytics snapshots.
type UsageSnapshotRepository struct {
	pool *pgxpool.Pool
}

func NewUsageSnapshotRepository(pool *pgxpool.Pool) *UsageSnapshotRepository {
	return &UsageSnapshotRepository{pool: pool}
}

func (r *UsageSnapshotRepository) Record(ctx context.Context, record UsageSnapshotRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO usage_snapshots (org_id, metric, value, recorded_at, recorded_date)
		VALUES ($1, $2, $3, $4, $4::date)
		ON CONFLICT (org_id, metric, recorded_date)
		DO UPDATE SET value = EXCLUDED.value, recorded_at = EXCLUDED.recorded_at
	`, record.OrgID, record.Metric, record.Value, record.RecordedAt)
	if err != nil {
		return fmt.Errorf("record usage snapshot: %w", err)
	}
	return nil
}

func (r *UsageSnapshotRepository) Increment(ctx context.Context, orgID uuid.UUID, metric string, recordedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO usage_snapshots (org_id, metric, value, recorded_at, recorded_date)
		VALUES ($1, $2, 1, $3, $3::date)
		ON CONFLICT (org_id, metric, recorded_date)
		DO UPDATE SET value = usage_snapshots.value + 1, recorded_at = EXCLUDED.recorded_at
	`, orgID, metric, recordedAt)
	if err != nil {
		return fmt.Errorf("increment usage snapshot: %w", err)
	}
	return nil
}

func (r *UsageSnapshotRepository) CurrentValue(ctx context.Context, orgID uuid.UUID, metric string, recordedAt time.Time) (int, error) {
	var value int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT value
			FROM usage_snapshots
			WHERE org_id = $1 AND metric = $2 AND recorded_date = $3::date
		), 0)
	`, orgID, metric, recordedAt).Scan(&value)
	if err != nil {
		return 0, fmt.Errorf("get current usage value: %w", err)
	}
	return value, nil
}
