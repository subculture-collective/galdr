package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

type BenchmarkPIISample struct {
	Email      string
	Name       string
	ExternalID string
}

type BenchmarkOrgMetrics struct {
	OrgID          uuid.UUID
	OrgName        string
	Industry       string
	CompanySize    int
	CustomerCount  int
	TotalMRR       int64
	AvgHealthScore float64
	AvgChurnRate   float64
	PIISamples     []BenchmarkPIISample
}

const unknownBenchmarkIndustry = "unknown"

var benchmarkIndustrySegments = map[string]string{
	"agency":      "agency",
	"e-commerce":  "e-commerce",
	"ecommerce":   "e-commerce",
	"education":   "education",
	"fintech":     "fintech",
	"healthcare":  "healthcare",
	"marketplace": "marketplace",
	"media":       "media",
	"other":       "other",
	"saas":        "saas",
}

type BenchmarkAnonymizer struct{}

func NewBenchmarkAnonymizer() *BenchmarkAnonymizer {
	return &BenchmarkAnonymizer{}
}

func (a *BenchmarkAnonymizer) Anonymize(metrics BenchmarkOrgMetrics) (*repository.BenchmarkContribution, error) {
	if metrics.OrgID == uuid.Nil {
		return nil, fmt.Errorf("org id is required")
	}
	if metrics.CustomerCount <= 0 {
		return nil, fmt.Errorf("customer count must be positive")
	}
	if metrics.AvgHealthScore < 0 || metrics.AvgHealthScore > 100 {
		return nil, fmt.Errorf("average health score out of range")
	}
	if metrics.AvgChurnRate < 0 || metrics.AvgChurnRate > 1 {
		return nil, fmt.Errorf("average churn rate out of range")
	}
	if metrics.TotalMRR < 0 {
		return nil, fmt.Errorf("total mrr must be nonnegative")
	}

	industry := NormalizeBenchmarkIndustry(metrics.Industry)

	return &repository.BenchmarkContribution{
		OrgID:               metrics.OrgID,
		Industry:            industry,
		CompanySizeBucket:   BucketCompanySize(metrics.CompanySize),
		AvgHealthScore:      metrics.AvgHealthScore,
		AvgMRR:              averageMRR(metrics.TotalMRR, metrics.CustomerCount),
		AvgChurnRate:        metrics.AvgChurnRate,
		CustomerCountBucket: BucketCustomerCount(metrics.CustomerCount),
		ContributedAt:       time.Now().UTC(),
	}, nil
}

func averageMRR(totalMRR int64, customerCount int) int64 {
	if customerCount <= 0 {
		return 0
	}
	return totalMRR / int64(customerCount)
}

// NormalizeBenchmarkIndustry maps free-form org input to safe benchmark segments.
func NormalizeBenchmarkIndustry(industry string) string {
	normalized := strings.ToLower(strings.TrimSpace(industry))
	if normalized == "" || strings.Contains(normalized, "@") || strings.Contains(normalized, ".") {
		return unknownBenchmarkIndustry
	}

	if canonical, ok := benchmarkIndustrySegments[normalized]; ok {
		return canonical
	}
	return unknownBenchmarkIndustry
}

func BucketCompanySize(size int) string {
	return benchmarkBucket(size)
}

func BucketCustomerCount(count int) string {
	return benchmarkBucket(count)
}

func benchmarkBucket(value int) string {
	switch {
	case value <= 10:
		return repository.BenchmarkBucket1To10
	case value <= 50:
		return repository.BenchmarkBucket11To50
	case value <= 200:
		return repository.BenchmarkBucket51To200
	case value <= 1000:
		return repository.BenchmarkBucket201To1000
	default:
		return repository.BenchmarkBucket1000Plus
	}
}

type BenchmarkOrgRepository interface {
	ListBenchmarkingEnabled(ctx context.Context) ([]repository.Organization, error)
}

type BenchmarkMetricsReader interface {
	CountCustomers(ctx context.Context, orgID uuid.UUID) (int, error)
	TotalMRR(ctx context.Context, orgID uuid.UUID) (int64, error)
	AverageHealthScore(ctx context.Context, orgID uuid.UUID) (float64, error)
	ChurnRate(ctx context.Context, orgID uuid.UUID) (float64, error)
}

type BenchmarkContributionWriter interface {
	CreateContribution(ctx context.Context, contribution *repository.BenchmarkContribution) error
}

type BenchmarkPipeline struct {
	organizations BenchmarkOrgRepository
	metrics       BenchmarkMetricsReader
	contributions BenchmarkContributionWriter
	anonymizer    *BenchmarkAnonymizer
}

func NewBenchmarkPipeline(
	organizations BenchmarkOrgRepository,
	metrics BenchmarkMetricsReader,
	contributions BenchmarkContributionWriter,
	anonymizer *BenchmarkAnonymizer,
) *BenchmarkPipeline {
	if anonymizer == nil {
		anonymizer = NewBenchmarkAnonymizer()
	}
	return &BenchmarkPipeline{
		organizations: organizations,
		metrics:       metrics,
		contributions: contributions,
		anonymizer:    anonymizer,
	}
}

func (p *BenchmarkPipeline) RunOnce(ctx context.Context) error {
	orgs, err := p.organizations.ListBenchmarkingEnabled(ctx)
	if err != nil {
		return fmt.Errorf("list benchmark opted-in orgs: %w", err)
	}

	for _, org := range orgs {
		if err := p.contributeOrg(ctx, org); err != nil {
			return err
		}
	}
	return nil
}

func (p *BenchmarkPipeline) contributeOrg(ctx context.Context, org repository.Organization) error {
	customerCount, err := p.metrics.CountCustomers(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("count benchmark customers: %w", err)
	}
	if customerCount == 0 {
		return nil
	}
	totalMRR, err := p.metrics.TotalMRR(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("sum benchmark mrr: %w", err)
	}
	avgScore, err := p.metrics.AverageHealthScore(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("average benchmark score: %w", err)
	}
	churnRate, err := p.metrics.ChurnRate(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("benchmark churn rate: %w", err)
	}

	contribution, err := p.anonymizer.Anonymize(BenchmarkOrgMetrics{
		OrgID:          org.ID,
		Industry:       org.Industry,
		CompanySize:    org.CompanySize,
		CustomerCount:  customerCount,
		TotalMRR:       totalMRR,
		AvgHealthScore: avgScore,
		AvgChurnRate:   churnRate,
	})
	if err != nil {
		return fmt.Errorf("anonymize benchmark contribution: %w", err)
	}

	if err := p.contributions.CreateContribution(ctx, contribution); err != nil {
		return fmt.Errorf("store benchmark contribution: %w", err)
	}
	return nil
}

type BenchmarkScheduler struct {
	pipeline *BenchmarkPipeline
	interval time.Duration
}

func NewBenchmarkScheduler(pipeline *BenchmarkPipeline, interval time.Duration) *BenchmarkScheduler {
	return &BenchmarkScheduler{pipeline: pipeline, interval: interval}
}

func (s *BenchmarkScheduler) Start(ctx context.Context) {
	slog.Info("benchmark scheduler started", "interval", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("benchmark scheduler stopped")
			return
		case <-ticker.C:
			if err := s.pipeline.RunOnce(ctx); err != nil {
				slog.Error("benchmark contribution run failed", "error", err)
			}
		}
	}
}
