package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/service"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

// IntegrationStripeHandler provides Stripe integration HTTP endpoints.
type IntegrationStripeHandler struct {
	oauthSvc *service.StripeOAuthService
	syncer   service.ConnectorSyncer
}

// NewIntegrationStripeHandler creates a new IntegrationStripeHandler.
func NewIntegrationStripeHandler(oauthSvc *service.StripeOAuthService, syncer service.ConnectorSyncer) *IntegrationStripeHandler {
	return &IntegrationStripeHandler{
		oauthSvc: oauthSvc,
		syncer:   syncer,
	}
}

// Connect handles GET /api/v1/integrations/stripe/connect.
// Returns the OAuth URL to redirect the user to Stripe.
func (h *IntegrationStripeHandler) Connect(w http.ResponseWriter, r *http.Request) {
	integrationConnect(w, r, h.oauthSvc.ConnectURL)
}

// Callback handles GET /api/v1/integrations/stripe/callback.
// Exchanges the code for tokens and initiates a full sync.
func (h *IntegrationStripeHandler) Callback(w http.ResponseWriter, r *http.Request) {
	integrationCallback(
		w,
		r,
		"stripe",
		"Stripe",
		"Stripe connected successfully. Initial sync started.",
		h.oauthSvc.ExchangeCode,
		h.runFullSync,
	)
}

// Status handles GET /api/v1/integrations/stripe/status.
func (h *IntegrationStripeHandler) Status(w http.ResponseWriter, r *http.Request) {
	integrationStatus(w, r, func(ctx context.Context, orgID uuid.UUID) (any, error) {
		return h.oauthSvc.GetStatus(ctx, orgID)
	})
}

// Disconnect handles DELETE /api/v1/integrations/stripe.
func (h *IntegrationStripeHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	integrationDisconnect(w, r, h.oauthSvc.Disconnect, "Stripe disconnected")
}

// TriggerSync handles POST /api/v1/integrations/stripe/sync.
func (h *IntegrationStripeHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	integrationTriggerSync(
		w,
		r,
		h.runFullSync,
		"sync started",
	)
}

func (h *IntegrationStripeHandler) runFullSync(ctx context.Context, orgID uuid.UUID) {
	if h.syncer == nil {
		slog.Error("stripe syncer is not configured")
		return
	}
	if _, err := h.syncer.Sync(ctx, "stripe", orgID, connectorsdk.SyncModeFull, nil); err != nil {
		slog.Error("stripe sync failed", "error", err)
	}
}

// WebhookStripeHandler provides Stripe webhook HTTP endpoints.
type WebhookStripeHandler struct {
	webhookSvc *service.StripeWebhookService
}

// NewWebhookStripeHandler creates a new WebhookStripeHandler.
func NewWebhookStripeHandler(webhookSvc *service.StripeWebhookService) *WebhookStripeHandler {
	return &WebhookStripeHandler{webhookSvc: webhookSvc}
}

// HandleWebhook handles POST /api/v1/webhooks/stripe.
func (h *WebhookStripeHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, webhookMaxBodyBytes)

	payload, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	if sigHeader == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("missing Stripe-Signature header"))
		return
	}

	if err := h.webhookSvc.HandleEvent(r.Context(), payload, sigHeader); err != nil {
		slog.Error("webhook processing error", "error", err)
		// Still return 200 to prevent Stripe from retrying
		writeJSON(w, http.StatusOK, map[string]string{"status": "error logged"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// StripeStatusResponse is returned by GET /api/v1/integrations/stripe/status.
type StripeStatusResponse struct {
	Status        string `json:"status"`
	AccountID     string `json:"account_id,omitempty"`
	LastSyncAt    string `json:"last_sync_at,omitempty"`
	LastSyncError string `json:"last_sync_error,omitempty"`
	CustomerCount int    `json:"customer_count,omitempty"`
}
