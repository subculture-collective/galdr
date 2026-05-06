package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/service"
)

type mockBenchmarkService struct {
	compareFn func(ctx context.Context, orgID uuid.UUID, industry, companySizeBucket string) (*service.BenchmarkComparisonResponse, error)
}

func (m *mockBenchmarkService) Compare(ctx context.Context, orgID uuid.UUID, industry, companySizeBucket string) (*service.BenchmarkComparisonResponse, error) {
	return m.compareFn(ctx, orgID, industry, companySizeBucket)
}

func TestBenchmarkCompareUnauthorized(t *testing.T) {
	h := NewBenchmarkHandler(&mockBenchmarkService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/benchmarks", nil)
	rr := httptest.NewRecorder()

	h.Compare(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestBenchmarkCompareSuccess(t *testing.T) {
	orgID := uuid.New()
	mock := &mockBenchmarkService{compareFn: func(ctx context.Context, gotOrgID uuid.UUID, industry, companySizeBucket string) (*service.BenchmarkComparisonResponse, error) {
		if gotOrgID != orgID {
			t.Fatalf("expected org %s, got %s", orgID, gotOrgID)
		}
		if industry != "SaaS" || companySizeBucket != "51-200" {
			t.Fatalf("expected query params, got %s %s", industry, companySizeBucket)
		}
		percentile := 78.0
		return &service.BenchmarkComparisonResponse{
			Participating: true,
			Industry:      "saas",
			Size:          "51-200",
			Percentile:    &percentile,
			Metrics: []service.BenchmarkMetricResponse{{
				Key:         "health_score",
				Label:       "Avg health score",
				Unit:        "score",
				YourValue:   82,
				Percentile:  &percentile,
				Benchmarks:  &service.BenchmarkPercentiles{P25: 61, P50: 70, P75: 79},
				SampleCount: 42,
			}},
		}, nil
	}}
	h := NewBenchmarkHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/benchmarks?industry=SaaS&size=51-200", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.Compare(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var response service.BenchmarkComparisonResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Participating || len(response.Metrics) != 1 {
		t.Fatalf("unexpected benchmark response: %+v", response)
	}
}
