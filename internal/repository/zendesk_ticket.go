package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ZendeskTicket represents a zendesk_tickets row.
type ZendeskTicket struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	CustomerID       *uuid.UUID
	ZendeskTicketID string
	ZendeskUserID   string
	Subject         string
	Status          string
	Priority        string
	Type            string
	CreatedAtRemote *time.Time
	UpdatedAtRemote *time.Time
	SolvedAt        *time.Time
	Metadata        map[string]any
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ZendeskTicketRepository handles zendesk_tickets database operations.
type ZendeskTicketRepository struct{ pool *pgxpool.Pool }

// NewZendeskTicketRepository creates a new ZendeskTicketRepository.
func NewZendeskTicketRepository(pool *pgxpool.Pool) *ZendeskTicketRepository { return &ZendeskTicketRepository{pool: pool} }

// Upsert creates or updates a Zendesk ticket by (org_id, zendesk_ticket_id).
func (r *ZendeskTicketRepository) Upsert(ctx context.Context, t *ZendeskTicket) error {
	query := `
		INSERT INTO zendesk_tickets (org_id, customer_id, zendesk_ticket_id, zendesk_user_id,
			subject, status, priority, type, created_at_remote, updated_at_remote, solved_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (org_id, zendesk_ticket_id) DO UPDATE SET
			customer_id = COALESCE(EXCLUDED.customer_id, zendesk_tickets.customer_id),
			zendesk_user_id = EXCLUDED.zendesk_user_id,
			subject = EXCLUDED.subject,
			status = EXCLUDED.status,
			priority = EXCLUDED.priority,
			type = EXCLUDED.type,
			created_at_remote = EXCLUDED.created_at_remote,
			updated_at_remote = EXCLUDED.updated_at_remote,
			solved_at = EXCLUDED.solved_at,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		t.OrgID, t.CustomerID, t.ZendeskTicketID, t.ZendeskUserID, t.Subject, t.Status,
		t.Priority, t.Type, t.CreatedAtRemote, t.UpdatedAtRemote, t.SolvedAt, t.Metadata,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// CountByOrgID returns Zendesk ticket count for an org.
func (r *ZendeskTicketRepository) CountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM zendesk_tickets WHERE org_id = $1`, orgID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count zendesk tickets: %w", err)
	}
	return count, nil
}

// CountByCustomerAndStatus returns open and solved ticket counts for a customer in a time window.
func (r *ZendeskTicketRepository) CountByCustomerAndStatus(ctx context.Context, customerID uuid.UUID, since time.Time) (open int, solved int, err error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status IN ('new', 'open', 'pending', 'hold') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status IN ('solved', 'closed') THEN 1 ELSE 0 END), 0)
		FROM zendesk_tickets
		WHERE customer_id = $1 AND created_at_remote >= $2`
	err = r.pool.QueryRow(ctx, query, customerID, since).Scan(&open, &solved)
	if err != nil {
		return 0, 0, fmt.Errorf("count zendesk tickets by status: %w", err)
	}
	return open, solved, nil
}

// AvgResolutionHours returns average ticket resolution time for a customer.
func (r *ZendeskTicketRepository) AvgResolutionHours(ctx context.Context, customerID uuid.UUID, since time.Time) (float64, error) {
	query := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (solved_at - created_at_remote)) / 3600), 0)
		FROM zendesk_tickets
		WHERE customer_id = $1 AND solved_at IS NOT NULL AND created_at_remote >= $2`
	var avg float64
	if err := r.pool.QueryRow(ctx, query, customerID, since).Scan(&avg); err != nil {
		return 0, fmt.Errorf("avg zendesk resolution hours: %w", err)
	}
	return avg, nil
}

// CountOpenByCustomer returns open ticket counts per customer for an org.
func (r *ZendeskTicketRepository) CountOpenByCustomer(ctx context.Context, orgID uuid.UUID) (map[uuid.UUID]int, error) {
	query := `
		SELECT customer_id, COUNT(*)
		FROM zendesk_tickets
		WHERE org_id = $1 AND customer_id IS NOT NULL AND status IN ('new', 'open', 'pending', 'hold')
		GROUP BY customer_id`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("count open zendesk tickets by customer: %w", err)
	}
	defer rows.Close()

	counts := map[uuid.UUID]int{}
	for rows.Next() {
		var customerID uuid.UUID
		var count int
		if err := rows.Scan(&customerID, &count); err != nil {
			return nil, fmt.Errorf("scan open zendesk ticket count: %w", err)
		}
		counts[customerID] = count
	}
	return counts, rows.Err()
}
