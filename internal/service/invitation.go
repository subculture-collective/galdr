package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
)

const invitationExpiryDays = 7

// CreateInvitationRequest holds input for creating an invitation.
type CreateInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// InvitationResponse is returned by invitation endpoints.
type InvitationResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// AcceptInvitationRequest holds input for accepting an invitation.
type AcceptInvitationRequest struct {
	Token     string `json:"token"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// InvitationService handles invitation logic.
type InvitationService struct {
	pool        *pgxpool.Pool
	invitations *repository.InvitationRepository
	orgs        *repository.OrganizationRepository
	users       *repository.UserRepository
	email       EmailService
	jwtMgr      *auth.JWTManager
}

// NewInvitationService creates a new InvitationService.
func NewInvitationService(pool *pgxpool.Pool, invitations *repository.InvitationRepository, orgs *repository.OrganizationRepository, users *repository.UserRepository, email EmailService, jwtMgr *auth.JWTManager) *InvitationService {
	return &InvitationService{
		pool:        pool,
		invitations: invitations,
		orgs:        orgs,
		users:       users,
		email:       email,
		jwtMgr:      jwtMgr,
	}
}

// Create creates a new invitation.
func (s *InvitationService) Create(ctx context.Context, orgID, inviterID uuid.UUID, req CreateInvitationRequest) (*InvitationResponse, error) {
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return nil, &ValidationError{Field: "email", Message: "email is required"}
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "member"
	}
	if role != "member" && role != "admin" {
		return nil, &ValidationError{Field: "role", Message: "role must be 'member' or 'admin'"}
	}

	// Check if user is already a member
	existingUser, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check user: %w", err)
	}
	if existingUser != nil {
		isMember, err := s.orgs.IsMember(ctx, existingUser.ID, orgID)
		if err != nil {
			return nil, fmt.Errorf("check membership: %w", err)
		}
		if isMember {
			return nil, &ConflictError{Resource: "invitation", Message: "user is already a member of this organization"}
		}
	}

	// Check for duplicate pending invitation
	hasPending, err := s.invitations.HasPendingInvitation(ctx, orgID, email)
	if err != nil {
		return nil, fmt.Errorf("check pending: %w", err)
	}
	if hasPending {
		return nil, &ConflictError{Resource: "invitation", Message: "a pending invitation already exists for this email"}
	}

	// Generate token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	inv := &repository.Invitation{
		OrgID:     orgID,
		Email:     email,
		Role:      role,
		Token:     token,
		InvitedBy: inviterID,
		ExpiresAt: time.Now().Add(invitationExpiryDays * 24 * time.Hour),
	}

	if err := s.invitations.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}

	// Send invitation email (best effort)
	if s.email != nil {
		if err := s.email.SendInvitation(ctx, SendInvitationParams{
			ToEmail:   email,
			Token:     token,
			OrgID:     orgID,
			InviterID: inviterID,
			Role:      role,
		}); err != nil {
			// Log but don't fail the invitation
			fmt.Printf("warning: failed to send invitation email: %v\n", err)
		}
	}

	return &InvitationResponse{
		ID:        inv.ID,
		Email:     inv.Email,
		Role:      inv.Role,
		Status:    inv.Status,
		ExpiresAt: inv.ExpiresAt,
		CreatedAt: inv.CreatedAt,
	}, nil
}

// ListPending returns all pending invitations for an org.
func (s *InvitationService) ListPending(ctx context.Context, orgID uuid.UUID) ([]InvitationResponse, error) {
	invitations, err := s.invitations.ListPendingByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}

	result := make([]InvitationResponse, len(invitations))
	for i, inv := range invitations {
		result[i] = InvitationResponse{
			ID:        inv.ID,
			Email:     inv.Email,
			Role:      inv.Role,
			Status:    inv.Status,
			ExpiresAt: inv.ExpiresAt,
			CreatedAt: inv.CreatedAt,
		}
	}
	return result, nil
}

// Revoke deletes an invitation.
func (s *InvitationService) Revoke(ctx context.Context, orgID, invitationID uuid.UUID) error {
	inv, err := s.invitations.GetByID(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("get invitation: %w", err)
	}
	if inv == nil {
		return &NotFoundError{Resource: "invitation", Message: "invitation not found"}
	}
	if inv.OrgID != orgID {
		return &ForbiddenError{Message: "invitation does not belong to this organization"}
	}

	return s.invitations.Delete(ctx, invitationID)
}

// Accept validates an invitation token and creates/links the user to the org.
func (s *InvitationService) Accept(ctx context.Context, req AcceptInvitationRequest) (*AuthResponse, error) {
	if strings.TrimSpace(req.Token) == "" {
		return nil, &ValidationError{Field: "token", Message: "token is required"}
	}

	inv, err := s.invitations.GetByToken(ctx, req.Token)
	if err != nil {
		return nil, fmt.Errorf("get invitation: %w", err)
	}
	if inv == nil {
		return nil, &NotFoundError{Resource: "invitation", Message: "invalid invitation token"}
	}
	if inv.Status == "accepted" {
		return nil, &ConflictError{Resource: "invitation", Message: "invitation has already been accepted"}
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, &ValidationError{Field: "token", Message: "invitation has expired"}
	}

	// Check if user already exists
	existingUser, err := s.users.GetByEmail(ctx, inv.Email)
	if err != nil {
		return nil, fmt.Errorf("check user: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var user *repository.User

	if existingUser != nil {
		// Existing user — link to org
		user = existingUser
	} else {
		// New user — validate required fields and create
		if strings.TrimSpace(req.Password) == "" {
			return nil, &ValidationError{Field: "password", Message: "password is required for new users"}
		}
		if err := validatePassword(req.Password); err != nil {
			return nil, &ValidationError{Field: "password", Message: err.Error()}
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}

		user = &repository.User{
			Email:        inv.Email,
			PasswordHash: string(hash),
			FirstName:    strings.TrimSpace(req.FirstName),
			LastName:     strings.TrimSpace(req.LastName),
		}

		if err := s.users.Create(ctx, tx, user); err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
	}

	// Add user to org with invited role
	if err := s.orgs.AddMember(ctx, tx, user.ID, inv.OrgID, inv.Role); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	// Mark invitation as accepted
	if err := s.invitations.MarkAccepted(ctx, tx, inv.ID); err != nil {
		return nil, fmt.Errorf("mark accepted: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	// Generate tokens for immediate login
	tokens, err := s.jwtMgr.GenerateTokenPair(user.ID, inv.OrgID, inv.Role)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Get org details for response
	org, err := s.orgs.GetByID(ctx, inv.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}

	return &AuthResponse{
		User: AuthUser{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		},
		Organization: AuthOrg{
			ID:       org.ID,
			Name:     org.Name,
			Slug:     org.Slug,
			Industry: org.Industry,
			Role:     inv.Role,
			Plan:     org.Plan,
		},
		Tokens: tokens,
	}, nil
}
