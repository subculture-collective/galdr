package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GenericWebhookConfig stores one org-owned generic webhook receiver mapping.
type GenericWebhookConfig struct {
	ID           uuid.UUID         `json:"id"`
	OrgID        uuid.UUID         `json:"org_id"`
	Name         string            `json:"name"`
	Secret       string            `json:"-"`
	FieldMapping map[string]string `json:"field_mapping"`
	IsActive     bool              `json:"is_active"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// GenericWebhookConfigRepository handles generic webhook configuration storage.
type GenericWebhookConfigRepository struct {
	pool *pgxpool.Pool
}

// NewGenericWebhookConfigRepository creates a GenericWebhookConfigRepository.
func NewGenericWebhookConfigRepository(pool *pgxpool.Pool) *GenericWebhookConfigRepository {
	return &GenericWebhookConfigRepository{pool: pool}
}

// Create inserts a generic webhook configuration.
func (r *GenericWebhookConfigRepository) Create(ctx context.Context, c *GenericWebhookConfig) error {
	query := `
		INSERT INTO generic_webhook_configs (id, org_id, name, secret, field_mapping, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.FieldMapping == nil {
		c.FieldMapping = map[string]string{}
	}
	return r.pool.QueryRow(ctx, query, c.ID, c.OrgID, c.Name, c.Secret, c.FieldMapping, c.IsActive).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// ListByOrg lists active and inactive configs for an org, excluding deleted rows.
func (r *GenericWebhookConfigRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*GenericWebhookConfig, error) {
	query := `
		SELECT id, org_id, name, secret, field_mapping, is_active, created_at, updated_at
		FROM generic_webhook_configs
		WHERE org_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list generic webhook configs: %w", err)
	}
	defer rows.Close()

	configs := []*GenericWebhookConfig{}
	for rows.Next() {
		c := &GenericWebhookConfig{}
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Name, &c.Secret, &c.FieldMapping, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan generic webhook config: %w", err)
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// GetByIDAndOrg retrieves one config by org and ID.
func (r *GenericWebhookConfigRepository) GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*GenericWebhookConfig, error) {
	query := `
		SELECT id, org_id, name, secret, field_mapping, is_active, created_at, updated_at
		FROM generic_webhook_configs
		WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL`
	return r.get(ctx, query, id, orgID)
}

// GetActiveByIDAndOrg retrieves one active config by org and ID.
func (r *GenericWebhookConfigRepository) GetActiveByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*GenericWebhookConfig, error) {
	query := `
		SELECT id, org_id, name, secret, field_mapping, is_active, created_at, updated_at
		FROM generic_webhook_configs
		WHERE id = $1 AND org_id = $2 AND is_active = TRUE AND deleted_at IS NULL`
	return r.get(ctx, query, id, orgID)
}

func (r *GenericWebhookConfigRepository) get(ctx context.Context, query string, args ...any) (*GenericWebhookConfig, error) {
	c := &GenericWebhookConfig{}
	err := r.pool.QueryRow(ctx, query, args...).Scan(&c.ID, &c.OrgID, &c.Name, &c.Secret, &c.FieldMapping, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get generic webhook config: %w", err)
	}
	return c, nil
}

// Update replaces mutable config fields.
func (r *GenericWebhookConfigRepository) Update(ctx context.Context, c *GenericWebhookConfig) error {
	query := `
		UPDATE generic_webhook_configs
		SET name = $3, secret = $4, field_mapping = $5, is_active = $6, updated_at = NOW()
		WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
		RETURNING updated_at`
	return r.pool.QueryRow(ctx, query, c.ID, c.OrgID, c.Name, c.Secret, c.FieldMapping, c.IsActive).Scan(&c.UpdatedAt)
}

// Delete soft-deletes a config.
func (r *GenericWebhookConfigRepository) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	query := `UPDATE generic_webhook_configs SET deleted_at = NOW(), is_active = FALSE WHERE org_id = $1 AND id = $2 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, orgID, id)
	if err != nil {
		return fmt.Errorf("delete generic webhook config: %w", err)
	}
	return nil
}
