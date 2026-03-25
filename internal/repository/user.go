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

// User represents a user row from the database.
type User struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  string
	FirstName     string
	LastName      string
	AvatarURL     string
	EmailVerified bool
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UserOrg represents a user_organizations join row.
type UserOrg struct {
	UserID uuid.UUID
	OrgID  uuid.UUID
	Role   string
}

// UserRepository handles user database operations.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user. Use tx for transactional operations.
func (r *UserRepository) Create(ctx context.Context, tx pgx.Tx, user *User) error {
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	return tx.QueryRow(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

// GetByEmail retrieves a user by email (case-insensitive via CITEXT).
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash,
			   COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(avatar_url, ''),
			   email_verified, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`

	u := &User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// EmailExists checks if an email is already taken.
func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return exists, nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, password_hash,
			   COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(avatar_url, ''),
			   email_verified, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`

	u := &User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.AvatarURL, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// UpdateLastLogin sets the last_login_at timestamp for a user.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

// GetUserOrgs returns all organizations a user belongs to.
func (r *UserRepository) GetUserOrgs(ctx context.Context, userID uuid.UUID) ([]UserOrg, error) {
	query := `SELECT user_id, org_id, role FROM user_organizations WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user orgs: %w", err)
	}
	defer rows.Close()

	var orgs []UserOrg
	for rows.Next() {
		var uo UserOrg
		if err := rows.Scan(&uo.UserID, &uo.OrgID, &uo.Role); err != nil {
			return nil, fmt.Errorf("scan user org: %w", err)
		}
		orgs = append(orgs, uo)
	}
	return orgs, rows.Err()
}

// UpdateProfile updates user profile fields.
func (r *UserRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, avatarURL string) error {
	query := `
		UPDATE users SET first_name = $2, last_name = $3, avatar_url = $4
		WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID, firstName, lastName, avatarURL)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	return nil
}

// UpdatePassword updates a user's password hash.
func (r *UserRepository) UpdatePassword(ctx context.Context, tx pgx.Tx, userID uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := tx.Exec(ctx, query, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

// GetUserOrgRole returns the role a user has in a specific org.
func (r *UserRepository) GetUserOrgRole(ctx context.Context, userID, orgID uuid.UUID) (string, error) {
	query := `SELECT role FROM user_organizations WHERE user_id = $1 AND org_id = $2`
	var role string
	err := r.pool.QueryRow(ctx, query, userID, orgID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get user org role: %w", err)
	}
	return role, nil
}
