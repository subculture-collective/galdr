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

// CustomerInsight stores one generated AI insight for a customer.
type CustomerInsight struct {
	ID          uuid.UUID      `json:"id"`
	OrgID       uuid.UUID      `json:"org_id"`
	CustomerID  uuid.UUID      `json:"customer_id"`
	InsightType string         `json:"insight_type"`
	Content     map[string]any `json:"content"`
	GeneratedAt time.Time      `json:"generated_at"`
	Model       string         `json:"model"`
	TokenCost   float64        `json:"token_cost"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// CustomerInsightRepository handles customer_insights persistence.
type CustomerInsightRepository struct {
	pool *pgxpool.Pool
}

// NewCustomerInsightRepository creates a customer insight repository.
func NewCustomerInsightRepository(pool *pgxpool.Pool) *CustomerInsightRepository {
	return &CustomerInsightRepository{pool: pool}
}

// Create inserts a generated insight.
func (r *CustomerInsightRepository) Create(ctx context.Context, insight *CustomerInsight) error {
	contentJSON, err := json.Marshal(insight.Content)
	if err != nil {
		return fmt.Errorf("marshal insight content: %w", err)
	}
	query := `
		INSERT INTO customer_insights (org_id, customer_id, insight_type, content, generated_at, model, token_cost)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, query,
		insight.OrgID, insight.CustomerID, insight.InsightType, contentJSON,
		insight.GeneratedAt, insight.Model, insight.TokenCost,
	).Scan(&insight.ID, &insight.CreatedAt, &insight.UpdatedAt)
}

// GetRecent returns the newest insight generated since the provided time.
func (r *CustomerInsightRepository) GetRecent(ctx context.Context, orgID, customerID uuid.UUID, insightType string, since time.Time) (*CustomerInsight, error) {
	query := `
		SELECT id, org_id, customer_id, insight_type, content, generated_at, model, token_cost, created_at, updated_at
		FROM customer_insights
		WHERE org_id = $1 AND customer_id = $2 AND insight_type = $3 AND generated_at >= $4
		ORDER BY generated_at DESC
		LIMIT 1`
	return scanCustomerInsight(r.pool.QueryRow(ctx, query, orgID, customerID, insightType, since))
}

// ListByCustomer lists recent insights for a customer.
func (r *CustomerInsightRepository) ListByCustomer(ctx context.Context, orgID, customerID uuid.UUID, limit int) ([]*CustomerInsight, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	query := `
		SELECT id, org_id, customer_id, insight_type, content, generated_at, model, token_cost, created_at, updated_at
		FROM customer_insights
		WHERE org_id = $1 AND customer_id = $2
		ORDER BY generated_at DESC
		LIMIT $3`
	rows, err := r.pool.Query(ctx, query, orgID, customerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list customer insights: %w", err)
	}
	defer rows.Close()

	var insights []*CustomerInsight
	for rows.Next() {
		insight, err := scanCustomerInsightRow(rows)
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}
	return insights, rows.Err()
}

type customerInsightScanner interface {
	Scan(dest ...any) error
}

func scanCustomerInsight(row customerInsightScanner) (*CustomerInsight, error) {
	insight, err := scanCustomerInsightRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return insight, err
}

func scanCustomerInsightRow(row customerInsightScanner) (*CustomerInsight, error) {
	insight := &CustomerInsight{}
	var contentJSON []byte
	if err := row.Scan(
		&insight.ID, &insight.OrgID, &insight.CustomerID, &insight.InsightType,
		&contentJSON, &insight.GeneratedAt, &insight.Model, &insight.TokenCost,
		&insight.CreatedAt, &insight.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(contentJSON, &insight.Content); err != nil {
		return nil, fmt.Errorf("unmarshal insight content: %w", err)
	}
	return insight, nil
}
