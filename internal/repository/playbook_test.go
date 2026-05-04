package repository

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestPlaybookTriggerTypeConstants verifies the expected trigger type values.
func TestPlaybookTriggerTypeConstants(t *testing.T) {
	cases := []struct {
		name     string
		got      string
		expected string
	}{
		{"score_threshold", PlaybookTriggerScoreThreshold, "score_threshold"},
		{"customer_event", PlaybookTriggerCustomerEvent, "customer_event"},
		{"schedule", PlaybookTriggerSchedule, "schedule"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.got)
			}
		})
	}
}

// TestPlaybookActionTypeConstants verifies the expected action type values.
func TestPlaybookActionTypeConstants(t *testing.T) {
	cases := []struct {
		name     string
		got      string
		expected string
	}{
		{"send_email", PlaybookActionSendEmail, "send_email"},
		{"internal_alert", PlaybookActionInternalAlert, "internal_alert"},
		{"tag_customer", PlaybookActionTagCustomer, "tag_customer"},
		{"webhook", PlaybookActionWebhook, "webhook"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.got)
			}
		})
	}
}

// TestPlaybookExecutionStatusConstants verifies the expected execution status values.
func TestPlaybookExecutionStatusConstants(t *testing.T) {
	cases := []struct {
		name     string
		got      string
		expected string
	}{
		{"pending", PlaybookExecutionPending, "pending"},
		{"running", PlaybookExecutionRunning, "running"},
		{"success", PlaybookExecutionSuccess, "success"},
		{"failed", PlaybookExecutionFailed, "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.got)
			}
		})
	}
}

// TestPlaybookModel verifies the Playbook struct fields can be populated.
func TestPlaybookModel(t *testing.T) {
	orgID := uuid.New()
	p := &Playbook{
		ID:            uuid.New(),
		OrgID:         orgID,
		Name:          "Re-engage at-risk customers",
		Description:   "Triggered when score drops below threshold",
		Enabled:       true,
		TriggerType:   PlaybookTriggerScoreThreshold,
		TriggerConfig: map[string]any{"threshold": 40, "direction": "below"},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if p.OrgID != orgID {
		t.Errorf("expected OrgID %s, got %s", orgID, p.OrgID)
	}
	if p.TriggerType != PlaybookTriggerScoreThreshold {
		t.Errorf("expected trigger_type %q, got %q", PlaybookTriggerScoreThreshold, p.TriggerType)
	}
	if !p.Enabled {
		t.Error("expected Enabled to be true")
	}
	if p.TriggerConfig["threshold"] != 40 {
		t.Errorf("expected threshold 40, got %v", p.TriggerConfig["threshold"])
	}
}

// TestPlaybookActionModel verifies the PlaybookAction struct fields can be populated.
func TestPlaybookActionModel(t *testing.T) {
	playbookID := uuid.New()
	a := &PlaybookAction{
		ID:           uuid.New(),
		PlaybookID:   playbookID,
		ActionType:   PlaybookActionSendEmail,
		ActionConfig: map[string]any{"template": "reengagement", "subject": "We miss you"},
		OrderIndex:   0,
	}

	if a.PlaybookID != playbookID {
		t.Errorf("expected PlaybookID %s, got %s", playbookID, a.PlaybookID)
	}
	if a.ActionType != PlaybookActionSendEmail {
		t.Errorf("expected action_type %q, got %q", PlaybookActionSendEmail, a.ActionType)
	}
	if a.OrderIndex != 0 {
		t.Errorf("expected OrderIndex 0, got %d", a.OrderIndex)
	}
}

// TestPlaybookExecutionModel verifies the PlaybookExecution struct fields can be populated.
func TestPlaybookExecutionModel(t *testing.T) {
	playbookID := uuid.New()
	customerID := uuid.New()
	e := &PlaybookExecution{
		ID:          uuid.New(),
		PlaybookID:  playbookID,
		CustomerID:  &customerID,
		TriggeredAt: time.Now(),
		Status:      PlaybookExecutionSuccess,
		Result:      map[string]any{"emails_sent": 1},
	}

	if e.PlaybookID != playbookID {
		t.Errorf("expected PlaybookID %s, got %s", playbookID, e.PlaybookID)
	}
	if e.CustomerID == nil || *e.CustomerID != customerID {
		t.Errorf("expected CustomerID %s, got %v", customerID, e.CustomerID)
	}
	if e.Status != PlaybookExecutionSuccess {
		t.Errorf("expected status %q, got %q", PlaybookExecutionSuccess, e.Status)
	}
}

// TestPlaybookExecutionNilCustomer verifies CustomerID can be nil (org-level triggers).
func TestPlaybookExecutionNilCustomer(t *testing.T) {
	e := &PlaybookExecution{
		ID:         uuid.New(),
		PlaybookID: uuid.New(),
		Status:     PlaybookExecutionPending,
		Result:     map[string]any{},
	}
	if e.CustomerID != nil {
		t.Errorf("expected nil CustomerID, got %v", e.CustomerID)
	}
}

// TestPlaybookMigrationUpFileContainsTables verifies the up migration defines all required tables.
func TestPlaybookMigrationUpFileContainsTables(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000023_create_playbooks.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)

	requiredTables := []string{"playbooks", "playbook_actions", "playbook_executions"}
	for _, table := range requiredTables {
		if !strings.Contains(sql, "CREATE TABLE "+table) {
			t.Errorf("migration up file missing CREATE TABLE %s", table)
		}
	}
}

// TestPlaybookMigrationUpFileContainsIndexes verifies that required indexes are present.
func TestPlaybookMigrationUpFileContainsIndexes(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000023_create_playbooks.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)

	requiredIndexes := []string{
		"idx_playbooks_org_id",
		"idx_playbook_actions_playbook_order",
		"idx_playbook_executions_playbook_id",
		"idx_playbook_executions_playbook_triggered",
	}
	for _, idx := range requiredIndexes {
		if !strings.Contains(sql, idx) {
			t.Errorf("migration up file missing index %s", idx)
		}
	}
}

// TestPlaybookMigrationDownFileDropsTables verifies the down migration drops all tables.
func TestPlaybookMigrationDownFileDropsTables(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000023_create_playbooks.down.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)

	requiredDrops := []string{
		"DROP TABLE IF EXISTS playbook_executions",
		"DROP TABLE IF EXISTS playbook_actions",
		"DROP TABLE IF EXISTS playbooks",
	}
	for _, stmt := range requiredDrops {
		if !strings.Contains(sql, stmt) {
			t.Errorf("migration down file missing statement: %s", stmt)
		}
	}
}

// TestPlaybookMigrationUpFileContainsJSONB verifies JSONB columns are defined.
func TestPlaybookMigrationUpFileContainsJSONB(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000023_create_playbooks.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)

	jsonbColumns := []string{"trigger_config", "action_config", "result"}
	for _, col := range jsonbColumns {
		if !strings.Contains(sql, col) {
			t.Errorf("migration up file missing JSONB column %s", col)
		}
	}
}
