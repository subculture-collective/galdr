package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresLLMUsageStore struct {
	pool *pgxpool.Pool
}

func NewPostgresLLMUsageStore(pool *pgxpool.Pool) *PostgresLLMUsageStore {
	return &PostgresLLMUsageStore{pool: pool}
}

func (s *PostgresLLMUsageStore) TrackLLMUsage(ctx context.Context, usage LLMUsage) error {
	createdAt := usage.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	requestType := normalizeLLMRequestType(usage.RequestType)
	model := usage.Model
	if model == "" {
		model = usage.Provider
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO llm_usage (org_id, request_type, model, input_tokens, output_tokens, cost_usd, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		usage.OrgID, requestType, model, usage.InputTokens, usage.OutputTokens, usage.CostUSD, createdAt,
	)
	if err != nil {
		return fmt.Errorf("insert llm usage: %w", err)
	}
	return nil
}

func (s *PostgresLLMUsageStore) SumLLMUsageCost(ctx context.Context, orgID uuid.UUID, start, end time.Time) (float64, error) {
	var total float64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM llm_usage
		WHERE org_id = $1 AND created_at >= $2 AND created_at < $3`, orgID, start, end).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum llm usage cost: %w", err)
	}
	return total, nil
}

func (s *PostgresLLMUsageStore) CountLLMUsageRequests(ctx context.Context, orgID uuid.UUID, start, end time.Time) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM llm_usage
		WHERE org_id = $1 AND created_at >= $2 AND created_at < $3`, orgID, start, end).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count llm usage requests: %w", err)
	}
	return count, nil
}

func (s *PostgresLLMUsageStore) SumLLMUsageTokens(ctx context.Context, orgID uuid.UUID, start, end time.Time) (int, int, error) {
	var inputTokens, outputTokens int64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0)
		FROM llm_usage
		WHERE org_id = $1 AND created_at >= $2 AND created_at < $3`, orgID, start, end).Scan(&inputTokens, &outputTokens)
	if err != nil {
		return 0, 0, fmt.Errorf("sum llm usage tokens: %w", err)
	}
	return int(inputTokens), int(outputTokens), nil
}
