package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
)

const bcryptCost = 12

// RegisterRequest holds the input for user registration.
type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	OrgName   string `json:"org_name"`
	Industry  string `json:"industry"`
}

// LoginRequest holds the input for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse holds the response for auth operations.
type AuthResponse struct {
	User         AuthUser        `json:"user"`
	Organization AuthOrg         `json:"organization"`
	Tokens       *auth.TokenPair `json:"tokens"`
}

// AuthUser is the user portion of an auth response.
type AuthUser struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

// AuthOrg is the organization portion of an auth response.
type AuthOrg struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	Industry string    `json:"industry"`
	Role     string    `json:"role"`
	Plan     string    `json:"plan"`
}

func authUserResponse(user *repository.User) AuthUser {
	return AuthUser{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}
}

func authOrgResponse(org *repository.Organization, role string) AuthOrg {
	return AuthOrg{
		ID:       org.ID,
		Name:     org.Name,
		Slug:     org.Slug,
		Industry: org.Industry,
		Role:     role,
		Plan:     org.Plan,
	}
}

// RefreshRequest holds the input for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// PasswordResetRequest holds the input for requesting a password reset.
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetCompleteRequest holds the input for completing a password reset.
type PasswordResetCompleteRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// AuthService handles authentication logic.
type AuthService struct {
	pool           *pgxpool.Pool
	users          *repository.UserRepository
	orgs           *repository.OrganizationRepository
	refreshTokens  *repository.RefreshTokenRepository
	passwordResets *repository.PasswordResetRepository
	jwtMgr         *auth.JWTManager
	refreshTTL     time.Duration
	email          EmailService
}

// NewAuthService creates a new AuthService.
func NewAuthService(pool *pgxpool.Pool, users *repository.UserRepository, orgs *repository.OrganizationRepository, refreshTokens *repository.RefreshTokenRepository, passwordResets *repository.PasswordResetRepository, jwtMgr *auth.JWTManager, refreshTTL time.Duration, email EmailService) *AuthService {
	return &AuthService{
		pool:           pool,
		users:          users,
		orgs:           orgs,
		refreshTokens:  refreshTokens,
		passwordResets: passwordResets,
		jwtMgr:         jwtMgr,
		refreshTTL:     refreshTTL,
		email:          email,
	}
}

// Register creates a new user and organization.
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	// Validate email
	if err := validateEmail(req.Email); err != nil {
		return nil, &ValidationError{Field: "email", Message: err.Error()}
	}

	// Validate password
	if err := validatePassword(req.Password); err != nil {
		return nil, &ValidationError{Field: "password", Message: err.Error()}
	}

	// Validate org name
	if strings.TrimSpace(req.OrgName) == "" {
		return nil, &ValidationError{Field: "org_name", Message: "organization name is required"}
	}
	industry, err := validateIndustry(req.Industry)
	if err != nil {
		return nil, err
	}

	// Check email uniqueness
	exists, err := s.users.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, &ConflictError{Resource: "user", Message: "email already registered"}
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &repository.User{
		Email:        strings.TrimSpace(req.Email),
		PasswordHash: string(hash),
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
	}

	slug := generateSlug(req.OrgName)
	// Handle slug collisions
	baseSlug := slug
	for i := 1; ; i++ {
		slugExists, err := s.orgs.SlugExists(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("check slug: %w", err)
		}
		if !slugExists {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	org := &repository.Organization{
		Name:     strings.TrimSpace(req.OrgName),
		Slug:     slug,
		Industry: industry,
	}

	// Create user + org + membership in a transaction
	err = s.createUserAndOrg(ctx, user, org)
	if err != nil {
		return nil, err
	}

	// Generate tokens
	tokens, err := s.jwtMgr.GenerateTokenPair(user.ID, org.ID, "owner")
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Store refresh token hash
	if err := s.storeRefreshToken(ctx, user.ID, tokens.RefreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &AuthResponse{
		User:         authUserResponse(user),
		Organization: authOrgResponse(org, "owner"),
		Tokens: tokens,
	}, nil
}

func (s *AuthService) createUserAndOrg(ctx context.Context, user *repository.User, org *repository.Organization) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.users.Create(ctx, tx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if err := s.orgs.Create(ctx, tx, org); err != nil {
		return fmt.Errorf("create org: %w", err)
	}

	if err := s.orgs.AddMember(ctx, tx, user.ID, org.ID, "owner"); err != nil {
		return fmt.Errorf("add owner: %w", err)
	}

	return tx.Commit(ctx)
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	// Validate inputs
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return nil, &AuthError{Message: "invalid email or password"}
	}

	user, err := s.users.GetByEmail(ctx, strings.TrimSpace(req.Email))
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		// Perform a dummy bcrypt compare to prevent timing attacks
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$12$000000000000000000000000000000000000000000000000000000"), []byte(req.Password))
		return nil, &AuthError{Message: "invalid email or password"}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, &AuthError{Message: "invalid email or password"}
	}

	// Get user's default org (first org)
	orgs, err := s.users.GetUserOrgs(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get user orgs: %w", err)
	}
	if len(orgs) == 0 {
		return nil, fmt.Errorf("user has no organizations")
	}

	defaultOrg := orgs[0]

	// Get org details
	orgDetails, err := s.orgs.GetByID(ctx, defaultOrg.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}

	// Update last login
	if err := s.users.UpdateLastLogin(ctx, user.ID); err != nil {
		slog.Warn("failed to update last login", "error", err, "user_id", user.ID)
	}

	// Generate tokens
	tokens, err := s.jwtMgr.GenerateTokenPair(user.ID, defaultOrg.OrgID, defaultOrg.Role)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Store refresh token hash
	if err := s.storeRefreshToken(ctx, user.ID, tokens.RefreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &AuthResponse{
		User:         authUserResponse(user),
		Organization: authOrgResponse(orgDetails, defaultOrg.Role),
		Tokens: tokens,
	}, nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	hasUpper := false
	hasLower := false
	hasDigit := false
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}
	return nil
}

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "org"
	}
	return slug
}

func (s *AuthService) storeRefreshToken(ctx context.Context, userID uuid.UUID, rawToken string) error {
	hash := repository.HashToken(rawToken)
	expiresAt := time.Now().Add(s.refreshTTL)
	return s.refreshTokens.Create(ctx, userID, hash, expiresAt)
}

// Refresh validates a refresh token and returns a new token pair (with rotation).
func (s *AuthService) Refresh(ctx context.Context, req RefreshRequest) (*AuthResponse, error) {
	if req.RefreshToken == "" {
		return nil, &AuthError{Message: "refresh token is required"}
	}

	hash := repository.HashToken(req.RefreshToken)
	rt, err := s.refreshTokens.GetByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	if rt == nil {
		return nil, &AuthError{Message: "invalid refresh token"}
	}
	if rt.RevokedAt != nil {
		return nil, &AuthError{Message: "refresh token has been revoked"}
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, &AuthError{Message: "refresh token has expired"}
	}

	// Revoke old token (rotation)
	if err := s.refreshTokens.Revoke(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	// Get user and their default org
	user, err := s.users.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, &AuthError{Message: "user not found"}
	}

	orgs, err := s.users.GetUserOrgs(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get user orgs: %w", err)
	}
	if len(orgs) == 0 {
		return nil, fmt.Errorf("user has no organizations")
	}

	defaultOrg := orgs[0]
	orgDetails, err := s.orgs.GetByID(ctx, defaultOrg.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}

	// Generate new token pair
	tokens, err := s.jwtMgr.GenerateTokenPair(user.ID, defaultOrg.OrgID, defaultOrg.Role)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Store new refresh token
	if err := s.storeRefreshToken(ctx, user.ID, tokens.RefreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &AuthResponse{
		User:         authUserResponse(user),
		Organization: authOrgResponse(orgDetails, defaultOrg.Role),
		Tokens: tokens,
	}, nil
}

// generateSecureToken generates a cryptographically random URL-safe base64 token.
func generateSecureToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// RequestPasswordReset always returns nil (no email enumeration).
// If the user exists, a reset email is sent.
func (s *AuthService) RequestPasswordReset(ctx context.Context, req PasswordResetRequest) error {
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return nil // don't reveal whether email exists
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		slog.Error("password reset lookup failed", "error", err)
		return nil
	}
	if user == nil {
		return nil // user not found — silently succeed
	}

	token, err := generateSecureToken(32)
	if err != nil {
		slog.Error("generate password reset token failed", "error", err)
		return nil
	}

	hash := repository.HashToken(token)
	expiresAt := time.Now().Add(1 * time.Hour)

	if err := s.passwordResets.Create(ctx, user.ID, hash, expiresAt); err != nil {
		slog.Error("store password reset token failed", "error", err)
		return nil
	}

	if s.email != nil {
		if err := s.email.SendPasswordReset(ctx, SendPasswordResetParams{
			ToEmail: user.Email,
			Token:   token,
		}); err != nil {
			slog.Error("send password reset email failed", "error", err)
		}
	}

	return nil
}

// CompletePasswordReset validates the token, updates the password, and revokes all refresh tokens.
func (s *AuthService) CompletePasswordReset(ctx context.Context, req PasswordResetCompleteRequest) error {
	if strings.TrimSpace(req.Token) == "" {
		return &ValidationError{Field: "token", Message: "token is required"}
	}
	if err := validatePassword(req.NewPassword); err != nil {
		return &ValidationError{Field: "new_password", Message: err.Error()}
	}

	hash := repository.HashToken(req.Token)
	pr, err := s.passwordResets.GetByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("get password reset: %w", err)
	}
	if pr == nil {
		return &AuthError{Message: "invalid or expired reset token"}
	}
	if pr.UsedAt != nil {
		return &AuthError{Message: "reset token has already been used"}
	}
	if time.Now().After(pr.ExpiresAt) {
		return &AuthError{Message: "reset token has expired"}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.users.UpdatePassword(ctx, tx, pr.UserID, string(passwordHash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if err := s.passwordResets.MarkUsed(ctx, tx, pr.ID); err != nil {
		return fmt.Errorf("mark reset used: %w", err)
	}

	if err := s.refreshTokens.RevokeAllForUser(ctx, tx, pr.UserID); err != nil {
		return fmt.Errorf("revoke refresh tokens: %w", err)
	}

	return tx.Commit(ctx)
}
