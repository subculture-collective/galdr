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

// CustomerNote is a note attached to a customer account.
type CustomerNote struct {
	ID               uuid.UUID
	CustomerID       uuid.UUID
	UserID           uuid.UUID
	Content          string
	AuthorFirstName  string
	AuthorLastName   string
	AuthorEmail      string
	AuthorAvatarURL  string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CustomerNoteRepository stores customer notes.
type CustomerNoteRepository struct {
	pool *pgxpool.Pool
}

// NewCustomerNoteRepository creates a new CustomerNoteRepository.
func NewCustomerNoteRepository(pool *pgxpool.Pool) *CustomerNoteRepository {
	return &CustomerNoteRepository{pool: pool}
}

// ListByCustomer returns notes newest first with author details.
func (r *CustomerNoteRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*CustomerNote, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.customer_id, n.user_id, n.content, n.created_at, n.updated_at,
		       COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), u.email::text,
		       COALESCE(u.avatar_url, '')
		FROM customer_notes n
		JOIN users u ON u.id = n.user_id
		WHERE n.customer_id = $1
		ORDER BY n.created_at DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("query customer notes: %w", err)
	}
	defer rows.Close()

	notes := make([]*CustomerNote, 0)
	for rows.Next() {
		note, err := scanCustomerNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customer notes: %w", err)
	}

	return notes, nil
}

// GetByIDForCustomer returns a single note scoped to a customer.
func (r *CustomerNoteRepository) GetByIDForCustomer(ctx context.Context, noteID, customerID uuid.UUID) (*CustomerNote, error) {
	note, err := scanCustomerNote(r.pool.QueryRow(ctx, `
		SELECT n.id, n.customer_id, n.user_id, n.content, n.created_at, n.updated_at,
		       COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), u.email::text,
		       COALESCE(u.avatar_url, '')
		FROM customer_notes n
		JOIN users u ON u.id = n.user_id
		WHERE n.id = $1 AND n.customer_id = $2`, noteID, customerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return note, nil
}

// Create inserts a customer note.
func (r *CustomerNoteRepository) Create(ctx context.Context, note *CustomerNote) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO customer_notes (customer_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`, note.CustomerID, note.UserID, note.Content).
		Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)
}

// Update changes note content.
func (r *CustomerNoteRepository) Update(ctx context.Context, note *CustomerNote) error {
	return r.pool.QueryRow(ctx, `
		UPDATE customer_notes
		SET content = $3
		WHERE id = $1 AND customer_id = $2
		RETURNING updated_at`, note.ID, note.CustomerID, note.Content).
		Scan(&note.UpdatedAt)
}

// Delete removes a note.
func (r *CustomerNoteRepository) Delete(ctx context.Context, noteID, customerID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM customer_notes WHERE id = $1 AND customer_id = $2`, noteID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

type noteRow interface {
	Scan(dest ...any) error
}

func scanCustomerNote(row noteRow) (*CustomerNote, error) {
	note := &CustomerNote{}
	if err := row.Scan(
		&note.ID,
		&note.CustomerID,
		&note.UserID,
		&note.Content,
		&note.CreatedAt,
		&note.UpdatedAt,
		&note.AuthorFirstName,
		&note.AuthorLastName,
		&note.AuthorEmail,
		&note.AuthorAvatarURL,
	); err != nil {
		return nil, err
	}
	return note, nil
}
