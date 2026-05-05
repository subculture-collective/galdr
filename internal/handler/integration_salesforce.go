package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"

	"github.com/onnwee/pulse-score/internal/service"
)

// IntegrationSalesforceHandler handles Salesforce-specific integration routes.
type IntegrationSalesforceHandler struct {
	oauthSvc *service.SalesforceOAuthService
	syncer   service.ConnectorSyncer
}

// NewIntegrationSalesforceHandler creates a Salesforce integration handler.
func NewIntegrationSalesforceHandler(oauthSvc *service.SalesforceOAuthService, syncer service.ConnectorSyncer) *IntegrationSalesforceHandler {
	return &IntegrationSalesforceHandler{oauthSvc: oauthSvc, syncer: syncer}
}

func (h *IntegrationSalesforceHandler) Connect(w http.ResponseWriter, r *http.Request) {
	integrationConnect(w, r, h.oauthSvc.ConnectURL)
}

func (h *IntegrationSalesforceHandler) Callback(w http.ResponseWriter, r *http.Request) {
	integrationCallback(w, r, "salesforce", "Salesforce", "Salesforce connected successfully. Initial sync started.", h.oauthSvc.ExchangeCode, h.runFullSync)
}

func (h *IntegrationSalesforceHandler) Status(w http.ResponseWriter, r *http.Request) {
	integrationStatus(w, r, func(ctx context.Context, orgID uuid.UUID) (any, error) {
		return h.oauthSvc.GetStatus(ctx, orgID)
	})
}

func (h *IntegrationSalesforceHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	integrationDisconnect(w, r, h.oauthSvc.Disconnect, "Salesforce disconnected")
}

func (h *IntegrationSalesforceHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	integrationTriggerSync(w, r, h.runFullSync, "Salesforce sync started")
}

func (h *IntegrationSalesforceHandler) runFullSync(ctx context.Context, orgID uuid.UUID) {
	if h.syncer == nil {
		slog.Error("salesforce syncer not configured", "org_id", orgID)
		return
	}
	if _, err := h.syncer.Sync(ctx, "salesforce", orgID, connectorsdk.SyncModeFull, nil); err != nil {
		slog.Error("salesforce full sync failed", "org_id", orgID, "error", err)
	}
}
