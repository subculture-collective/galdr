package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

func TestBenchmarkBucketCompanySize(t *testing.T) {
	cases := []struct {
		name     string
		input    int
		expected string
	}{
		{"zero defaults smallest anonymous bucket", 0, repository.BenchmarkBucket1To10},
		{"one", 1, repository.BenchmarkBucket1To10},
		{"ten", 10, repository.BenchmarkBucket1To10},
		{"eleven", 11, repository.BenchmarkBucket11To50},
		{"fifty", 50, repository.BenchmarkBucket11To50},
		{"fifty one", 51, repository.BenchmarkBucket51To200},
		{"two hundred", 200, repository.BenchmarkBucket51To200},
		{"two hundred one", 201, repository.BenchmarkBucket201To1000},
		{"thousand", 1000, repository.BenchmarkBucket201To1000},
		{"thousand one", 1001, repository.BenchmarkBucket1000Plus},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BucketCompanySize(tc.input); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestBenchmarkInsightNotificationsDetectBelowMedianDrop(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repo := &fakeBenchmarkInsightRepo{
		pairs: []repository.BenchmarkContributionPair{
			{
				Previous: benchmarkContributionForOrg(orgID, "saas", repository.BenchmarkBucket11To50, 72, 1000, 0.10, 2),
				Current:  benchmarkContributionForOrg(orgID, "saas", repository.BenchmarkBucket11To50, 48, 1000, 0.10, 2),
			},
		},
		aggregates: []repository.BenchmarkAggregate{
			{Industry: "saas", CompanySizeBucket: repository.BenchmarkBucket11To50, MetricName: repository.BenchmarkMetricHealthScore, P25: 40, P50: 60, P75: 80, P90: 90},
		},
		members: map[uuid.UUID][]repository.OrgMember{
			orgID: {{UserID: userID, Email: "owner@example.com", Role: "owner"}},
		},
		prefs: map[uuid.UUID]*repository.NotificationPreference{
			userID: {UserID: userID, OrgID: orgID, InAppEnabled: true, EmailEnabled: true},
		},
	}
	notifier := NewBenchmarkInsightNotificationService(BenchmarkInsightNotificationDeps{
		Contributions: repo,
		Aggregates:     repo,
		Members:        repo,
		Notifications:  repo,
		Preferences:    repo,
	})

	if err := notifier.RunOnce(context.Background()); err != nil {
		t.Fatalf("run notifier: %v", err)
	}

	if len(repo.notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(repo.notifications))
	}
	notification := repo.notifications[0]
	if notification.Type != "benchmark_insight" {
		t.Fatalf("expected benchmark_insight type, got %q", notification.Type)
	}
	if !strings.Contains(notification.Message, "dropped below the industry median") {
		t.Fatalf("expected below-median message, got %q", notification.Message)
	}
	if notification.Data["metric_name"] != repository.BenchmarkMetricHealthScore {
		t.Fatalf("expected metric data, got %#v", notification.Data)
	}
}

func TestBenchmarkInsightWeeklyDigestEmailsOptedInMembers(t *testing.T) {
	orgID := uuid.New()
	weeklyUserID := uuid.New()
	disabledUserID := uuid.New()
	repo := &fakeBenchmarkInsightRepo{
		members: map[uuid.UUID][]repository.OrgMember{
			orgID: {
				{UserID: weeklyUserID, Email: "weekly@example.com", Role: "owner"},
				{UserID: disabledUserID, Email: "disabled@example.com", Role: "admin"},
			},
		},
		prefs: map[uuid.UUID]*repository.NotificationPreference{
			weeklyUserID:   {UserID: weeklyUserID, OrgID: orgID, EmailEnabled: true, DigestEnabled: true, DigestFrequency: "weekly"},
			disabledUserID: {UserID: disabledUserID, OrgID: orgID, EmailEnabled: true, DigestEnabled: false, DigestFrequency: "weekly"},
		},
	}
	emails := &fakeBenchmarkEmailSender{}
	notifier := NewBenchmarkInsightNotificationService(BenchmarkInsightNotificationDeps{
		Members:     repo,
		Preferences: repo,
		Emails:      emails,
	})

	err := notifier.SendWeeklyDigest(context.Background(), []BenchmarkWeeklyDigestSummary{
		{OrgID: orgID, OrgName: "Acme", MetricName: repository.BenchmarkMetricHealthScore, PreviousPercentile: 58, CurrentPercentile: 65},
	})
	if err != nil {
		t.Fatalf("send digest: %v", err)
	}

	if len(emails.sent) != 1 {
		t.Fatalf("expected 1 digest email, got %d", len(emails.sent))
	}
	if emails.sent[0].To != "weekly@example.com" {
		t.Fatalf("expected weekly recipient, got %q", emails.sent[0].To)
	}
	if !strings.Contains(emails.sent[0].TextBody, "ranked at P65 - up from P58") {
		t.Fatalf("expected benchmark percentile copy, got %q", emails.sent[0].TextBody)
	}
}

func TestBenchmarkInsightWeeklyDigestBuildsLatestSummaries(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repo := &fakeBenchmarkInsightRepo{
		pairs: []repository.BenchmarkContributionPair{
			{
				Previous: benchmarkContributionForOrg(orgID, "saas", repository.BenchmarkBucket11To50, 58, 1000, 0.10, 2),
				Current:  benchmarkContributionForOrg(orgID, "saas", repository.BenchmarkBucket11To50, 65, 1000, 0.10, 2),
			},
		},
		aggregates: []repository.BenchmarkAggregate{
			{Industry: "saas", CompanySizeBucket: repository.BenchmarkBucket11To50, MetricName: repository.BenchmarkMetricHealthScore, P25: 25, P50: 50, P75: 75, P90: 90},
		},
		members: map[uuid.UUID][]repository.OrgMember{
			orgID: {{UserID: userID, Email: "weekly@example.com", Role: "owner"}},
		},
		prefs: map[uuid.UUID]*repository.NotificationPreference{
			userID: {UserID: userID, OrgID: orgID, EmailEnabled: true, DigestEnabled: true, DigestFrequency: "weekly"},
		},
	}
	emails := &fakeBenchmarkEmailSender{}
	notifier := NewBenchmarkInsightNotificationService(BenchmarkInsightNotificationDeps{
		Contributions: repo,
		Aggregates:     repo,
		Members:        repo,
		Preferences:    repo,
		Emails:         emails,
	})

	if err := notifier.SendWeeklyDigestFromLatest(context.Background()); err != nil {
		t.Fatalf("send latest digest: %v", err)
	}

	if len(emails.sent) != 1 {
		t.Fatalf("expected 1 digest email, got %d", len(emails.sent))
	}
	if !strings.Contains(emails.sent[0].TextBody, "This week your health score ranked at P50 - unchanged from P50") {
		t.Fatalf("expected generated digest copy, got %q", emails.sent[0].TextBody)
	}
}

func TestBenchmarkWorkflowRunsContributionAggregationAndInsightsInOrder(t *testing.T) {
	var calls []string
	workflow := NewBenchmarkWorkflow(
		&fakeBenchmarkRunner{name: "contributions", calls: &calls},
		&fakeBenchmarkRunner{name: "aggregates", calls: &calls},
		&fakeBenchmarkRunner{name: "insights", calls: &calls},
	)

	if err := workflow.RunOnce(context.Background()); err != nil {
		t.Fatalf("run workflow: %v", err)
	}

	got := strings.Join(calls, ",")
	if got != "contributions,aggregates,insights" {
		t.Fatalf("expected workflow order, got %q", got)
	}
}

func TestBenchmarkAnonymizerStripsPIIAndBucketsRawMetrics(t *testing.T) {
	orgID := uuid.New()
	anonymizer := NewBenchmarkAnonymizer()

	contribution, err := anonymizer.Anonymize(BenchmarkOrgMetrics{
		OrgID:          orgID,
		OrgName:        "Acme Secret Co",
		Industry:       "SaaS",
		CompanySize:    42,
		CustomerCount:  147,
		TotalMRR:       987654,
		AvgHealthScore: 81.25,
		AvgChurnRate:   0.08,
		PIISamples: []BenchmarkPIISample{
			{Email: "ceo@acme.example", Name: "Ada Lovelace", ExternalID: "cus_secret"},
		},
	})
	if err != nil {
		t.Fatalf("anonymize failed: %v", err)
	}

	if contribution.OrgID != orgID {
		t.Errorf("expected org id preserved for internal dedupe")
	}
	if contribution.Industry != "saas" {
		t.Errorf("expected normalized industry saas, got %q", contribution.Industry)
	}
	if contribution.CompanySizeBucket != repository.BenchmarkBucket11To50 {
		t.Errorf("expected company size bucket 11-50, got %q", contribution.CompanySizeBucket)
	}
	if contribution.CustomerCountBucket != repository.BenchmarkBucket51To200 {
		t.Errorf("expected customer count bucket 51-200, got %q", contribution.CustomerCountBucket)
	}
	if contribution.AvgMRR != 6718 {
		t.Errorf("expected avg mrr 6718, got %d", contribution.AvgMRR)
	}
}

func TestBenchmarkIndustrySegmentsUsePredefinedOrganizationIndustries(t *testing.T) {
	for _, tc := range []struct {
		industry string
		segment  string
	}{
		{"SaaS", "saas"},
		{"E-commerce", "e-commerce"},
		{"Fintech", "fintech"},
		{"Healthcare", "healthcare"},
		{"Education", "education"},
		{"Media", "media"},
		{"Marketplace", "marketplace"},
		{"Agency", "agency"},
		{"Other", "other"},
	} {
		t.Run(tc.industry, func(t *testing.T) {
			if got := NormalizeBenchmarkIndustry(tc.industry); got != tc.segment {
				t.Fatalf("expected %q segment, got %q", tc.segment, got)
			}
		})
	}
}

func TestBenchmarkIndustrySegmentsRejectFreeFormAliases(t *testing.T) {
	for _, industry := range []string{"AI", "Artificial Intelligence", "Software", "Consumer"} {
		t.Run(industry, func(t *testing.T) {
			if got := NormalizeBenchmarkIndustry(industry); got != unknownBenchmarkIndustry {
				t.Fatalf("expected free-form industry to be anonymized as unknown, got %q", got)
			}
		})
	}
}

func TestBenchmarkAnonymizerRejectsPIIIndustrySegments(t *testing.T) {
	anonymizer := NewBenchmarkAnonymizer()

	contribution, err := anonymizer.Anonymize(BenchmarkOrgMetrics{
		OrgID:          uuid.New(),
		Industry:       "Acme Secret Co ceo@acme.example",
		CompanySize:    42,
		CustomerCount:  100,
		TotalMRR:       100000,
		AvgHealthScore: 72,
		AvgChurnRate:   0.05,
	})
	if err != nil {
		t.Fatalf("anonymize failed: %v", err)
	}

	if contribution.Industry != "unknown" {
		t.Fatalf("expected PII-like industry to be anonymized as unknown, got %q", contribution.Industry)
	}
}

func TestBenchmarkAnonymizerRejectsNonpositiveCustomerCounts(t *testing.T) {
	anonymizer := NewBenchmarkAnonymizer()

	for _, tc := range []struct {
		name          string
		customerCount int
	}{
		{"zero", 0},
		{"negative", -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metrics := validBenchmarkOrgMetrics()
			metrics.CustomerCount = tc.customerCount

			contribution, err := anonymizer.Anonymize(metrics)

			if err == nil {
				t.Fatal("expected anonymizer to reject nonpositive customer count")
			}
			if contribution != nil {
				t.Fatal("expected no contribution for nonpositive customer count")
			}
		})
	}
}

func TestBenchmarkAnonymizerRejectsNegativeMRR(t *testing.T) {
	anonymizer := NewBenchmarkAnonymizer()

	metrics := validBenchmarkOrgMetrics()
	metrics.TotalMRR = -1

	contribution, err := anonymizer.Anonymize(metrics)

	if err == nil {
		t.Fatal("expected anonymizer to reject negative MRR")
	}
	if contribution != nil {
		t.Fatal("expected no contribution for negative MRR")
	}
}

func validBenchmarkOrgMetrics() BenchmarkOrgMetrics {
	return BenchmarkOrgMetrics{
		OrgID:          uuid.New(),
		Industry:       "SaaS",
		CompanySize:    25,
		CustomerCount:  10,
		TotalMRR:       100000,
		AvgHealthScore: 70,
		AvgChurnRate:   0.1,
	}
}

func TestBenchmarkPipelineSkipsOptedOutOrganizations(t *testing.T) {
	optedInOrg := repository.Organization{ID: uuid.New(), Industry: "saas", CompanySize: 25, BenchmarkingEnabled: true}
	optedOutOrg := repository.Organization{ID: uuid.New(), Industry: "fintech", CompanySize: 80, BenchmarkingEnabled: false}
	orgs := &fakeBenchmarkOrgRepo{orgs: []repository.Organization{optedInOrg, optedOutOrg}}
	metrics := &fakeBenchmarkMetricsRepo{
		customerCounts: map[uuid.UUID]int{optedInOrg.ID: 24},
		totalMRR:       map[uuid.UUID]int64{optedInOrg.ID: 240000},
		avgScores:      map[uuid.UUID]float64{optedInOrg.ID: 72.4},
		churnRates:     map[uuid.UUID]float64{optedInOrg.ID: 0.11},
		integrations:   map[uuid.UUID]int{optedInOrg.ID: 2},
	}
	contributions := &fakeBenchmarkContributionRepo{}
	pipeline := NewBenchmarkPipeline(orgs, metrics, contributions, NewBenchmarkAnonymizer())

	if err := pipeline.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once failed: %v", err)
	}

	if len(contributions.created) != 1 {
		t.Fatalf("expected 1 contribution, got %d", len(contributions.created))
	}
	if contributions.created[0].OrgID != optedInOrg.ID {
		t.Errorf("expected opted-in org contribution only")
	}
	if contributions.created[0].CompanySizeBucket != repository.BenchmarkBucket11To50 {
		t.Errorf("expected anonymized company size bucket, got %q", contributions.created[0].CompanySizeBucket)
	}
	if contributions.created[0].ActiveIntegrationCount != 2 {
		t.Errorf("expected active integration count 2, got %d", contributions.created[0].ActiveIntegrationCount)
	}
}

func TestBenchmarkPipelineSkipsOrganizationsWithoutCustomers(t *testing.T) {
	org := repository.Organization{ID: uuid.New(), Industry: "saas", CompanySize: 25, BenchmarkingEnabled: true}
	orgs := &fakeBenchmarkOrgRepo{orgs: []repository.Organization{org}}
	metrics := &fakeBenchmarkMetricsRepo{
		customerCounts: map[uuid.UUID]int{org.ID: 0},
	}
	contributions := &fakeBenchmarkContributionRepo{}
	pipeline := NewBenchmarkPipeline(orgs, metrics, contributions, NewBenchmarkAnonymizer())

	if err := pipeline.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once failed: %v", err)
	}

	if len(contributions.created) != 0 {
		t.Fatalf("expected no contribution for org without customers, got %d", len(contributions.created))
	}
}

func TestBenchmarkPipelineSkipsEnabledOrganizationsWithoutIndustry(t *testing.T) {
	classifiedOrg := repository.Organization{ID: uuid.New(), Industry: "SaaS", CompanySize: 25, BenchmarkingEnabled: true}
	unclassifiedOrg := repository.Organization{ID: uuid.New(), Industry: " ", CompanySize: 80, BenchmarkingEnabled: true}
	orgs := &fakeBenchmarkOrgRepo{orgs: []repository.Organization{classifiedOrg, unclassifiedOrg}}
	metrics := &fakeBenchmarkMetricsRepo{
		customerCounts: map[uuid.UUID]int{classifiedOrg.ID: 24},
		totalMRR:       map[uuid.UUID]int64{classifiedOrg.ID: 240000},
		avgScores:      map[uuid.UUID]float64{classifiedOrg.ID: 72.4},
		churnRates:     map[uuid.UUID]float64{classifiedOrg.ID: 0.11},
	}
	contributions := &fakeBenchmarkContributionRepo{}
	pipeline := NewBenchmarkPipeline(orgs, metrics, contributions, NewBenchmarkAnonymizer())

	if err := pipeline.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once failed: %v", err)
	}

	if len(contributions.created) != 1 {
		t.Fatalf("expected 1 classified contribution, got %d", len(contributions.created))
	}
	if contributions.created[0].OrgID != classifiedOrg.ID {
		t.Fatalf("expected only classified org contribution")
	}
}

func TestBenchmarkAggregationServiceCalculatesSegmentPercentiles(t *testing.T) {
	segmentIndustry := "saas"
	segmentBucket := repository.BenchmarkBucket11To50
	contributions := &fakeBenchmarkAggregateRepo{
		contributions: []repository.BenchmarkContribution{
			benchmarkContribution(segmentIndustry, segmentBucket, 10, 1000, 0.10, 1),
			benchmarkContribution(segmentIndustry, segmentBucket, 20, 2000, 0.20, 2),
			benchmarkContribution(segmentIndustry, segmentBucket, 30, 3000, 0.30, 3),
			benchmarkContribution(segmentIndustry, segmentBucket, 40, 4000, 0.40, 4),
			benchmarkContribution(segmentIndustry, segmentBucket, 50, 5000, 0.50, 5),
			benchmarkContribution("fintech", repository.BenchmarkBucket51To200, 90, 9000, 0.90, 9),
		},
	}
	service := NewBenchmarkAggregationService(contributions, contributions)

	if err := service.RunOnce(context.Background()); err != nil {
		t.Fatalf("aggregate once failed: %v", err)
	}

	health := contributions.aggregateByMetric(repository.BenchmarkMetricHealthScore)
	if health == nil {
		t.Fatal("expected health score aggregate")
	}
	if health.Industry != segmentIndustry || health.CompanySizeBucket != segmentBucket {
		t.Fatalf("expected saas 11-50 segment, got %s %s", health.Industry, health.CompanySizeBucket)
	}
	if health.SampleCount != 5 {
		t.Fatalf("expected sample count 5, got %d", health.SampleCount)
	}
	assertFloatEqual(t, health.P25, 20)
	assertFloatEqual(t, health.P50, 30)
	assertFloatEqual(t, health.P75, 40)
	assertFloatEqual(t, health.P90, 46)

	mrr := contributions.aggregateByMetric(repository.BenchmarkMetricMRRPerCustomer)
	if mrr == nil {
		t.Fatal("expected mrr per customer aggregate")
	}
	assertFloatEqual(t, mrr.P90, 4600)

	churn := contributions.aggregateByMetric(repository.BenchmarkMetricChurnRate)
	if churn == nil {
		t.Fatal("expected churn rate aggregate")
	}
	assertFloatEqual(t, churn.P50, 0.30)

	usage := contributions.aggregateByMetric(repository.BenchmarkMetricIntegrationUsage)
	if usage == nil {
		t.Fatal("expected integration usage aggregate")
	}
	assertFloatEqual(t, usage.P75, 4)
}

func TestBenchmarkAggregationServiceSkipsSegmentsBelowMinimumSampleSize(t *testing.T) {
	repo := &fakeBenchmarkAggregateRepo{
		contributions: []repository.BenchmarkContribution{
			benchmarkContribution("saas", repository.BenchmarkBucket11To50, 10, 1000, 0.10, 1),
			benchmarkContribution("saas", repository.BenchmarkBucket11To50, 20, 2000, 0.20, 2),
			benchmarkContribution("saas", repository.BenchmarkBucket11To50, 30, 3000, 0.30, 3),
			benchmarkContribution("saas", repository.BenchmarkBucket11To50, 40, 4000, 0.40, 4),
		},
	}
	service := NewBenchmarkAggregationService(repo, repo)

	if err := service.RunOnce(context.Background()); err != nil {
		t.Fatalf("aggregate once failed: %v", err)
	}
	if len(repo.aggregates) != 0 {
		t.Fatalf("expected no aggregates below minimum sample size, got %d", len(repo.aggregates))
	}
}

func TestBenchmarkAggregationServiceExcludesStaleAndOutlierContributions(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeBenchmarkAggregateRepo{
		contributions: []repository.BenchmarkContribution{
			benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 10, 1000, 0.10, 1, now.Add(-1*time.Hour)),
			benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 11, 1100, 0.11, 1, now.Add(-2*time.Hour)),
			benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 12, 1200, 0.12, 1, now.Add(-3*time.Hour)),
			benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 13, 1300, 0.13, 1, now.Add(-4*time.Hour)),
			benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 14, 1400, 0.14, 1, now.Add(-5*time.Hour)),
			benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 100, 10000, 0.90, 9, now.Add(-6*time.Hour)),
			benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 60, 6000, 0.60, 6, now.Add(-31*24*time.Hour)),
		},
	}
	service := NewBenchmarkAggregationService(repo, repo)

	if err := service.RunOnce(context.Background()); err != nil {
		t.Fatalf("aggregate once failed: %v", err)
	}

	health := repo.aggregateByMetric(repository.BenchmarkMetricHealthScore)
	if health == nil {
		t.Fatal("expected health score aggregate")
	}
	if health.SampleCount != 5 {
		t.Fatalf("expected stale and outlier contributions excluded, got sample count %d", health.SampleCount)
	}
	assertFloatEqual(t, health.P50, 12)
	if health.QualityScore <= 0 || health.QualityScore > 100 {
		t.Fatalf("expected quality score in 1..100, got %.2f", health.QualityScore)
	}
	if health.QualityLevel == "" {
		t.Fatal("expected quality level")
	}
}

func TestBenchmarkAggregationServiceExcludesInvalidPersistedContributions(t *testing.T) {
	now := time.Now().UTC()
	valid := []repository.BenchmarkContribution{
		benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 50, 5000, 0.10, 1, now.Add(-1*time.Hour)),
		benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 51, 5100, 0.11, 1, now.Add(-2*time.Hour)),
		benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 52, 5200, 0.12, 1, now.Add(-3*time.Hour)),
		benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 53, 5300, 0.13, 1, now.Add(-4*time.Hour)),
		benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 54, 5400, 0.14, 1, now.Add(-5*time.Hour)),
	}
	invalidNegativeMRR := benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 55, -1, 0.15, 1, now.Add(-6*time.Hour))
	invalidScore := benchmarkContributionAt("saas", repository.BenchmarkBucket11To50, 101, 5500, 0.15, 1, now.Add(-7*time.Hour))
	repo := &fakeBenchmarkAggregateRepo{contributions: append(valid, invalidNegativeMRR, invalidScore)}
	service := NewBenchmarkAggregationService(repo, repo)

	if err := service.RunOnce(context.Background()); err != nil {
		t.Fatalf("aggregate once failed: %v", err)
	}

	health := repo.aggregateByMetric(repository.BenchmarkMetricHealthScore)
	if health == nil {
		t.Fatal("expected aggregate from valid persisted contributions")
	}
	if health.SampleCount != len(valid) {
		t.Fatalf("expected invalid persisted contributions excluded, got sample count %d", health.SampleCount)
	}
	assertFloatEqual(t, health.P50, 52)
}

func benchmarkContribution(industry, bucket string, score float64, mrr int64, churnRate float64, integrationCount int) repository.BenchmarkContribution {
	return benchmarkContributionAt(industry, bucket, score, mrr, churnRate, integrationCount, time.Now().UTC())
}

func benchmarkContributionForOrg(orgID uuid.UUID, industry, bucket string, score float64, mrr int64, churnRate float64, integrationCount int) repository.BenchmarkContribution {
	contribution := benchmarkContribution(industry, bucket, score, mrr, churnRate, integrationCount)
	contribution.OrgID = orgID
	contribution.ID = uuid.New()
	return contribution
}

func benchmarkContributionAt(industry, bucket string, score float64, mrr int64, churnRate float64, integrationCount int, contributedAt time.Time) repository.BenchmarkContribution {
	return repository.BenchmarkContribution{
		OrgID:                  uuid.New(),
		Industry:               industry,
		CompanySizeBucket:      bucket,
		AvgHealthScore:         score,
		AvgMRR:                 mrr,
		AvgChurnRate:           churnRate,
		ActiveIntegrationCount: integrationCount,
		ContributedAt:          contributedAt,
	}
}

func assertFloatEqual(t *testing.T, got, expected float64) {
	t.Helper()
	if got != expected {
		t.Fatalf("expected %.4f, got %.4f", expected, got)
	}
}

type fakeBenchmarkOrgRepo struct {
	orgs []repository.Organization
}

func (f *fakeBenchmarkOrgRepo) ListBenchmarkingEnabled(ctx context.Context) ([]repository.Organization, error) {
	var enabled []repository.Organization
	for _, org := range f.orgs {
		if org.BenchmarkingEnabled {
			enabled = append(enabled, org)
		}
	}
	return enabled, nil
}

type fakeBenchmarkMetricsRepo struct {
	customerCounts map[uuid.UUID]int
	totalMRR       map[uuid.UUID]int64
	avgScores      map[uuid.UUID]float64
	churnRates     map[uuid.UUID]float64
	integrations   map[uuid.UUID]int
}

func (f *fakeBenchmarkMetricsRepo) CountCustomers(ctx context.Context, orgID uuid.UUID) (int, error) {
	return f.customerCounts[orgID], nil
}

func (f *fakeBenchmarkMetricsRepo) TotalMRR(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return f.totalMRR[orgID], nil
}

func (f *fakeBenchmarkMetricsRepo) AverageHealthScore(ctx context.Context, orgID uuid.UUID) (float64, error) {
	return f.avgScores[orgID], nil
}

func (f *fakeBenchmarkMetricsRepo) ChurnRate(ctx context.Context, orgID uuid.UUID) (float64, error) {
	return f.churnRates[orgID], nil
}

func (f *fakeBenchmarkMetricsRepo) ActiveIntegrationCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	return f.integrations[orgID], nil
}

type fakeBenchmarkContributionRepo struct {
	created []*repository.BenchmarkContribution
}

func (f *fakeBenchmarkContributionRepo) CreateContribution(ctx context.Context, contribution *repository.BenchmarkContribution) error {
	f.created = append(f.created, contribution)
	return nil
}

type fakeBenchmarkAggregateRepo struct {
	contributions []repository.BenchmarkContribution
	aggregates    []*repository.BenchmarkAggregate
}

func (f *fakeBenchmarkAggregateRepo) ListLatestContributions(ctx context.Context) ([]repository.BenchmarkContribution, error) {
	return f.contributions, nil
}

func (f *fakeBenchmarkAggregateRepo) CreateAggregate(ctx context.Context, aggregate *repository.BenchmarkAggregate) error {
	f.aggregates = append(f.aggregates, aggregate)
	return nil
}

func (f *fakeBenchmarkAggregateRepo) aggregateByMetric(metric string) *repository.BenchmarkAggregate {
	for _, aggregate := range f.aggregates {
		if aggregate.MetricName == metric {
			return aggregate
		}
	}
	return nil
}

type fakeBenchmarkInsightRepo struct {
	pairs         []repository.BenchmarkContributionPair
	aggregates    []repository.BenchmarkAggregate
	members       map[uuid.UUID][]repository.OrgMember
	prefs         map[uuid.UUID]*repository.NotificationPreference
	notifications []*repository.Notification
}

func (f *fakeBenchmarkInsightRepo) ListLatestContributionPairs(ctx context.Context) ([]repository.BenchmarkContributionPair, error) {
	return f.pairs, nil
}

func (f *fakeBenchmarkInsightRepo) ListLatestAggregates(ctx context.Context) ([]repository.BenchmarkAggregate, error) {
	return f.aggregates, nil
}

func (f *fakeBenchmarkInsightRepo) ListMembers(ctx context.Context, orgID uuid.UUID) ([]repository.OrgMember, error) {
	return f.members[orgID], nil
}

func (f *fakeBenchmarkInsightRepo) Get(ctx context.Context, userID, orgID uuid.UUID) (*repository.NotificationPreference, error) {
	if pref, ok := f.prefs[userID]; ok {
		return pref, nil
	}
	return &repository.NotificationPreference{
		UserID:          userID,
		OrgID:           orgID,
		EmailEnabled:    true,
		InAppEnabled:    true,
		DigestFrequency: benchmarkDigestFrequencyWeekly,
	}, nil
}

func (f *fakeBenchmarkInsightRepo) Create(ctx context.Context, notification *repository.Notification) error {
	f.notifications = append(f.notifications, notification)
	return nil
}

type fakeBenchmarkEmailSender struct {
	sent []SendEmailParams
}

func (f *fakeBenchmarkEmailSender) SendEmail(ctx context.Context, params SendEmailParams) (string, error) {
	f.sent = append(f.sent, params)
	return "message-id", nil
}

type fakeBenchmarkRunner struct {
	name  string
	calls *[]string
}

func (f *fakeBenchmarkRunner) RunOnce(ctx context.Context) error {
	*f.calls = append(*f.calls, f.name)
	return nil
}
