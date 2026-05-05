package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SavedView stores a named customer-list filter configuration.
type SavedView struct {
	ID        uuid.UUID      `json:"id"`
	OrgID     uuid.UUID      `json:"org_id"`
	UserID    uuid.UUID      `json:"user_id"`
	Name      string         `json:"name"`
	Filters   map[string]any `json:"filters"`
	IsShared  bool           `json:"is_shared"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// SavedViewRepository handles saved_views database operations.
type SavedViewRepository struct {
	pool *pgxpool.Pool
}

// NewSavedViewRepository creates a new SavedViewRepository.
func NewSavedViewRepository(pool *pgxpool.Pool) *SavedViewRepository {
	return &SavedViewRepository{pool: pool}
}

// ListVisible returns shared org views and private views for the current user.
func (r *SavedViewRepository) ListVisible(ctx context.Context, orgID, userID uuid.UUID) ([]*SavedView, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, user_id, name, filters, is_shared, created_at, updated_at
		FROM saved_views
		WHERE org_id = $1 AND (is_shared = TRUE OR user_id = $2)
		ORDER BY is_shared DESC, name ASC, created_at DESC
	`, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("query saved views: %w", err)
	}
	defer rows.Close()

	var views []*SavedView
	for rows.Next() {
		view := &SavedView{}
		if err := rows.Scan(&view.ID, &view.OrgID, &view.UserID, &view.Name, &view.Filters, &view.IsShared, &view.CreatedAt, &view.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan saved view: %w", err)
		}
		views = append(views, view)
	}

	return views, nil
}

// GetVisible returns one shared org view or private user view.
func (r *SavedViewRepository) GetVisible(ctx context.Context, id, orgID, userID uuid.UUID) (*SavedView, error) {
	view := &SavedView{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, user_id, name, filters, is_shared, created_at, updated_at
		FROM saved_views
		WHERE id = $1 AND org_id = $2 AND (is_shared = TRUE OR user_id = $3)
	`, id, orgID, userID).Scan(&view.ID, &view.OrgID, &view.UserID, &view.Name, &view.Filters, &view.IsShared, &view.CreatedAt, &view.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query saved view: %w", err)
	}
	return view, nil
}

// Create inserts a saved view.
func (r *SavedViewRepository) Create(ctx context.Context, view *SavedView) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO saved_views (org_id, user_id, name, filters, is_shared)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, view.OrgID, view.UserID, view.Name, view.Filters, view.IsShared).Scan(&view.ID, &view.CreatedAt, &view.UpdatedAt)
}

// UpdateOwned updates a view only when owned by the current user.
func (r *SavedViewRepository) UpdateOwned(ctx context.Context, view *SavedView) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE saved_views
		SET name = $1, filters = $2, is_shared = $3, updated_at = NOW()
		WHERE id = $4 AND org_id = $5 AND user_id = $6
	`, view.Name, view.Filters, view.IsShared, view.ID, view.OrgID, view.UserID)
	if err != nil {
		return fmt.Errorf("update saved view: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteOwned deletes a view only when owned by the current user.
func (r *SavedViewRepository) DeleteOwned(ctx context.Context, id, orgID, userID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `
		DELETE FROM saved_views WHERE id = $1 AND org_id = $2 AND user_id = $3
	`, id, orgID, userID)
	if err != nil {
		return fmt.Errorf("delete saved view: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
