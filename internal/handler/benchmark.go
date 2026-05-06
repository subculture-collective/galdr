package handler

import (
	"net/http"

	"github.com/onnwee/pulse-score/internal/auth"
)

type BenchmarkHandler struct {
	benchmarks benchmarkServicer
}

func NewBenchmarkHandler(benchmarks benchmarkServicer) *BenchmarkHandler {
	return &BenchmarkHandler{benchmarks: benchmarks}
}

func (h *BenchmarkHandler) Compare(w http.ResponseWriter, r *http.Request) {
	orgID, ok := auth.GetOrgID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse("unauthorized"))
		return
	}

	response, err := h.benchmarks.Compare(
		r.Context(),
		orgID,
		r.URL.Query().Get("industry"),
		r.URL.Query().Get("size"),
	)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}
