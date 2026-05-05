package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeLLMPlanResolver struct{ tier string }

func (f fakeLLMPlanResolver) GetCurrentPlan(ctx context.Context, orgID uuid.UUID) (string, error) {
	return f.tier, nil
}

type fakeLLMUsageRepo struct {
	cost     float64
	dayCount int
	monCount int
	tracked  []LLMUsage
}

func (f *fakeLLMUsageRepo) TrackLLMUsage(ctx context.Context, usage LLMUsage) error {
	f.tracked = append(f.tracked, usage)
	f.cost += usage.CostUSD
	f.dayCount++
	f.monCount++
	return nil
}

func (f *fakeLLMUsageRepo) SumLLMUsageCost(ctx context.Context, orgID uuid.UUID, start, end time.Time) (float64, error) {
	return f.cost, nil
}

func (f *fakeLLMUsageRepo) CountLLMUsageRequests(ctx context.Context, orgID uuid.UUID, start, end time.Time) (int, error) {
	if end.Sub(start) <= 24*time.Hour {
		return f.dayCount, nil
	}
	return f.monCount, nil
}

func TestLLMUsageServiceEnforcesTierBudget(t *testing.T) {
	svc := NewLLMUsageService(&fakeLLMUsageRepo{cost: 4.99}, fakeLLMPlanResolver{tier: "growth"}, nil)

	err := svc.CheckLLMUsage(context.Background(), uuid.New(), 0.02, false)
	if !errors.Is(err, ErrLLMBudgetExceeded) {
		t.Fatalf("expected budget exceeded, got %v", err)
	}
}

func TestLLMUsageServiceRequiresConfirmationForManualRegenerationOverBudget(t *testing.T) {
	svc := NewLLMUsageService(&fakeLLMUsageRepo{cost: 4.99}, fakeLLMPlanResolver{tier: "growth"}, nil)

	err := svc.CheckLLMUsage(context.Background(), uuid.New(), 0.02, true)
	if !errors.Is(err, ErrLLMBudgetConfirmationRequired) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
}

func TestLLMUsageServiceEnforcesDailyRequestLimit(t *testing.T) {
	svc := NewLLMUsageService(&fakeLLMUsageRepo{dayCount: 50}, fakeLLMPlanResolver{tier: "growth"}, nil)

	err := svc.CheckLLMUsage(context.Background(), uuid.New(), 0.01, false)
	if !errors.Is(err, ErrLLMRateLimited) {
		t.Fatalf("expected rate limit, got %v", err)
	}
}

func TestLLMUsageServiceReportsBudgetWarningAtEightyPercent(t *testing.T) {
	svc := NewLLMUsageService(&fakeLLMUsageRepo{cost: 4.00, dayCount: 12, monCount: 20}, fakeLLMPlanResolver{tier: "growth"}, nil)

	summary, err := svc.GetLLMUsageSummary(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected summary, got %v", err)
	}
	if !summary.BudgetWarning || summary.BudgetPercentUsed != 0.8 || summary.DailyRequestLimit != 50 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
