package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// SavedViewHandler provides customer saved-view endpoints.
type SavedViewHandler struct {
	viewService savedViewServicer
}

// NewSavedViewHandler creates a new SavedViewHandler.
func NewSavedViewHandler(viewService savedViewServicer) *SavedViewHandler {
	return &SavedViewHandler{viewService: viewService}
}

// List handles GET /api/v1/customers/saved-views.
func (h *SavedViewHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := orgUserFromContext(w, r)
	if !ok {
		return
	}

	views, err := h.viewService.List(r.Context(), orgID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"views": views})
}

// Get handles GET /api/v1/customers/saved-views/{id}.
func (h *SavedViewHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := orgUserFromContext(w, r)
	if !ok {
		return
	}
	id, ok := savedViewIDFromRequest(w, r)
	if !ok {
		return
	}

	view, err := h.viewService.GetByID(r.Context(), id, orgID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// Create handles POST /api/v1/customers/saved-views.
func (h *SavedViewHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := orgUserFromContext(w, r)
	if !ok {
		return
	}

	var req service.CreateSavedViewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	view, err := h.viewService.Create(r.Context(), orgID, userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// Update handles PATCH /api/v1/customers/saved-views/{id}.
func (h *SavedViewHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := orgUserFromContext(w, r)
	if !ok {
		return
	}
	id, ok := savedViewIDFromRequest(w, r)
	if !ok {
		return
	}

	var req service.UpdateSavedViewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	view, err := h.viewService.Update(r.Context(), id, orgID, userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// Delete handles DELETE /api/v1/customers/saved-views/{id}.
func (h *SavedViewHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := orgUserFromContext(w, r)
	if !ok {
		return
	}
	id, ok := savedViewIDFromRequest(w, r)
	if !ok {
		return
	}

	if err := h.viewService.Delete(r.Context(), id, orgID, userID); err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func orgUserFromContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return uuid.Nil, uuid.Nil, false
	}
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, userID, true
}

func savedViewIDFromRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid saved view ID"))
		return uuid.Nil, false
	}
	return id, true
}
