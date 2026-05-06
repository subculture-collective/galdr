package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/onnwee/pulse-score/internal/repository"
)

func TestPlaybookServiceCreatePersistsNestedActions(t *testing.T) {
	orgID := uuid.New()
	playbooks := &mockPlaybookRepo{}
	actions := &mockPlaybookActionRepo{}
	svc := NewPlaybookService(playbooks, actions)

	created, err := svc.Create(context.Background(), orgID, CreatePlaybookRequest{
		Name:          "  Save at-risk customers  ",
		Description:   "Notify CSMs",
		TriggerType:   repository.PlaybookTriggerScoreThreshold,
		TriggerConfig: map[string]any{"threshold": float64(50), "direction": "below"},
		Actions: []PlaybookActionRequest{
			{ActionType: repository.PlaybookActionInternalAlert, ActionConfig: map[string]any{"message": "Customer is at risk"}},
			{ActionType: repository.PlaybookActionTagCustomer, ActionConfig: map[string]any{"tag": "at-risk"}},
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.Name != "Save at-risk customers" {
		t.Fatalf("expected trimmed name, got %q", created.Name)
	}
	if !created.Enabled {
		t.Fatal("expected playbook enabled by default")
	}
	if len(created.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(created.Actions))
	}
	if actions.created[0].OrderIndex != 0 || actions.created[1].OrderIndex != 1 {
		t.Fatalf("expected action order indexes from request order, got %d and %d", actions.created[0].OrderIndex, actions.created[1].OrderIndex)
	}
	if playbooks.created.OrgID != orgID {
		t.Fatalf("expected org scoped create")
	}
}

func TestPlaybookServiceRejectsInvalidTriggerConfig(t *testing.T) {
	svc := NewPlaybookService(&mockPlaybookRepo{}, &mockPlaybookActionRepo{})

	_, err := svc.Create(context.Background(), uuid.New(), CreatePlaybookRequest{
		Name:          "Bad trigger",
		TriggerType:   repository.PlaybookTriggerScoreThreshold,
		TriggerConfig: map[string]any{"threshold": float64(150)},
	})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %T %v", err, err)
	}
	if validationErr.Field != "trigger_config.threshold" {
		t.Fatalf("expected threshold field, got %q", validationErr.Field)
	}
}

func TestPlaybookServiceUpdateReplacesNestedActions(t *testing.T) {
	orgID := uuid.New()
	playbookID := uuid.New()
	playbooks := &mockPlaybookRepo{existing: &repository.Playbook{ID: playbookID, OrgID: orgID, Name: "Old", Enabled: true}}
	actions := &mockPlaybookActionRepo{}
	svc := NewPlaybookService(playbooks, actions)

	updated, err := svc.Update(context.Background(), playbookID, orgID, UpdatePlaybookRequest{
		Name:          "Renewal save",
		TriggerType:   repository.PlaybookTriggerCustomerEvent,
		TriggerConfig: map[string]any{"event_type": "subscription.canceled"},
		Actions: []PlaybookActionRequest{
			{ActionType: repository.PlaybookActionSendEmail, ActionConfig: map[string]any{"subject": "Can we help?", "template": "renewal-save"}},
		},
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.TriggerType != repository.PlaybookTriggerCustomerEvent {
		t.Fatalf("expected updated trigger type, got %q", updated.TriggerType)
	}
	if actions.deletedPlaybookID != playbookID {
		t.Fatalf("expected actions deleted for playbook %s, got %s", playbookID, actions.deletedPlaybookID)
	}
	if len(updated.Actions) != 1 || updated.Actions[0].ActionType != repository.PlaybookActionSendEmail {
		t.Fatalf("expected replacement email action, got %#v", updated.Actions)
	}
}

func TestPlaybookServiceGetUsesTenantScope(t *testing.T) {
	orgID := uuid.New()
	playbookID := uuid.New()
	playbooks := &mockPlaybookRepo{}
	svc := NewPlaybookService(playbooks, &mockPlaybookActionRepo{})

	_, err := svc.GetByID(context.Background(), playbookID, orgID)
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected not found, got %T %v", err, err)
	}
	if playbooks.gotOrgID != orgID || playbooks.gotID != playbookID {
		t.Fatalf("expected tenant scoped lookup")
	}
}

func TestPlaybookServiceSetEnabled(t *testing.T) {
	orgID := uuid.New()
	playbookID := uuid.New()
	playbooks := &mockPlaybookRepo{existing: &repository.Playbook{ID: playbookID, OrgID: orgID, Name: "PB", Enabled: true, TriggerType: repository.PlaybookTriggerCustomerEvent, TriggerConfig: map[string]any{"event_type": "login"}}}
	svc := NewPlaybookService(playbooks, &mockPlaybookActionRepo{})

	updated, err := svc.SetEnabled(context.Background(), playbookID, orgID, SetPlaybookEnabledRequest{Enabled: false})
	if err != nil {
		t.Fatalf("SetEnabled returned error: %v", err)
	}
	if updated.Enabled {
		t.Fatal("expected disabled playbook")
	}
}

type mockPlaybookRepo struct {
	created *repository.Playbook
	existing *repository.Playbook
	listed []*repository.Playbook
	gotID uuid.UUID
	gotOrgID uuid.UUID
}

func (m *mockPlaybookRepo) List(ctx context.Context, orgID uuid.UUID) ([]*repository.Playbook, error) {
	m.gotOrgID = orgID
	return m.listed, nil
}

func (m *mockPlaybookRepo) GetByID(ctx context.Context, id, orgID uuid.UUID) (*repository.Playbook, error) {
	m.gotID = id
	m.gotOrgID = orgID
	if m.existing == nil {
		return nil, nil
	}
	copy := *m.existing
	return &copy, nil
}

func (m *mockPlaybookRepo) Create(ctx context.Context, p *repository.Playbook) error {
	copy := *p
	copy.ID = uuid.New()
	m.created = &copy
	*p = copy
	return nil
}

func (m *mockPlaybookRepo) Update(ctx context.Context, p *repository.Playbook) error {
	if m.existing == nil {
		return pgx.ErrNoRows
	}
	copy := *p
	m.existing = &copy
	return nil
}

func (m *mockPlaybookRepo) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	if m.existing == nil {
		return pgx.ErrNoRows
	}
	m.gotID = id
	m.gotOrgID = orgID
	return nil
}

type mockPlaybookActionRepo struct {
	created []*repository.PlaybookAction
	listed []*repository.PlaybookAction
	deletedPlaybookID uuid.UUID
}

func (m *mockPlaybookActionRepo) ListByPlaybook(ctx context.Context, playbookID uuid.UUID) ([]*repository.PlaybookAction, error) {
	if len(m.listed) > 0 {
		return m.listed, nil
	}
	return m.created, nil
}

func (m *mockPlaybookActionRepo) Create(ctx context.Context, a *repository.PlaybookAction) error {
	copy := *a
	copy.ID = uuid.New()
	m.created = append(m.created, &copy)
	*a = copy
	return nil
}

func (m *mockPlaybookActionRepo) DeleteByPlaybook(ctx context.Context, playbookID uuid.UUID) error {
	m.deletedPlaybookID = playbookID
	m.created = nil
	return nil
}
