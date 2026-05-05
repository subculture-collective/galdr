package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/service"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

// IntegrationZendeskHandler provides Zendesk integration HTTP endpoints.
type IntegrationZendeskHandler struct {
	oauthSvc *service.ZendeskOAuthService
	syncer   service.ConnectorSyncer
}

// NewIntegrationZendeskHandler creates a new IntegrationZendeskHandler.
func NewIntegrationZendeskHandler(oauthSvc *service.ZendeskOAuthService, syncer service.ConnectorSyncer) *IntegrationZendeskHandler {
	return &IntegrationZendeskHandler{oauthSvc: oauthSvc, syncer: syncer}
}

// Connect handles GET /api/v1/integrations/zendesk/connect?subdomain=acme.
func (h *IntegrationZendeskHandler) Connect(w http.ResponseWriter, r *http.Request) {
	orgID, ok := integrationOrgID(w, r)
	if !ok {
		return
	}
	connectURL, err := h.oauthSvc.ConnectURL(orgID, r.URL.Query().Get("subdomain"))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": connectURL})
}

// Callback handles GET /api/v1/integrations/zendesk/callback.
func (h *IntegrationZendeskHandler) Callback(w http.ResponseWriter, r *http.Request) {
	integrationCallback(w, r, "zendesk", "Zendesk", "Zendesk connected successfully. Initial sync started.", h.oauthSvc.ExchangeCode, h.runFullSync)
}

// Status handles GET /api/v1/integrations/zendesk/status.
func (h *IntegrationZendeskHandler) Status(w http.ResponseWriter, r *http.Request) {
	integrationStatus(w, r, func(ctx context.Context, orgID uuid.UUID) (any, error) {
		return h.oauthSvc.GetStatus(ctx, orgID)
	})
}

// Disconnect handles DELETE /api/v1/integrations/zendesk.
func (h *IntegrationZendeskHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	integrationDisconnect(w, r, h.oauthSvc.Disconnect, "Zendesk disconnected")
}

// TriggerSync handles POST /api/v1/integrations/zendesk/sync.
func (h *IntegrationZendeskHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	integrationTriggerSync(w, r, h.runFullSync, "Zendesk sync started")
}

func (h *IntegrationZendeskHandler) runFullSync(ctx context.Context, orgID uuid.UUID) {
	if h.syncer == nil {
		slog.Error("zendesk syncer is not configured")
		return
	}
	if _, err := h.syncer.Sync(ctx, "zendesk", orgID, connectorsdk.SyncModeFull, nil); err != nil {
		slog.Error("zendesk sync failed", "error", err)
	}
}
