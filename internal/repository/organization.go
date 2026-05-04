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

// Organization represents an organizations row.
type Organization struct {
	ID               uuid.UUID
	Name             string
	Slug             string
	Industry         string
	Plan             string
	StripeCustomerID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// OrganizationRepository handles organization database operations.
type OrganizationRepository struct {
	pool *pgxpool.Pool
}

// NewOrganizationRepository creates a new OrganizationRepository.
func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

// Create inserts a new organization inside a transaction.
func (r *OrganizationRepository) Create(ctx context.Context, tx pgx.Tx, org *Organization) error {
	query := `
		INSERT INTO organizations (id, name, slug)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at`

	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}

	return tx.QueryRow(ctx, query, org.ID, org.Name, org.Slug).Scan(&org.CreatedAt, &org.UpdatedAt)
}

// SlugExists checks if an organization slug is already taken.
func (r *OrganizationRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM organizations WHERE slug = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check slug exists: %w", err)
	}
	return exists, nil
}

// AddMember adds a user to an organization with a given role inside a transaction.
func (r *OrganizationRepository) AddMember(ctx context.Context, tx pgx.Tx, userID, orgID uuid.UUID, role string) error {
	query := `INSERT INTO user_organizations (user_id, org_id, role) VALUES ($1, $2, $3)`
	_, err := tx.Exec(ctx, query, userID, orgID, role)
	if err != nil {
		return fmt.Errorf("add org member: %w", err)
	}
	return nil
}

// GetByID retrieves an organization by ID.
func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	query := `
		SELECT id, name, slug, COALESCE(industry, ''), plan, COALESCE(stripe_customer_id, ''),
			   created_at, updated_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL`

	o := &Organization{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.Name, &o.Slug, &o.Industry, &o.Plan, &o.StripeCustomerID,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org by id: %w", err)
	}
	return o, nil
}

// IsMember checks if a user is a member of an organization.
func (r *OrganizationRepository) IsMember(ctx context.Context, userID, orgID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_organizations WHERE user_id = $1 AND org_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, orgID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check org membership: %w", err)
	}
	return exists, nil
}

// OrganizationWithStats holds an org with member and customer counts.
type OrganizationWithStats struct {
	Organization
	MemberCount   int `json:"member_count"`
	CustomerCount int `json:"customer_count"`
}

// GetWithStats retrieves an org with member and customer counts.
func (r *OrganizationRepository) GetWithStats(ctx context.Context, orgID uuid.UUID) (*OrganizationWithStats, error) {
	query := `
		SELECT o.id, o.name, o.slug, COALESCE(o.industry, ''), o.plan, COALESCE(o.stripe_customer_id, ''),
			o.created_at, o.updated_at,
			(SELECT COUNT(*) FROM user_organizations WHERE org_id = o.id) AS member_count,
			(SELECT COUNT(*) FROM customers WHERE org_id = o.id AND deleted_at IS NULL) AS customer_count
		FROM organizations o
		WHERE o.id = $1 AND o.deleted_at IS NULL`

	ows := &OrganizationWithStats{}
	err := r.pool.QueryRow(ctx, query, orgID).Scan(
		&ows.ID, &ows.Name, &ows.Slug, &ows.Industry, &ows.Plan, &ows.StripeCustomerID,
		&ows.CreatedAt, &ows.UpdatedAt,
		&ows.MemberCount, &ows.CustomerCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org with stats: %w", err)
	}
	return ows, nil
}

// Update updates an org's profile fields.
func (r *OrganizationRepository) Update(ctx context.Context, orgID uuid.UUID, name, slug, industry string) error {
	query := `UPDATE organizations SET name = $2, slug = $3, industry = NULLIF($4, '') WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, orgID, name, slug, industry)
	if err != nil {
		return fmt.Errorf("update org: %w", err)
	}
	return nil
}

// UpdatePlan updates an org's billing plan.
func (r *OrganizationRepository) UpdatePlan(ctx context.Context, orgID uuid.UUID, plan string) error {
	query := `UPDATE organizations SET plan = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, orgID, plan)
	if err != nil {
		return fmt.Errorf("update org plan: %w", err)
	}
	return nil
}

// UpdatePlanTx updates an org's billing plan inside an existing transaction.
func (r *OrganizationRepository) UpdatePlanTx(ctx context.Context, tx pgx.Tx, orgID uuid.UUID, plan string) error {
	query := `UPDATE organizations SET plan = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := tx.Exec(ctx, query, orgID, plan)
	if err != nil {
		return fmt.Errorf("update org plan in tx: %w", err)
	}
	return nil
}

// UpdateStripeCustomerID updates an org's Stripe customer ID.
func (r *OrganizationRepository) UpdateStripeCustomerID(ctx context.Context, orgID uuid.UUID, stripeCustomerID string) error {
	query := `UPDATE organizations SET stripe_customer_id = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, orgID, stripeCustomerID)
	if err != nil {
		return fmt.Errorf("update org stripe customer id: %w", err)
	}
	return nil
}

// UpdateStripeCustomerIDTx updates an org's Stripe customer ID inside an existing transaction.
func (r *OrganizationRepository) UpdateStripeCustomerIDTx(ctx context.Context, tx pgx.Tx, orgID uuid.UUID, stripeCustomerID string) error {
	query := `UPDATE organizations SET stripe_customer_id = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := tx.Exec(ctx, query, orgID, stripeCustomerID)
	if err != nil {
		return fmt.Errorf("update org stripe customer id in tx: %w", err)
	}
	return nil
}

// GetByStripeCustomerID retrieves an organization by Stripe customer ID.
func (r *OrganizationRepository) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*Organization, error) {
	query := `
		SELECT id, name, slug, COALESCE(industry, ''), plan, COALESCE(stripe_customer_id, ''), created_at, updated_at
		FROM organizations
		WHERE stripe_customer_id = $1 AND deleted_at IS NULL
		LIMIT 1`

	o := &Organization{}
	err := r.pool.QueryRow(ctx, query, stripeCustomerID).Scan(
		&o.ID,
		&o.Name,
		&o.Slug,
		&o.Industry,
		&o.Plan,
		&o.StripeCustomerID,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org by stripe customer id: %w", err)
	}
	return o, nil
}

// CountMembers returns the number of members in an org.
func (r *OrganizationRepository) CountMembers(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM user_organizations WHERE org_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count members: %w", err)
	}
	return count, nil
}

// OrgMember holds a member of an org with user details.
type OrgMember struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	AvatarURL string    `json:"avatar_url"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}

// ListMembers returns all members of an org with user details.
func (r *OrganizationRepository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]OrgMember, error) {
	query := `
		SELECT u.id, u.email, u.first_name, u.last_name, COALESCE(u.avatar_url, ''),
			uo.role, uo.created_at
		FROM user_organizations uo
		JOIN users u ON u.id = uo.user_id
		WHERE uo.org_id = $1 AND u.deleted_at IS NULL
		ORDER BY uo.created_at`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []OrgMember
	for rows.Next() {
		var m OrgMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.FirstName, &m.LastName, &m.AvatarURL, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// GetMemberRole returns the role of a user in an org.
func (r *OrganizationRepository) GetMemberRole(ctx context.Context, orgID, userID uuid.UUID) (string, error) {
	query := `SELECT role FROM user_organizations WHERE org_id = $1 AND user_id = $2`
	var role string
	err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get member role: %w", err)
	}
	return role, nil
}

// UpdateMemberRole updates a user's role in an org.
func (r *OrganizationRepository) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	query := `UPDATE user_organizations SET role = $3 WHERE org_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, orgID, userID, role)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	return nil
}

// RemoveMember removes a user from an org.
func (r *OrganizationRepository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	query := `DELETE FROM user_organizations WHERE org_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, orgID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// CountOwners returns the number of users with the "owner" role in an org.
func (r *OrganizationRepository) CountOwners(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM user_organizations WHERE org_id = $1 AND role = 'owner'`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count owners: %w", err)
	}
	return count, nil
}
