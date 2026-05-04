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

// CustomerEvent represents a customer_events row.
type CustomerEvent struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	CustomerID      uuid.UUID
	EventType       string
	Source          string
	ExternalEventID string
	OccurredAt      time.Time
	Data            map[string]any
	CreatedAt       time.Time
}

// CustomerEventRepository handles customer_events database operations.
type CustomerEventRepository struct {
	pool *pgxpool.Pool
}

// NewCustomerEventRepository creates a new CustomerEventRepository.
func NewCustomerEventRepository(pool *pgxpool.Pool) *CustomerEventRepository {
	return &CustomerEventRepository{pool: pool}
}

// Upsert creates a customer event (idempotent by org_id, source, external_event_id).
func (r *CustomerEventRepository) Upsert(ctx context.Context, e *CustomerEvent) error {
	query := `
		INSERT INTO customer_events (org_id, customer_id, event_type, source, external_event_id, occurred_at, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (org_id, source, external_event_id) DO NOTHING
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		e.OrgID, e.CustomerID, e.EventType, e.Source, e.ExternalEventID, e.OccurredAt, e.Data,
	).Scan(&e.ID, &e.CreatedAt)
	// ON CONFLICT DO NOTHING returns no rows — that's fine
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// ListByCustomerAndType returns events for a customer of a specific type since a given time.
func (r *CustomerEventRepository) ListByCustomerAndType(ctx context.Context, customerID uuid.UUID, eventType string, since time.Time) ([]*CustomerEvent, error) {
	query := `
		SELECT id, org_id, customer_id, event_type, source, COALESCE(external_event_id, ''),
			occurred_at, COALESCE(data, '{}'), created_at
		FROM customer_events
		WHERE customer_id = $1 AND event_type = $2 AND occurred_at >= $3
		ORDER BY occurred_at DESC`

	rows, err := r.pool.Query(ctx, query, customerID, eventType, since)
	if err != nil {
		return nil, fmt.Errorf("list customer events by type: %w", err)
	}
	defer rows.Close()

	return scanCustomerEvents(rows)
}

// CountEventsByTypeForOrg returns event counts per customer for a given event type and time window.
func (r *CustomerEventRepository) CountEventsByTypeForOrg(ctx context.Context, orgID uuid.UUID, eventType string, since time.Time) (map[uuid.UUID]int, error) {
	query := `
		SELECT customer_id, COUNT(*)
		FROM customer_events
		WHERE org_id = $1 AND event_type = $2 AND occurred_at >= $3
		GROUP BY customer_id`

	rows, err := r.pool.Query(ctx, query, orgID, eventType, since)
	if err != nil {
		return nil, fmt.Errorf("count events by type for org: %w", err)
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var customerID uuid.UUID
		var count int
		if err := rows.Scan(&customerID, &count); err != nil {
			return nil, fmt.Errorf("scan event count: %w", err)
		}
		counts[customerID] = count
	}
	return counts, rows.Err()
}

// ListByCustomer returns events for a customer ordered by occurred_at desc.
func (r *CustomerEventRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]*CustomerEvent, error) {
	query := `
		SELECT id, org_id, customer_id, event_type, source, COALESCE(external_event_id, ''),
			occurred_at, COALESCE(data, '{}'), created_at
		FROM customer_events
		WHERE customer_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, customerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list customer events: %w", err)
	}
	defer rows.Close()

	return scanCustomerEvents(rows)
}

// ListByCustomerSince returns events for a customer since a given time.
func (r *CustomerEventRepository) ListByCustomerSince(ctx context.Context, customerID uuid.UUID, since time.Time) ([]*CustomerEvent, error) {
	query := `
		SELECT id, org_id, customer_id, event_type, source, COALESCE(external_event_id, ''),
			occurred_at, COALESCE(data, '{}'), created_at
		FROM customer_events
		WHERE customer_id = $1 AND occurred_at >= $2
		ORDER BY occurred_at DESC`

	rows, err := r.pool.Query(ctx, query, customerID, since)
	if err != nil {
		return nil, fmt.Errorf("list customer events since: %w", err)
	}
	defer rows.Close()

	return scanCustomerEvents(rows)
}

func scanCustomerEvents(rows pgx.Rows) ([]*CustomerEvent, error) {
	var events []*CustomerEvent
	for rows.Next() {
		e := &CustomerEvent{}
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.CustomerID, &e.EventType, &e.Source, &e.ExternalEventID,
			&e.OccurredAt, &e.Data, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan customer event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// EventListParams holds pagination and filter params for event listing.
type EventListParams struct {
	CustomerID uuid.UUID
	OrgID      uuid.UUID
	Page       int
	PerPage    int
	EventType  string
	From       time.Time
	To         time.Time
}

// EventListResult holds paginated event list results.
type EventListResult struct {
	Events     []*CustomerEvent
	Total      int
	Page       int
	PerPage    int
	TotalPages int
}

// ListPaginated returns a paginated list of events for a customer with optional filters.
func (r *CustomerEventRepository) ListPaginated(ctx context.Context, params EventListParams) (*EventListResult, error) {
	where := "customer_id = $1 AND org_id = $2"
	args := []any{params.CustomerID, params.OrgID}
	argIdx := 3

	if params.EventType != "" {
		where += fmt.Sprintf(" AND event_type = $%d", argIdx)
		args = append(args, params.EventType)
		argIdx++
	}
	if !params.From.IsZero() {
		where += fmt.Sprintf(" AND occurred_at >= $%d", argIdx)
		args = append(args, params.From)
		argIdx++
	}
	if !params.To.IsZero() {
		where += fmt.Sprintf(" AND occurred_at <= $%d", argIdx)
		args = append(args, params.To)
		argIdx++
	}

	// Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM customer_events WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count events: %w", err)
	}

	totalPages := 0
	if params.PerPage > 0 {
		totalPages = (total + params.PerPage - 1) / params.PerPage
	}

	// Data
	offset := (params.Page - 1) * params.PerPage
	dataQuery := fmt.Sprintf(`
		SELECT id, org_id, customer_id, event_type, source, COALESCE(external_event_id, ''),
			occurred_at, COALESCE(data, '{}'), created_at
		FROM customer_events
		WHERE %s
		ORDER BY occurred_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, params.PerPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list paginated events: %w", err)
	}
	defer rows.Close()

	var events []*CustomerEvent
	for rows.Next() {
		e := &CustomerEvent{}
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.CustomerID, &e.EventType, &e.Source, &e.ExternalEventID,
			&e.OccurredAt, &e.Data, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &EventListResult{
		Events:     events,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}
