package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

type playbookServicer interface {
	List(ctx context.Context, orgID uuid.UUID) ([]*service.PlaybookResponse, error)
	GetByID(ctx context.Context, id, orgID uuid.UUID) (*service.PlaybookResponse, error)
	Create(ctx context.Context, orgID uuid.UUID, req service.CreatePlaybookRequest) (*service.PlaybookResponse, error)
	Update(ctx context.Context, id, orgID uuid.UUID, req service.UpdatePlaybookRequest) (*service.PlaybookResponse, error)
	Delete(ctx context.Context, id, orgID uuid.UUID) error
	SetEnabled(ctx context.Context, id, orgID uuid.UUID, req service.SetPlaybookEnabledRequest) (*service.PlaybookResponse, error)
}

// PlaybookHandler provides playbook CRUD endpoints.
type PlaybookHandler struct {
	playbooks playbookServicer
}

func NewPlaybookHandler(playbooks playbookServicer) *PlaybookHandler {
	return &PlaybookHandler{playbooks: playbooks}
}

// List handles GET /api/v1/playbooks.
func (h *PlaybookHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromRequest(w, r)
	if !ok {
		return
	}
	playbooks, err := h.playbooks.List(r.Context(), orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"playbooks": playbooks})
}

// Get handles GET /api/v1/playbooks/{id}.
func (h *PlaybookHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, id, ok := orgIDAndPlaybookID(w, r)
	if !ok {
		return
	}
	playbook, err := h.playbooks.GetByID(r.Context(), id, orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, playbook)
}

// Create handles POST /api/v1/playbooks.
func (h *PlaybookHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromRequest(w, r)
	if !ok {
		return
	}
	var req service.CreatePlaybookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}
	playbook, err := h.playbooks.Create(r.Context(), orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, playbook)
}

// Update handles PUT /api/v1/playbooks/{id}.
func (h *PlaybookHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, id, ok := orgIDAndPlaybookID(w, r)
	if !ok {
		return
	}
	var req service.UpdatePlaybookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}
	playbook, err := h.playbooks.Update(r.Context(), id, orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, playbook)
}

// Delete handles DELETE /api/v1/playbooks/{id}.
func (h *PlaybookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, id, ok := orgIDAndPlaybookID(w, r)
	if !ok {
		return
	}
	if err := h.playbooks.Delete(r.Context(), id, orgID); err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// SetEnabled handles PUT /api/v1/playbooks/{id}/enable.
func (h *PlaybookHandler) SetEnabled(w http.ResponseWriter, r *http.Request) {
	orgID, id, ok := orgIDAndPlaybookID(w, r)
	if !ok {
		return
	}
	var req service.SetPlaybookEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}
	playbook, err := h.playbooks.SetEnabled(r.Context(), id, orgID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, playbook)
}

func orgIDAndPlaybookID(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	orgID, ok := orgIDFromRequest(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid playbook ID"))
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, id, true
}

func orgIDFromRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return uuid.Nil, false
	}
	return orgID, true
}
