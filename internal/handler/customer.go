package handler

import (
	"encoding/json"
	"errors"
	"io"
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
	insightService  customerInsightServicer
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(cs customerServicer) *CustomerHandler {
	return &CustomerHandler{customerService: cs}
}

// NewCustomerHandlerWithInsights creates a CustomerHandler with AI insight endpoints enabled.
func NewCustomerHandlerWithInsights(cs customerServicer, insights customerInsightServicer) *CustomerHandler {
	return &CustomerHandler{customerService: cs, insightService: insights}
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
		OrgID:    orgID,
		Page:     page,
		PerPage:  perPage,
		Sort:     q.Get("sort"),
		Order:    q.Get("order"),
		Risk:     q.Get("risk"),
		Search:   q.Get("search"),
		Source:   q.Get("source"),
		Assignee: q.Get("assignee"),
	}
	if params.Assignee == "me" {
		if userID, ok := auth.GetUserID(r.Context()); ok {
			params.AssigneeUserID = userID
		}
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

	customerID, err := parseCustomerID(r)
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

// ListEvents handles GET /api/v1/customers/{id}/events.
func (h *CustomerHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerID, err := parseCustomerID(r)
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

// ListAssignments handles GET /api/v1/customers/{id}/assignments.
func (h *CustomerHandler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerID, ok := parseUUIDParam(w, r, "id", "invalid customer ID")
	if !ok {
		return
	}

	resp, err := h.customerService.ListAssignments(r.Context(), customerID, orgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// AssignCustomer handles POST /api/v1/customers/{id}/assignments.
func (h *CustomerHandler) AssignCustomer(w http.ResponseWriter, r *http.Request) {
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

	var req service.CustomerAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	resp, err := h.customerService.AssignCustomer(r.Context(), customerID, orgID, req.UserID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// UnassignCustomer handles DELETE /api/v1/customers/{id}/assignments/{userID}.
func (h *CustomerHandler) UnassignCustomer(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerID, ok := parseUUIDParam(w, r, "id", "invalid customer ID")
	if !ok {
		return
	}
	assigneeID, ok := parseUUIDParam(w, r, "userID", "invalid user ID")
	if !ok {
		return
	}

	if err := h.customerService.UnassignCustomer(r.Context(), customerID, orgID, assigneeID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

// GenerateInsight handles POST /api/v1/customers/{id}/insights.
func (h *CustomerHandler) GenerateInsight(w http.ResponseWriter, r *http.Request) {
	if h.insightService == nil {
		writeJSON(w, http.StatusNotFound, errorResponse("insights not configured"))
		return
	}
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerID, err := parseCustomerID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid customer ID"))
		return
	}

	var opts service.InsightGenerationOptions
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&opts); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
			return
		}
	}
	opts.Trigger = service.InsightTriggerManual

	resp, err := h.insightService.GenerateCustomerInsight(r.Context(), orgID, customerID, opts)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	status := http.StatusCreated
	if resp != nil && resp.Cached {
		status = http.StatusOK
	}
	writeJSON(w, status, resp)
}

// ListInsights handles GET /api/v1/customers/{id}/insights.
func (h *CustomerHandler) ListInsights(w http.ResponseWriter, r *http.Request) {
	if h.insightService == nil {
		writeJSON(w, http.StatusNotFound, errorResponse("insights not configured"))
		return
	}
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	customerID, err := parseCustomerID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid customer ID"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	insights, err := h.insightService.ListCustomerInsights(r.Context(), orgID, customerID, limit)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"insights": insights})
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

func parseCustomerID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}
