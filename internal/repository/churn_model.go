package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChurnModelVersion stores a trained churn model artifact and its quality metrics.
type ChurnModelVersion struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	Version      string
	FeatureNames []string
	Weights      []float64
	Bias         float64
	Cutoff       float64
	Metrics      map[string]float64
	TrainedAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ChurnModelRepository handles churn_models database operations.
type ChurnModelRepository struct {
	pool *pgxpool.Pool
}

// NewChurnModelRepository creates a new ChurnModelRepository.
func NewChurnModelRepository(pool *pgxpool.Pool) *ChurnModelRepository {
	return &ChurnModelRepository{pool: pool}
}

// Save stores or updates a trained model version for an organization.
func (r *ChurnModelRepository) Save(ctx context.Context, model *ChurnModelVersion) error {
	if model == nil {
		return fmt.Errorf("churn model version is required")
	}
	featureNamesJSON, err := json.Marshal(model.FeatureNames)
	if err != nil {
		return fmt.Errorf("marshal churn model feature names: %w", err)
	}
	weightsJSON, err := json.Marshal(model.Weights)
	if err != nil {
		return fmt.Errorf("marshal churn model weights: %w", err)
	}
	metricsJSON, err := json.Marshal(model.Metrics)
	if err != nil {
		return fmt.Errorf("marshal churn model metrics: %w", err)
	}

	query := `
		INSERT INTO churn_models (org_id, version, feature_names, weights, bias, cutoff, metrics, trained_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (org_id, version) DO UPDATE SET
			feature_names = EXCLUDED.feature_names,
			weights = EXCLUDED.weights,
			bias = EXCLUDED.bias,
			cutoff = EXCLUDED.cutoff,
			metrics = EXCLUDED.metrics,
			trained_at = EXCLUDED.trained_at
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		model.OrgID, model.Version, featureNamesJSON, weightsJSON, model.Bias,
		model.Cutoff, metricsJSON, model.TrainedAt,
	).Scan(&model.ID, &model.CreatedAt, &model.UpdatedAt)
}

// GetLatestByOrg returns the newest trained model for an organization.
func (r *ChurnModelRepository) GetLatestByOrg(ctx context.Context, orgID uuid.UUID) (*ChurnModelVersion, error) {
	query := `
		SELECT id, org_id, version, feature_names, weights, bias, cutoff, metrics, trained_at, created_at, updated_at
		FROM churn_models
		WHERE org_id = $1
		ORDER BY trained_at DESC
		LIMIT 1`

	return r.scanModel(ctx, query, orgID)
}

// GetByVersion returns a specific trained model version for an organization.
func (r *ChurnModelRepository) GetByVersion(ctx context.Context, orgID uuid.UUID, version string) (*ChurnModelVersion, error) {
	query := `
		SELECT id, org_id, version, feature_names, weights, bias, cutoff, metrics, trained_at, created_at, updated_at
		FROM churn_models
		WHERE org_id = $1 AND version = $2`

	return r.scanModel(ctx, query, orgID, version)
}

func (r *ChurnModelRepository) scanModel(ctx context.Context, query string, args ...any) (*ChurnModelVersion, error) {
	model := &ChurnModelVersion{}
	var featureNamesJSON, weightsJSON, metricsJSON []byte
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&model.ID, &model.OrgID, &model.Version, &featureNamesJSON, &weightsJSON,
		&model.Bias, &model.Cutoff, &metricsJSON, &model.TrainedAt,
		&model.CreatedAt, &model.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get churn model: %w", err)
	}
	if err := json.Unmarshal(featureNamesJSON, &model.FeatureNames); err != nil {
		return nil, fmt.Errorf("unmarshal churn model feature names: %w", err)
	}
	if err := json.Unmarshal(weightsJSON, &model.Weights); err != nil {
		return nil, fmt.Errorf("unmarshal churn model weights: %w", err)
	}
	if err := json.Unmarshal(metricsJSON, &model.Metrics); err != nil {
		return nil, fmt.Errorf("unmarshal churn model metrics: %w", err)
	}
	return model, nil
}
