package repository

import (
	"os"
	"strings"
	"testing"
)

const churnModelsMigration = "../../migrations/000030_create_churn_models"

func TestChurnModelMigrationUpFileContainsModelVersionStore(t *testing.T) {
	data, err := os.ReadFile(churnModelsMigration + ".up.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)

	required := []string{
		"CREATE TABLE churn_models",
		"version       TEXT NOT NULL",
		"feature_names JSONB NOT NULL",
		"weights       JSONB NOT NULL",
		"metrics       JSONB NOT NULL",
		"UNIQUE (org_id, version)",
		"idx_churn_models_org_trained",
		"set_churn_models_updated_at",
	}
	for _, want := range required {
		if !strings.Contains(sql, want) {
			t.Errorf("migration up file missing %s", want)
		}
	}
}

func TestChurnModelMigrationDownFileDropsModelVersionStore(t *testing.T) {
	data, err := os.ReadFile(churnModelsMigration + ".down.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)

	required := []string{
		"DROP TRIGGER IF EXISTS set_churn_models_updated_at ON churn_models",
		"DROP TABLE IF EXISTS churn_models",
	}
	for _, want := range required {
		if !strings.Contains(sql, want) {
			t.Errorf("migration down file missing %s", want)
		}
	}
}
