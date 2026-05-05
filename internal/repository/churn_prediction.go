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

// ChurnRiskFactor describes one feature contributing to churn risk.
type ChurnRiskFactor struct {
	Feature      string  `json:"feature"`
	Contribution float64 `json:"contribution"`
}

// ChurnPrediction stores the current churn-risk prediction for a customer.
type ChurnPrediction struct {
	ID           uuid.UUID         `json:"id"`
	OrgID        uuid.UUID         `json:"org_id"`
	CustomerID   uuid.UUID         `json:"customer_id"`
	Probability  float64           `json:"probability"`
	Confidence   float64           `json:"confidence"`
	RiskFactors  []ChurnRiskFactor `json:"risk_factors"`
	ModelVersion string            `json:"model_version"`
	PredictedAt  time.Time         `json:"predicted_at"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ChurnPredictionRepository handles churn_predictions database operations.
type ChurnPredictionRepository struct {
	pool *pgxpool.Pool
}

// NewChurnPredictionRepository creates a new ChurnPredictionRepository.
func NewChurnPredictionRepository(pool *pgxpool.Pool) *ChurnPredictionRepository {
	return &ChurnPredictionRepository{pool: pool}
}

// Upsert stores the latest churn prediction for a customer.
func (r *ChurnPredictionRepository) Upsert(ctx context.Context, prediction *ChurnPrediction) error {
	riskFactorsJSON, err := json.Marshal(prediction.RiskFactors)
	if err != nil {
		return fmt.Errorf("marshal churn risk factors: %w", err)
	}

	query := `
		INSERT INTO churn_predictions (org_id, customer_id, probability, confidence, risk_factors, model_version, predicted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (customer_id) DO UPDATE SET
			probability = EXCLUDED.probability,
			confidence = EXCLUDED.confidence,
			risk_factors = EXCLUDED.risk_factors,
			model_version = EXCLUDED.model_version,
			predicted_at = EXCLUDED.predicted_at
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		prediction.OrgID,
		prediction.CustomerID,
		prediction.Probability,
		prediction.Confidence,
		riskFactorsJSON,
		prediction.ModelVersion,
		prediction.PredictedAt,
	).Scan(&prediction.ID, &prediction.CreatedAt, &prediction.UpdatedAt)
}

// GetByCustomerID retrieves the current churn prediction for a customer.
func (r *ChurnPredictionRepository) GetByCustomerID(ctx context.Context, customerID, orgID uuid.UUID) (*ChurnPrediction, error) {
	query := `
		SELECT id, org_id, customer_id, probability, confidence, risk_factors, model_version, predicted_at, created_at, updated_at
		FROM churn_predictions
		WHERE customer_id = $1 AND org_id = $2`

	prediction := &ChurnPrediction{}
	var riskFactorsJSON []byte
	err := r.pool.QueryRow(ctx, query, customerID, orgID).Scan(
		&prediction.ID,
		&prediction.OrgID,
		&prediction.CustomerID,
		&prediction.Probability,
		&prediction.Confidence,
		&riskFactorsJSON,
		&prediction.ModelVersion,
		&prediction.PredictedAt,
		&prediction.CreatedAt,
		&prediction.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get churn prediction: %w", err)
	}
	if err := json.Unmarshal(riskFactorsJSON, &prediction.RiskFactors); err != nil {
		return nil, fmt.Errorf("unmarshal churn risk factors: %w", err)
	}
	return prediction, nil
}
