package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/service"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

// IntegrationHubSpotHandler provides HubSpot integration HTTP endpoints.
type IntegrationHubSpotHandler struct {
	oauthSvc *service.HubSpotOAuthService
	syncer   service.ConnectorSyncer
}

// NewIntegrationHubSpotHandler creates a new IntegrationHubSpotHandler.
func NewIntegrationHubSpotHandler(oauthSvc *service.HubSpotOAuthService, syncer service.ConnectorSyncer) *IntegrationHubSpotHandler {
	return &IntegrationHubSpotHandler{
		oauthSvc: oauthSvc,
		syncer:   syncer,
	}
}

// Connect handles GET /api/v1/integrations/hubspot/connect.
func (h *IntegrationHubSpotHandler) Connect(w http.ResponseWriter, r *http.Request) {
	integrationConnect(w, r, h.oauthSvc.ConnectURL)
}

// Callback handles GET /api/v1/integrations/hubspot/callback.
func (h *IntegrationHubSpotHandler) Callback(w http.ResponseWriter, r *http.Request) {
	integrationCallback(
		w,
		r,
		"hubspot",
		"HubSpot",
		"HubSpot connected successfully. Initial sync started.",
		h.oauthSvc.ExchangeCode,
		h.runFullSync,
	)
}

// Status handles GET /api/v1/integrations/hubspot/status.
func (h *IntegrationHubSpotHandler) Status(w http.ResponseWriter, r *http.Request) {
	integrationStatus(w, r, func(ctx context.Context, orgID uuid.UUID) (any, error) {
		return h.oauthSvc.GetStatus(ctx, orgID)
	})
}

// Disconnect handles DELETE /api/v1/integrations/hubspot.
func (h *IntegrationHubSpotHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	integrationDisconnect(w, r, h.oauthSvc.Disconnect, "HubSpot disconnected")
}

// TriggerSync handles POST /api/v1/integrations/hubspot/sync.
func (h *IntegrationHubSpotHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	integrationTriggerSync(
		w,
		r,
		h.runFullSync,
		"HubSpot sync started",
	)
}

func (h *IntegrationHubSpotHandler) runFullSync(ctx context.Context, orgID uuid.UUID) {
	if h.syncer == nil {
		slog.Error("hubspot syncer is not configured")
		return
	}
	if _, err := h.syncer.Sync(ctx, "hubspot", orgID, connectorsdk.SyncModeFull, nil); err != nil {
		slog.Error("hubspot sync failed", "error", err)
	}
}

// WebhookHubSpotHandler provides HubSpot webhook HTTP endpoints.
type WebhookHubSpotHandler struct {
	webhookSvc *service.HubSpotWebhookService
}

// NewWebhookHubSpotHandler creates a new WebhookHubSpotHandler.
func NewWebhookHubSpotHandler(webhookSvc *service.HubSpotWebhookService) *WebhookHubSpotHandler {
	return &WebhookHubSpotHandler{webhookSvc: webhookSvc}
}

// HandleWebhook handles POST /api/v1/webhooks/hubspot.
func (h *WebhookHubSpotHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, webhookMaxBodyBytes)

	payload, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	sigHeader := r.Header.Get("X-HubSpot-Signature-v3")
	timestamp := r.Header.Get("X-HubSpot-Request-Timestamp")

	if err := h.webhookSvc.VerifySignature(payload, sigHeader, timestamp, r.Method, r.URL.RequestURI()); err != nil {
		slog.Warn("hubspot webhook signature verification failed", "error", err)
		writeJSON(w, http.StatusUnauthorized, errorResponse("invalid signature"))
		return
	}

	var events []service.HubSpotWebhookEvent
	if err := json.Unmarshal(payload, &events); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	if err := h.webhookSvc.ProcessEvents(r.Context(), events); err != nil {
		slog.Error("hubspot webhook processing error", "error", err)
	}

	// Always return 200 to prevent HubSpot retries
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
