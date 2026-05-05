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

// CustomerAssignment links a customer account to an org member.
type CustomerAssignment struct {
	CustomerID        uuid.UUID
	UserID            uuid.UUID
	AssignedAt        time.Time
	AssignedBy        uuid.UUID
	AssigneeFirstName string
	AssigneeLastName  string
	AssigneeEmail     string
	AssigneeAvatarURL string
	AssignedByEmail   string
}

// CustomerAssignmentRepository stores account assignments.
type CustomerAssignmentRepository struct {
	pool *pgxpool.Pool
}

const customerAssignmentSelectColumns = `
	a.customer_id, a.user_id, a.assigned_at, a.assigned_by,
	COALESCE(assignee.first_name, ''), COALESCE(assignee.last_name, ''), assignee.email::text,
	COALESCE(assignee.avatar_url, ''), assigner.email::text`

// NewCustomerAssignmentRepository creates a new CustomerAssignmentRepository.
func NewCustomerAssignmentRepository(pool *pgxpool.Pool) *CustomerAssignmentRepository {
	return &CustomerAssignmentRepository{pool: pool}
}

// ListByCustomer returns all assignees for a customer.
func (r *CustomerAssignmentRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*CustomerAssignment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+customerAssignmentSelectColumns+`
		FROM customer_assignments a
		JOIN users assignee ON assignee.id = a.user_id
		JOIN users assigner ON assigner.id = a.assigned_by
		WHERE a.customer_id = $1
		ORDER BY a.assigned_at DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("query customer assignments: %w", err)
	}
	defer rows.Close()

	assignments := make([]*CustomerAssignment, 0)
	for rows.Next() {
		assignment, err := scanCustomerAssignment(rows)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customer assignments: %w", err)
	}

	return assignments, nil
}

// Get returns one assignment.
func (r *CustomerAssignmentRepository) Get(ctx context.Context, customerID, userID uuid.UUID) (*CustomerAssignment, error) {
	assignment, err := scanCustomerAssignment(r.pool.QueryRow(ctx, `
		SELECT `+customerAssignmentSelectColumns+`
		FROM customer_assignments a
		JOIN users assignee ON assignee.id = a.user_id
		JOIN users assigner ON assigner.id = a.assigned_by
		WHERE a.customer_id = $1 AND a.user_id = $2`, customerID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return assignment, nil
}

// Upsert creates or refreshes an assignment.
func (r *CustomerAssignmentRepository) Upsert(ctx context.Context, customerID, userID, assignedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO customer_assignments (customer_id, user_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (customer_id, user_id) DO UPDATE SET
			assigned_at = NOW(),
			assigned_by = EXCLUDED.assigned_by`, customerID, userID, assignedBy)
	return err
}

// Delete removes an assignment.
func (r *CustomerAssignmentRepository) Delete(ctx context.Context, customerID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM customer_assignments WHERE customer_id = $1 AND user_id = $2`, customerID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// UserInOrg checks the assignee belongs to the customer's org.
func (r *CustomerAssignmentRepository) UserInOrg(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_organizations
			WHERE org_id = $1 AND user_id = $2
		)`, orgID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check assignment user org: %w", err)
	}
	return exists, nil
}

type assignmentRow interface {
	Scan(dest ...any) error
}

func scanCustomerAssignment(row assignmentRow) (*CustomerAssignment, error) {
	assignment := &CustomerAssignment{}
	if err := row.Scan(
		&assignment.CustomerID,
		&assignment.UserID,
		&assignment.AssignedAt,
		&assignment.AssignedBy,
		&assignment.AssigneeFirstName,
		&assignment.AssigneeLastName,
		&assignment.AssigneeEmail,
		&assignment.AssigneeAvatarURL,
		&assignment.AssignedByEmail,
	); err != nil {
		return nil, err
	}
	return assignment, nil
}
