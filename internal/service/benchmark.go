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

var benchmarkIndustryAliases = map[string]string{
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

	if canonical, ok := benchmarkIndustryAliases[normalized]; ok {
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

const (
	benchmarkInsightNotificationType = "benchmark_insight"
	benchmarkDigestFrequencyWeekly   = "weekly"
)

type BenchmarkContributionPairReader interface {
	ListLatestContributionPairs(ctx context.Context) ([]repository.BenchmarkContributionPair, error)
}

type BenchmarkAggregateReader interface {
	ListLatestAggregates(ctx context.Context) ([]repository.BenchmarkAggregate, error)
}

type BenchmarkMemberReader interface {
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]repository.OrgMember, error)
}

type BenchmarkNotificationWriter interface {
	Create(ctx context.Context, notification *repository.Notification) error
}

type BenchmarkPreferenceReader interface {
	Get(ctx context.Context, userID, orgID uuid.UUID) (*repository.NotificationPreference, error)
}

type BenchmarkEmailSender interface {
	SendEmail(ctx context.Context, params SendEmailParams) (string, error)
}

type BenchmarkInsightNotificationDeps struct {
	Contributions BenchmarkContributionPairReader
	Aggregates     BenchmarkAggregateReader
	Members        BenchmarkMemberReader
	Notifications  BenchmarkNotificationWriter
	Preferences    BenchmarkPreferenceReader
	Emails         BenchmarkEmailSender
}

type BenchmarkInsightNotificationService struct {
	contributions BenchmarkContributionPairReader
	aggregates     BenchmarkAggregateReader
	members        BenchmarkMemberReader
	notifications  BenchmarkNotificationWriter
	preferences    BenchmarkPreferenceReader
	emails         BenchmarkEmailSender
}

type BenchmarkWeeklyDigestSummary struct {
	OrgID              uuid.UUID
	OrgName            string
	MetricName         string
	PreviousPercentile float64
	CurrentPercentile  float64
}

type benchmarkInsight struct {
	orgID              uuid.UUID
	metricName         string
	message            string
	previousValue      float64
	currentValue       float64
	previousPercentile float64
	currentPercentile  float64
}

func NewBenchmarkInsightNotificationService(deps BenchmarkInsightNotificationDeps) *BenchmarkInsightNotificationService {
	return &BenchmarkInsightNotificationService{
		contributions: deps.Contributions,
		aggregates:     deps.Aggregates,
		members:        deps.Members,
		notifications:  deps.Notifications,
		preferences:    deps.Preferences,
		emails:         deps.Emails,
	}
}

func (s *BenchmarkInsightNotificationService) RunOnce(ctx context.Context) error {
	if s.contributions == nil || s.aggregates == nil || s.members == nil || s.notifications == nil {
		return nil
	}
	pairs, err := s.contributions.ListLatestContributionPairs(ctx)
	if err != nil {
		return fmt.Errorf("list benchmark contribution pairs: %w", err)
	}
	aggregates, err := s.aggregates.ListLatestAggregates(ctx)
	if err != nil {
		return fmt.Errorf("list benchmark aggregates: %w", err)
	}
	aggregateByKey := benchmarkAggregateByKey(aggregates)

	for _, pair := range pairs {
		for _, metric := range benchmarkContributionMetrics(pair.Previous, pair.Current) {
			aggregate, ok := aggregateByKey[benchmarkAggregateKeyFor(pair.Current, metric.name)]
			if !ok {
				continue
			}
			insight, ok := benchmarkInsightFromMetric(pair.Current.OrgID, metric, aggregate)
			if !ok {
				continue
			}
			if err := s.notifyMembers(ctx, insight); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *BenchmarkInsightNotificationService) SendWeeklyDigest(ctx context.Context, summaries []BenchmarkWeeklyDigestSummary) error {
	if s.members == nil || s.emails == nil {
		return nil
	}
	for _, summary := range summaries {
		members, err := s.members.ListMembers(ctx, summary.OrgID)
		if err != nil {
			return fmt.Errorf("list benchmark digest members: %w", err)
		}
		for _, member := range members {
			if !s.shouldSendDigest(ctx, member.UserID, summary.OrgID) || strings.TrimSpace(member.Email) == "" {
				continue
			}
			if _, err := s.emails.SendEmail(ctx, SendEmailParams{
				To:       member.Email,
				Subject:  "Your weekly benchmark summary",
				TextBody: benchmarkDigestText(summary),
				HTMLBody: benchmarkDigestHTML(summary),
			}); err != nil {
				return fmt.Errorf("send benchmark digest email: %w", err)
			}
		}
	}
	return nil
}

func (s *BenchmarkInsightNotificationService) SendWeeklyDigestFromLatest(ctx context.Context) error {
	if s.contributions == nil || s.aggregates == nil {
		return nil
	}
	pairs, err := s.contributions.ListLatestContributionPairs(ctx)
	if err != nil {
		return fmt.Errorf("list benchmark digest contribution pairs: %w", err)
	}
	aggregates, err := s.aggregates.ListLatestAggregates(ctx)
	if err != nil {
		return fmt.Errorf("list benchmark digest aggregates: %w", err)
	}
	aggregateByKey := benchmarkAggregateByKey(aggregates)

	summaries := make([]BenchmarkWeeklyDigestSummary, 0, len(pairs))
	for _, pair := range pairs {
		summary, ok := benchmarkDigestSummaryFromPair(pair, aggregateByKey)
		if !ok {
			continue
		}
		summaries = append(summaries, summary)
	}
	return s.SendWeeklyDigest(ctx, summaries)
}

func (s *BenchmarkInsightNotificationService) notifyMembers(ctx context.Context, insight benchmarkInsight) error {
	members, err := s.members.ListMembers(ctx, insight.orgID)
	if err != nil {
		return fmt.Errorf("list benchmark notification members: %w", err)
	}
	for _, member := range members {
		pref := s.preference(ctx, member.UserID, insight.orgID)
		if pref.InAppEnabled {
			notification := &repository.Notification{
				UserID:  member.UserID,
				OrgID:   insight.orgID,
				Type:    benchmarkInsightNotificationType,
				Title:   "Benchmark insight",
				Message: insight.message,
				Data: map[string]any{
					"metric_name":         insight.metricName,
					"previous_value":      insight.previousValue,
					"current_value":       insight.currentValue,
					"previous_percentile": insight.previousPercentile,
					"current_percentile":  insight.currentPercentile,
				},
			}
			if err := s.notifications.Create(ctx, notification); err != nil {
				return fmt.Errorf("create benchmark insight notification: %w", err)
			}
		}
		if pref.EmailEnabled && s.emails != nil && strings.TrimSpace(member.Email) != "" {
			if _, err := s.emails.SendEmail(ctx, SendEmailParams{
				To:       member.Email,
				Subject:  "Benchmark insight",
				TextBody: insight.message,
				HTMLBody: "<p>" + insight.message + "</p>",
			}); err != nil {
				return fmt.Errorf("send benchmark insight email: %w", err)
			}
		}
	}
	return nil
}

func (s *BenchmarkInsightNotificationService) preference(ctx context.Context, userID, orgID uuid.UUID) *repository.NotificationPreference {
	if s.preferences == nil {
		return defaultBenchmarkNotificationPreference(userID, orgID)
	}
	pref, err := s.preferences.Get(ctx, userID, orgID)
	if err != nil || pref == nil {
		return defaultBenchmarkNotificationPreference(userID, orgID)
	}
	return pref
}

func (s *BenchmarkInsightNotificationService) shouldSendDigest(ctx context.Context, userID, orgID uuid.UUID) bool {
	pref := s.preference(ctx, userID, orgID)
	return pref.EmailEnabled && pref.DigestEnabled && pref.DigestFrequency == benchmarkDigestFrequencyWeekly
}

func defaultBenchmarkNotificationPreference(userID, orgID uuid.UUID) *repository.NotificationPreference {
	return &repository.NotificationPreference{
		UserID:          userID,
		OrgID:           orgID,
		EmailEnabled:    true,
		InAppEnabled:    true,
		DigestFrequency: benchmarkDigestFrequencyWeekly,
	}
}

type benchmarkAggregateKey struct {
	industry          string
	companySizeBucket string
	metricName        string
}

func benchmarkAggregateByKey(aggregates []repository.BenchmarkAggregate) map[benchmarkAggregateKey]repository.BenchmarkAggregate {
	byKey := make(map[benchmarkAggregateKey]repository.BenchmarkAggregate, len(aggregates))
	for _, aggregate := range aggregates {
		byKey[benchmarkAggregateKey{
			industry:          aggregate.Industry,
			companySizeBucket: aggregate.CompanySizeBucket,
			metricName:        aggregate.MetricName,
		}] = aggregate
	}
	return byKey
}

func benchmarkAggregateKeyFor(contribution repository.BenchmarkContribution, metricName string) benchmarkAggregateKey {
	return benchmarkAggregateKey{
		industry:          contribution.Industry,
		companySizeBucket: contribution.CompanySizeBucket,
		metricName:        metricName,
	}
}

func benchmarkDigestSummaryFromPair(pair repository.BenchmarkContributionPair, aggregates map[benchmarkAggregateKey]repository.BenchmarkAggregate) (BenchmarkWeeklyDigestSummary, bool) {
	aggregate, ok := aggregates[benchmarkAggregateKeyFor(pair.Current, repository.BenchmarkMetricHealthScore)]
	if !ok {
		return BenchmarkWeeklyDigestSummary{}, false
	}
	return BenchmarkWeeklyDigestSummary{
		OrgID:              pair.Current.OrgID,
		MetricName:         repository.BenchmarkMetricHealthScore,
		PreviousPercentile: benchmarkPosition(pair.Previous.AvgHealthScore, aggregate),
		CurrentPercentile:  benchmarkPosition(pair.Current.AvgHealthScore, aggregate),
	}, true
}

type benchmarkContributionMetric struct {
	name     string
	previous float64
	current  float64
}

func benchmarkContributionMetrics(previous, current repository.BenchmarkContribution) []benchmarkContributionMetric {
	return []benchmarkContributionMetric{
		{name: repository.BenchmarkMetricHealthScore, previous: previous.AvgHealthScore, current: current.AvgHealthScore},
		{name: repository.BenchmarkMetricMRRPerCustomer, previous: float64(previous.AvgMRR), current: float64(current.AvgMRR)},
		{name: repository.BenchmarkMetricChurnRate, previous: previous.AvgChurnRate, current: current.AvgChurnRate},
		{name: repository.BenchmarkMetricIntegrationUsage, previous: float64(previous.ActiveIntegrationCount), current: float64(current.ActiveIntegrationCount)},
	}
}

func benchmarkInsightFromMetric(orgID uuid.UUID, metric benchmarkContributionMetric, aggregate repository.BenchmarkAggregate) (benchmarkInsight, bool) {
	previousPercentile := benchmarkPosition(metric.previous, aggregate)
	currentPercentile := benchmarkPosition(metric.current, aggregate)
	switch {
	case metric.previous >= aggregate.P50 && metric.current < aggregate.P50:
		return benchmarkInsight{
			orgID:              orgID,
			metricName:         metric.name,
			message:            fmt.Sprintf("Your %s dropped below the industry median (P50).", benchmarkMetricLabel(metric.name)),
			previousValue:      metric.previous,
			currentValue:       metric.current,
			previousPercentile: previousPercentile,
			currentPercentile:  currentPercentile,
		}, true
	case metric.previous < aggregate.P75 && metric.current >= aggregate.P75:
		return benchmarkInsight{
			orgID:              orgID,
			metricName:         metric.name,
			message:            fmt.Sprintf("Your %s rose above the industry P75 benchmark.", benchmarkMetricLabel(metric.name)),
			previousValue:      metric.previous,
			currentValue:       metric.current,
			previousPercentile: previousPercentile,
			currentPercentile:  currentPercentile,
		}, true
	default:
		return benchmarkInsight{}, false
	}
}

func benchmarkPosition(value float64, aggregate repository.BenchmarkAggregate) float64 {
	switch {
	case value < aggregate.P25:
		return 10
	case value < aggregate.P50:
		return 25
	case value < aggregate.P75:
		return 50
	case value < aggregate.P90:
		return 75
	default:
		return 90
	}
}

func benchmarkMetricLabel(metricName string) string {
	switch metricName {
	case repository.BenchmarkMetricHealthScore:
		return "health score"
	case repository.BenchmarkMetricMRRPerCustomer:
		return "MRR per customer"
	case repository.BenchmarkMetricChurnRate:
		return "churn rate"
	case repository.BenchmarkMetricIntegrationUsage:
		return "integration usage"
	default:
		return strings.ReplaceAll(metricName, "_", " ")
	}
}

func benchmarkDigestText(summary BenchmarkWeeklyDigestSummary) string {
	direction := "unchanged from"
	if summary.CurrentPercentile > summary.PreviousPercentile {
		direction = "up from"
	} else if summary.CurrentPercentile < summary.PreviousPercentile {
		direction = "down from"
	}
	return fmt.Sprintf("This week your %s ranked at P%.0f - %s P%.0f.", benchmarkMetricLabel(summary.MetricName), summary.CurrentPercentile, direction, summary.PreviousPercentile)
}

func benchmarkDigestHTML(summary BenchmarkWeeklyDigestSummary) string {
	return "<p>" + benchmarkDigestText(summary) + "</p>"
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

type BenchmarkRunner interface {
	RunOnce(ctx context.Context) error
}

type BenchmarkRunnerFunc func(ctx context.Context) error

func (f BenchmarkRunnerFunc) RunOnce(ctx context.Context) error {
	return f(ctx)
}

type BenchmarkWorkflow struct {
	runners []BenchmarkRunner
}

func NewBenchmarkWorkflow(runners ...BenchmarkRunner) *BenchmarkWorkflow {
	return &BenchmarkWorkflow{runners: runners}
}

func (w *BenchmarkWorkflow) RunOnce(ctx context.Context) error {
	for _, runner := range w.runners {
		if runner == nil {
			continue
		}
		if err := runner.RunOnce(ctx); err != nil {
			return fmt.Errorf("run benchmark workflow: %w", err)
		}
	}
	return nil
}

type BenchmarkScheduler struct {
	runner   BenchmarkRunner
	interval time.Duration
}

func NewBenchmarkScheduler(runner BenchmarkRunner, interval time.Duration) *BenchmarkScheduler {
	return &BenchmarkScheduler{runner: runner, interval: interval}
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
			if s.runner == nil {
				continue
			}
			if err := s.runner.RunOnce(ctx); err != nil {
				slog.Error("benchmark scheduler run failed", "error", err)
			}
		}
	}
}
