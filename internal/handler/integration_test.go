package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

type mockIntegrationService struct {
	listFn       func(ctx context.Context, orgID uuid.UUID) ([]service.IntegrationSummary, error)
	healthFn     func(ctx context.Context, orgID uuid.UUID) (*service.IntegrationHealthResponse, error)
	connectFn    func(ctx context.Context, orgID uuid.UUID, provider string, req service.ConnectIntegrationRequest) (*connectorsdk.AuthResult, error)
	getStatusFn  func(ctx context.Context, orgID uuid.UUID, provider string) (*service.IntegrationStatus, error)
	triggerSyncFn func(ctx context.Context, orgID uuid.UUID, provider string) error
	disconnectFn func(ctx context.Context, orgID uuid.UUID, provider string) error
}

func (m *mockIntegrationService) List(ctx context.Context, orgID uuid.UUID) ([]service.IntegrationSummary, error) {
	return m.listFn(ctx, orgID)
}

func (m *mockIntegrationService) GetHealth(ctx context.Context, orgID uuid.UUID) (*service.IntegrationHealthResponse, error) {
	return m.healthFn(ctx, orgID)
}

func (m *mockIntegrationService) Connect(ctx context.Context, orgID uuid.UUID, provider string, req service.ConnectIntegrationRequest) (*connectorsdk.AuthResult, error) {
	return m.connectFn(ctx, orgID, provider, req)
}

func (m *mockIntegrationService) GetStatus(ctx context.Context, orgID uuid.UUID, provider string) (*service.IntegrationStatus, error) {
	return m.getStatusFn(ctx, orgID, provider)
}

func (m *mockIntegrationService) TriggerSync(ctx context.Context, orgID uuid.UUID, provider string) error {
	return m.triggerSyncFn(ctx, orgID, provider)
}

func (m *mockIntegrationService) Disconnect(ctx context.Context, orgID uuid.UUID, provider string) error {
	return m.disconnectFn(ctx, orgID, provider)
}

func TestIntegrationList_Unauthorized(t *testing.T) {
	h := NewIntegrationHandler(&mockIntegrationService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestIntegrationList_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockIntegrationService{
		listFn: func(ctx context.Context, oID uuid.UUID) ([]service.IntegrationSummary, error) {
			return []service.IntegrationSummary{
				{Provider: "stripe", Status: "active"},
			}, nil
		},
	}

	h := NewIntegrationHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestIntegrationList_ServiceError(t *testing.T) {
	orgID := uuid.New()
	mock := &mockIntegrationService{
		listFn: func(ctx context.Context, oID uuid.UUID) ([]service.IntegrationSummary, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	h := NewIntegrationHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestIntegrationHealth_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockIntegrationService{
		healthFn: func(ctx context.Context, oID uuid.UUID) (*service.IntegrationHealthResponse, error) {
			if oID != orgID {
				t.Errorf("expected org %s, got %s", orgID, oID)
			}
			return &service.IntegrationHealthResponse{
				StaleAfterHours: 24,
				Integrations: []service.IntegrationHealthSummary{
					{Provider: "stripe", Status: "active", HealthStatus: "healthy"},
				},
			}, nil
		},
	}

	h := NewIntegrationHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/health", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestIntegrationConnect_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockIntegrationService{
		connectFn: func(ctx context.Context, oID uuid.UUID, provider string, req service.ConnectIntegrationRequest) (*connectorsdk.AuthResult, error) {
			if provider != "posthog" {
				t.Errorf("expected posthog provider, got %s", provider)
			}
			if req.APIKey != "phx_test" || req.ProjectID != "123" {
				t.Errorf("unexpected connect request: %#v", req)
			}
			return &connectorsdk.AuthResult{ExternalAccountID: "123", Metadata: map[string]string{"project_id": "123"}}, nil
		},
	}

	h := NewIntegrationHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/posthog/connect", bytes.NewBufferString(`{"api_key":"phx_test","project_id":"123"}`))
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "provider", "posthog")
	rr := httptest.NewRecorder()

	h.Connect(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestIntegrationGetStatus_Unauthorized(t *testing.T) {
	h := NewIntegrationHandler(&mockIntegrationService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/stripe/status", nil)
	req = withChiParam(req, "provider", "stripe")
	rr := httptest.NewRecorder()

	h.GetStatus(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestIntegrationGetStatus_EmptyProvider(t *testing.T) {
	orgID := uuid.New()
	h := NewIntegrationHandler(&mockIntegrationService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations//status", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "provider", "")
	rr := httptest.NewRecorder()

	h.GetStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestIntegrationGetStatus_NotFound(t *testing.T) {
	orgID := uuid.New()
	mock := &mockIntegrationService{
		getStatusFn: func(ctx context.Context, oID uuid.UUID, provider string) (*service.IntegrationStatus, error) {
			return nil, &service.NotFoundError{Resource: "integration", Message: "no hubspot integration found"}
		},
	}

	h := NewIntegrationHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/hubspot/status", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "provider", "hubspot")
	rr := httptest.NewRecorder()

	h.GetStatus(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestIntegrationGetStatus_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockIntegrationService{
		getStatusFn: func(ctx context.Context, oID uuid.UUID, provider string) (*service.IntegrationStatus, error) {
			if provider != "stripe" {
				t.Errorf("expected provider stripe, got %s", provider)
			}
			return &service.IntegrationStatus{
				IntegrationSummary: service.IntegrationSummary{Provider: "stripe", Status: "active"},
			}, nil
		},
	}

	h := NewIntegrationHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/stripe/status", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "provider", "stripe")
	rr := httptest.NewRecorder()

	h.GetStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestIntegrationTriggerSync_Unauthorized(t *testing.T) {
	h := NewIntegrationHandler(&mockIntegrationService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/stripe/sync", nil)
	req = withChiParam(req, "provider", "stripe")
	rr := httptest.NewRecorder()

	h.TriggerSync(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestIntegrationTriggerSync_EmptyProvider(t *testing.T) {
	orgID := uuid.New()
	h := NewIntegrationHandler(&mockIntegrationService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations//sync", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "provider", "")
	rr := httptest.NewRecorder()

	h.TriggerSync(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestIntegrationTriggerSync_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockIntegrationService{
		triggerSyncFn: func(ctx context.Context, oID uuid.UUID, provider string) error {
			return nil
		},
	}

	h := NewIntegrationHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/stripe/sync", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "provider", "stripe")
	rr := httptest.NewRecorder()

	h.TriggerSync(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
}

func TestIntegrationDisconnect_Unauthorized(t *testing.T) {
	h := NewIntegrationHandler(&mockIntegrationService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/stripe", nil)
	req = withChiParam(req, "provider", "stripe")
	rr := httptest.NewRecorder()

	h.Disconnect(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestIntegrationDisconnect_EmptyProvider(t *testing.T) {
	orgID := uuid.New()
	h := NewIntegrationHandler(&mockIntegrationService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "provider", "")
	rr := httptest.NewRecorder()

	h.Disconnect(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestIntegrationDisconnect_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockIntegrationService{
		disconnectFn: func(ctx context.Context, oID uuid.UUID, provider string) error {
			return nil
		},
	}

	h := NewIntegrationHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/stripe", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "provider", "stripe")
	rr := httptest.NewRecorder()

	h.Disconnect(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestIntegrationDisconnect_NotFound(t *testing.T) {
	orgID := uuid.New()
	mock := &mockIntegrationService{
		disconnectFn: func(ctx context.Context, oID uuid.UUID, provider string) error {
			return &service.NotFoundError{Resource: "integration", Message: "no stripe integration found"}
		},
	}

	h := NewIntegrationHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/integrations/stripe", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "provider", "stripe")
	rr := httptest.NewRecorder()

	h.Disconnect(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
