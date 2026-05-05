package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresLLMBudgetNotifier struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPostgresLLMBudgetNotifier(pool *pgxpool.Pool) *PostgresLLMBudgetNotifier {
	return &PostgresLLMBudgetNotifier{pool: pool, now: time.Now}
}

func (n *PostgresLLMBudgetNotifier) NotifyLLMBudgetWarning(ctx context.Context, warning LLMBudgetWarning) error {
	now := n.now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
	_, err := n.pool.Exec(ctx, `
		INSERT INTO notifications (user_id, org_id, type, title, message, data)
		SELECT uo.user_id, uo.org_id, 'llm_budget_warning', 'AI budget warning',
		       'Your organization has used at least 80% of its monthly AI budget.',
		       jsonb_build_object('period_start', $2::text, 'tier', $3::text, 'budget_usd', $4::numeric, 'monthly_cost_usd', $5::numeric, 'percent_used', $6::numeric)
		FROM user_organizations uo
		WHERE uo.org_id = $1
		  AND uo.role IN ('owner', 'admin')
		  AND NOT EXISTS (
		      SELECT 1 FROM notifications n
		      WHERE n.user_id = uo.user_id
		        AND n.org_id = uo.org_id
		        AND n.type = 'llm_budget_warning'
		        AND n.data->>'period_start' = $2::text
		  )`, warning.OrgID, periodStart, warning.Tier, warning.BudgetUSD, warning.MonthlyCostUSD, warning.PercentUsed)
	if err != nil {
		return fmt.Errorf("create llm budget warning notifications: %w", err)
	}
	return nil
}
