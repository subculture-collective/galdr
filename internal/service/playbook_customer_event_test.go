package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/repository"
)

func TestPlaybookCustomerEventTriggerFiresOnMatchingEventType(t *testing.T) {
	orgID := uuid.New()
	customerID := uuid.New()
	playbookID := uuid.New()
	store := &fakeCustomerEventTriggerStore{
		playbooks: []*repository.Playbook{
			{
				ID:            playbookID,
				OrgID:         orgID,
				Enabled:       true,
				TriggerType:   repository.PlaybookTriggerCustomerEvent,
				TriggerConfig: map[string]any{"event_types": []any{"payment.failed"}},
			},
		},
	}
	trigger := NewPlaybookCustomerEventTriggerService(store, store)

	err := trigger.EvaluateCustomerEvent(context.Background(), &repository.CustomerEvent{
		ID:              uuid.New(),
		OrgID:           orgID,
		CustomerID:      customerID,
		EventType:       "payment.failed",
		Source:          "stripe",
		ExternalEventID: "evt_1",
		OccurredAt:      time.Now(),
		Data:            map[string]any{"amount_cents": 1500},
	})
	if err != nil {
		t.Fatalf("EvaluateCustomerEvent returned error: %v", err)
	}
	if len(store.executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(store.executions))
	}
	execution := store.executions[0]
	if execution.PlaybookID != playbookID {
		t.Fatalf("expected playbook %s, got %s", playbookID, execution.PlaybookID)
	}
	if execution.CustomerID == nil || *execution.CustomerID != customerID {
		t.Fatalf("expected customer %s, got %v", customerID, execution.CustomerID)
	}
	if execution.Status != repository.PlaybookExecutionPending {
		t.Fatalf("expected pending execution, got %s", execution.Status)
	}
	if execution.Result["event_type"] != "payment.failed" {
		t.Fatalf("expected event type result, got %+v", execution.Result)
	}
}

func TestPlaybookCustomerEventTriggerSupportsMultipleEventTypes(t *testing.T) {
	orgID := uuid.New()
	store := &fakeCustomerEventTriggerStore{
		playbooks: []*repository.Playbook{
			{
				ID:            uuid.New(),
				OrgID:         orgID,
				Enabled:       true,
				TriggerType:   repository.PlaybookTriggerCustomerEvent,
				TriggerConfig: map[string]any{"event_types": []any{"payment.failed", "ticket.opened"}},
			},
		},
	}
	trigger := NewPlaybookCustomerEventTriggerService(store, store)

	for _, eventType := range []string{"ticket.opened", "subscription.cancelled"} {
		err := trigger.EvaluateCustomerEvent(context.Background(), &repository.CustomerEvent{
			ID:         uuid.New(),
			OrgID:      orgID,
			CustomerID: uuid.New(),
			EventType:  eventType,
			Source:     "test",
			Data:       map[string]any{},
		})
		if err != nil {
			t.Fatalf("EvaluateCustomerEvent returned error: %v", err)
		}
	}

	if len(store.executions) != 1 {
		t.Fatalf("expected only matching event type to execute once, got %d", len(store.executions))
	}
}

func TestPlaybookCustomerEventTriggerFiltersConditions(t *testing.T) {
	orgID := uuid.New()
	store := &fakeCustomerEventTriggerStore{
		playbooks: []*repository.Playbook{
			{
				ID:          uuid.New(),
				OrgID:       orgID,
				Enabled:     true,
				TriggerType: repository.PlaybookTriggerCustomerEvent,
				TriggerConfig: map[string]any{
					"event_types": []any{"payment.failed"},
					"conditions": []any{
						map[string]any{"field": "amount_cents", "operator": "greater_than", "value": 1000},
						map[string]any{"field": "currency", "operator": "equals", "value": "usd"},
					},
				},
			},
		},
	}
	trigger := NewPlaybookCustomerEventTriggerService(store, store)

	lowAmount := &repository.CustomerEvent{
		ID:         uuid.New(),
		OrgID:      orgID,
		CustomerID: uuid.New(),
		EventType:  "payment.failed",
		Data:       map[string]any{"amount_cents": 500, "currency": "usd"},
	}
	matching := &repository.CustomerEvent{
		ID:         uuid.New(),
		OrgID:      orgID,
		CustomerID: uuid.New(),
		EventType:  "payment.failed",
		Data:       map[string]any{"amount_cents": 1500, "currency": "usd"},
	}

	if err := trigger.EvaluateCustomerEvent(context.Background(), lowAmount); err != nil {
		t.Fatalf("EvaluateCustomerEvent returned error: %v", err)
	}
	if err := trigger.EvaluateCustomerEvent(context.Background(), matching); err != nil {
		t.Fatalf("EvaluateCustomerEvent returned error: %v", err)
	}
	if len(store.executions) != 1 {
		t.Fatalf("expected only condition-matching event to execute once, got %d", len(store.executions))
	}
}

func TestPlaybookCustomerEventTriggerRequestsOnlyEnabledPlaybooks(t *testing.T) {
	orgID := uuid.New()
	store := &fakeCustomerEventTriggerStore{}
	trigger := NewPlaybookCustomerEventTriggerService(store, store)

	err := trigger.EvaluateCustomerEvent(context.Background(), &repository.CustomerEvent{
		ID:         uuid.New(),
		OrgID:      orgID,
		CustomerID: uuid.New(),
		EventType:  "payment.failed",
		Data:       map[string]any{},
	})
	if err != nil {
		t.Fatalf("EvaluateCustomerEvent returned error: %v", err)
	}
	if store.gotTriggerType != repository.PlaybookTriggerCustomerEvent {
		t.Fatalf("expected customer event trigger query, got %q", store.gotTriggerType)
	}
	if store.gotOrgID != orgID {
		t.Fatalf("expected org %s, got %s", orgID, store.gotOrgID)
	}
	if len(store.executions) != 0 {
		t.Fatalf("expected no executions without enabled playbooks, got %d", len(store.executions))
	}
}

type fakeCustomerEventTriggerStore struct {
	playbooks      []*repository.Playbook
	executions     []*repository.PlaybookExecution
	gotOrgID       uuid.UUID
	gotTriggerType string
}

func (f *fakeCustomerEventTriggerStore) ListEnabledByTrigger(_ context.Context, orgID uuid.UUID, triggerType string) ([]*repository.Playbook, error) {
	f.gotOrgID = orgID
	f.gotTriggerType = triggerType
	return f.playbooks, nil
}

func (f *fakeCustomerEventTriggerStore) Create(_ context.Context, execution *repository.PlaybookExecution) error {
	f.executions = append(f.executions, execution)
	return nil
}
