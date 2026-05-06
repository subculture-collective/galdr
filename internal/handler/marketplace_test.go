package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

type mockMarketplaceService struct {
	registerFn      func(ctx context.Context, developerID uuid.UUID, req service.RegisterConnectorRequest) (*repository.MarketplaceConnector, error)
	listPublishedFn func(ctx context.Context) ([]*repository.MarketplaceConnector, error)
	getPublishedFn  func(ctx context.Context, id string) (*repository.MarketplaceConnector, error)
	installFn       func(ctx context.Context, orgID uuid.UUID, id string, req service.InstallConnectorRequest) (*repository.ConnectorInstallation, error)
	listReviewQueueFn func(ctx context.Context) ([]*repository.MarketplaceConnector, error)
	reviewFn        func(ctx context.Context, reviewerID uuid.UUID, id, version string, req service.ConnectorReviewRequest) (*repository.ConnectorReviewResult, error)
	rejectFn        func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error)
	publishFn       func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error)
}

func (m *mockMarketplaceService) Register(ctx context.Context, developerID uuid.UUID, req service.RegisterConnectorRequest) (*repository.MarketplaceConnector, error) {
	return m.registerFn(ctx, developerID, req)
}

func (m *mockMarketplaceService) ListPublished(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
	return m.listPublishedFn(ctx)
}

func (m *mockMarketplaceService) GetPublished(ctx context.Context, id string) (*repository.MarketplaceConnector, error) {
	return m.getPublishedFn(ctx, id)
}

func (m *mockMarketplaceService) Install(ctx context.Context, orgID uuid.UUID, id string, req service.InstallConnectorRequest) (*repository.ConnectorInstallation, error) {
	return m.installFn(ctx, orgID, id, req)
}

func (m *mockMarketplaceService) ListReviewQueue(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
	return m.listReviewQueueFn(ctx)
}

func (m *mockMarketplaceService) Review(ctx context.Context, reviewerID uuid.UUID, id, version string, req service.ConnectorReviewRequest) (*repository.ConnectorReviewResult, error) {
	return m.reviewFn(ctx, reviewerID, id, version, req)
}

func (m *mockMarketplaceService) Reject(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
	return m.rejectFn(ctx, id, version)
}

func (m *mockMarketplaceService) Publish(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
	return m.publishFn(ctx, id, version)
}

func TestMarketplaceRegister_Unauthorized(t *testing.T) {
	h := NewMarketplaceHandler(&mockMarketplaceService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/connectors", nil)
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMarketplaceRegister_Success(t *testing.T) {
	developerID := uuid.New()
	manifest := validMarketplaceManifest("mock-crm", "1.0.0")
	body, err := json.Marshal(service.RegisterConnectorRequest{Manifest: manifest, Status: repository.MarketplaceConnectorStatusSubmitted})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	mock := &mockMarketplaceService{
		registerFn: func(ctx context.Context, dID uuid.UUID, req service.RegisterConnectorRequest) (*repository.MarketplaceConnector, error) {
			if dID != developerID {
				t.Fatalf("expected developer id %s, got %s", developerID, dID)
			}
			if req.Manifest.ID != manifest.ID || req.Status != repository.MarketplaceConnectorStatusSubmitted {
				t.Fatal("request not decoded")
			}
			return marketplaceConnector(manifest.ID, manifest.Version, repository.MarketplaceConnectorStatusSubmitted), nil
		},
	}

	h := NewMarketplaceHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/connectors", bytes.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), developerID))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestMarketplaceListPublished_Success(t *testing.T) {
	mock := &mockMarketplaceService{
		listPublishedFn: func(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
			return []*repository.MarketplaceConnector{marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusPublished)}, nil
		},
	}

	h := NewMarketplaceHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/marketplace/connectors", nil)
	rr := httptest.NewRecorder()

	h.ListPublished(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var res struct {
		Connectors []repository.MarketplaceConnector `json:"connectors"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(res.Connectors) != 1 || res.Connectors[0].ID != "mock-crm" {
		t.Fatalf("unexpected connectors response: %+v", res.Connectors)
	}
}

func TestMarketplaceGetPublished_EmptyID(t *testing.T) {
	h := NewMarketplaceHandler(&mockMarketplaceService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/marketplace/connectors/", nil)
	req = withChiParam(req, "id", "")
	rr := httptest.NewRecorder()

	h.GetPublished(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestMarketplaceInstall_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockMarketplaceService{
		installFn: func(ctx context.Context, oID uuid.UUID, id string, req service.InstallConnectorRequest) (*repository.ConnectorInstallation, error) {
			if oID != orgID || id != "mock-crm" {
				t.Fatalf("unexpected install target %s %s", oID, id)
			}
			return &repository.ConnectorInstallation{ID: uuid.New(), OrgID: orgID, ConnectorID: id, ConnectorVersion: "1.0.0", Status: repository.ConnectorInstallationStatusActive}, nil
		},
	}

	h := NewMarketplaceHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/connectors/mock-crm/install", bytes.NewReader([]byte(`{"config":{"api_key":"secret"}}`)))
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", "mock-crm")
	rr := httptest.NewRecorder()

	h.Install(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestMarketplaceInstall_AllowsEmptyConfig(t *testing.T) {
	orgID := uuid.New()
	mock := &mockMarketplaceService{
		installFn: func(ctx context.Context, oID uuid.UUID, id string, req service.InstallConnectorRequest) (*repository.ConnectorInstallation, error) {
			if oID != orgID || id != "mock-crm" {
				t.Fatalf("unexpected install target %s %s", oID, id)
			}
			if req.Config != nil {
				t.Fatalf("expected empty config, got %+v", req.Config)
			}
			return &repository.ConnectorInstallation{ID: uuid.New(), OrgID: orgID, ConnectorID: id, ConnectorVersion: "1.0.0", Status: repository.ConnectorInstallationStatusActive}, nil
		},
	}

	h := NewMarketplaceHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/connectors/mock-crm/install", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", "mock-crm")
	rr := httptest.NewRecorder()

	h.Install(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestMarketplaceInstall_Unauthorized(t *testing.T) {
	h := NewMarketplaceHandler(&mockMarketplaceService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/connectors/mock-crm/install", nil)
	req = withChiParam(req, "id", "mock-crm")
	rr := httptest.NewRecorder()

	h.Install(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMarketplaceReview_Success(t *testing.T) {
	reviewerID := uuid.New()
	body := []byte(`{"checklist":{"data_access_justified":true,"error_handling_ready":true,"documentation_ready":true}}`)
	mock := &mockMarketplaceService{
		reviewFn: func(ctx context.Context, rID uuid.UUID, id, version string, req service.ConnectorReviewRequest) (*repository.ConnectorReviewResult, error) {
			if rID != reviewerID || id != "mock-crm" || version != "1.0.0" {
				t.Fatalf("unexpected review target %s %s %s", rID, id, version)
			}
			if !req.Checklist.DataAccessJustified || !req.Checklist.ErrorHandlingReady || !req.Checklist.DocumentationReady {
				t.Fatal("review checklist not decoded")
			}
			return &repository.ConnectorReviewResult{
				ID:               uuid.New(),
				ConnectorID:      id,
				ConnectorVersion: version,
				ReviewerID:       rID,
				Status:           repository.ConnectorReviewStatusApproved,
			}, nil
		},
	}

	h := NewMarketplaceHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/connectors/mock-crm/versions/1.0.0/review", bytes.NewReader(body))
	req = req.WithContext(auth.WithUserID(req.Context(), reviewerID))
	req = withChiParam(req, "id", "mock-crm")
	req = withChiParam(req, "version", "1.0.0")
	rr := httptest.NewRecorder()

	h.Review(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestMarketplaceReview_Unauthorized(t *testing.T) {
	h := NewMarketplaceHandler(&mockMarketplaceService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/connectors/mock-crm/versions/1.0.0/review", nil)
	req = withChiParam(req, "id", "mock-crm")
	req = withChiParam(req, "version", "1.0.0")
	rr := httptest.NewRecorder()

	h.Review(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMarketplaceListReviewQueue_Success(t *testing.T) {
	mock := &mockMarketplaceService{
		listReviewQueueFn: func(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
			return []*repository.MarketplaceConnector{marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusSubmitted)}, nil
		},
	}

	h := NewMarketplaceHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/marketplace/connectors/review-queue", nil)
	rr := httptest.NewRecorder()

	h.ListReviewQueue(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var res struct {
		Connectors []repository.MarketplaceConnector `json:"connectors"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(res.Connectors) != 1 || res.Connectors[0].Status != repository.MarketplaceConnectorStatusSubmitted {
		t.Fatalf("unexpected queue response: %+v", res.Connectors)
	}
}

func TestMarketplaceReject_Success(t *testing.T) {
	mock := &mockMarketplaceService{
		rejectFn: func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
			if id != "mock-crm" || version != "1.0.0" {
				t.Fatalf("unexpected reject target %s %s", id, version)
			}
			return marketplaceConnector(id, version, repository.MarketplaceConnectorStatusRejected), nil
		},
	}

	h := NewMarketplaceHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/connectors/mock-crm/versions/1.0.0/reject", nil)
	req = withChiParam(req, "id", "mock-crm")
	req = withChiParam(req, "version", "1.0.0")
	rr := httptest.NewRecorder()

	h.Reject(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestMarketplacePublish_Success(t *testing.T) {
	mock := &mockMarketplaceService{
		publishFn: func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
			if id != "mock-crm" || version != "1.0.0" {
				t.Fatalf("unexpected publish target %s %s", id, version)
			}
			return marketplaceConnector(id, version, repository.MarketplaceConnectorStatusPublished), nil
		},
	}

	h := NewMarketplaceHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/connectors/mock-crm/versions/1.0.0/publish", nil)
	req = withChiParam(req, "id", "mock-crm")
	req = withChiParam(req, "version", "1.0.0")
	rr := httptest.NewRecorder()

	h.Publish(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func validMarketplaceManifest(id, version string) connectorsdk.ConnectorManifest {
	return connectorsdk.ConnectorManifest{
		ID:          id,
		Name:        "Mock CRM",
		Version:     version,
		Description: "Mock CRM connector",
		Auth: connectorsdk.AuthConfig{
			Type: connectorsdk.AuthTypeAPIKey,
			APIKey: &connectorsdk.APIKeyConfig{
				HeaderName: "Authorization",
				Prefix:     "Bearer",
			},
		},
		Sync: connectorsdk.SyncConfig{
			SupportedModes: []connectorsdk.SyncMode{connectorsdk.SyncModeFull, connectorsdk.SyncModeIncremental},
			DefaultMode:    connectorsdk.SyncModeFull,
			Resources: []connectorsdk.ResourceConfig{
				{Name: "customers", Description: "CRM customers", Required: true},
			},
		},
	}
}

func marketplaceConnector(id, version, status string) *repository.MarketplaceConnector {
	return &repository.MarketplaceConnector{
		ID:          id,
		Version:     version,
		DeveloperID: uuid.New(),
		Name:        "Mock CRM",
		Description: "Mock CRM connector",
		Manifest:    validMarketplaceManifest(id, version),
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
