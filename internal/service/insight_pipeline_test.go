package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service/prompts"
)

func TestInsightPipelineGeneratesStoresAndCachesCustomerInsight(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	orgID := uuid.New()
	customerID := uuid.New()
	llm := &fakeInsightLLM{text: validInsightJSON("customer_analysis")}
	insights := &fakeInsightStore{}
	p := newTestInsightPipeline(now, llm, insights, orgID, customerID)

	first, err := p.GenerateCustomerInsight(ctx, orgID, customerID, InsightGenerationOptions{})
	if err != nil {
		t.Fatalf("generate insight: %v", err)
	}
	if first.Cached {
		t.Fatal("first insight should not be cached")
	}
	if first.Insight.Model != "gpt-4o-mini" {
		t.Fatalf("expected model from LLM provider, got %q", first.Insight.Model)
	}
	if llm.calls != 1 || len(insights.saved) != 1 {
		t.Fatalf("expected one LLM call and one save, got calls=%d saves=%d", llm.calls, len(insights.saved))
	}

	second, err := p.GenerateCustomerInsight(ctx, orgID, customerID, InsightGenerationOptions{})
	if err != nil {
		t.Fatalf("generate cached insight: %v", err)
	}
	if !second.Cached {
		t.Fatal("second insight should use recent cached insight")
	}
	if llm.calls != 1 || len(insights.saved) != 1 {
		t.Fatalf("cache miss caused extra work: calls=%d saves=%d", llm.calls, len(insights.saved))
	}

	_, err = p.GenerateCustomerInsight(ctx, orgID, customerID, InsightGenerationOptions{Force: true})
	if err != nil {
		t.Fatalf("force generate insight: %v", err)
	}
	if llm.calls != 2 || len(insights.saved) != 2 {
		t.Fatalf("force should bypass cache: calls=%d saves=%d", llm.calls, len(insights.saved))
	}
}

func TestInsightPipelineProcessesBatchAtRiskCustomersFirst(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	orgID := uuid.New()
	redCustomer := uuid.New()
	yellowCustomer := uuid.New()
	greenCustomer := uuid.New()
	llm := &fakeInsightLLM{text: validInsightJSON("customer_analysis")}
	insights := &fakeInsightStore{}
	p := newTestInsightPipeline(now, llm, insights, orgID, redCustomer)
	p.health.scores = []*repository.HealthScore{
		{OrgID: orgID, CustomerID: redCustomer, OverallScore: 20, RiskLevel: "red", Factors: map[string]float64{"payment_recency": 0.2}, CalculatedAt: now},
		{OrgID: orgID, CustomerID: yellowCustomer, OverallScore: 55, RiskLevel: "yellow", Factors: map[string]float64{"payment_recency": 0.5}, CalculatedAt: now},
		{OrgID: orgID, CustomerID: greenCustomer, OverallScore: 85, RiskLevel: "green", Factors: map[string]float64{"payment_recency": 0.9}, CalculatedAt: now},
	}
	p.customers.byID[yellowCustomer] = &repository.Customer{ID: yellowCustomer, OrgID: orgID, Name: "Beta", Source: "stripe", CreatedAt: now.Add(-48 * time.Hour)}
	p.customers.byID[greenCustomer] = &repository.Customer{ID: greenCustomer, OrgID: orgID, Name: "Gamma", Source: "stripe", CreatedAt: now.Add(-48 * time.Hour)}

	resp, err := p.ProcessBatch(ctx, orgID, 2)
	if err != nil {
		t.Fatalf("process batch: %v", err)
	}
	if resp.Processed != 2 || resp.Failed != 0 {
		t.Fatalf("expected two processed, got %+v", resp)
	}
	if len(insights.saved) != 2 || insights.saved[0].CustomerID != redCustomer || insights.saved[1].CustomerID != yellowCustomer {
		t.Fatalf("expected red then yellow processing, got %+v", insights.saved)
	}
}

func TestInsightPipelineShouldGenerateForScoreChange(t *testing.T) {
	prev := &repository.HealthScore{OverallScore: 72, RiskLevel: "green"}
	curr := &repository.HealthScore{OverallScore: 60, RiskLevel: "yellow"}
	if trigger, ok := InsightTriggerForScoreChange(prev, curr, 10); !ok || trigger != InsightTriggerRiskLevelChanged {
		t.Fatalf("expected risk transition trigger, got trigger=%q ok=%v", trigger, ok)
	}

	curr = &repository.HealthScore{OverallScore: 61, RiskLevel: "green"}
	if trigger, ok := InsightTriggerForScoreChange(prev, curr, 10); !ok || trigger != InsightTriggerScoreDrop {
		t.Fatalf("expected score drop trigger, got trigger=%q ok=%v", trigger, ok)
	}

	curr = &repository.HealthScore{OverallScore: 65, RiskLevel: "green"}
	if _, ok := InsightTriggerForScoreChange(prev, curr, 10); ok {
		t.Fatal("small score movement should not trigger insight generation")
	}
}

func TestInsightPipelineGeneratesForScoreChangeTrigger(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	orgID := uuid.New()
	customerID := uuid.New()
	llm := &fakeInsightLLM{text: validInsightJSON("customer_analysis")}
	insights := &fakeInsightStore{}
	p := newTestInsightPipeline(now, llm, insights, orgID, customerID)
	previous := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 82, RiskLevel: "green"}
	current := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 62, RiskLevel: "yellow"}

	resp, triggered, err := p.GenerateForScoreChange(ctx, previous, current)
	if err != nil {
		t.Fatalf("generate for score change: %v", err)
	}
	if !triggered {
		t.Fatal("expected score change to trigger insight generation")
	}
	if resp == nil || resp.Insight == nil || len(insights.saved) != 1 || llm.calls != 1 {
		t.Fatalf("expected one generated insight, resp=%+v calls=%d saves=%d", resp, llm.calls, len(insights.saved))
	}
}

func newTestInsightPipeline(now time.Time, llm *fakeInsightLLM, insights *fakeInsightStore, orgID, customerID uuid.UUID) *InsightPipeline {
	customers := &fakeInsightCustomers{byID: map[uuid.UUID]*repository.Customer{
		customerID: {ID: customerID, OrgID: orgID, Name: "Acme", Email: "a@example.com", CompanyName: "Acme", Source: "stripe", MRRCents: 10000, Currency: "usd", CreatedAt: now.Add(-72 * time.Hour)},
	}}
	health := &fakeInsightHealth{scores: []*repository.HealthScore{
		{OrgID: orgID, CustomerID: customerID, OverallScore: 41, RiskLevel: "yellow", Factors: map[string]float64{"payment_recency": 0.4}, CalculatedAt: now},
	}}
	events := &fakeInsightEvents{events: []*repository.CustomerEvent{
		{OrgID: orgID, CustomerID: customerID, EventType: "score.changed", Source: "health_scoring", OccurredAt: now.Add(-time.Hour), Data: map[string]any{"delta": -12}},
	}}
	return NewInsightPipeline(InsightPipelineDeps{
		Customers:    customers,
		HealthScores: health,
		Events:       events,
		Insights:     insights,
		LLM:          llm,
		Now:          func() time.Time { return now },
	})
}

type fakeInsightLLM struct {
	text  string
	calls int
}

func (f *fakeInsightLLM) Complete(ctx context.Context, req LLMCompletionRequest) (*LLMCompletionResponse, error) {
	f.calls++
	return &LLMCompletionResponse{Text: f.text, Provider: "gpt-4o-mini", InputTokens: 100, OutputTokens: 50, CostUSD: 0.001}, nil
}

type fakeInsightCustomers struct{ byID map[uuid.UUID]*repository.Customer }

func (f *fakeInsightCustomers) GetByIDAndOrg(ctx context.Context, customerID, orgID uuid.UUID) (*repository.Customer, error) {
	c := f.byID[customerID]
	if c == nil || c.OrgID != orgID {
		return nil, nil
	}
	return c, nil
}

type fakeInsightHealth struct{ scores []*repository.HealthScore }

func (f *fakeInsightHealth) GetByCustomerID(ctx context.Context, customerID, orgID uuid.UUID) (*repository.HealthScore, error) {
	for _, s := range f.scores {
		if s.CustomerID == customerID && s.OrgID == orgID {
			return s, nil
		}
	}
	return nil, nil
}

func (f *fakeInsightHealth) ListByOrg(ctx context.Context, orgID uuid.UUID, filters repository.HealthScoreFilters) ([]*repository.HealthScore, error) {
	limit := filters.Limit
	if limit <= 0 || limit > len(f.scores) {
		limit = len(f.scores)
	}
	return f.scores[:limit], nil
}

func (f *fakeInsightHealth) GetScoreAtTime(ctx context.Context, customerID, orgID uuid.UUID, at time.Time) (*repository.HealthScore, error) {
	return nil, nil
}

type fakeInsightEvents struct{ events []*repository.CustomerEvent }

func (f *fakeInsightEvents) ListByCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]*repository.CustomerEvent, error) {
	return f.events, nil
}

type fakeInsightStore struct{ saved []*repository.CustomerInsight }

func (f *fakeInsightStore) GetRecent(ctx context.Context, orgID, customerID uuid.UUID, insightType string, since time.Time) (*repository.CustomerInsight, error) {
	for i := len(f.saved) - 1; i >= 0; i-- {
		insight := f.saved[i]
		if insight.OrgID == orgID && insight.CustomerID == customerID && insight.InsightType == insightType && insight.GeneratedAt.After(since) {
			return insight, nil
		}
	}
	return nil, nil
}

func (f *fakeInsightStore) Create(ctx context.Context, insight *repository.CustomerInsight) error {
	if insight.ID == uuid.Nil {
		insight.ID = uuid.New()
	}
	f.saved = append(f.saved, insight)
	return nil
}

func (f *fakeInsightStore) ListByCustomer(ctx context.Context, orgID, customerID uuid.UUID, limit int) ([]*repository.CustomerInsight, error) {
	return f.saved, nil
}

func validInsightJSON(insightType string) string {
	data, _ := json.Marshal(prompts.StructuredInsight{
		InsightType: insightType,
		Summary:     "Payment risk increased after recent score drop.",
		RiskLevel:   "yellow",
		Confidence:  "high",
		Signals: []prompts.InsightSignal{{
			Name:     "Payment recency",
			Evidence: "Payment recency factor is below target.",
			Impact:   "negative",
		}},
		Recommendations: []prompts.Recommendation{{
			Action:    "Review billing health with the account owner.",
			Owner:     "customer_success",
			Priority:  "high",
			Rationale: "Recent score decline suggests churn risk.",
		}},
	})
	return string(data)
}
