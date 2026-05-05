package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

// MarketplaceHandler handles connector registration and discovery APIs.
type MarketplaceHandler struct {
	service marketplaceServicer
}

func NewMarketplaceHandler(service marketplaceServicer) *MarketplaceHandler {
	return &MarketplaceHandler{service: service}
}

func (h *MarketplaceHandler) Register(w http.ResponseWriter, r *http.Request) {
	developerID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	var req service.RegisterConnectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
		return
	}

	connector, err := h.service.Register(r.Context(), developerID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, connector)
}

func (h *MarketplaceHandler) ListPublished(w http.ResponseWriter, r *http.Request) {
	connectors, err := h.service.ListPublished(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connectors": connectors})
}

func (h *MarketplaceHandler) GetPublished(w http.ResponseWriter, r *http.Request) {
	id := connectorIDParam(r)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("connector id is required"))
		return
	}

	connector, err := h.service.GetPublished(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, connector)
}

func (h *MarketplaceHandler) Install(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}
	id := connectorIDParam(r)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("connector id is required"))
		return
	}

	var req service.InstallConnectorRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			writeJSON(w, http.StatusBadRequest, errorResponse("invalid request body"))
			return
		}
	}

	installation, err := h.service.Install(r.Context(), orgID, id, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, installation)
}

func connectorIDParam(r *http.Request) string {
	return strings.TrimSpace(chi.URLParam(r, "id"))
}
