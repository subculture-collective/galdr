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

// CustomerFeature stores the current churn-model feature vector for a customer.
type CustomerFeature struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	CustomerID   uuid.UUID
	Features     map[string]float64
	CalculatedAt time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CustomerFeatureRepository handles customer_features database operations.
type CustomerFeatureRepository struct {
	pool *pgxpool.Pool
}

// NewCustomerFeatureRepository creates a new CustomerFeatureRepository.
func NewCustomerFeatureRepository(pool *pgxpool.Pool) *CustomerFeatureRepository {
	return &CustomerFeatureRepository{pool: pool}
}

// Upsert stores the latest feature vector for a customer.
func (r *CustomerFeatureRepository) Upsert(ctx context.Context, feature *CustomerFeature) error {
	featuresJSON, err := json.Marshal(feature.Features)
	if err != nil {
		return fmt.Errorf("marshal customer features: %w", err)
	}

	query := `
		INSERT INTO customer_features (org_id, customer_id, features, calculated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (customer_id) DO UPDATE SET
			features = EXCLUDED.features,
			calculated_at = EXCLUDED.calculated_at
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		feature.OrgID, feature.CustomerID, featuresJSON, feature.CalculatedAt,
	).Scan(&feature.ID, &feature.CreatedAt, &feature.UpdatedAt)
}

// GetByCustomerID retrieves the current feature vector for a customer.
func (r *CustomerFeatureRepository) GetByCustomerID(ctx context.Context, customerID, orgID uuid.UUID) (*CustomerFeature, error) {
	query := `
		SELECT id, org_id, customer_id, features, calculated_at, created_at, updated_at
		FROM customer_features
		WHERE customer_id = $1 AND org_id = $2`

	feature := &CustomerFeature{}
	var featuresJSON []byte
	err := r.pool.QueryRow(ctx, query, customerID, orgID).Scan(
		&feature.ID, &feature.OrgID, &feature.CustomerID, &featuresJSON,
		&feature.CalculatedAt, &feature.CreatedAt, &feature.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer features: %w", err)
	}
	if err := json.Unmarshal(featuresJSON, &feature.Features); err != nil {
		return nil, fmt.Errorf("unmarshal customer features: %w", err)
	}
	return feature, nil
}
