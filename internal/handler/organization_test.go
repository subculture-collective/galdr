package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

type mockOrganizationService struct {
	getCurrentFn    func(ctx context.Context, orgID uuid.UUID) (*service.OrgDetailResponse, error)
	updateCurrentFn func(ctx context.Context, orgID uuid.UUID, req service.UpdateOrgRequest) (*service.OrgDetailResponse, error)
	createFn        func(ctx context.Context, userID uuid.UUID, req service.CreateOrgRequest) (*service.OrgResponse, error)
}

func (m *mockOrganizationService) GetCurrent(ctx context.Context, orgID uuid.UUID) (*service.OrgDetailResponse, error) {
	return m.getCurrentFn(ctx, orgID)
}

func (m *mockOrganizationService) UpdateCurrent(ctx context.Context, orgID uuid.UUID, req service.UpdateOrgRequest) (*service.OrgDetailResponse, error) {
	return m.updateCurrentFn(ctx, orgID, req)
}

func (m *mockOrganizationService) Create(ctx context.Context, userID uuid.UUID, req service.CreateOrgRequest) (*service.OrgResponse, error) {
	return m.createFn(ctx, userID, req)
}

func TestOrganizationGetCurrent_Unauthorized(t *testing.T) {
	h := NewOrganizationHandler(&mockOrganizationService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrent(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestOrganizationGetCurrent_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockOrganizationService{
		getCurrentFn: func(ctx context.Context, oID uuid.UUID) (*service.OrgDetailResponse, error) {
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			return &service.OrgDetailResponse{
				ID:            orgID,
				Name:          "Test Org",
				Slug:          "test-org",
				Plan:          "free",
				MemberCount:   3,
				CustomerCount: 100,
			}, nil
		},
	}

	h := NewOrganizationHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/current", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.GetCurrent(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp service.OrgDetailResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got %s", resp.Name)
	}
	if resp.MemberCount != 3 {
		t.Errorf("expected 3 members, got %d", resp.MemberCount)
	}
}

func TestOrganizationGetCurrent_NotFound(t *testing.T) {
	orgID := uuid.New()
	mock := &mockOrganizationService{
		getCurrentFn: func(ctx context.Context, oID uuid.UUID) (*service.OrgDetailResponse, error) {
			return nil, &service.NotFoundError{Resource: "organization", Message: "organization not found"}
		},
	}

	h := NewOrganizationHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/current", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.GetCurrent(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestOrganizationUpdateCurrent_Unauthorized(t *testing.T) {
	h := NewOrganizationHandler(&mockOrganizationService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/current", nil)
	rr := httptest.NewRecorder()

	h.UpdateCurrent(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestOrganizationUpdateCurrent_InvalidBody(t *testing.T) {
	orgID := uuid.New()
	h := NewOrganizationHandler(&mockOrganizationService{})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/current", bytes.NewBufferString("{bad"))
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.UpdateCurrent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestOrganizationUpdateCurrent_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockOrganizationService{
		updateCurrentFn: func(ctx context.Context, oID uuid.UUID, req service.UpdateOrgRequest) (*service.OrgDetailResponse, error) {
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			if req.Name == nil || *req.Name != "New Name" {
				t.Errorf("expected name 'New Name', got %v", req.Name)
			}
			return &service.OrgDetailResponse{
				ID:   orgID,
				Name: "New Name",
				Slug: "new-name",
			}, nil
		},
	}

	h := NewOrganizationHandler(mock)
	body, _ := json.Marshal(map[string]string{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/current", bytes.NewReader(body))
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.UpdateCurrent(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestOrganizationUpdateCurrent_ValidationError(t *testing.T) {
	orgID := uuid.New()
	mock := &mockOrganizationService{
		updateCurrentFn: func(ctx context.Context, oID uuid.UUID, req service.UpdateOrgRequest) (*service.OrgDetailResponse, error) {
			return nil, &service.ValidationError{Field: "name", Message: "organization name is required"}
		},
	}

	h := NewOrganizationHandler(mock)
	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/current", bytes.NewReader(body))
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.UpdateCurrent(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestOrganizationCreate_Unauthorized(t *testing.T) {
	h := NewOrganizationHandler(&mockOrganizationService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", nil)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestOrganizationCreate_InvalidBody(t *testing.T) {
	userID := uuid.New()
	h := NewOrganizationHandler(&mockOrganizationService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewBufferString("{bad"))
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestOrganizationCreate_Success(t *testing.T) {
	userID := uuid.New()
	newOrgID := uuid.New()
	mock := &mockOrganizationService{
		createFn: func(ctx context.Context, uID uuid.UUID, req service.CreateOrgRequest) (*service.OrgResponse, error) {
			if uID != userID {
				t.Errorf("expected userID %s, got %s", userID, uID)
			}
			if req.Name != "New Org" {
				t.Errorf("expected name 'New Org', got %s", req.Name)
			}
			return &service.OrgResponse{
				ID:   newOrgID,
				Name: "New Org",
				Slug: "new-org",
				Plan: "free",
			}, nil
		},
	}

	h := NewOrganizationHandler(mock)
	body, _ := json.Marshal(map[string]string{"name": "New Org"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}

	var resp service.OrgResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Name != "New Org" {
		t.Errorf("expected name 'New Org', got %s", resp.Name)
	}
}

func TestOrganizationCreate_ValidationError(t *testing.T) {
	userID := uuid.New()
	mock := &mockOrganizationService{
		createFn: func(ctx context.Context, uID uuid.UUID, req service.CreateOrgRequest) (*service.OrgResponse, error) {
			return nil, &service.ValidationError{Field: "name", Message: "organization name is required"}
		},
	}

	h := NewOrganizationHandler(mock)
	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestOrganizationCreate_InternalError(t *testing.T) {
	userID := uuid.New()
	mock := &mockOrganizationService{
		createFn: func(ctx context.Context, uID uuid.UUID, req service.CreateOrgRequest) (*service.OrgResponse, error) {
			return nil, fmt.Errorf("database connection failed")
		},
	}

	h := NewOrganizationHandler(mock)
	body, _ := json.Marshal(map[string]string{"name": "Org"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations", bytes.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
