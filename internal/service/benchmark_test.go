package service

import (
	"context"
	"testing"

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
			contribution, err := anonymizer.Anonymize(BenchmarkOrgMetrics{
				OrgID:          uuid.New(),
				Industry:       "SaaS",
				CompanySize:    25,
				CustomerCount:  tc.customerCount,
				TotalMRR:       0,
				AvgHealthScore: 0,
				AvgChurnRate:   0,
			})

			if err == nil {
				t.Fatal("expected anonymizer to reject nonpositive customer count")
			}
			if contribution != nil {
				t.Fatal("expected no contribution for nonpositive customer count")
			}
		})
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

type fakeBenchmarkContributionRepo struct {
	created []*repository.BenchmarkContribution
}

func (f *fakeBenchmarkContributionRepo) CreateContribution(ctx context.Context, contribution *repository.BenchmarkContribution) error {
	f.created = append(f.created, contribution)
	return nil
}
