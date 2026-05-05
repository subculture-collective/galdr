package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FeatureOverride represents an org-specific feature or limit override.
type FeatureOverride struct {
	OrgID         uuid.UUID
	FeatureName   string
	Enabled       *bool
	LimitOverride *int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// FeatureOverrideRepository handles feature_overrides database operations.
type FeatureOverrideRepository struct {
	pool *pgxpool.Pool
}

func NewFeatureOverrideRepository(pool *pgxpool.Pool) *FeatureOverrideRepository {
	return &FeatureOverrideRepository{pool: pool}
}

func (r *FeatureOverrideRepository) GetByOrgAndFeature(ctx context.Context, orgID uuid.UUID, featureName string) (*FeatureOverride, error) {
	query := `
		SELECT org_id, feature_name, enabled, limit_override, created_at, updated_at
		FROM feature_overrides
		WHERE org_id = $1 AND feature_name = $2`

	override := &FeatureOverride{}
	err := r.pool.QueryRow(ctx, query, orgID, normalizeFeatureOverrideName(featureName)).Scan(
		&override.OrgID,
		&override.FeatureName,
		&override.Enabled,
		&override.LimitOverride,
		&override.CreatedAt,
		&override.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get feature override: %w", err)
	}
	return override, nil
}

func (r *FeatureOverrideRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]FeatureOverride, error) {
	query := `
		SELECT org_id, feature_name, enabled, limit_override, created_at, updated_at
		FROM feature_overrides
		WHERE org_id = $1
		ORDER BY feature_name`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list feature overrides: %w", err)
	}
	defer rows.Close()

	var overrides []FeatureOverride
	for rows.Next() {
		var override FeatureOverride
		if err := rows.Scan(
			&override.OrgID,
			&override.FeatureName,
			&override.Enabled,
			&override.LimitOverride,
			&override.CreatedAt,
			&override.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan feature override: %w", err)
		}
		overrides = append(overrides, override)
	}
	return overrides, rows.Err()
}

func normalizeFeatureOverrideName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
