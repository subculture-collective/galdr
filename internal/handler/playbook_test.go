package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

func TestPlaybookHandlerCreate(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	svc := &mockPlaybookService{response: &service.PlaybookResponse{Playbook: repository.Playbook{ID: uuid.New(), OrgID: orgID, Name: "At-risk", Enabled: true}}}
	h := NewPlaybookHandler(svc)
	body := `{"name":"At-risk","trigger_type":"score_threshold","trigger_config":{"threshold":50},"actions":[{"action_type":"internal_alert","action_config":{"message":"At risk"}}]}`
	req := authedPlaybookRequest(http.MethodPost, "/playbooks", body, orgID, userID)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.created.OrgID != orgID || svc.created.Request.Name != "At-risk" {
		t.Fatalf("expected create called with org and body, got %#v", svc.created)
	}
}

func TestPlaybookHandlerListWrapsResponse(t *testing.T) {
	orgID := uuid.New()
	svc := &mockPlaybookService{list: []*service.PlaybookResponse{{Playbook: repository.Playbook{ID: uuid.New(), OrgID: orgID, Name: "One"}}}}
	h := NewPlaybookHandler(svc)
	req := authedPlaybookRequest(http.MethodGet, "/playbooks", "", orgID, uuid.New())
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var payload struct { Playbooks []service.PlaybookResponse `json:"playbooks"` }
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Playbooks) != 1 || payload.Playbooks[0].Name != "One" {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestPlaybookHandlerGetInvalidID(t *testing.T) {
	h := NewPlaybookHandler(&mockPlaybookService{})
	req := authedPlaybookRequest(http.MethodGet, "/playbooks/nope", "", uuid.New(), uuid.New())
	req = withChiParam(req, "id", "nope")
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPlaybookHandlerUpdate(t *testing.T) {
	orgID := uuid.New()
	playbookID := uuid.New()
	svc := &mockPlaybookService{response: &service.PlaybookResponse{Playbook: repository.Playbook{ID: playbookID, OrgID: orgID, Name: "Updated"}}}
	h := NewPlaybookHandler(svc)
	req := authedPlaybookRequest(http.MethodPut, "/playbooks/"+playbookID.String(), `{"name":"Updated","trigger_type":"customer_event","trigger_config":{"event_type":"login"}}`, orgID, uuid.New())
	req = withChiParam(req, "id", playbookID.String())
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.updated.ID != playbookID || svc.updated.Request.Name != "Updated" {
		t.Fatalf("expected update call, got %#v", svc.updated)
	}
}

func TestPlaybookHandlerSetEnabled(t *testing.T) {
	orgID := uuid.New()
	playbookID := uuid.New()
	svc := &mockPlaybookService{response: &service.PlaybookResponse{Playbook: repository.Playbook{ID: playbookID, OrgID: orgID, Enabled: false}}}
	h := NewPlaybookHandler(svc)
	req := authedPlaybookRequest(http.MethodPut, "/playbooks/"+playbookID.String()+"/enable", `{"enabled":false}`, orgID, uuid.New())
	req = withChiParam(req, "id", playbookID.String())
	rr := httptest.NewRecorder()

	h.SetEnabled(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if svc.enabled.Request.Enabled {
		t.Fatal("expected disabled request")
	}
}

func TestPlaybookHandlerDeleteNotFound(t *testing.T) {
	orgID := uuid.New()
	playbookID := uuid.New()
	svc := &mockPlaybookService{err: &service.NotFoundError{Resource: "playbook", Message: "playbook not found"}}
	h := NewPlaybookHandler(svc)
	req := authedPlaybookRequest(http.MethodDelete, "/playbooks/"+playbookID.String(), "", orgID, uuid.New())
	req = withChiParam(req, "id", playbookID.String())
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func authedPlaybookRequest(method, path, body string, orgID, userID uuid.UUID) *http.Request {
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reader)
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	return req.WithContext(ctx)
}

type playbookCall[T any] struct {
	ID uuid.UUID
	OrgID uuid.UUID
	Request T
}

type mockPlaybookService struct {
	response *service.PlaybookResponse
	list []*service.PlaybookResponse
	err error
	created playbookCall[service.CreatePlaybookRequest]
	updated playbookCall[service.UpdatePlaybookRequest]
	enabled playbookCall[service.SetPlaybookEnabledRequest]
}

func (m *mockPlaybookService) List(ctx context.Context, orgID uuid.UUID) ([]*service.PlaybookResponse, error) {
	return m.list, m.err
}

func (m *mockPlaybookService) GetByID(ctx context.Context, id, orgID uuid.UUID) (*service.PlaybookResponse, error) {
	return m.response, m.err
}

func (m *mockPlaybookService) Create(ctx context.Context, orgID uuid.UUID, req service.CreatePlaybookRequest) (*service.PlaybookResponse, error) {
	m.created = playbookCall[service.CreatePlaybookRequest]{OrgID: orgID, Request: req}
	return m.response, m.err
}

func (m *mockPlaybookService) Update(ctx context.Context, id, orgID uuid.UUID, req service.UpdatePlaybookRequest) (*service.PlaybookResponse, error) {
	m.updated = playbookCall[service.UpdatePlaybookRequest]{ID: id, OrgID: orgID, Request: req}
	return m.response, m.err
}

func (m *mockPlaybookService) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	return m.err
}

func (m *mockPlaybookService) SetEnabled(ctx context.Context, id, orgID uuid.UUID, req service.SetPlaybookEnabledRequest) (*service.PlaybookResponse, error) {
	m.enabled = playbookCall[service.SetPlaybookEnabledRequest]{ID: id, OrgID: orgID, Request: req}
	return m.response, m.err
}

var _ playbookServicer = (*mockPlaybookService)(nil)

func TestPlaybookHandlerUnauthorized(t *testing.T) {
	h := NewPlaybookHandler(&mockPlaybookService{})
	req := httptest.NewRequest(http.MethodGet, "/playbooks", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestPlaybookHandlerValidationError(t *testing.T) {
	svc := &mockPlaybookService{err: &service.ValidationError{Field: "name", Message: "name is required"}}
	h := NewPlaybookHandler(svc)
	req := authedPlaybookRequest(http.MethodPost, "/playbooks", `{"name":""}`, uuid.New(), uuid.New())
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestPlaybookHandlerInternalError(t *testing.T) {
	svc := &mockPlaybookService{err: errors.New("boom")}
	h := NewPlaybookHandler(svc)
	req := authedPlaybookRequest(http.MethodGet, "/playbooks", "", uuid.New(), uuid.New())
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
