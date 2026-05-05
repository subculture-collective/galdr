package repository

import (
	"os"
	"strings"
	"testing"
)

func TestCustomerFeatureMigrationUpFileContainsFeatureStore(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000026_create_customer_features.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)

	required := []string{
		"CREATE TABLE customer_features",
		"features      JSONB NOT NULL",
		"UNIQUE (customer_id)",
		"idx_customer_features_org_calculated",
		"set_customer_features_updated_at",
	}
	for _, want := range required {
		if !strings.Contains(sql, want) {
			t.Errorf("migration up file missing %s", want)
		}
	}
}

func TestCustomerFeatureMigrationDownFileDropsFeatureStore(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000026_create_customer_features.down.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)

	required := []string{
		"DROP TRIGGER IF EXISTS set_customer_features_updated_at ON customer_features",
		"DROP TABLE IF EXISTS customer_features",
	}
	for _, want := range required {
		if !strings.Contains(sql, want) {
			t.Errorf("migration down file missing %s", want)
		}
	}
}
