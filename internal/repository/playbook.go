package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Trigger types for playbooks.
const (
	PlaybookTriggerScoreThreshold = "score_threshold"
	PlaybookTriggerCustomerEvent  = "customer_event"
	PlaybookTriggerSchedule       = "schedule"
)

// Action types for playbook actions.
const (
	PlaybookActionSendEmail      = "send_email"
	PlaybookActionInternalAlert  = "internal_alert"
	PlaybookActionTagCustomer    = "tag_customer"
	PlaybookActionWebhook        = "webhook"
)

// Execution statuses for playbook executions.
const (
	PlaybookExecutionPending = "pending"
	PlaybookExecutionRunning = "running"
	PlaybookExecutionSuccess = "success"
	PlaybookExecutionFailed  = "failed"
)

// Playbook represents a playbooks row.
type Playbook struct {
	ID            uuid.UUID      `json:"id"`
	OrgID         uuid.UUID      `json:"org_id"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Enabled       bool           `json:"enabled"`
	TriggerType   string         `json:"trigger_type"`
	TriggerConfig map[string]any `json:"trigger_config"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// PlaybookAction represents a playbook_actions row.
type PlaybookAction struct {
	ID           uuid.UUID      `json:"id"`
	PlaybookID   uuid.UUID      `json:"playbook_id"`
	ActionType   string         `json:"action_type"`
	ActionConfig map[string]any `json:"action_config"`
	OrderIndex   int            `json:"order_index"`
}

// PlaybookExecution represents a playbook_executions row.
type PlaybookExecution struct {
	ID          uuid.UUID      `json:"id"`
	PlaybookID  uuid.UUID      `json:"playbook_id"`
	CustomerID  *uuid.UUID     `json:"customer_id,omitempty"`
	TriggeredAt time.Time      `json:"triggered_at"`
	Status      string         `json:"status"`
	Result      map[string]any `json:"result"`
}

// PlaybookRepository handles playbook database operations.
type PlaybookRepository struct {
	pool *pgxpool.Pool
}

// NewPlaybookRepository creates a new PlaybookRepository.
func NewPlaybookRepository(pool *pgxpool.Pool) *PlaybookRepository {
	return &PlaybookRepository{pool: pool}
}

// CountByOrg returns the number of playbooks configured for an organization.
func (r *PlaybookRepository) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM playbooks WHERE org_id = $1`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count playbooks: %w", err)
	}
	return count, nil
}

// List returns all playbooks for an organization.
func (r *PlaybookRepository) List(ctx context.Context, orgID uuid.UUID) ([]*Playbook, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, name, description, enabled, trigger_type, trigger_config, created_at, updated_at
		FROM playbooks
		WHERE org_id = $1
		ORDER BY created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("query playbooks: %w", err)
	}
	defer rows.Close()

	var playbooks []*Playbook
	for rows.Next() {
		p := &Playbook{}
		if err := rows.Scan(
			&p.ID, &p.OrgID, &p.Name, &p.Description,
			&p.Enabled, &p.TriggerType, &p.TriggerConfig,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan playbook: %w", err)
		}
		playbooks = append(playbooks, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playbooks: %w", err)
	}
	return playbooks, nil
}

// GetByID returns a single playbook by ID scoped to an org.
func (r *PlaybookRepository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*Playbook, error) {
	p := &Playbook{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, name, description, enabled, trigger_type, trigger_config, created_at, updated_at
		FROM playbooks
		WHERE id = $1 AND org_id = $2
	`, id, orgID).Scan(
		&p.ID, &p.OrgID, &p.Name, &p.Description,
		&p.Enabled, &p.TriggerType, &p.TriggerConfig,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query playbook: %w", err)
	}
	return p, nil
}

// Create inserts a new playbook.
func (r *PlaybookRepository) Create(ctx context.Context, p *Playbook) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO playbooks (org_id, name, description, enabled, trigger_type, trigger_config)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, p.OrgID, p.Name, p.Description, p.Enabled, p.TriggerType, p.TriggerConfig,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

// Update updates an existing playbook.
func (r *PlaybookRepository) Update(ctx context.Context, p *Playbook) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE playbooks
		SET name = $1, description = $2, enabled = $3, trigger_type = $4, trigger_config = $5, updated_at = NOW()
		WHERE id = $6 AND org_id = $7
	`, p.Name, p.Description, p.Enabled, p.TriggerType, p.TriggerConfig, p.ID, p.OrgID)
	if err != nil {
		return fmt.Errorf("update playbook: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Delete deletes a playbook and its associated actions/executions (via CASCADE).
func (r *PlaybookRepository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `
		DELETE FROM playbooks WHERE id = $1 AND org_id = $2
	`, id, orgID)
	if err != nil {
		return fmt.Errorf("delete playbook: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// PlaybookActionRepository handles playbook_actions database operations.
type PlaybookActionRepository struct {
	pool *pgxpool.Pool
}

// NewPlaybookActionRepository creates a new PlaybookActionRepository.
func NewPlaybookActionRepository(pool *pgxpool.Pool) *PlaybookActionRepository {
	return &PlaybookActionRepository{pool: pool}
}

// ListByPlaybook returns all actions for a playbook ordered by order_index.
func (r *PlaybookActionRepository) ListByPlaybook(ctx context.Context, playbookID uuid.UUID) ([]*PlaybookAction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, playbook_id, action_type, action_config, order_index
		FROM playbook_actions
		WHERE playbook_id = $1
		ORDER BY order_index ASC
	`, playbookID)
	if err != nil {
		return nil, fmt.Errorf("query playbook actions: %w", err)
	}
	defer rows.Close()

	var actions []*PlaybookAction
	for rows.Next() {
		a := &PlaybookAction{}
		if err := rows.Scan(&a.ID, &a.PlaybookID, &a.ActionType, &a.ActionConfig, &a.OrderIndex); err != nil {
			return nil, fmt.Errorf("scan playbook action: %w", err)
		}
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playbook actions: %w", err)
	}
	return actions, nil
}

// Create inserts a new playbook action.
func (r *PlaybookActionRepository) Create(ctx context.Context, a *PlaybookAction) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO playbook_actions (playbook_id, action_type, action_config, order_index)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, a.PlaybookID, a.ActionType, a.ActionConfig, a.OrderIndex,
	).Scan(&a.ID)
}

// DeleteByPlaybook deletes all actions for a playbook.
func (r *PlaybookActionRepository) DeleteByPlaybook(ctx context.Context, playbookID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM playbook_actions WHERE playbook_id = $1`, playbookID)
	if err != nil {
		return fmt.Errorf("delete playbook actions: %w", err)
	}
	return nil
}

// PlaybookExecutionRepository handles playbook_executions database operations.
type PlaybookExecutionRepository struct {
	pool *pgxpool.Pool
}

// NewPlaybookExecutionRepository creates a new PlaybookExecutionRepository.
func NewPlaybookExecutionRepository(pool *pgxpool.Pool) *PlaybookExecutionRepository {
	return &PlaybookExecutionRepository{pool: pool}
}

// ListByPlaybook returns execution history for a playbook, most recent first.
func (r *PlaybookExecutionRepository) ListByPlaybook(ctx context.Context, playbookID uuid.UUID) ([]*PlaybookExecution, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, playbook_id, customer_id, triggered_at, status, result
		FROM playbook_executions
		WHERE playbook_id = $1
		ORDER BY triggered_at DESC
	`, playbookID)
	if err != nil {
		return nil, fmt.Errorf("query playbook executions: %w", err)
	}
	defer rows.Close()

	var executions []*PlaybookExecution
	for rows.Next() {
		e := &PlaybookExecution{}
		if err := rows.Scan(&e.ID, &e.PlaybookID, &e.CustomerID, &e.TriggeredAt, &e.Status, &e.Result); err != nil {
			return nil, fmt.Errorf("scan playbook execution: %w", err)
		}
		executions = append(executions, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playbook executions: %w", err)
	}
	return executions, nil
}

// Create inserts a new playbook execution record.
func (r *PlaybookExecutionRepository) Create(ctx context.Context, e *PlaybookExecution) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO playbook_executions (playbook_id, customer_id, status, result)
		VALUES ($1, $2, $3, $4)
		RETURNING id, triggered_at
	`, e.PlaybookID, e.CustomerID, e.Status, e.Result,
	).Scan(&e.ID, &e.TriggeredAt)
}

// UpdateStatus updates the status and result of an execution.
func (r *PlaybookExecutionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, result map[string]any) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE playbook_executions SET status = $1, result = $2 WHERE id = $3
	`, status, result, id)
	if err != nil {
		return fmt.Errorf("update playbook execution status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
