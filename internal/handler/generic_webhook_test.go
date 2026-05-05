package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

type mockGenericWebhookService struct {
	createFn  func(ctx context.Context, orgID uuid.UUID, req service.GenericWebhookConfigRequest) (*repository.GenericWebhookConfig, error)
	processFn func(ctx context.Context, orgSlug string, webhookID uuid.UUID, payload []byte, signature string) (*service.GenericWebhookProcessResult, error)
	testFn    func(ctx context.Context, orgID uuid.UUID, req service.GenericWebhookTestRequest) (*service.GenericWebhookTestResult, error)
}

func (m *mockGenericWebhookService) Create(ctx context.Context, orgID uuid.UUID, req service.GenericWebhookConfigRequest) (*repository.GenericWebhookConfig, error) {
	return m.createFn(ctx, orgID, req)
}

func (m *mockGenericWebhookService) List(ctx context.Context, orgID uuid.UUID) ([]*repository.GenericWebhookConfig, error) {
	return nil, nil
}

func (m *mockGenericWebhookService) Get(ctx context.Context, orgID, id uuid.UUID) (*repository.GenericWebhookConfig, error) {
	return nil, nil
}

func (m *mockGenericWebhookService) Update(ctx context.Context, orgID, id uuid.UUID, req service.GenericWebhookConfigRequest) (*repository.GenericWebhookConfig, error) {
	return nil, nil
}

func (m *mockGenericWebhookService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return nil
}

func (m *mockGenericWebhookService) Process(ctx context.Context, orgSlug string, webhookID uuid.UUID, payload []byte, signature string) (*service.GenericWebhookProcessResult, error) {
	return m.processFn(ctx, orgSlug, webhookID, payload, signature)
}

func (m *mockGenericWebhookService) Test(ctx context.Context, orgID uuid.UUID, req service.GenericWebhookTestRequest) (*service.GenericWebhookTestResult, error) {
	return m.testFn(ctx, orgID, req)
}

func TestGenericWebhookHandlerCreateConfig(t *testing.T) {
	orgID := uuid.New()
	webhookID := uuid.New()
	mock := &mockGenericWebhookService{
		createFn: func(ctx context.Context, gotOrgID uuid.UUID, req service.GenericWebhookConfigRequest) (*repository.GenericWebhookConfig, error) {
			if gotOrgID != orgID {
				t.Fatalf("expected org %s, got %s", orgID, gotOrgID)
			}
			if req.Name != "Billing feed" || req.Secret != "top-secret" || req.FieldMapping["customer_email"] != "$.data.email" {
				t.Fatalf("request not decoded: %+v", req)
			}
			return &repository.GenericWebhookConfig{ID: webhookID, OrgID: orgID, Name: req.Name, FieldMapping: req.FieldMapping, IsActive: true}, nil
		},
	}
	h := NewGenericWebhookHandler(mock)
	body := []byte(`{"name":"Billing feed","secret":"top-secret","field_mapping":{"customer_email":"$.data.email","event_type":"$.event"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/generic-webhooks", bytes.NewReader(body))
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestGenericWebhookHandlerProcessUsesSignatureHeader(t *testing.T) {
	webhookID := uuid.New()
	mock := &mockGenericWebhookService{
		processFn: func(ctx context.Context, orgSlug string, gotWebhookID uuid.UUID, payload []byte, signature string) (*service.GenericWebhookProcessResult, error) {
			if orgSlug != "acme" || gotWebhookID != webhookID || signature != "sha256=abc" || string(payload) != `{"event":"login"}` {
				t.Fatalf("unexpected process args slug=%s id=%s sig=%s payload=%s", orgSlug, gotWebhookID, signature, payload)
			}
			return &service.GenericWebhookProcessResult{EventID: uuid.New(), CustomerID: uuid.New(), EventType: "login"}, nil
		},
	}
	h := NewGenericWebhookHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/generic/acme/"+webhookID.String(), bytes.NewReader([]byte(`{"event":"login"}`)))
	req.Header.Set("X-PulseScore-Signature", "sha256=abc")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org_slug", "acme")
	rctx.URLParams.Add("webhook_id", webhookID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.Process(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGenericWebhookHandlerTestMapping(t *testing.T) {
	orgID := uuid.New()
	mock := &mockGenericWebhookService{
		testFn: func(ctx context.Context, gotOrgID uuid.UUID, req service.GenericWebhookTestRequest) (*service.GenericWebhookTestResult, error) {
			if gotOrgID != orgID || req.FieldMapping["event_type"] != "$.event" {
				t.Fatalf("unexpected test args: org=%s req=%+v", gotOrgID, req)
			}
			return &service.GenericWebhookTestResult{Mapped: map[string]any{"event_type": "login"}}, nil
		},
	}
	h := NewGenericWebhookHandler(mock)
	body, _ := json.Marshal(service.GenericWebhookTestRequest{Payload: map[string]any{"event": "login"}, FieldMapping: map[string]string{"event_type": "$.event"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/generic-webhooks/test", bytes.NewReader(body))
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Test(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
