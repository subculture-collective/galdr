package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service/prompts"
)

const (
	InsightTypeCustomerAnalysis      = "customer_analysis"
	InsightTriggerManual             = "manual"
	InsightTriggerScoreDrop          = "score_drop"
	InsightTriggerRiskLevelChanged   = "risk_level_changed"
	InsightRequestTypeCustomerInsight = "customer_insight"
	defaultInsightCacheTTL           = 24 * time.Hour
	defaultInsightBatchLimit         = 50
	defaultScoreDropInsightThreshold = 10
)

type insightCustomerStore interface {
	GetByIDAndOrg(ctx context.Context, customerID, orgID uuid.UUID) (*repository.Customer, error)
}

type insightHealthStore interface {
	GetByCustomerID(ctx context.Context, customerID, orgID uuid.UUID) (*repository.HealthScore, error)
	GetScoreAtTime(ctx context.Context, customerID, orgID uuid.UUID, at time.Time) (*repository.HealthScore, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, filters repository.HealthScoreFilters) ([]*repository.HealthScore, error)
}

type insightEventStore interface {
	ListByCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]*repository.CustomerEvent, error)
}

type insightStore interface {
	Create(ctx context.Context, insight *repository.CustomerInsight) error
	GetRecent(ctx context.Context, orgID, customerID uuid.UUID, insightType string, since time.Time) (*repository.CustomerInsight, error)
	ListByCustomer(ctx context.Context, orgID, customerID uuid.UUID, limit int) ([]*repository.CustomerInsight, error)
}

type insightLLM interface {
	Complete(ctx context.Context, req LLMCompletionRequest) (*LLMCompletionResponse, error)
}

// InsightPipelineDeps contains dependencies for per-customer AI analysis.
type InsightPipelineDeps struct {
	Customers    insightCustomerStore
	HealthScores insightHealthStore
	Events       insightEventStore
	Insights     insightStore
	LLM          insightLLM
	Now          func() time.Time
	CacheTTL     time.Duration
	BatchLimit   int
}

// InsightPipeline generates and caches customer insights.
type InsightPipeline struct {
	customers    insightCustomerStore
	healthScores insightHealthStore
	events       insightEventStore
	insights     insightStore
	llm          insightLLM
	now          func() time.Time
	cacheTTL     time.Duration
	batchLimit   int
}

// InsightGenerationOptions controls insight generation behavior.
type InsightGenerationOptions struct {
	Force       bool   `json:"force"`
	Trigger     string `json:"trigger"`
	InsightType string `json:"insight_type"`
}

// CustomerInsightResponse wraps a generated or cached insight.
type CustomerInsightResponse struct {
	Insight *repository.CustomerInsight `json:"insight"`
	Cached  bool                        `json:"cached"`
}

// InsightBatchResponse summarizes batch insight generation.
type InsightBatchResponse struct {
	Processed int      `json:"processed"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

// NewInsightPipeline creates a per-customer insight pipeline.
func NewInsightPipeline(deps InsightPipelineDeps) *InsightPipeline {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	cacheTTL := deps.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultInsightCacheTTL
	}
	batchLimit := deps.BatchLimit
	if batchLimit <= 0 {
		batchLimit = defaultInsightBatchLimit
	}
	return &InsightPipeline{
		customers:    deps.Customers,
		healthScores: deps.HealthScores,
		events:       deps.Events,
		insights:     deps.Insights,
		llm:          deps.LLM,
		now:          now,
		cacheTTL:     cacheTTL,
		batchLimit:   batchLimit,
	}
}

// GenerateCustomerInsight creates a customer insight or returns a fresh cached one.
func (p *InsightPipeline) GenerateCustomerInsight(ctx context.Context, orgID, customerID uuid.UUID, opts InsightGenerationOptions) (*CustomerInsightResponse, error) {
	if !p.configured() {
		return nil, fmt.Errorf("insight pipeline not configured")
	}
	insightType := insightTypeOrDefault(opts.InsightType)
	if !opts.Force {
		cached, err := p.insights.GetRecent(ctx, orgID, customerID, insightType, p.now().Add(-p.cacheTTL))
		if err != nil {
			return nil, fmt.Errorf("get cached insight: %w", err)
		}
		if cached != nil {
			return &CustomerInsightResponse{Insight: cached, Cached: true}, nil
		}
	}

	customer, err := p.customers.GetByIDAndOrg(ctx, customerID, orgID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, &NotFoundError{Resource: "customer", Message: "customer not found"}
	}
	healthScore, err := p.healthScores.GetByCustomerID(ctx, customerID, orgID)
	if err != nil {
		return nil, fmt.Errorf("get health score: %w", err)
	}
	if healthScore == nil {
		return nil, &NotFoundError{Resource: "health_score", Message: "health score not found"}
	}
	events, err := p.events.ListByCustomer(ctx, customerID, 10)
	if err != nil {
		return nil, fmt.Errorf("list customer events: %w", err)
	}

	data := p.buildPromptData(ctx, customer, healthScore, events)
	completion, err := p.llm.Complete(ctx, LLMCompletionRequest{
		OrgID:              orgID,
		TemplateName:       string(prompts.CustomerAnalysisTemplate),
		TemplateData:       data,
		MaxTokens:          prompts.Specs()[prompts.CustomerAnalysisTemplate].MaxOutputTokens,
		RequestType:        InsightRequestTypeCustomerInsight,
		ManualRegeneration: opts.Force,
	})
	if err != nil {
		return nil, fmt.Errorf("generate insight: %w", err)
	}
	structured, err := prompts.ParseStructuredOutput([]byte(completion.Text))
	if err != nil {
		return nil, fmt.Errorf("parse insight: %w", err)
	}
	content, err := structuredInsightContent(structured)
	if err != nil {
		return nil, err
	}
	if opts.Trigger != "" {
		content["trigger"] = opts.Trigger
	}
	insight := &repository.CustomerInsight{
		OrgID:       orgID,
		CustomerID:  customerID,
		InsightType: structured.InsightType,
		Content:     content,
		GeneratedAt: p.now(),
		Model:       completion.Provider,
		TokenCost:   completion.CostUSD,
	}
	if err := p.insights.Create(ctx, insight); err != nil {
		return nil, fmt.Errorf("store insight: %w", err)
	}
	return &CustomerInsightResponse{Insight: insight}, nil
}

// ListCustomerInsights returns recent customer insights.
func (p *InsightPipeline) ListCustomerInsights(ctx context.Context, orgID, customerID uuid.UUID, limit int) ([]*repository.CustomerInsight, error) {
	if p == nil || p.insights == nil {
		return nil, fmt.Errorf("insight pipeline not configured")
	}
	return p.insights.ListByCustomer(ctx, orgID, customerID, limit)
}

// GenerateForScoreChange generates an insight when a score change crosses insight thresholds.
func (p *InsightPipeline) GenerateForScoreChange(ctx context.Context, previous, current *repository.HealthScore) (*CustomerInsightResponse, bool, error) {
	trigger, ok := InsightTriggerForScoreChange(previous, current, 0)
	if !ok {
		return nil, false, nil
	}
	resp, err := p.GenerateCustomerInsight(ctx, current.OrgID, current.CustomerID, InsightGenerationOptions{Trigger: trigger})
	return resp, true, err
}

// ProcessBatch generates insights for lowest-score customers first.
func (p *InsightPipeline) ProcessBatch(ctx context.Context, orgID uuid.UUID, limit int) (*InsightBatchResponse, error) {
	if !p.configured() {
		return nil, fmt.Errorf("insight pipeline not configured")
	}
	if limit <= 0 || limit > p.batchLimit {
		limit = p.batchLimit
	}
	scores, err := p.healthScores.ListByOrg(ctx, orgID, repository.HealthScoreFilters{Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("list at-risk scores: %w", err)
	}
	resp := &InsightBatchResponse{}
	for _, score := range scores {
		if _, err := p.GenerateCustomerInsight(ctx, orgID, score.CustomerID, InsightGenerationOptions{}); err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, err.Error())
			continue
		}
		resp.Processed++
	}
	return resp, nil
}

func (p *InsightPipeline) configured() bool {
	return p != nil && p.customers != nil && p.healthScores != nil && p.events != nil && p.insights != nil && p.llm != nil
}

func insightTypeOrDefault(insightType string) string {
	if insightType == "" {
		return InsightTypeCustomerAnalysis
	}
	return insightType
}

func (p *InsightPipeline) buildPromptData(ctx context.Context, customer *repository.Customer, healthScore *repository.HealthScore, events []*repository.CustomerEvent) prompts.CustomerAnalysisData {
	now := p.now()
	return prompts.CustomerAnalysisData{
		Customer: prompts.CustomerSnapshot{
			Name:        customer.Name,
			Email:       customer.Email,
			CompanyName: customer.CompanyName,
			Source:      customer.Source,
			MRRCents:    customer.MRRCents,
			Currency:    customer.Currency,
			Tenure:      now.Sub(customer.CreatedAt),
		},
		HealthScore: prompts.HealthScoreSnapshot{
			OverallScore:   healthScore.OverallScore,
			RiskLevel:      healthScore.RiskLevel,
			ScoreChange7d:  p.scoreChange(ctx, healthScore, now.AddDate(0, 0, -7)),
			ScoreChange30d: p.scoreChange(ctx, healthScore, now.AddDate(0, 0, -30)),
			Factors:        scoreFactors(healthScore.Factors),
		},
		Events: eventSummaries(events, now),
	}
}

func (p *InsightPipeline) scoreChange(ctx context.Context, current *repository.HealthScore, at time.Time) int {
	previous, err := p.healthScores.GetScoreAtTime(ctx, current.CustomerID, current.OrgID, at)
	if err != nil || previous == nil {
		return 0
	}
	return current.OverallScore - previous.OverallScore
}

func scoreFactors(factors map[string]float64) []prompts.ScoreFactor {
	names := make([]string, 0, len(factors))
	for name := range factors {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]prompts.ScoreFactor, 0, len(names))
	for _, name := range names {
		result = append(result, prompts.ScoreFactor{Name: name, Score: factors[name]})
	}
	return result
}

func eventSummaries(events []*repository.CustomerEvent, now time.Time) []prompts.CustomerEventSummary {
	result := make([]prompts.CustomerEventSummary, 0, len(events))
	for _, event := range events {
		result = append(result, prompts.CustomerEventSummary{
			Type:    event.EventType,
			Source:  event.Source,
			AgeDays: int(now.Sub(event.OccurredAt).Hours() / 24),
			Summary: fmt.Sprint(event.Data),
		})
	}
	return result
}

func structuredInsightContent(insight *prompts.StructuredInsight) (map[string]any, error) {
	data, err := json.Marshal(insight)
	if err != nil {
		return nil, fmt.Errorf("marshal structured insight: %w", err)
	}
	var content map[string]any
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("unmarshal structured insight content: %w", err)
	}
	return content, nil
}

// InsightTriggerForScoreChange returns whether a score change should generate an insight.
func InsightTriggerForScoreChange(previous, current *repository.HealthScore, threshold int) (string, bool) {
	if previous == nil || current == nil {
		return "", false
	}
	if threshold <= 0 {
		threshold = defaultScoreDropInsightThreshold
	}
	if current.RiskLevel != previous.RiskLevel {
		return InsightTriggerRiskLevelChanged, true
	}
	if previous.OverallScore-current.OverallScore >= threshold {
		return InsightTriggerScoreDrop, true
	}
	return "", false
}
