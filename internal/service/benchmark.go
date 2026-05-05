package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
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
	OrgID                  uuid.UUID
	OrgName                string
	Industry               string
	CompanySize            int
	CustomerCount          int
	TotalMRR               int64
	AvgHealthScore         float64
	AvgChurnRate           float64
	ActiveIntegrationCount int
	PIISamples             []BenchmarkPIISample
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
	if metrics.ActiveIntegrationCount < 0 {
		return nil, fmt.Errorf("active integration count must be nonnegative")
	}

	industry := NormalizeBenchmarkIndustry(metrics.Industry)

	return &repository.BenchmarkContribution{
		OrgID:                  metrics.OrgID,
		Industry:               industry,
		CompanySizeBucket:      BucketCompanySize(metrics.CompanySize),
		AvgHealthScore:         metrics.AvgHealthScore,
		AvgMRR:                 averageMRR(metrics.TotalMRR, metrics.CustomerCount),
		AvgChurnRate:           metrics.AvgChurnRate,
		ActiveIntegrationCount: metrics.ActiveIntegrationCount,
		CustomerCountBucket:    BucketCustomerCount(metrics.CustomerCount),
		ContributedAt:          time.Now().UTC(),
	}, nil
}

func averageMRR(totalMRR int64, customerCount int) int64 {
	if customerCount <= 0 {
		return 0
	}
	return totalMRR / int64(customerCount)
}

// NormalizeBenchmarkIndustry maps organization industry labels to safe benchmark segments.
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
	ActiveIntegrationCount(ctx context.Context, orgID uuid.UUID) (int, error)
}

type BenchmarkContributionWriter interface {
	CreateContribution(ctx context.Context, contribution *repository.BenchmarkContribution) error
}

type BenchmarkContributionReader interface {
	ListLatestContributions(ctx context.Context) ([]repository.BenchmarkContribution, error)
}

type BenchmarkAggregateWriter interface {
	CreateAggregate(ctx context.Context, aggregate *repository.BenchmarkAggregate) error
}

const (
	benchmarkMinimumSampleSize         = 5
	benchmarkContributionFreshnessWindow = 30 * 24 * time.Hour
	benchmarkQualityTargetSampleSize   = 20
	benchmarkQualitySampleWeight       = 0.5
	benchmarkQualityRecencyWeight      = 0.3
	benchmarkQualityVarianceWeight     = 0.2
	benchmarkQualityHighThreshold      = 80
	benchmarkQualityMediumThreshold    = 60
)

const (
	benchmarkQualityLevelHigh   = "high"
	benchmarkQualityLevelMedium = "medium"
	benchmarkQualityLevelLow    = "low"
)

type BenchmarkAggregationService struct {
	contributions BenchmarkContributionReader
	aggregates    BenchmarkAggregateWriter
}

func NewBenchmarkAggregationService(
	contributions BenchmarkContributionReader,
	aggregates BenchmarkAggregateWriter,
) *BenchmarkAggregationService {
	return &BenchmarkAggregationService{contributions: contributions, aggregates: aggregates}
}

func (s *BenchmarkAggregationService) RunOnce(ctx context.Context) error {
	contributions, err := s.contributions.ListLatestContributions(ctx)
	if err != nil {
		return fmt.Errorf("list benchmark contributions: %w", err)
	}

	calculatedAt := time.Now().UTC()
	validContributions := validBenchmarkContributions(contributions, calculatedAt)
	segments := groupBenchmarkContributions(validContributions)
	for _, segment := range segments {
		if len(segment.contributions) < benchmarkMinimumSampleSize {
			continue
		}
		for _, metric := range benchmarkMetricValues(segment.contributions) {
			if len(metric.observations) < benchmarkMinimumSampleSize {
				continue
			}
			aggregate := benchmarkAggregate(segment, metric, calculatedAt)
			if err := s.aggregates.CreateAggregate(ctx, aggregate); err != nil {
				return fmt.Errorf("create benchmark aggregate: %w", err)
			}
		}
	}

	return nil
}

type benchmarkSegment struct {
	industry            string
	companySizeBucket   string
	contributions       []repository.BenchmarkContribution
}

type benchmarkSegmentKey struct {
	industry          string
	companySizeBucket string
}

type benchmarkMetricValueSet struct {
	name         string
	observations []benchmarkMetricObservation
}

type benchmarkMetricObservation struct {
	value         float64
	contributedAt time.Time
}

func validBenchmarkContributions(contributions []repository.BenchmarkContribution, now time.Time) []repository.BenchmarkContribution {
	valid := make([]repository.BenchmarkContribution, 0, len(contributions))
	freshAfter := now.Add(-benchmarkContributionFreshnessWindow)
	for _, contribution := range contributions {
		if !isFreshBenchmarkContribution(contribution, freshAfter) || !isValidBenchmarkContribution(contribution) {
			continue
		}
		valid = append(valid, contribution)
	}
	return valid
}

func isFreshBenchmarkContribution(contribution repository.BenchmarkContribution, freshAfter time.Time) bool {
	if contribution.ContributedAt.IsZero() {
		return false
	}
	return !contribution.ContributedAt.Before(freshAfter)
}

func isValidBenchmarkContribution(contribution repository.BenchmarkContribution) bool {
	return contribution.OrgID != uuid.Nil &&
		strings.TrimSpace(contribution.Industry) != "" &&
		strings.TrimSpace(contribution.CompanySizeBucket) != "" &&
		contribution.AvgHealthScore >= 0 && contribution.AvgHealthScore <= 100 &&
		contribution.AvgMRR >= 0 &&
		contribution.AvgChurnRate >= 0 && contribution.AvgChurnRate <= 1 &&
		contribution.ActiveIntegrationCount >= 0
}

func groupBenchmarkContributions(contributions []repository.BenchmarkContribution) []benchmarkSegment {
	segmentByKey := make(map[benchmarkSegmentKey]*benchmarkSegment)
	for _, contribution := range contributions {
		key := benchmarkSegmentKey{
			industry:          contribution.Industry,
			companySizeBucket: contribution.CompanySizeBucket,
		}
		segment, ok := segmentByKey[key]
		if !ok {
			segment = &benchmarkSegment{
				industry:          contribution.Industry,
				companySizeBucket: contribution.CompanySizeBucket,
			}
			segmentByKey[key] = segment
		}
		segment.contributions = append(segment.contributions, contribution)
	}

	segments := make([]benchmarkSegment, 0, len(segmentByKey))
	for _, segment := range segmentByKey {
		segments = append(segments, *segment)
	}
	sort.Slice(segments, func(i, j int) bool {
		if segments[i].industry == segments[j].industry {
			return segments[i].companySizeBucket < segments[j].companySizeBucket
		}
		return segments[i].industry < segments[j].industry
	})
	return segments
}

func benchmarkMetricValues(contributions []repository.BenchmarkContribution) []benchmarkMetricValueSet {
	healthScores := make([]benchmarkMetricObservation, 0, len(contributions))
	mrrPerCustomer := make([]benchmarkMetricObservation, 0, len(contributions))
	churnRates := make([]benchmarkMetricObservation, 0, len(contributions))
	integrationUsage := make([]benchmarkMetricObservation, 0, len(contributions))

	for _, contribution := range contributions {
		healthScores = append(healthScores, benchmarkObservation(contribution.AvgHealthScore, contribution.ContributedAt))
		mrrPerCustomer = append(mrrPerCustomer, benchmarkObservation(float64(contribution.AvgMRR), contribution.ContributedAt))
		churnRates = append(churnRates, benchmarkObservation(contribution.AvgChurnRate, contribution.ContributedAt))
		integrationUsage = append(integrationUsage, benchmarkObservation(float64(contribution.ActiveIntegrationCount), contribution.ContributedAt))
	}

	return []benchmarkMetricValueSet{
		{name: repository.BenchmarkMetricHealthScore, observations: excludeBenchmarkOutliers(healthScores)},
		{name: repository.BenchmarkMetricMRRPerCustomer, observations: excludeBenchmarkOutliers(mrrPerCustomer)},
		{name: repository.BenchmarkMetricChurnRate, observations: excludeBenchmarkOutliers(churnRates)},
		{name: repository.BenchmarkMetricIntegrationUsage, observations: excludeBenchmarkOutliers(integrationUsage)},
	}
}

func benchmarkObservation(value float64, contributedAt time.Time) benchmarkMetricObservation {
	return benchmarkMetricObservation{value: value, contributedAt: contributedAt}
}

func benchmarkAggregate(segment benchmarkSegment, metric benchmarkMetricValueSet, calculatedAt time.Time) *repository.BenchmarkAggregate {
	values := benchmarkObservationValues(metric.observations)
	sort.Float64s(values)
	qualityScore := benchmarkQualityScore(metric.observations, calculatedAt)

	return &repository.BenchmarkAggregate{
		Industry:          segment.industry,
		CompanySizeBucket: segment.companySizeBucket,
		MetricName:        metric.name,
		P25:               percentile(values, 0.25),
		P50:               percentile(values, 0.50),
		P75:               percentile(values, 0.75),
		P90:               percentile(values, 0.90),
		SampleCount:       len(values),
		QualityScore:      qualityScore,
		QualityLevel:      benchmarkQualityLevel(qualityScore),
		CalculatedAt:      calculatedAt,
	}
}

func benchmarkObservationValues(observations []benchmarkMetricObservation) []float64 {
	values := make([]float64, 0, len(observations))
	for _, observation := range observations {
		values = append(values, observation.value)
	}
	return values
}

func excludeBenchmarkOutliers(observations []benchmarkMetricObservation) []benchmarkMetricObservation {
	if len(observations) < benchmarkMinimumSampleSize {
		return observations
	}
	values := benchmarkObservationValues(observations)
	sort.Float64s(values)
	q1 := percentile(values, 0.25)
	q3 := percentile(values, 0.75)
	iqr := q3 - q1
	if iqr == 0 {
		return observations
	}
	lowerFence := q1 - 1.5*iqr
	upperFence := q3 + 1.5*iqr

	filtered := make([]benchmarkMetricObservation, 0, len(observations))
	for _, observation := range observations {
		if observation.value < lowerFence || observation.value > upperFence {
			continue
		}
		filtered = append(filtered, observation)
	}
	return filtered
}

func benchmarkQualityScore(observations []benchmarkMetricObservation, now time.Time) float64 {
	if len(observations) == 0 {
		return 0
	}
	sampleScore := math.Min(float64(len(observations))/benchmarkQualityTargetSampleSize, 1)
	recencyScore := benchmarkRecencyQuality(observations, now)
	varianceScore := benchmarkVarianceQuality(benchmarkObservationValues(observations))
	score := (benchmarkQualitySampleWeight * sampleScore) +
		(benchmarkQualityRecencyWeight * recencyScore) +
		(benchmarkQualityVarianceWeight * varianceScore)
	return math.Round(score * 100)
}

func benchmarkRecencyQuality(observations []benchmarkMetricObservation, now time.Time) float64 {
	var totalAge time.Duration
	for _, observation := range observations {
		age := now.Sub(observation.contributedAt)
		if age < 0 {
			age = 0
		}
		totalAge += age
	}
	avgAge := totalAge / time.Duration(len(observations))
	return math.Max(0, 1-(float64(avgAge)/float64(benchmarkContributionFreshnessWindow)))
}

func benchmarkVarianceQuality(values []float64) float64 {
	if len(values) < 2 {
		return 1
	}
	average := averageFloat64(values)
	var variance float64
	for _, value := range values {
		delta := value - average
		variance += delta * delta
	}
	stddev := math.Sqrt(variance / float64(len(values)))
	if average == 0 {
		if stddev == 0 {
			return 1
		}
		return 0
	}
	coefficient := math.Abs(stddev / average)
	return math.Max(0, 1-math.Min(coefficient, 1))
}

func averageFloat64(values []float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func benchmarkQualityLevel(score float64) string {
	switch {
	case score >= benchmarkQualityHighThreshold:
		return benchmarkQualityLevelHigh
	case score >= benchmarkQualityMediumThreshold:
		return benchmarkQualityLevelMedium
	default:
		return benchmarkQualityLevelLow
	}
}

func percentile(sortedValues []float64, quantile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if len(sortedValues) == 1 {
		return sortedValues[0]
	}

	position := quantile * float64(len(sortedValues)-1)
	lowerIndex := int(position)
	upperIndex := lowerIndex + 1
	if upperIndex >= len(sortedValues) {
		return sortedValues[lowerIndex]
	}

	fraction := position - float64(lowerIndex)
	value := sortedValues[lowerIndex] + (sortedValues[upperIndex]-sortedValues[lowerIndex])*fraction
	return math.Round(value*10000) / 10000
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
		if strings.TrimSpace(org.Industry) == "" {
			slog.Warn("skipping benchmark contribution without industry", "org_id", org.ID)
			continue
		}
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
	integrationCount, err := p.metrics.ActiveIntegrationCount(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("count benchmark active integrations: %w", err)
	}

	contribution, err := p.anonymizer.Anonymize(BenchmarkOrgMetrics{
		OrgID:                  org.ID,
		Industry:               org.Industry,
		CompanySize:            org.CompanySize,
		CustomerCount:          customerCount,
		TotalMRR:               totalMRR,
		AvgHealthScore:         avgScore,
		AvgChurnRate:           churnRate,
		ActiveIntegrationCount: integrationCount,
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
