package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

func TestZendeskConnectRequiresSubdomain(t *testing.T) {
	orgID := uuid.New()
	h := NewIntegrationZendeskHandler(
		service.NewZendeskOAuthService(service.ZendeskOAuthConfig{ClientID: "client"}, nil),
		nil,
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/zendesk/connect", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Connect(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestZendeskConnectSuccess(t *testing.T) {
	orgID := uuid.New()
	h := NewIntegrationZendeskHandler(
		service.NewZendeskOAuthService(service.ZendeskOAuthConfig{
			ClientID:         "client",
			OAuthRedirectURL: "http://localhost/callback",
		}, nil),
		nil,
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/zendesk/connect?subdomain=acme", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Connect(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
