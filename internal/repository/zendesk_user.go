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

// ZendeskUser represents a zendesk_users row.
type ZendeskUser struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	CustomerID    *uuid.UUID
	ZendeskUserID string
	Email         string
	Name          string
	Role          string
	Metadata      map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ZendeskUserRepository handles zendesk_users database operations.
type ZendeskUserRepository struct{ pool *pgxpool.Pool }

// NewZendeskUserRepository creates a new ZendeskUserRepository.
func NewZendeskUserRepository(pool *pgxpool.Pool) *ZendeskUserRepository { return &ZendeskUserRepository{pool: pool} }

// Upsert creates or updates a Zendesk user by (org_id, zendesk_user_id).
func (r *ZendeskUserRepository) Upsert(ctx context.Context, u *ZendeskUser) error {
	query := `
		INSERT INTO zendesk_users (org_id, customer_id, zendesk_user_id, email, name, role, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (org_id, zendesk_user_id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, zendesk_users.customer_id),
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			role = EXCLUDED.role,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query, u.OrgID, u.CustomerID, u.ZendeskUserID, u.Email, u.Name, u.Role, u.Metadata).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

// GetByZendeskID returns a Zendesk user by external ID.
func (r *ZendeskUserRepository) GetByZendeskID(ctx context.Context, orgID uuid.UUID, zendeskUserID string) (*ZendeskUser, error) {
	query := `
		SELECT id, org_id, customer_id, zendesk_user_id, COALESCE(email, ''), COALESCE(name, ''),
			COALESCE(role, ''), COALESCE(metadata, '{}'), created_at, updated_at
		FROM zendesk_users
		WHERE org_id = $1 AND zendesk_user_id = $2`
	u := &ZendeskUser{}
	err := r.pool.QueryRow(ctx, query, orgID, zendeskUserID).Scan(&u.ID, &u.OrgID, &u.CustomerID, &u.ZendeskUserID, &u.Email, &u.Name, &u.Role, &u.Metadata, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get zendesk user by id: %w", err)
	}
	return u, nil
}

// CountByOrgID returns Zendesk user count for an org.
func (r *ZendeskUserRepository) CountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM zendesk_users WHERE org_id = $1`, orgID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count zendesk users: %w", err)
	}
	return count, nil
}

// LinkCustomer sets the customer_id for a Zendesk user.
func (r *ZendeskUserRepository) LinkCustomer(ctx context.Context, id, customerID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE zendesk_users SET customer_id = $2, updated_at = NOW() WHERE id = $1`, id, customerID)
	if err != nil {
		return fmt.Errorf("link zendesk user to customer: %w", err)
	}
	return nil
}
