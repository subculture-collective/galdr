package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

// CustomerHandler provides customer HTTP endpoints.
type CustomerHandler struct {
	customerService customerServicer
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(cs customerServicer) *CustomerHandler {
	return &CustomerHandler{customerService: cs}
}

// List handles GET /api/v1/customers.
func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	params := repository.CustomerListParams{
		OrgID:     orgID,
		Page:      page,
		PerPage:   perPage,
		Sort:      q.Get("sort"),
		Order:     q.Get("order"),
		Risk:      q.Get("risk"),
		ChurnRisk: q.Get("churn_risk"),
		Search:    q.Get("search"),
		Source:    q.Get("source"),
	}

	resp, err := h.customerService.List(r.Context(), params)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetDetail handles GET /api/v1/customers/{id}.
func (h *CustomerHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerIDStr := chi.URLParam(r, "id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid customer ID"))
		return
	}

	detail, err := h.customerService.GetDetail(r.Context(), customerID, orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// GetChurnPrediction handles GET /api/v1/customers/{id}/churn-prediction.
func (h *CustomerHandler) GetChurnPrediction(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerIDStr := chi.URLParam(r, "id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid customer ID"))
		return
	}

	prediction, err := h.customerService.GetChurnPrediction(r.Context(), customerID, orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, prediction)
}

// ListEvents handles GET /api/v1/customers/{id}/events.
func (h *CustomerHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerIDStr := chi.URLParam(r, "id")
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid customer ID"))
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	params := repository.EventListParams{
		CustomerID: customerID,
		OrgID:      orgID,
		Page:       page,
		PerPage:    perPage,
		EventType:  q.Get("type"),
	}

	if fromStr := q.Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			params.From = t
		}
	}
	if toStr := q.Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			params.To = t
		}
	}

	resp, err := h.customerService.ListEvents(r.Context(), params)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListNotes handles GET /api/v1/customers/{id}/notes.
func (h *CustomerHandler) ListNotes(w http.ResponseWriter, r *http.Request) {
	orgID, userID, role, ok := notesActorContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerID, ok := parseUUIDParam(w, r, "id", "invalid customer ID")
	if !ok {
		return
	}

	resp, err := h.customerService.ListNotes(r.Context(), customerID, orgID, userID, role)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateNote handles POST /api/v1/customers/{id}/notes.
func (h *CustomerHandler) CreateNote(w http.ResponseWriter, r *http.Request) {
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

	customerID, ok := parseUUIDParam(w, r, "id", "invalid customer ID")
	if !ok {
		return
	}

	var req service.CustomerNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.customerService.CreateNote(r.Context(), customerID, orgID, userID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// UpdateNote handles PUT /api/v1/customers/{id}/notes/{noteID}.
func (h *CustomerHandler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	orgID, userID, role, ok := notesActorContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerID, ok := parseUUIDParam(w, r, "id", "invalid customer ID")
	if !ok {
		return
	}
	noteID, ok := parseUUIDParam(w, r, "noteID", "invalid note ID")
	if !ok {
		return
	}

	var req service.CustomerNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.customerService.UpdateNote(r.Context(), customerID, noteID, orgID, userID, role, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// DeleteNote handles DELETE /api/v1/customers/{id}/notes/{noteID}.
func (h *CustomerHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	orgID, userID, role, ok := notesActorContext(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerID, ok := parseUUIDParam(w, r, "id", "invalid customer ID")
	if !ok {
		return
	}
	noteID, ok := parseUUIDParam(w, r, "noteID", "invalid note ID")
	if !ok {
		return
	}

	if err := h.customerService.DeleteNote(r.Context(), customerID, noteID, orgID, userID, role); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func notesActorContext(r *http.Request) (uuid.UUID, uuid.UUID, string, bool) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		return uuid.Nil, uuid.Nil, "", false
	}
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		return uuid.Nil, uuid.Nil, "", false
	}
	role, ok := auth.GetRole(r.Context())
	if !ok {
		return uuid.Nil, uuid.Nil, "", false
	}
	return orgID, userID, role, true
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, name, msg string) (uuid.UUID, bool) {
	value, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse(msg))
		return uuid.Nil, false
	}
	return value, true
}
