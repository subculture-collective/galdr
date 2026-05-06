package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/repository"
)

func TestPlaybookThresholdTriggerFiresOnlyOnDropsBelowCrossing(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	customerID := uuid.New()
	playbookID := uuid.New()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	store := &fakeThresholdPlaybookStore{
		playbooks: []*repository.Playbook{
			{
				ID:          playbookID,
				OrgID:       orgID,
				Name:        "At-risk save",
				Enabled:     true,
				TriggerType: repository.PlaybookTriggerScoreThreshold,
				TriggerConfig: map[string]any{
					"threshold": 40,
					"direction": "drops_below",
				},
			},
		},
	}
	trigger := NewPlaybookThresholdTriggerService(PlaybookThresholdTriggerConfig{
		Playbooks:  store,
		Executions: store,
		Now:        func() time.Time { return now },
	})

	previous := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 41, RiskLevel: "yellow"}
	current := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 39, RiskLevel: "red"}
	if err := trigger.EvaluateScoreChange(ctx, previous, current); err != nil {
		t.Fatalf("expected crossing evaluation success, got %v", err)
	}
	if len(store.created) != 1 {
		t.Fatalf("expected one execution for crossing, got %d", len(store.created))
	}
	created := store.created[0]
	if created.PlaybookID != playbookID || created.CustomerID == nil || *created.CustomerID != customerID {
		t.Fatalf("unexpected execution target: %+v", created)
	}
	if created.Status != repository.PlaybookExecutionPending {
		t.Fatalf("expected pending execution, got %q", created.Status)
	}
	if created.Result["trigger"] != repository.PlaybookTriggerScoreThreshold {
		t.Fatalf("expected trigger metadata, got %+v", created.Result)
	}

	previous = current
	current = &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 35, RiskLevel: "red"}
	if err := trigger.EvaluateScoreChange(ctx, previous, current); err != nil {
		t.Fatalf("expected non-crossing evaluation success, got %v", err)
	}
	if len(store.created) != 1 {
		t.Fatalf("expected no continuous firing, got %d executions", len(store.created))
	}
}

func TestPlaybookThresholdTriggerRespectsRisesAboveDirection(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	customerID := uuid.New()
	playbookID := uuid.New()
	store := &fakeThresholdPlaybookStore{
		playbooks: []*repository.Playbook{
			{
				ID:          playbookID,
				OrgID:       orgID,
				Name:        "Expansion celebration",
				Enabled:     true,
				TriggerType: repository.PlaybookTriggerScoreThreshold,
				TriggerConfig: map[string]any{
					"threshold": 80,
					"direction": "rises_above",
				},
			},
		},
	}
	trigger := NewPlaybookThresholdTriggerService(PlaybookThresholdTriggerConfig{Playbooks: store, Executions: store})

	previous := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 79, RiskLevel: "green"}
	current := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 81, RiskLevel: "green"}
	if err := trigger.EvaluateScoreChange(ctx, previous, current); err != nil {
		t.Fatalf("expected rises-above evaluation success, got %v", err)
	}
	if len(store.created) != 1 {
		t.Fatalf("expected rises-above execution, got %d", len(store.created))
	}

	previous = &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 85, RiskLevel: "green"}
	current = &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 75, RiskLevel: "green"}
	if err := trigger.EvaluateScoreChange(ctx, previous, current); err != nil {
		t.Fatalf("expected opposite direction evaluation success, got %v", err)
	}
	if len(store.created) != 1 {
		t.Fatalf("expected opposite direction not to fire, got %d", len(store.created))
	}
}

func TestPlaybookThresholdTriggerSkipsRecentCustomerExecution(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	customerID := uuid.New()
	playbookID := uuid.New()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	store := &fakeThresholdPlaybookStore{
		playbooks: []*repository.Playbook{
			{
				ID:          playbookID,
				OrgID:       orgID,
				Name:        "At-risk save",
				Enabled:     true,
				TriggerType: repository.PlaybookTriggerScoreThreshold,
				TriggerConfig: map[string]any{
					"threshold":      40,
					"direction":      "drops_below",
					"cooldown_hours": 24,
				},
			},
		},
		recent: map[thresholdRecentExecution]time.Time{{playbookID: playbookID, customerID: customerID}: now.Add(-time.Hour)},
	}
	trigger := NewPlaybookThresholdTriggerService(PlaybookThresholdTriggerConfig{
		Playbooks:  store,
		Executions: store,
		Now:        func() time.Time { return now },
	})

	previous := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 40, RiskLevel: "yellow"}
	current := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 39, RiskLevel: "red"}
	if err := trigger.EvaluateScoreChange(ctx, previous, current); err != nil {
		t.Fatalf("expected cooldown evaluation success, got %v", err)
	}
	if len(store.created) != 0 {
		t.Fatalf("expected cooldown to suppress execution, got %d", len(store.created))
	}
}

func TestPlaybookThresholdTriggerCooldownIsCustomerScoped(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	customerID := uuid.New()
	otherCustomerID := uuid.New()
	playbookID := uuid.New()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	store := &fakeThresholdPlaybookStore{
		playbooks: []*repository.Playbook{
			{
				ID:          playbookID,
				OrgID:       orgID,
				Name:        "At-risk save",
				Enabled:     true,
				TriggerType: repository.PlaybookTriggerScoreThreshold,
				TriggerConfig: map[string]any{
					"threshold": 40,
					"direction": "drops_below",
				},
			},
		},
		recent: map[thresholdRecentExecution]time.Time{{playbookID: playbookID, customerID: otherCustomerID}: now.Add(-time.Hour)},
	}
	trigger := NewPlaybookThresholdTriggerService(PlaybookThresholdTriggerConfig{
		Playbooks:  store,
		Executions: store,
		Now:        func() time.Time { return now },
	})

	previous := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 40, RiskLevel: "yellow"}
	current := &repository.HealthScore{OrgID: orgID, CustomerID: customerID, OverallScore: 39, RiskLevel: "red"}
	if err := trigger.EvaluateScoreChange(ctx, previous, current); err != nil {
		t.Fatalf("expected customer-scoped cooldown evaluation success, got %v", err)
	}
	if len(store.created) != 1 {
		t.Fatalf("expected different customer's cooldown not to suppress execution, got %d", len(store.created))
	}
}

type thresholdRecentExecution struct {
	playbookID uuid.UUID
	customerID uuid.UUID
}

type fakeThresholdPlaybookStore struct {
	playbooks []*repository.Playbook
	created   []*repository.PlaybookExecution
	recent    map[thresholdRecentExecution]time.Time
}

func (f *fakeThresholdPlaybookStore) ListEnabledByTrigger(ctx context.Context, orgID uuid.UUID, triggerType string) ([]*repository.Playbook, error) {
	var result []*repository.Playbook
	for _, playbook := range f.playbooks {
		if playbook.OrgID == orgID && playbook.Enabled && playbook.TriggerType == triggerType {
			result = append(result, playbook)
		}
	}
	return result, nil
}

func (f *fakeThresholdPlaybookStore) HasRecentCustomerExecution(ctx context.Context, playbookID, customerID uuid.UUID, since time.Time) (bool, error) {
	if f.recent == nil {
		return false, nil
	}
	triggeredAt, ok := f.recent[thresholdRecentExecution{playbookID: playbookID, customerID: customerID}]
	return ok && !triggeredAt.Before(since), nil
}

func (f *fakeThresholdPlaybookStore) Create(ctx context.Context, execution *repository.PlaybookExecution) error {
	f.created = append(f.created, execution)
	return nil
}
