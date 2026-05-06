package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	planmodel "github.com/onnwee/pulse-score/internal/billing"
)

const llmBudgetWarningThreshold = 0.8

var (
	ErrLLMBudgetExceeded             = errors.New("llm monthly budget exceeded")
	ErrLLMBudgetConfirmationRequired = errors.New("llm budget confirmation required")
)

type LLMPlanResolver interface {
	GetCurrentPlan(ctx context.Context, orgID uuid.UUID) (string, error)
}

type LLMUsageRepository interface {
	TrackLLMUsage(ctx context.Context, usage LLMUsage) error
	SumLLMUsageCost(ctx context.Context, orgID uuid.UUID, start, end time.Time) (float64, error)
	CountLLMUsageRequests(ctx context.Context, orgID uuid.UUID, start, end time.Time) (int, error)
	SumLLMUsageTokens(ctx context.Context, orgID uuid.UUID, start, end time.Time) (int, int, error)
}

type LLMBudgetWarningNotifier interface {
	NotifyLLMBudgetWarning(ctx context.Context, warning LLMBudgetWarning) error
}

type LLMUsageSummary struct {
	Tier              string  `json:"tier"`
	MonthlyCostUSD    float64 `json:"monthly_cost_usd"`
	BudgetUSD         float64 `json:"budget_usd"`
	BudgetPercentUsed float64 `json:"budget_percent_used"`
	BudgetWarning     bool    `json:"budget_warning"`
	RequestsToday     int     `json:"requests_today"`
	DailyRequestLimit  int     `json:"daily_request_limit"`
	RequestsThisMonth  int     `json:"requests_this_month"`
	RemainingBudgetUSD float64 `json:"remaining_budget_usd"`
	InputTokensMonth   int     `json:"input_tokens_month,omitempty"`
	OutputTokensMonth  int     `json:"output_tokens_month,omitempty"`
}

type LLMBudgetWarning struct {
	OrgID          uuid.UUID
	Tier           string
	BudgetUSD      float64
	MonthlyCostUSD float64
	PercentUsed    float64
}

type LLMUsageService struct {
	usage    LLMUsageRepository
	plans    LLMPlanResolver
	notifier LLMBudgetWarningNotifier
	now      func() time.Time
}

func NewLLMUsageService(usage LLMUsageRepository, plans LLMPlanResolver, notifier LLMBudgetWarningNotifier) *LLMUsageService {
	return &LLMUsageService{usage: usage, plans: plans, notifier: notifier, now: time.Now}
}

func (s *LLMUsageService) CheckLLMUsage(ctx context.Context, orgID uuid.UUID, estimatedCostUSD float64, manualRegeneration bool) error {
	limits, summary, err := s.currentUsage(ctx, orgID)
	if err != nil {
		return err
	}
	if limits.DailyRequests >= 0 && summary.RequestsToday >= limits.DailyRequests {
		return ErrLLMRateLimited
	}
	if limits.BudgetUSD >= 0 && summary.MonthlyCostUSD+estimatedCostUSD >= limits.BudgetUSD {
		if manualRegeneration {
			return ErrLLMBudgetConfirmationRequired
		}
		return ErrLLMBudgetExceeded
	}
	return nil
}

func (s *LLMUsageService) RecordLLMUsage(ctx context.Context, usage LLMUsage) (*LLMUsageSummary, error) {
	if err := s.usage.TrackLLMUsage(ctx, usage); err != nil {
		return nil, err
	}
	return s.GetLLMUsageSummary(ctx, usage.OrgID)
}

func (s *LLMUsageService) NotifyLLMBudgetWarning(ctx context.Context, warning LLMBudgetWarning) error {
	if s.notifier == nil {
		return nil
	}
	return s.notifier.NotifyLLMBudgetWarning(ctx, warning)
}

func (s *LLMUsageService) GetLLMUsageSummary(ctx context.Context, orgID uuid.UUID) (*LLMUsageSummary, error) {
	_, summary, err := s.currentUsage(ctx, orgID)
	return summary, err
}

func (s *LLMUsageService) currentUsage(ctx context.Context, orgID uuid.UUID) (llmTierLimits, *LLMUsageSummary, error) {
	if s.usage == nil || s.plans == nil {
		return llmTierLimits{}, nil, errors.New("llm usage service dependencies are required")
	}
	tier, err := s.plans.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return llmTierLimits{}, nil, fmt.Errorf("get current plan: %w", err)
	}
	limits := limitsForLLMTier(tier)
	now := s.now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonth := monthStart.AddDate(0, 1, 0)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	nextDay := dayStart.AddDate(0, 0, 1)

	monthlyCost, err := s.usage.SumLLMUsageCost(ctx, orgID, monthStart, nextMonth)
	if err != nil {
		return limits, nil, fmt.Errorf("sum llm usage cost: %w", err)
	}
	requestsToday, err := s.usage.CountLLMUsageRequests(ctx, orgID, dayStart, nextDay)
	if err != nil {
		return limits, nil, fmt.Errorf("count daily llm requests: %w", err)
	}
	requestsMonth, err := s.usage.CountLLMUsageRequests(ctx, orgID, monthStart, nextMonth)
	if err != nil {
		return limits, nil, fmt.Errorf("count monthly llm requests: %w", err)
	}
	inputTokensMonth, outputTokensMonth, err := s.usage.SumLLMUsageTokens(ctx, orgID, monthStart, nextMonth)
	if err != nil {
		return limits, nil, fmt.Errorf("sum monthly llm tokens: %w", err)
	}

	percent := 0.0
	if limits.BudgetUSD > 0 {
		percent = monthlyCost / limits.BudgetUSD
	}
	remaining := math.Max(limits.BudgetUSD-monthlyCost, 0)
	return limits, &LLMUsageSummary{
		Tier:              string(planmodel.NormalizeTier(tier)),
		MonthlyCostUSD:    monthlyCost,
		BudgetUSD:         limits.BudgetUSD,
		BudgetPercentUsed: percent,
		BudgetWarning:     limits.BudgetUSD > 0 && percent >= llmBudgetWarningThreshold,
		RequestsToday:     requestsToday,
		DailyRequestLimit:  limits.DailyRequests,
		RequestsThisMonth:  requestsMonth,
		RemainingBudgetUSD: remaining,
		InputTokensMonth:   inputTokensMonth,
		OutputTokensMonth:  outputTokensMonth,
	}, nil
}

type llmTierLimits struct {
	BudgetUSD     float64
	DailyRequests int
}

func limitsForLLMTier(tier string) llmTierLimits {
	switch planmodel.NormalizeTier(tier) {
	case planmodel.TierGrowth:
		return llmTierLimits{BudgetUSD: 5, DailyRequests: 50}
	case planmodel.TierScale:
		return llmTierLimits{BudgetUSD: 50, DailyRequests: 500}
	default:
		return llmTierLimits{BudgetUSD: 0, DailyRequests: 0}
	}
}
