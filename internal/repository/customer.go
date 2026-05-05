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

// Customer represents a customers row.
type Customer struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	ExternalID  string
	Source      string
	Email       string
	Name        string
	CompanyName string
	MRRCents    int
	Currency    string
	FirstSeenAt *time.Time
	LastSeenAt  *time.Time
	Metadata    map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// CustomerRepository handles customer database operations.
type CustomerRepository struct {
	pool *pgxpool.Pool
}

// NewCustomerRepository creates a new CustomerRepository.
func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: pool}
}

// UpsertByExternal creates or updates a customer by (org_id, source, external_id).
func (r *CustomerRepository) UpsertByExternal(ctx context.Context, c *Customer) error {
	query := `
		INSERT INTO customers (org_id, external_id, source, email, name, company_name, currency, first_seen_at, last_seen_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (org_id, source, external_id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			company_name = EXCLUDED.company_name,
			currency = EXCLUDED.currency,
			last_seen_at = EXCLUDED.last_seen_at,
			metadata = EXCLUDED.metadata,
			deleted_at = NULL
		RETURNING id, mrr_cents, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		c.OrgID, c.ExternalID, c.Source, c.Email, c.Name, c.CompanyName,
		c.Currency, c.FirstSeenAt, c.LastSeenAt, c.Metadata,
	).Scan(&c.ID, &c.MRRCents, &c.CreatedAt, &c.UpdatedAt)
}

// GetByExternalID retrieves a customer by (org_id, source, external_id).
func (r *CustomerRepository) GetByExternalID(ctx context.Context, orgID uuid.UUID, source, externalID string) (*Customer, error) {
	query := `
		SELECT id, org_id, external_id, source, COALESCE(email, ''), COALESCE(name, ''),
			COALESCE(company_name, ''), mrr_cents, currency,
			first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at, deleted_at
		FROM customers
		WHERE org_id = $1 AND source = $2 AND external_id = $3 AND deleted_at IS NULL`

	c := &Customer{}
	err := r.pool.QueryRow(ctx, query, orgID, source, externalID).Scan(
		&c.ID, &c.OrgID, &c.ExternalID, &c.Source, &c.Email, &c.Name,
		&c.CompanyName, &c.MRRCents, &c.Currency,
		&c.FirstSeenAt, &c.LastSeenAt, &c.Metadata, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer by external id: %w", err)
	}
	return c, nil
}

// GetByEmail retrieves a customer by email within an organization.
func (r *CustomerRepository) GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*Customer, error) {
	query := `
		SELECT id, org_id, external_id, source, COALESCE(email, ''), COALESCE(name, ''),
			COALESCE(company_name, ''), mrr_cents, currency,
			first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at, deleted_at
		FROM customers
		WHERE org_id = $1 AND email = $2 AND deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT 1`

	c := &Customer{}
	err := r.pool.QueryRow(ctx, query, orgID, email).Scan(
		&c.ID, &c.OrgID, &c.ExternalID, &c.Source, &c.Email, &c.Name,
		&c.CompanyName, &c.MRRCents, &c.Currency,
		&c.FirstSeenAt, &c.LastSeenAt, &c.Metadata, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer by email: %w", err)
	}
	return c, nil
}

// GetByID retrieves a customer by ID.
func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	query := `
		SELECT id, org_id, external_id, source, COALESCE(email, ''), COALESCE(name, ''),
			COALESCE(company_name, ''), mrr_cents, currency,
			first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at, deleted_at
		FROM customers
		WHERE id = $1 AND deleted_at IS NULL`

	c := &Customer{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.OrgID, &c.ExternalID, &c.Source, &c.Email, &c.Name,
		&c.CompanyName, &c.MRRCents, &c.Currency,
		&c.FirstSeenAt, &c.LastSeenAt, &c.Metadata, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer by id: %w", err)
	}
	return c, nil
}

// GetByIDAndOrg retrieves a customer by ID within an organization.
func (r *CustomerRepository) GetByIDAndOrg(ctx context.Context, customerID, orgID uuid.UUID) (*Customer, error) {
	query := `
		SELECT id, org_id, external_id, source, COALESCE(email, ''), COALESCE(name, ''),
			COALESCE(company_name, ''), mrr_cents, currency,
			first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at, deleted_at
		FROM customers
		WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL`

	c := &Customer{}
	err := r.pool.QueryRow(ctx, query, customerID, orgID).Scan(
		&c.ID, &c.OrgID, &c.ExternalID, &c.Source, &c.Email, &c.Name,
		&c.CompanyName, &c.MRRCents, &c.Currency,
		&c.FirstSeenAt, &c.LastSeenAt, &c.Metadata, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get customer by id and org: %w", err)
	}
	return c, nil
}

// SoftDelete marks a customer as deleted.
func (r *CustomerRepository) SoftDelete(ctx context.Context, orgID uuid.UUID, source, externalID string) error {
	query := `UPDATE customers SET deleted_at = NOW() WHERE org_id = $1 AND source = $2 AND external_id = $3 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, orgID, source, externalID)
	if err != nil {
		return fmt.Errorf("soft delete customer: %w", err)
	}
	return nil
}

// UpdateMRR updates the mrr_cents for a customer.
func (r *CustomerRepository) UpdateMRR(ctx context.Context, customerID uuid.UUID, mrrCents int) error {
	query := `UPDATE customers SET mrr_cents = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, customerID, mrrCents)
	if err != nil {
		return fmt.Errorf("update mrr: %w", err)
	}
	return nil
}

// UpdateCompanyAndMetadata updates a customer's company name and metadata.
// Metadata is shallow-merged into existing JSONB metadata.
func (r *CustomerRepository) UpdateCompanyAndMetadata(ctx context.Context, customerID uuid.UUID, companyName string, metadata map[string]any) error {
	query := `
		UPDATE customers
		SET
			company_name = CASE WHEN $2 = '' THEN company_name ELSE $2 END,
			metadata = COALESCE(metadata, '{}'::jsonb) || COALESCE($3::jsonb, '{}'::jsonb),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.pool.Exec(ctx, query, customerID, companyName, metadata)
	if err != nil {
		return fmt.Errorf("update customer company and metadata: %w", err)
	}
	return nil
}

// ListByOrg retrieves all non-deleted customers for an org.
func (r *CustomerRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*Customer, error) {
	query := `
		SELECT id, org_id, external_id, source, COALESCE(email, ''), COALESCE(name, ''),
			COALESCE(company_name, ''), mrr_cents, currency,
			first_seen_at, last_seen_at, COALESCE(metadata, '{}'), created_at, updated_at, deleted_at
		FROM customers
		WHERE org_id = $1 AND deleted_at IS NULL
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	var customers []*Customer
	for rows.Next() {
		c := &Customer{}
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.ExternalID, &c.Source, &c.Email, &c.Name,
			&c.CompanyName, &c.MRRCents, &c.Currency,
			&c.FirstSeenAt, &c.LastSeenAt, &c.Metadata, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

// FindDuplicatesByEmail returns groups of active customers that share the same email within an org.
func (r *CustomerRepository) FindDuplicatesByEmail(ctx context.Context, orgID uuid.UUID) ([][]*Customer, error) {
	query := `
		WITH duplicate_emails AS (
			SELECT email
			FROM customers
			WHERE org_id = $1 AND deleted_at IS NULL AND COALESCE(email, '') <> ''
			GROUP BY email
			HAVING COUNT(*) > 1
		)
		SELECT c.id, c.org_id, c.external_id, c.source, COALESCE(c.email, ''), COALESCE(c.name, ''),
			COALESCE(c.company_name, ''), c.mrr_cents, c.currency,
			c.first_seen_at, c.last_seen_at, COALESCE(c.metadata, '{}'), c.created_at, c.updated_at, c.deleted_at
		FROM customers c
		JOIN duplicate_emails d ON d.email = c.email
		WHERE c.org_id = $1 AND c.deleted_at IS NULL
		ORDER BY c.email, c.first_seen_at NULLS LAST, c.created_at`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("find duplicate customers by email: %w", err)
	}
	defer rows.Close()

	groupsByEmail := map[string][]*Customer{}
	emailOrder := make([]string, 0)

	for rows.Next() {
		c := &Customer{}
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.ExternalID, &c.Source, &c.Email, &c.Name,
			&c.CompanyName, &c.MRRCents, &c.Currency,
			&c.FirstSeenAt, &c.LastSeenAt, &c.Metadata, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan duplicate customer: %w", err)
		}

		if _, exists := groupsByEmail[c.Email]; !exists {
			emailOrder = append(emailOrder, c.Email)
		}
		groupsByEmail[c.Email] = append(groupsByEmail[c.Email], c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	groups := make([][]*Customer, 0, len(emailOrder))
	for _, email := range emailOrder {
		groups = append(groups, groupsByEmail[email])
	}

	return groups, nil
}

// CountByOrg returns the number of active customers for an org.
func (r *CustomerRepository) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM customers WHERE org_id = $1 AND deleted_at IS NULL`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count customers: %w", err)
	}
	return count, nil
}

// TotalMRRByOrg returns the sum of mrr_cents for all active customers in an org.
func (r *CustomerRepository) TotalMRRByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	query := `SELECT COALESCE(SUM(mrr_cents), 0) FROM customers WHERE org_id = $1 AND deleted_at IS NULL`
	var total int64
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("total mrr: %w", err)
	}
	return total, nil
}

// CustomerListParams holds pagination and filter params for customer listing.
type CustomerListParams struct {
	OrgID   uuid.UUID
	Page    int
	PerPage int
	Sort    string
	Order   string
	Risk    string
	Search  string
	Source  string
}

// CustomerWithScore holds a customer with its health score data.
type CustomerWithScore struct {
	Customer
	OverallScore *int
	RiskLevel    *string
}

// CustomerListResult holds paginated customer list results.
type CustomerListResult struct {
	Customers  []CustomerWithScore
	Total      int
	Page       int
	PerPage    int
	TotalPages int
}

// ListWithScores returns a paginated customer list with health score join.
func (r *CustomerRepository) ListWithScores(ctx context.Context, params CustomerListParams) (*CustomerListResult, error) {
	// Build WHERE clauses
	where := "c.org_id = $1 AND c.deleted_at IS NULL"
	args := []any{params.OrgID}
	argIdx := 2

	if params.Risk != "" {
		where += fmt.Sprintf(" AND hs.risk_level = $%d", argIdx)
		args = append(args, params.Risk)
		argIdx++
	}
	if params.Search != "" {
		where += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.email ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}
	if params.Source != "" {
		where += fmt.Sprintf(" AND c.source = $%d", argIdx)
		args = append(args, params.Source)
		argIdx++
	}

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM customers c LEFT JOIN health_scores hs ON c.id = hs.customer_id WHERE %s`, where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count customers with scores: %w", err)
	}

	totalPages := 0
	if params.PerPage > 0 {
		totalPages = (total + params.PerPage - 1) / params.PerPage
	}

	// Sort validation
	sortColumn := "c.name"
	sortAllowlist := map[string]string{
		"name":      "c.name",
		"mrr":       "c.mrr_cents",
		"score":     "hs.overall_score",
		"last_seen": "c.last_seen_at",
	}
	if col, ok := sortAllowlist[params.Sort]; ok {
		sortColumn = col
	}

	order := "ASC"
	if params.Order == "desc" {
		order = "DESC"
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT c.id, c.org_id, c.external_id, c.source, COALESCE(c.email, ''), COALESCE(c.name, ''),
			COALESCE(c.company_name, ''), c.mrr_cents, c.currency,
			c.first_seen_at, c.last_seen_at, COALESCE(c.metadata, '{}'), c.created_at, c.updated_at, c.deleted_at,
			hs.overall_score, hs.risk_level
		FROM customers c
		LEFT JOIN health_scores hs ON c.id = hs.customer_id
		WHERE %s
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d`,
		where, sortColumn, order, argIdx, argIdx+1)

	offset := (params.Page - 1) * params.PerPage
	args = append(args, params.PerPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list customers with scores: %w", err)
	}
	defer rows.Close()

	var customers []CustomerWithScore
	for rows.Next() {
		cs := CustomerWithScore{}
		if err := rows.Scan(
			&cs.ID, &cs.OrgID, &cs.ExternalID, &cs.Source, &cs.Email, &cs.Name,
			&cs.CompanyName, &cs.MRRCents, &cs.Currency,
			&cs.FirstSeenAt, &cs.LastSeenAt, &cs.Metadata, &cs.CreatedAt, &cs.UpdatedAt, &cs.DeletedAt,
			&cs.OverallScore, &cs.RiskLevel,
		); err != nil {
			return nil, fmt.Errorf("scan customer with score: %w", err)
		}
		customers = append(customers, cs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &CustomerListResult{
		Customers:  customers,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}
