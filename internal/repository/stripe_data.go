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

// StripeSubscription represents a stripe_subscriptions row.
type StripeSubscription struct {
	ID                   uuid.UUID
	OrgID                uuid.UUID
	CustomerID           uuid.UUID
	StripeSubscriptionID string
	Status               string
	PlanName             string
	AmountCents          int
	Currency             string
	Interval             string
	CurrentPeriodStart   *time.Time
	CurrentPeriodEnd     *time.Time
	CanceledAt           *time.Time
	Metadata             map[string]any
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// StripePayment represents a stripe_payments row.
type StripePayment struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	CustomerID      uuid.UUID
	StripePaymentID string
	AmountCents     int
	Currency        string
	Status          string
	FailureCode     string
	FailureMessage  string
	PaidAt          *time.Time
	CreatedAt       time.Time
}

// StripeSubscriptionRepository handles stripe_subscriptions database operations.
type StripeSubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewStripeSubscriptionRepository creates a new StripeSubscriptionRepository.
func NewStripeSubscriptionRepository(pool *pgxpool.Pool) *StripeSubscriptionRepository {
	return &StripeSubscriptionRepository{pool: pool}
}

// Upsert creates or updates a stripe subscription.
func (r *StripeSubscriptionRepository) Upsert(ctx context.Context, s *StripeSubscription) error {
	query := `
		INSERT INTO stripe_subscriptions (org_id, customer_id, stripe_subscription_id, status, plan_name,
			amount_cents, currency, interval, current_period_start, current_period_end, canceled_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (stripe_subscription_id) DO UPDATE SET
			status = EXCLUDED.status,
			plan_name = EXCLUDED.plan_name,
			amount_cents = EXCLUDED.amount_cents,
			currency = EXCLUDED.currency,
			interval = EXCLUDED.interval,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end = EXCLUDED.current_period_end,
			canceled_at = EXCLUDED.canceled_at,
			metadata = EXCLUDED.metadata
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		s.OrgID, s.CustomerID, s.StripeSubscriptionID, s.Status, s.PlanName,
		s.AmountCents, s.Currency, s.Interval, s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.CanceledAt, s.Metadata,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

// ListActiveByCustomer returns active subscriptions for a customer.
func (r *StripeSubscriptionRepository) ListActiveByCustomer(ctx context.Context, customerID uuid.UUID) ([]*StripeSubscription, error) {
	query := `
		SELECT id, org_id, customer_id, stripe_subscription_id, status, COALESCE(plan_name, ''),
			amount_cents, currency, COALESCE(interval, ''), current_period_start, current_period_end,
			canceled_at, COALESCE(metadata, '{}'), created_at, updated_at
		FROM stripe_subscriptions
		WHERE customer_id = $1 AND status IN ('active', 'trialing', 'past_due')
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("list active subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*StripeSubscription
	for rows.Next() {
		s := &StripeSubscription{}
		if err := rows.Scan(
			&s.ID, &s.OrgID, &s.CustomerID, &s.StripeSubscriptionID, &s.Status, &s.PlanName,
			&s.AmountCents, &s.Currency, &s.Interval, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
			&s.CanceledAt, &s.Metadata, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// ListByCustomer returns all subscriptions for a customer.
func (r *StripeSubscriptionRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*StripeSubscription, error) {
	query := `
		SELECT id, org_id, customer_id, stripe_subscription_id, status, COALESCE(plan_name, ''),
			amount_cents, currency, COALESCE(interval, ''), current_period_start, current_period_end,
			canceled_at, COALESCE(metadata, '{}'), created_at, updated_at
		FROM stripe_subscriptions
		WHERE customer_id = $1
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("list customer subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*StripeSubscription
	for rows.Next() {
		s := &StripeSubscription{}
		if err := rows.Scan(
			&s.ID, &s.OrgID, &s.CustomerID, &s.StripeSubscriptionID, &s.Status, &s.PlanName,
			&s.AmountCents, &s.Currency, &s.Interval, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
			&s.CanceledAt, &s.Metadata, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan customer subscription: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// ListByOrg returns all subscriptions for an org.
func (r *StripeSubscriptionRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*StripeSubscription, error) {
	query := `
		SELECT id, org_id, customer_id, stripe_subscription_id, status, COALESCE(plan_name, ''),
			amount_cents, currency, COALESCE(interval, ''), current_period_start, current_period_end,
			canceled_at, COALESCE(metadata, '{}'), created_at, updated_at
		FROM stripe_subscriptions
		WHERE org_id = $1
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list org subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*StripeSubscription
	for rows.Next() {
		s := &StripeSubscription{}
		if err := rows.Scan(
			&s.ID, &s.OrgID, &s.CustomerID, &s.StripeSubscriptionID, &s.Status, &s.PlanName,
			&s.AmountCents, &s.Currency, &s.Interval, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
			&s.CanceledAt, &s.Metadata, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// GetByStripeID retrieves a subscription by its Stripe ID.
func (r *StripeSubscriptionRepository) GetByStripeID(ctx context.Context, stripeSubID string) (*StripeSubscription, error) {
	query := `
		SELECT id, org_id, customer_id, stripe_subscription_id, status, COALESCE(plan_name, ''),
			amount_cents, currency, COALESCE(interval, ''), current_period_start, current_period_end,
			canceled_at, COALESCE(metadata, '{}'), created_at, updated_at
		FROM stripe_subscriptions
		WHERE stripe_subscription_id = $1`

	s := &StripeSubscription{}
	err := r.pool.QueryRow(ctx, query, stripeSubID).Scan(
		&s.ID, &s.OrgID, &s.CustomerID, &s.StripeSubscriptionID, &s.Status, &s.PlanName,
		&s.AmountCents, &s.Currency, &s.Interval, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
		&s.CanceledAt, &s.Metadata, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription by stripe id: %w", err)
	}
	return s, nil
}

// DeleteByStripeID removes a subscription by its Stripe ID.
func (r *StripeSubscriptionRepository) DeleteByStripeID(ctx context.Context, stripeSubID string) error {
	query := `DELETE FROM stripe_subscriptions WHERE stripe_subscription_id = $1`
	_, err := r.pool.Exec(ctx, query, stripeSubID)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	return nil
}

// StripePaymentRepository handles stripe_payments database operations.
type StripePaymentRepository struct {
	pool *pgxpool.Pool
}

// NewStripePaymentRepository creates a new StripePaymentRepository.
func NewStripePaymentRepository(pool *pgxpool.Pool) *StripePaymentRepository {
	return &StripePaymentRepository{pool: pool}
}

// Upsert creates or updates a stripe payment.
func (r *StripePaymentRepository) Upsert(ctx context.Context, p *StripePayment) error {
	query := `
		INSERT INTO stripe_payments (org_id, customer_id, stripe_payment_id, amount_cents, currency, status,
			failure_code, failure_message, paid_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (stripe_payment_id) DO UPDATE SET
			status = EXCLUDED.status,
			failure_code = EXCLUDED.failure_code,
			failure_message = EXCLUDED.failure_message,
			paid_at = EXCLUDED.paid_at
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		p.OrgID, p.CustomerID, p.StripePaymentID, p.AmountCents, p.Currency, p.Status,
		p.FailureCode, p.FailureMessage, p.PaidAt,
	).Scan(&p.ID, &p.CreatedAt)
}

// ListByCustomer returns payments for a customer ordered by paid_at descending.
func (r *StripePaymentRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*StripePayment, error) {
	query := `
		SELECT id, org_id, customer_id, stripe_payment_id, amount_cents, currency, status,
			COALESCE(failure_code, ''), COALESCE(failure_message, ''), paid_at, created_at
		FROM stripe_payments
		WHERE customer_id = $1
		ORDER BY COALESCE(paid_at, created_at) DESC`

	rows, err := r.pool.Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []*StripePayment
	for rows.Next() {
		p := &StripePayment{}
		if err := rows.Scan(
			&p.ID, &p.OrgID, &p.CustomerID, &p.StripePaymentID, &p.AmountCents, &p.Currency, &p.Status,
			&p.FailureCode, &p.FailureMessage, &p.PaidAt, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

// CountFailedByCustomerInWindow returns the count of failed payments for a customer in a time window.
func (r *StripePaymentRepository) CountFailedByCustomerInWindow(ctx context.Context, customerID uuid.UUID, since time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM stripe_payments WHERE customer_id = $1 AND status = 'failed' AND COALESCE(paid_at, created_at) >= $2`
	var count int
	err := r.pool.QueryRow(ctx, query, customerID, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count failed payments: %w", err)
	}
	return count, nil
}

// CountByCustomerInWindow returns the total count of payments for a customer in a time window.
func (r *StripePaymentRepository) CountByCustomerInWindow(ctx context.Context, customerID uuid.UUID, since time.Time) (int, error) {
	query := `SELECT COUNT(*) FROM stripe_payments WHERE customer_id = $1 AND COALESCE(paid_at, created_at) >= $2`
	var count int
	err := r.pool.QueryRow(ctx, query, customerID, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count payments: %w", err)
	}
	return count, nil
}

// GetLastSuccessfulPayment returns the most recent successful payment for a customer.
func (r *StripePaymentRepository) GetLastSuccessfulPayment(ctx context.Context, customerID uuid.UUID) (*StripePayment, error) {
	query := `
		SELECT id, org_id, customer_id, stripe_payment_id, amount_cents, currency, status,
			COALESCE(failure_code, ''), COALESCE(failure_message, ''), paid_at, created_at
		FROM stripe_payments
		WHERE customer_id = $1 AND status = 'succeeded'
		ORDER BY paid_at DESC
		LIMIT 1`

	p := &StripePayment{}
	err := r.pool.QueryRow(ctx, query, customerID).Scan(
		&p.ID, &p.OrgID, &p.CustomerID, &p.StripePaymentID, &p.AmountCents, &p.Currency, &p.Status,
		&p.FailureCode, &p.FailureMessage, &p.PaidAt, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get last successful payment: %w", err)
	}
	return p, nil
}

// CountConsecutiveFailures returns the number of consecutive failed payments (most recent first).
func (r *StripePaymentRepository) CountConsecutiveFailures(ctx context.Context, customerID uuid.UUID) (int, error) {
	query := `
		WITH ordered_payments AS (
			SELECT status, ROW_NUMBER() OVER (ORDER BY COALESCE(paid_at, created_at) DESC) AS rn
			FROM stripe_payments WHERE customer_id = $1
		)
		SELECT COUNT(*) FROM ordered_payments
		WHERE status = 'failed' AND rn <= (
			SELECT COALESCE(MIN(rn) - 1, (SELECT COUNT(*) FROM ordered_payments))
			FROM ordered_payments WHERE status != 'failed'
		)`
	var count int
	err := r.pool.QueryRow(ctx, query, customerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count consecutive failures: %w", err)
	}
	return count, nil
}
