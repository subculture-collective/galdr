package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const benchmarkMigrationPath = "../../migrations/000026_create_benchmarking"

func TestBenchmarkBucketConstants(t *testing.T) {
	cases := []struct {
		name     string
		got      string
		expected string
	}{
		{"1-10", BenchmarkBucket1To10, "1-10"},
		{"11-50", BenchmarkBucket11To50, "11-50"},
		{"51-200", BenchmarkBucket51To200, "51-200"},
		{"201-1000", BenchmarkBucket201To1000, "201-1000"},
		{"1000+", BenchmarkBucket1000Plus, "1000+"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.got)
			}
		})
	}
}

func TestBenchmarkContributionModelContainsOnlyOrgLevelFields(t *testing.T) {
	orgID := uuid.New()
	contribution := &BenchmarkContribution{
		ID:                  uuid.New(),
		OrgID:               orgID,
		Industry:            "saas",
		CompanySizeBucket:   BenchmarkBucket11To50,
		AvgHealthScore:      74.5,
		AvgMRR:              125000,
		AvgChurnRate:        0.12,
		CustomerCountBucket: BenchmarkBucket51To200,
		ContributedAt:       time.Now(),
	}

	if contribution.OrgID != orgID {
		t.Errorf("expected OrgID %s, got %s", orgID, contribution.OrgID)
	}
	if contribution.CompanySizeBucket != BenchmarkBucket11To50 {
		t.Errorf("expected company size bucket %q, got %q", BenchmarkBucket11To50, contribution.CompanySizeBucket)
	}
	if contribution.CustomerCountBucket != BenchmarkBucket51To200 {
		t.Errorf("expected customer count bucket %q, got %q", BenchmarkBucket51To200, contribution.CustomerCountBucket)
	}
	if contribution.AvgHealthScore != 74.5 {
		t.Errorf("expected avg health score 74.5, got %f", contribution.AvgHealthScore)
	}
}

func TestBenchmarkMigrationUpFileContainsTablesAndOptInFields(t *testing.T) {
	sql := readMigrationSQL(t, benchmarkMigrationPath+".up.sql")

	required := []string{
		"benchmarking_enabled",
		"company_size",
		"CREATE TABLE benchmark_contributions",
		"CREATE TABLE benchmark_aggregates",
		"industry",
		"customer_count_bucket",
		"avg_health_score",
		"avg_mrr",
		"avg_churn_rate",
	}
	for _, item := range required {
		if !strings.Contains(sql, item) {
			t.Errorf("migration up file missing %s", item)
		}
	}
}

func TestBenchmarkMigrationUpFilePreventsIndividualCustomerData(t *testing.T) {
	sql := readMigrationSQL(t, benchmarkMigrationPath+".up.sql")

	forbidden := []string{"customer_id ", "email ", "full_name ", "external_id ", "metadata "}
	for _, item := range forbidden {
		if strings.Contains(sql, item) {
			t.Errorf("benchmark migration must not include PII field %s", item)
		}
	}
}

func TestBenchmarkMigrationDownFileDropsTablesAndOptInFields(t *testing.T) {
	sql := readMigrationSQL(t, benchmarkMigrationPath+".down.sql")

	required := []string{
		"DROP TABLE IF EXISTS benchmark_aggregates",
		"DROP TABLE IF EXISTS benchmark_contributions",
		"DROP COLUMN IF EXISTS benchmarking_enabled",
		"DROP COLUMN IF EXISTS company_size",
	}
	for _, item := range required {
		if !strings.Contains(sql, item) {
			t.Errorf("migration down file missing %s", item)
		}
	}
}

func TestBenchmarkMigrationDoesNotOwnIndustryClassification(t *testing.T) {
	upSQL := readMigrationSQL(t, benchmarkMigrationPath+".up.sql")
	downSQL := readMigrationSQL(t, benchmarkMigrationPath+".down.sql")

	if strings.Contains(upSQL, "ADD COLUMN industry") {
		t.Error("benchmark migration must not add organizations.industry; industry classification owns that column")
	}
	if strings.Contains(upSQL, "ADD COLUMN IF NOT EXISTS industry") {
		t.Error("benchmark migration must not create organizations.industry; industry classification owns that column")
	}
	if strings.Contains(downSQL, "DROP COLUMN IF EXISTS industry") {
		t.Error("benchmark migration must not drop organizations.industry; industry classification owns that column")
	}
}
