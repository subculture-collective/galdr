package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	core "github.com/onnwee/pulse-score/internal/service"
	billing "github.com/onnwee/pulse-score/internal/service/billing"
)

type billingCheckoutServicer interface {
	CreateCheckoutSession(ctx context.Context, orgID, userID uuid.UUID, req billing.CreateCheckoutSessionRequest) (*billing.CreateCheckoutSessionResponse, error)
}

type billingPortalServicer interface {
	CreatePortalSession(ctx context.Context, orgID uuid.UUID) (*billing.PortalSessionResponse, error)
	CancelAtPeriodEnd(ctx context.Context, orgID uuid.UUID) error
}

type billingSubscriptionServicer interface {
	GetSubscriptionSummary(ctx context.Context, orgID uuid.UUID) (*billing.SubscriptionSummary, error)
}

type billingPlanChangeServicer interface {
	ChangePlan(ctx context.Context, orgID, userID uuid.UUID, req billing.ChangePlanRequest) (*billing.ChangePlanResponse, error)
}

type billingWebhookServicer interface {
	HandleEvent(ctx context.Context, payload []byte, sigHeader string) error
}

// BillingHandler provides PulseScore billing endpoints.
type BillingHandler struct {
	checkoutSvc     billingCheckoutServicer
	portalSvc       billingPortalServicer
	subscriptionSvc billingSubscriptionServicer
	planChangeSvc   billingPlanChangeServicer
}

func NewBillingHandler(
	checkoutSvc billingCheckoutServicer,
	portalSvc billingPortalServicer,
	subscriptionSvc billingSubscriptionServicer,
	planChangeSvc billingPlanChangeServicer,
) *BillingHandler {
	return &BillingHandler{
		checkoutSvc:     checkoutSvc,
		portalSvc:       portalSvc,
		subscriptionSvc: subscriptionSvc,
		planChangeSvc:   planChangeSvc,
	}
}

// CreateCheckout handles POST /api/v1/billing/checkout.
func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req billing.CreateCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.checkoutSvc.CreateCheckoutSession(r.Context(), orgID, userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreatePortalSession handles POST /api/v1/billing/portal-session.
func (h *BillingHandler) CreatePortalSession(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	resp, err := h.portalSvc.CreatePortalSession(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetSubscription handles GET /api/v1/billing/subscription.
func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	resp, err := h.subscriptionSvc.GetSubscriptionSummary(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// CancelSubscription handles POST /api/v1/billing/cancel.
func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	if err := h.portalSvc.CancelAtPeriodEnd(r.Context(), orgID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancel_at_period_end"})
}

// ChangePlan handles POST /api/v1/billing/change-plan.
func (h *BillingHandler) ChangePlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req billing.ChangePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.planChangeSvc.ChangePlan(r.Context(), orgID, userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// WebhookStripeBillingHandler handles Stripe billing webhooks.
type WebhookStripeBillingHandler struct {
	webhookSvc billingWebhookServicer
}

func NewWebhookStripeBillingHandler(webhookSvc billingWebhookServicer) *WebhookStripeBillingHandler {
	return &WebhookStripeBillingHandler{webhookSvc: webhookSvc}
}

// HandleWebhook handles POST /api/v1/webhooks/stripe-billing.
func (h *WebhookStripeBillingHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
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
		var validationErr *core.ValidationError
		if errors.As(err, &validationErr) {
			writeJSON(w, http.StatusBadRequest, errorResponse(validationErr.Message))
			return
		}

		slog.Error("billing webhook processing error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
