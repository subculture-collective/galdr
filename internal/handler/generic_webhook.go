package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

const genericWebhookSignatureHeader = "X-PulseScore-Signature"

// GenericWebhookHandler provides generic webhook config and receiver endpoints.
type GenericWebhookHandler struct {
	service genericWebhookServicer
}

// NewGenericWebhookHandler creates a GenericWebhookHandler.
func NewGenericWebhookHandler(service genericWebhookServicer) *GenericWebhookHandler {
	return &GenericWebhookHandler{service: service}
}

// Create handles POST /api/v1/integrations/generic-webhooks.
func (h *GenericWebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}
	var req service.GenericWebhookConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}
	cfg, err := h.service.Create(r.Context(), orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, cfg)
}

// List handles GET /api/v1/integrations/generic-webhooks.
func (h *GenericWebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}
	configs, err := h.service.List(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhooks": configs})
}

// Get handles GET /api/v1/integrations/generic-webhooks/{id}.
func (h *GenericWebhookHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, id, ok := h.orgAndWebhookID(w, r)
	if !ok {
		return
	}
	cfg, err := h.service.Get(r.Context(), orgID, id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// Update handles PATCH /api/v1/integrations/generic-webhooks/{id}.
func (h *GenericWebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, id, ok := h.orgAndWebhookID(w, r)
	if !ok {
		return
	}
	var req service.GenericWebhookConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}
	cfg, err := h.service.Update(r.Context(), orgID, id, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// Delete handles DELETE /api/v1/integrations/generic-webhooks/{id}.
func (h *GenericWebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, id, ok := h.orgAndWebhookID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), orgID, id); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Process handles POST /api/v1/webhooks/generic/{org_slug}/{webhook_id}.
func (h *GenericWebhookHandler) Process(w http.ResponseWriter, r *http.Request) {
	orgSlug := chi.URLParam(r, "org_slug")
	webhookID, err := uuid.Parse(chi.URLParam(r, "webhook_id"))
	if orgSlug == "" || err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid webhook route"))
		return
	}
	payload, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}
	result, err := h.service.Process(r.Context(), orgSlug, webhookID, payload, r.Header.Get(genericWebhookSignatureHeader))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Test handles POST /api/v1/integrations/generic-webhooks/test.
func (h *GenericWebhookHandler) Test(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}
	var req service.GenericWebhookTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}
	result, err := h.service.Test(r.Context(), orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *GenericWebhookHandler) orgAndWebhookID(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid webhook id"))
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, id, true
}
