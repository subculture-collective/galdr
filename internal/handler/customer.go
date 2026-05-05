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
		OrgID:   orgID,
		Page:    page,
		PerPage: perPage,
		Sort:    q.Get("sort"),
		Order:   q.Get("order"),
		Risk:    q.Get("risk"),
		Search:  q.Get("search"),
		Source:  q.Get("source"),
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

func parseCustomerID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}
