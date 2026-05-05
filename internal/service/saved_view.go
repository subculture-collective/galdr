package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/onnwee/pulse-score/internal/repository"
)

const maxSavedViewNameLen = 120

// SavedViewFilters is the supported customer filter payload stored as JSONB.
type SavedViewFilters struct {
	Search    string   `json:"search,omitempty"`
	RiskLevel string   `json:"risk_level,omitempty"`
	Source    string   `json:"source,omitempty"`
	Sort      string   `json:"sort,omitempty"`
	Order     string   `json:"order,omitempty"`
	Assignee  string   `json:"assignee,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// CreateSavedViewRequest holds input for creating a saved view.
type CreateSavedViewRequest struct {
	Name     string           `json:"name"`
	Filters  SavedViewFilters `json:"filters"`
	IsShared bool             `json:"is_shared"`
}

// UpdateSavedViewRequest holds input for updating a saved view.
type UpdateSavedViewRequest struct {
	Name     *string           `json:"name"`
	Filters  *SavedViewFilters `json:"filters"`
	IsShared *bool             `json:"is_shared"`
}

// SavedViewService handles saved customer views.
type SavedViewService struct {
	repo *repository.SavedViewRepository
}

// NewSavedViewService creates a new SavedViewService.
func NewSavedViewService(repo *repository.SavedViewRepository) *SavedViewService {
	return &SavedViewService{repo: repo}
}

// List returns all views visible to a user.
func (s *SavedViewService) List(ctx context.Context, orgID, userID uuid.UUID) ([]*repository.SavedView, error) {
	views, err := s.repo.ListVisible(ctx, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("list saved views: %w", err)
	}
	return views, nil
}

// GetByID returns a visible saved view.
func (s *SavedViewService) GetByID(ctx context.Context, id, orgID, userID uuid.UUID) (*repository.SavedView, error) {
	view, err := s.repo.GetVisible(ctx, id, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("get saved view: %w", err)
	}
	if view == nil {
		return nil, &NotFoundError{Resource: "saved_view", Message: "saved view not found"}
	}
	return view, nil
}

// Create creates a saved view.
func (s *SavedViewService) Create(ctx context.Context, orgID, userID uuid.UUID, req CreateSavedViewRequest) (*repository.SavedView, error) {
	name, err := validateSavedViewName(req.Name)
	if err != nil {
		return nil, err
	}

	view := &repository.SavedView{
		OrgID:    orgID,
		UserID:   userID,
		Name:     name,
		Filters:  filtersToMap(req.Filters),
		IsShared: req.IsShared,
	}
	if err := s.repo.Create(ctx, view); err != nil {
		return nil, fmt.Errorf("create saved view: %w", err)
	}
	return view, nil
}

// Update updates an owned saved view.
func (s *SavedViewService) Update(ctx context.Context, id, orgID, userID uuid.UUID, req UpdateSavedViewRequest) (*repository.SavedView, error) {
	view, err := s.repo.GetVisible(ctx, id, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("get saved view: %w", err)
	}
	if view == nil {
		return nil, &NotFoundError{Resource: "saved_view", Message: "saved view not found"}
	}
	if view.UserID != userID {
		return nil, &ForbiddenError{Message: "only the view owner can modify this saved view"}
	}

	if req.Name != nil {
		name, err := validateSavedViewName(*req.Name)
		if err != nil {
			return nil, err
		}
		view.Name = name
	}
	if req.Filters != nil {
		view.Filters = filtersToMap(*req.Filters)
	}
	if req.IsShared != nil {
		view.IsShared = *req.IsShared
	}

	if err := s.repo.UpdateOwned(ctx, view); err != nil {
		if err == pgx.ErrNoRows {
			return nil, &NotFoundError{Resource: "saved_view", Message: "saved view not found"}
		}
		return nil, fmt.Errorf("update saved view: %w", err)
	}
	return s.repo.GetVisible(ctx, id, orgID, userID)
}

// Delete deletes an owned saved view.
func (s *SavedViewService) Delete(ctx context.Context, id, orgID, userID uuid.UUID) error {
	view, err := s.repo.GetVisible(ctx, id, orgID, userID)
	if err != nil {
		return fmt.Errorf("get saved view: %w", err)
	}
	if view == nil {
		return &NotFoundError{Resource: "saved_view", Message: "saved view not found"}
	}
	if view.UserID != userID {
		return &ForbiddenError{Message: "only the view owner can delete this saved view"}
	}

	if err := s.repo.DeleteOwned(ctx, id, orgID, userID); err != nil {
		if err == pgx.ErrNoRows {
			return &NotFoundError{Resource: "saved_view", Message: "saved view not found"}
		}
		return fmt.Errorf("delete saved view: %w", err)
	}
	return nil
}

func validateSavedViewName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(trimmed) > maxSavedViewNameLen {
		return "", &ValidationError{Field: "name", Message: "name must be 120 characters or fewer"}
	}
	return trimmed, nil
}

func filtersToMap(filters SavedViewFilters) map[string]any {
	result := make(map[string]any)
	if filters.Search != "" {
		result["search"] = filters.Search
	}
	if filters.RiskLevel != "" {
		result["risk_level"] = filters.RiskLevel
	}
	if filters.Source != "" {
		result["source"] = filters.Source
	}
	if filters.Sort != "" {
		result["sort"] = filters.Sort
	}
	if filters.Order != "" {
		result["order"] = filters.Order
	}
	if filters.Assignee != "" {
		result["assignee"] = filters.Assignee
	}
	if len(filters.Tags) > 0 {
		result["tags"] = filters.Tags
	}
	return result
}
