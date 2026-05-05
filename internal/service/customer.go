package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"

	"github.com/onnwee/pulse-score/internal/repository"
)

// CustomerService handles customer-related business logic.
type CustomerService struct {
	customerRepo *repository.CustomerRepository
	healthRepo   *repository.HealthScoreRepository
	subRepo      *repository.StripeSubscriptionRepository
	eventRepo    *repository.CustomerEventRepository
	noteRepo     *repository.CustomerNoteRepository
	assignRepo   *repository.CustomerAssignmentRepository
}

// NewCustomerService creates a new CustomerService.
func NewCustomerService(
	cr *repository.CustomerRepository,
	hr *repository.HealthScoreRepository,
	sr *repository.StripeSubscriptionRepository,
	er *repository.CustomerEventRepository,
	nr *repository.CustomerNoteRepository,
	ar *repository.CustomerAssignmentRepository,
) *CustomerService {
	return &CustomerService{
		customerRepo: cr,
		healthRepo:   hr,
		subRepo:      sr,
		eventRepo:    er,
		noteRepo:     nr,
		assignRepo:   ar,
	}
}

// CustomerListResponse is the JSON response for customer list.
type CustomerListResponse struct {
	Customers  []CustomerListItem `json:"customers"`
	Pagination PaginationMeta     `json:"pagination"`
}

// CustomerListItem is a single customer in the list response.
type CustomerListItem struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	CompanyName  string     `json:"company_name"`
	MRRCents     int        `json:"mrr_cents"`
	Source       string     `json:"source"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
	OverallScore *int       `json:"overall_score"`
	RiskLevel    *string    `json:"risk_level"`
}

// PaginationMeta holds pagination metadata for list responses.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// List returns a paginated list of customers with health scores.
func (s *CustomerService) List(ctx context.Context, params repository.CustomerListParams) (*CustomerListResponse, error) {
	// Validate and default params
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 25
	}
	if params.PerPage > 100 {
		params.PerPage = 100
	}

	validSorts := map[string]bool{"name": true, "mrr": true, "score": true, "last_seen": true}
	if !validSorts[params.Sort] {
		params.Sort = "name"
	}
	if params.Order != "asc" && params.Order != "desc" {
		params.Order = "asc"
	}

	validRisks := map[string]bool{"green": true, "yellow": true, "red": true, "low": true, "medium": true, "high": true, "critical": true}
	if params.Risk != "" && !validRisks[params.Risk] {
		return nil, &ValidationError{Field: "risk", Message: "invalid risk level"}
	}
	if params.Assignee == "me" && params.AssigneeUserID == uuid.Nil {
		return nil, &ValidationError{Field: "assignee", Message: "current user is required"}
	}
	if params.Assignee != "" && params.Assignee != "me" && params.Assignee != "unassigned" {
		if _, err := uuid.Parse(params.Assignee); err != nil {
			return nil, &ValidationError{Field: "assignee", Message: "invalid assignee"}
		}
	}

	result, err := s.customerRepo.ListWithScores(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	items := make([]CustomerListItem, len(result.Customers))
	for i, c := range result.Customers {
		items[i] = CustomerListItem{
			ID:           c.ID,
			Name:         c.Name,
			Email:        c.Email,
			CompanyName:  c.CompanyName,
			MRRCents:     c.MRRCents,
			Source:       c.Source,
			LastSeenAt:   c.LastSeenAt,
			OverallScore: c.OverallScore,
			RiskLevel:    c.RiskLevel,
		}
	}

	return &CustomerListResponse{
		Customers: items,
		Pagination: PaginationMeta{
			Page:       result.Page,
			PerPage:    result.PerPage,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	}, nil
}

// CustomerDetail is the full detail response for a customer.
type CustomerDetail struct {
	Customer      CustomerInfo             `json:"customer"`
	HealthScore   *HealthScoreDetail       `json:"health_score"`
	Subscriptions []*SubscriptionInfo      `json:"subscriptions"`
	RecentEvents  []*EventInfo             `json:"recent_events"`
	Assignments   []CustomerAssignmentResponse `json:"assignments"`
}

// CustomerInfo holds customer info fields.
type CustomerInfo struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	CompanyName string         `json:"company_name"`
	MRRCents    int            `json:"mrr_cents"`
	Currency    string         `json:"currency"`
	Source      string         `json:"source"`
	ExternalID  string         `json:"external_id"`
	FirstSeenAt *time.Time     `json:"first_seen_at"`
	LastSeenAt  *time.Time     `json:"last_seen_at"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
}

// HealthScoreDetail holds health score info with factor breakdown.
type HealthScoreDetail struct {
	OverallScore int                `json:"overall_score"`
	RiskLevel    string             `json:"risk_level"`
	Factors      map[string]float64 `json:"factors"`
	CalculatedAt time.Time          `json:"calculated_at"`
}

// SubscriptionInfo holds subscription info.
type SubscriptionInfo struct {
	ID                   uuid.UUID  `json:"id"`
	StripeSubscriptionID string     `json:"stripe_subscription_id"`
	Status               string     `json:"status"`
	PlanName             string     `json:"plan_name"`
	AmountCents          int        `json:"amount_cents"`
	Currency             string     `json:"currency"`
	Interval             string     `json:"interval"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end"`
}

// EventInfo holds event info.
type EventInfo struct {
	ID         uuid.UUID      `json:"id"`
	EventType  string         `json:"event_type"`
	Source     string         `json:"source"`
	OccurredAt time.Time      `json:"occurred_at"`
	Data       map[string]any `json:"data"`
}

// CustomerNoteRequest is the write payload for customer notes.
type CustomerNoteRequest struct {
	Content string `json:"content"`
}

// CustomerNotesResponse is the JSON response for note listing.
type CustomerNotesResponse struct {
	Notes []CustomerNoteResponse `json:"notes"`
}

// CustomerNoteResponse represents a note with author and permissions.
type CustomerNoteResponse struct {
	ID         uuid.UUID          `json:"id"`
	CustomerID uuid.UUID          `json:"customer_id"`
	UserID     uuid.UUID          `json:"user_id"`
	Author     CustomerNoteAuthor `json:"author"`
	Content    string             `json:"content"`
	CanEdit    bool               `json:"can_edit"`
	CanDelete  bool               `json:"can_delete"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// CustomerNoteAuthor is displayed with note metadata.
type CustomerNoteAuthor struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
}

// CustomerAssignmentRequest is the assignment write payload.
type CustomerAssignmentRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

// CustomerAssignmentResponse represents a customer assignee.
type CustomerAssignmentResponse struct {
	CustomerID  uuid.UUID                `json:"customer_id"`
	UserID      uuid.UUID                `json:"user_id"`
	Assignee    CustomerAssignmentUser   `json:"assignee"`
	AssignedAt  time.Time                `json:"assigned_at"`
	AssignedBy  uuid.UUID                `json:"assigned_by"`
}

// CustomerAssignmentsResponse is the JSON response for assignment listing.
type CustomerAssignmentsResponse struct {
	Assignments []CustomerAssignmentResponse `json:"assignments"`
}

// CustomerAssignmentUser is displayed with assignment metadata.
type CustomerAssignmentUser struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
}

// GetDetail returns the full detail for a customer.
func (s *CustomerService) GetDetail(ctx context.Context, customerID, orgID uuid.UUID) (*CustomerDetail, error) {
	customer, err := s.customerRepo.GetByIDAndOrg(ctx, customerID, orgID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, &NotFoundError{Resource: "customer", Message: "customer not found"}
	}

	var (
		healthScore   *repository.HealthScore
		subscriptions []*repository.StripeSubscription
		events        []*repository.CustomerEvent
		assignments   []*repository.CustomerAssignment
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		healthScore, err = s.healthRepo.GetByCustomerID(gctx, customerID, orgID)
		return err
	})

	g.Go(func() error {
		var err error
		subscriptions, err = s.subRepo.ListActiveByCustomer(gctx, customerID)
		return err
	})

	g.Go(func() error {
		var err error
		events, err = s.eventRepo.ListByCustomer(gctx, customerID, 10)
		return err
	})

	g.Go(func() error {
		var err error
		assignments, err = s.assignRepo.ListByCustomer(gctx, customerID)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("load customer detail: %w", err)
	}

	detail := &CustomerDetail{
		Customer: CustomerInfo{
			ID:          customer.ID,
			Name:        customer.Name,
			Email:       customer.Email,
			CompanyName: customer.CompanyName,
			MRRCents:    customer.MRRCents,
			Currency:    customer.Currency,
			Source:      customer.Source,
			ExternalID:  customer.ExternalID,
			FirstSeenAt: customer.FirstSeenAt,
			LastSeenAt:  customer.LastSeenAt,
			Metadata:    customer.Metadata,
			CreatedAt:   customer.CreatedAt,
		},
	}

	if healthScore != nil {
		detail.HealthScore = &HealthScoreDetail{
			OverallScore: healthScore.OverallScore,
			RiskLevel:    healthScore.RiskLevel,
			Factors:      healthScore.Factors,
			CalculatedAt: healthScore.CalculatedAt,
		}
	}

	subInfos := make([]*SubscriptionInfo, len(subscriptions))
	for i, sub := range subscriptions {
		subInfos[i] = &SubscriptionInfo{
			ID:                   sub.ID,
			StripeSubscriptionID: sub.StripeSubscriptionID,
			Status:               sub.Status,
			PlanName:             sub.PlanName,
			AmountCents:          sub.AmountCents,
			Currency:             sub.Currency,
			Interval:             sub.Interval,
			CurrentPeriodEnd:     sub.CurrentPeriodEnd,
		}
	}
	detail.Subscriptions = subInfos

	eventInfos := make([]*EventInfo, len(events))
	for i, e := range events {
		eventInfos[i] = &EventInfo{
			ID:         e.ID,
			EventType:  e.EventType,
			Source:     e.Source,
			OccurredAt: e.OccurredAt,
			Data:       e.Data,
		}
	}
	detail.RecentEvents = eventInfos

	detail.Assignments = mapCustomerAssignments(assignments)

	return detail, nil
}

// ListAssignments returns all current assignees for a customer.
func (s *CustomerService) ListAssignments(ctx context.Context, customerID, orgID uuid.UUID) (*CustomerAssignmentsResponse, error) {
	if err := s.ensureCustomerInOrg(ctx, customerID, orgID); err != nil {
		return nil, err
	}

	assignments, err := s.assignRepo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list customer assignments: %w", err)
	}

	return &CustomerAssignmentsResponse{Assignments: mapCustomerAssignments(assignments)}, nil
}

// AssignCustomer assigns an org member to a customer.
func (s *CustomerService) AssignCustomer(ctx context.Context, customerID, orgID, assigneeID, assignedBy uuid.UUID) (*CustomerAssignmentResponse, error) {
	if assigneeID == uuid.Nil {
		return nil, &ValidationError{Field: "user_id", Message: "user_id is required"}
	}
	if err := s.ensureCustomerInOrg(ctx, customerID, orgID); err != nil {
		return nil, err
	}
	belongs, err := s.assignRepo.UserInOrg(ctx, orgID, assigneeID)
	if err != nil {
		return nil, err
	}
	if !belongs {
		return nil, &ValidationError{Field: "user_id", Message: "assignee must belong to organization"}
	}

	if err := s.assignRepo.Upsert(ctx, customerID, assigneeID, assignedBy); err != nil {
		return nil, fmt.Errorf("assign customer: %w", err)
	}
	assignment, err := s.assignRepo.Get(ctx, customerID, assigneeID)
	if err != nil {
		return nil, fmt.Errorf("get customer assignment: %w", err)
	}
	if assignment == nil {
		return nil, &NotFoundError{Resource: "customer_assignment", Message: "customer assignment not found"}
	}

	resp := mapCustomerAssignment(assignment)
	return &resp, nil
}

// UnassignCustomer removes an assignee from a customer.
func (s *CustomerService) UnassignCustomer(ctx context.Context, customerID, orgID, assigneeID uuid.UUID) error {
	if assigneeID == uuid.Nil {
		return &ValidationError{Field: "user_id", Message: "user_id is required"}
	}
	if err := s.ensureCustomerInOrg(ctx, customerID, orgID); err != nil {
		return err
	}
	if err := s.assignRepo.Delete(ctx, customerID, assigneeID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &NotFoundError{Resource: "customer_assignment", Message: "customer assignment not found"}
		}
		return fmt.Errorf("unassign customer: %w", err)
	}
	return nil
}

// EventListResponse is the JSON response for event listing.
type EventListResponse struct {
	Events     []*EventInfo   `json:"events"`
	Pagination PaginationMeta `json:"pagination"`
}

// ListEvents returns a paginated list of events for a customer.
func (s *CustomerService) ListEvents(ctx context.Context, params repository.EventListParams) (*EventListResponse, error) {
	// Validate params
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 25
	}
	if params.PerPage > 100 {
		params.PerPage = 100
	}
	if !params.From.IsZero() && !params.To.IsZero() && params.From.After(params.To) {
		return nil, &ValidationError{Field: "from", Message: "from must be before to"}
	}

	// Verify customer exists and belongs to org
	customer, err := s.customerRepo.GetByIDAndOrg(ctx, params.CustomerID, params.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, &NotFoundError{Resource: "customer", Message: "customer not found"}
	}

	result, err := s.eventRepo.ListPaginated(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	eventInfos := make([]*EventInfo, len(result.Events))
	for i, e := range result.Events {
		eventInfos[i] = &EventInfo{
			ID:         e.ID,
			EventType:  e.EventType,
			Source:     e.Source,
			OccurredAt: e.OccurredAt,
			Data:       e.Data,
		}
	}

	return &EventListResponse{
		Events: eventInfos,
		Pagination: PaginationMeta{
			Page:       result.Page,
			PerPage:    result.PerPage,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	}, nil
}

// ListNotes returns notes for a customer, newest first.
func (s *CustomerService) ListNotes(ctx context.Context, customerID, orgID, actorID uuid.UUID, actorRole string) (*CustomerNotesResponse, error) {
	if err := s.ensureCustomerInOrg(ctx, customerID, orgID); err != nil {
		return nil, err
	}

	notes, err := s.noteRepo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list customer notes: %w", err)
	}

	resp := make([]CustomerNoteResponse, len(notes))
	for i, note := range notes {
		resp[i] = mapCustomerNote(note, actorID, actorRole)
	}

	return &CustomerNotesResponse{Notes: resp}, nil
}

// CreateNote creates a note for a customer.
func (s *CustomerService) CreateNote(ctx context.Context, customerID, orgID, userID uuid.UUID, req CustomerNoteRequest) (*CustomerNoteResponse, error) {
	content, err := validateCustomerNoteContent(req.Content)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCustomerInOrg(ctx, customerID, orgID); err != nil {
		return nil, err
	}

	note := &repository.CustomerNote{CustomerID: customerID, UserID: userID, Content: content}
	if err := s.noteRepo.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("create customer note: %w", err)
	}

	created, err := s.noteRepo.GetByIDForCustomer(ctx, note.ID, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer note: %w", err)
	}
	if created == nil {
		return nil, &NotFoundError{Resource: "customer_note", Message: "customer note not found"}
	}

	resp := mapCustomerNote(created, userID, "member")
	return &resp, nil
}

// UpdateNote updates a note when actor owns it or has admin privileges.
func (s *CustomerService) UpdateNote(ctx context.Context, customerID, noteID, orgID, userID uuid.UUID, actorRole string, req CustomerNoteRequest) (*CustomerNoteResponse, error) {
	content, err := validateCustomerNoteContent(req.Content)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCustomerInOrg(ctx, customerID, orgID); err != nil {
		return nil, err
	}

	note, err := s.noteRepo.GetByIDForCustomer(ctx, noteID, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer note: %w", err)
	}
	if note == nil {
		return nil, &NotFoundError{Resource: "customer_note", Message: "customer note not found"}
	}
	if !canManageCustomerNote(userID, actorRole, note.UserID) {
		return nil, &ForbiddenError{Message: "cannot edit this note"}
	}

	note.Content = content
	if err := s.noteRepo.Update(ctx, note); err != nil {
		return nil, fmt.Errorf("update customer note: %w", err)
	}

	updated := mapCustomerNote(note, userID, actorRole)
	return &updated, nil
}

// DeleteNote deletes a note when actor owns it or has admin privileges.
func (s *CustomerService) DeleteNote(ctx context.Context, customerID, noteID, orgID, userID uuid.UUID, actorRole string) error {
	if err := s.ensureCustomerInOrg(ctx, customerID, orgID); err != nil {
		return err
	}

	note, err := s.noteRepo.GetByIDForCustomer(ctx, noteID, customerID)
	if err != nil {
		return fmt.Errorf("get customer note: %w", err)
	}
	if note == nil {
		return &NotFoundError{Resource: "customer_note", Message: "customer note not found"}
	}
	if !canManageCustomerNote(userID, actorRole, note.UserID) {
		return &ForbiddenError{Message: "cannot delete this note"}
	}

	if err := s.noteRepo.Delete(ctx, noteID, customerID); err != nil {
		return fmt.Errorf("delete customer note: %w", err)
	}
	return nil
}

func (s *CustomerService) ensureCustomerInOrg(ctx context.Context, customerID, orgID uuid.UUID) error {
	customer, err := s.customerRepo.GetByIDAndOrg(ctx, customerID, orgID)
	if err != nil {
		return fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return &NotFoundError{Resource: "customer", Message: "customer not found"}
	}
	return nil
}

func validateCustomerNoteContent(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", &ValidationError{Field: "content", Message: "content is required"}
	}
	if len(trimmed) > 10000 {
		return "", &ValidationError{Field: "content", Message: "content must be 10000 characters or fewer"}
	}
	return trimmed, nil
}

func mapCustomerNote(note *repository.CustomerNote, actorID uuid.UUID, actorRole string) CustomerNoteResponse {
	canManage := canManageCustomerNote(actorID, actorRole, note.UserID)
	return CustomerNoteResponse{
		ID:         note.ID,
		CustomerID: note.CustomerID,
		UserID:     note.UserID,
		Author: CustomerNoteAuthor{
			ID:        note.UserID,
			Name:      customerNoteAuthorName(note),
			Email:     note.AuthorEmail,
			AvatarURL: note.AuthorAvatarURL,
		},
		Content:   note.Content,
		CanEdit:   canManage,
		CanDelete: canManage,
		CreatedAt: note.CreatedAt,
		UpdatedAt: note.UpdatedAt,
	}
}

func mapCustomerAssignment(assignment *repository.CustomerAssignment) CustomerAssignmentResponse {
	return CustomerAssignmentResponse{
		CustomerID: assignment.CustomerID,
		UserID:     assignment.UserID,
		Assignee: CustomerAssignmentUser{
			ID:        assignment.UserID,
			Name:      customerAssignmentUserName(assignment),
			Email:     assignment.AssigneeEmail,
			AvatarURL: assignment.AssigneeAvatarURL,
		},
		AssignedAt: assignment.AssignedAt,
		AssignedBy: assignment.AssignedBy,
	}
}

func mapCustomerAssignments(assignments []*repository.CustomerAssignment) []CustomerAssignmentResponse {
	responses := make([]CustomerAssignmentResponse, len(assignments))
	for i, assignment := range assignments {
		responses[i] = mapCustomerAssignment(assignment)
	}
	return responses
}

func customerAssignmentUserName(assignment *repository.CustomerAssignment) string {
	name := strings.TrimSpace(strings.Join([]string{assignment.AssigneeFirstName, assignment.AssigneeLastName}, " "))
	if name != "" {
		return name
	}
	return assignment.AssigneeEmail
}

func customerNoteAuthorName(note *repository.CustomerNote) string {
	name := strings.TrimSpace(strings.Join([]string{note.AuthorFirstName, note.AuthorLastName}, " "))
	if name != "" {
		return name
	}
	return note.AuthorEmail
}

func canManageCustomerNote(actorID uuid.UUID, actorRole string, authorID uuid.UUID) bool {
	return actorID == authorID || actorRole == "admin" || actorRole == "owner"
}
