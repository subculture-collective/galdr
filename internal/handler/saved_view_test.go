package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

type mockSavedViewService struct {
	listFn    func(ctx context.Context, orgID, userID uuid.UUID) ([]*repository.SavedView, error)
	getByIDFn func(ctx context.Context, id, orgID, userID uuid.UUID) (*repository.SavedView, error)
	createFn  func(ctx context.Context, orgID, userID uuid.UUID, req service.CreateSavedViewRequest) (*repository.SavedView, error)
	updateFn  func(ctx context.Context, id, orgID, userID uuid.UUID, req service.UpdateSavedViewRequest) (*repository.SavedView, error)
	deleteFn  func(ctx context.Context, id, orgID, userID uuid.UUID) error
}

func (m *mockSavedViewService) List(ctx context.Context, orgID, userID uuid.UUID) ([]*repository.SavedView, error) {
	return m.listFn(ctx, orgID, userID)
}

func (m *mockSavedViewService) GetByID(ctx context.Context, id, orgID, userID uuid.UUID) (*repository.SavedView, error) {
	return m.getByIDFn(ctx, id, orgID, userID)
}

func (m *mockSavedViewService) Create(ctx context.Context, orgID, userID uuid.UUID, req service.CreateSavedViewRequest) (*repository.SavedView, error) {
	return m.createFn(ctx, orgID, userID, req)
}

func (m *mockSavedViewService) Update(ctx context.Context, id, orgID, userID uuid.UUID, req service.UpdateSavedViewRequest) (*repository.SavedView, error) {
	return m.updateFn(ctx, id, orgID, userID, req)
}

func (m *mockSavedViewService) Delete(ctx context.Context, id, orgID, userID uuid.UUID) error {
	return m.deleteFn(ctx, id, orgID, userID)
}

func TestSavedViewList_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	viewID := uuid.New()
	mock := &mockSavedViewService{
		listFn: func(ctx context.Context, oID, uID uuid.UUID) ([]*repository.SavedView, error) {
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			if uID != userID {
				t.Errorf("expected userID %s, got %s", userID, uID)
			}
			return []*repository.SavedView{{ID: viewID, Name: "At risk", Filters: map[string]any{"risk_level": "red"}}}, nil
		},
	}

	h := NewSavedViewHandler(mock)
	req := authedSavedViewRequest(http.MethodGet, "/api/v1/customers/saved-views", nil, orgID, userID)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Views []repository.SavedView `json:"views"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Views) != 1 || resp.Views[0].Filters["risk_level"] != "red" {
		t.Fatalf("expected saved risk filter, got %#v", resp.Views)
	}
}

func TestSavedViewCreate_CapturesFiltersAndSharing(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	viewID := uuid.New()
	mock := &mockSavedViewService{
		createFn: func(ctx context.Context, oID, uID uuid.UUID, req service.CreateSavedViewRequest) (*repository.SavedView, error) {
			if oID != orgID || uID != userID {
				t.Fatalf("expected org/user IDs passed through")
			}
			if req.Name != "Red accounts" {
				t.Errorf("expected name Red accounts, got %s", req.Name)
			}
			if req.Filters.RiskLevel != "red" || req.Filters.Source != "stripe" || req.Filters.Sort != "mrr" {
				t.Errorf("unexpected filters: %#v", req.Filters)
			}
			if !req.IsShared {
				t.Errorf("expected shared view")
			}
			return &repository.SavedView{ID: viewID, Name: req.Name, Filters: map[string]any{"risk_level": req.Filters.RiskLevel}, IsShared: req.IsShared}, nil
		},
	}

	h := NewSavedViewHandler(mock)
	body, _ := json.Marshal(map[string]any{
		"name":      "Red accounts",
		"is_shared": true,
		"filters": map[string]any{
			"search":     "acme",
			"risk_level": "red",
			"source":     "stripe",
			"sort":       "mrr",
			"order":      "desc",
			"assignee":   userID.String(),
			"tags":       []string{"enterprise", "renewal"},
		},
	})
	req := authedSavedViewRequest(http.MethodPost, "/api/v1/customers/saved-views", bytes.NewReader(body), orgID, userID)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestSavedViewUpdate_UsesIDAndPatchBody(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	viewID := uuid.New()
	name := "Expansion pipeline"
	isShared := false
	mock := &mockSavedViewService{
		updateFn: func(ctx context.Context, id, oID, uID uuid.UUID, req service.UpdateSavedViewRequest) (*repository.SavedView, error) {
			if id != viewID || oID != orgID || uID != userID {
				t.Fatalf("expected route/org/user IDs passed through")
			}
			if req.Name == nil || *req.Name != name {
				t.Fatalf("expected name patch")
			}
			if req.IsShared == nil || *req.IsShared != isShared {
				t.Fatalf("expected shared patch")
			}
			if req.Filters == nil || req.Filters.Search != "beta" {
				t.Fatalf("expected filter patch, got %#v", req.Filters)
			}
			return &repository.SavedView{ID: viewID, Name: name, Filters: map[string]any{"search": "beta"}}, nil
		},
	}

	h := NewSavedViewHandler(mock)
	body, _ := json.Marshal(map[string]any{"name": name, "is_shared": isShared, "filters": map[string]any{"search": "beta"}})
	req := authedSavedViewRequest(http.MethodPatch, "/api/v1/customers/saved-views/"+viewID.String(), bytes.NewReader(body), orgID, userID)
	req = withChiParam(req, "id", viewID.String())
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestSavedViewDelete_ForbiddenFromService(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	viewID := uuid.New()
	mock := &mockSavedViewService{
		deleteFn: func(ctx context.Context, id, oID, uID uuid.UUID) error {
			return &service.ForbiddenError{Message: "only the view owner can delete this saved view"}
		},
	}

	h := NewSavedViewHandler(mock)
	req := authedSavedViewRequest(http.MethodDelete, "/api/v1/customers/saved-views/"+viewID.String(), nil, orgID, userID)
	req = withChiParam(req, "id", viewID.String())
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestSavedViewCreate_UnauthorizedWithoutUser(t *testing.T) {
	orgID := uuid.New()
	h := NewSavedViewHandler(&mockSavedViewService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers/saved-views", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func authedSavedViewRequest(method, target string, body *bytes.Reader, orgID, userID uuid.UUID) *http.Request {
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, body)
	}
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	return req.WithContext(ctx)
}
