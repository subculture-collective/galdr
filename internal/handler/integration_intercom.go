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

// IntegrationIntercomHandler provides Intercom integration HTTP endpoints.
type IntegrationIntercomHandler struct {
	oauthSvc *service.IntercomOAuthService
	syncer   service.ConnectorSyncer
}

// NewIntegrationIntercomHandler creates a new IntegrationIntercomHandler.
func NewIntegrationIntercomHandler(oauthSvc *service.IntercomOAuthService, syncer service.ConnectorSyncer) *IntegrationIntercomHandler {
	return &IntegrationIntercomHandler{
		oauthSvc: oauthSvc,
		syncer:   syncer,
	}
}

// Connect handles GET /api/v1/integrations/intercom/connect.
func (h *IntegrationIntercomHandler) Connect(w http.ResponseWriter, r *http.Request) {
	integrationConnect(w, r, h.oauthSvc.ConnectURL)
}

// Callback handles GET /api/v1/integrations/intercom/callback.
func (h *IntegrationIntercomHandler) Callback(w http.ResponseWriter, r *http.Request) {
	integrationCallback(
		w,
		r,
		"intercom",
		"Intercom",
		"Intercom connected successfully. Initial sync started.",
		h.oauthSvc.ExchangeCode,
		h.runFullSync,
	)
}

// Status handles GET /api/v1/integrations/intercom/status.
func (h *IntegrationIntercomHandler) Status(w http.ResponseWriter, r *http.Request) {
	integrationStatus(w, r, func(ctx context.Context, orgID uuid.UUID) (any, error) {
		return h.oauthSvc.GetStatus(ctx, orgID)
	})
}

// Disconnect handles DELETE /api/v1/integrations/intercom.
func (h *IntegrationIntercomHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	integrationDisconnect(w, r, h.oauthSvc.Disconnect, "Intercom disconnected")
}

// TriggerSync handles POST /api/v1/integrations/intercom/sync.
func (h *IntegrationIntercomHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	integrationTriggerSync(
		w,
		r,
		h.runFullSync,
		"Intercom sync started",
	)
}

func (h *IntegrationIntercomHandler) runFullSync(ctx context.Context, orgID uuid.UUID) {
	if h.syncer == nil {
		slog.Error("intercom syncer is not configured")
		return
	}
	if _, err := h.syncer.Sync(ctx, "intercom", orgID, connectorsdk.SyncModeFull, nil); err != nil {
		slog.Error("intercom sync failed", "error", err)
	}
}

// WebhookIntercomHandler provides Intercom webhook HTTP endpoints.
type WebhookIntercomHandler struct {
	webhookSvc *service.IntercomWebhookService
}

// NewWebhookIntercomHandler creates a new WebhookIntercomHandler.
func NewWebhookIntercomHandler(webhookSvc *service.IntercomWebhookService) *WebhookIntercomHandler {
	return &WebhookIntercomHandler{webhookSvc: webhookSvc}
}

// HandleWebhook handles POST /api/v1/webhooks/intercom.
func (h *WebhookIntercomHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, webhookMaxBodyBytes)

	payload, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	sigHeader := r.Header.Get("X-Hub-Signature")

	if err := h.webhookSvc.VerifySignature(payload, sigHeader); err != nil {
		slog.Warn("intercom webhook signature verification failed", "error", err)
		writeJSON(w, http.StatusUnauthorized, errorResponse("invalid signature"))
		return
	}

	var event service.IntercomWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	if err := h.webhookSvc.ProcessEvent(r.Context(), event); err != nil {
		slog.Error("intercom webhook processing error", "error", err)
	}

	// Always return 200 to prevent Intercom retries
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
