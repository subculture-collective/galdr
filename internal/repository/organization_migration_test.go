package repository

import (
	"os"
	"strings"
	"testing"
)

func readMigrationSQL(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	return string(data)
}

func TestOrganizationIndustryMigrationAddsNullableColumn(t *testing.T) {
	sql := readMigrationSQL(t, "../../migrations/000024_add_organization_industry.up.sql")

	if !strings.Contains(sql, "ADD COLUMN IF NOT EXISTS industry VARCHAR(50)") {
		t.Error("migration up file must add nullable organizations.industry column")
	}
	if strings.Contains(sql, "industry VARCHAR(50) NOT NULL") {
		t.Error("organizations.industry must allow NULL for existing organizations")
	}
}

func TestOrganizationIndustryMigrationAddsSegmentationIndex(t *testing.T) {
	sql := readMigrationSQL(t, "../../migrations/000024_add_organization_industry.up.sql")

	if !strings.Contains(sql, "idx_organizations_industry") {
		t.Error("migration up file must add industry index for benchmark segmentation")
	}
	if !strings.Contains(sql, "WHERE industry IS NOT NULL") {
		t.Error("industry segmentation index should only include classified organizations")
	}
}

func TestOrganizationIndustryMigrationRestrictsPredefinedOptions(t *testing.T) {
	sql := readMigrationSQL(t, "../../migrations/000024_add_organization_industry.up.sql")

	for _, industry := range []string{"SaaS", "E-commerce", "Fintech", "Healthcare", "Education", "Media", "Marketplace", "Agency", "Other"} {
		if !strings.Contains(sql, "'"+industry+"'") {
			t.Errorf("migration up file must allow predefined industry %q", industry)
		}
	}
	if !strings.Contains(sql, "CHECK (industry IS NULL OR industry IN") {
		t.Error("migration up file must reject unknown industry classifications")
	}
}

func TestOrganizationIndustryMigrationDownRemovesIndexAndColumn(t *testing.T) {
	sql := readMigrationSQL(t, "../../migrations/000024_add_organization_industry.down.sql")

	requiredStatements := []string{
		"DROP INDEX IF EXISTS idx_organizations_industry",
		"DROP CONSTRAINT IF EXISTS organizations_industry_check",
		"DROP COLUMN IF EXISTS industry",
	}
	for _, stmt := range requiredStatements {
		if !strings.Contains(sql, stmt) {
			t.Errorf("migration down file missing statement: %s", stmt)
		}
	}
}
