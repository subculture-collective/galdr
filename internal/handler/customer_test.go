package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

type mockCustomerService struct {
	listFn       func(ctx context.Context, params repository.CustomerListParams) (*service.CustomerListResponse, error)
	getDetailFn  func(ctx context.Context, customerID, orgID uuid.UUID) (*service.CustomerDetail, error)
	listEventsFn func(ctx context.Context, params repository.EventListParams) (*service.EventListResponse, error)
	listNotesFn  func(ctx context.Context, customerID, orgID, actorID uuid.UUID, actorRole string) (*service.CustomerNotesResponse, error)
	createNoteFn func(ctx context.Context, customerID, orgID, userID uuid.UUID, req service.CustomerNoteRequest) (*service.CustomerNoteResponse, error)
	updateNoteFn func(ctx context.Context, customerID, noteID, orgID, userID uuid.UUID, actorRole string, req service.CustomerNoteRequest) (*service.CustomerNoteResponse, error)
	deleteNoteFn func(ctx context.Context, customerID, noteID, orgID, userID uuid.UUID, actorRole string) error
}

func (m *mockCustomerService) List(ctx context.Context, params repository.CustomerListParams) (*service.CustomerListResponse, error) {
	return m.listFn(ctx, params)
}

func (m *mockCustomerService) GetDetail(ctx context.Context, customerID, orgID uuid.UUID) (*service.CustomerDetail, error) {
	return m.getDetailFn(ctx, customerID, orgID)
}

func (m *mockCustomerService) ListEvents(ctx context.Context, params repository.EventListParams) (*service.EventListResponse, error) {
	return m.listEventsFn(ctx, params)
}

func (m *mockCustomerService) ListNotes(ctx context.Context, customerID, orgID, actorID uuid.UUID, actorRole string) (*service.CustomerNotesResponse, error) {
	return m.listNotesFn(ctx, customerID, orgID, actorID, actorRole)
}

func (m *mockCustomerService) CreateNote(ctx context.Context, customerID, orgID, userID uuid.UUID, req service.CustomerNoteRequest) (*service.CustomerNoteResponse, error) {
	return m.createNoteFn(ctx, customerID, orgID, userID, req)
}

func (m *mockCustomerService) UpdateNote(ctx context.Context, customerID, noteID, orgID, userID uuid.UUID, actorRole string, req service.CustomerNoteRequest) (*service.CustomerNoteResponse, error) {
	return m.updateNoteFn(ctx, customerID, noteID, orgID, userID, actorRole, req)
}

func (m *mockCustomerService) DeleteNote(ctx context.Context, customerID, noteID, orgID, userID uuid.UUID, actorRole string) error {
	return m.deleteNoteFn(ctx, customerID, noteID, orgID, userID, actorRole)
}

func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestCustomerList_Unauthorized(t *testing.T) {
	h := NewCustomerHandler(&mockCustomerService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestCustomerList_Success(t *testing.T) {
	orgID := uuid.New()
	mock := &mockCustomerService{
		listFn: func(ctx context.Context, params repository.CustomerListParams) (*service.CustomerListResponse, error) {
			if params.OrgID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, params.OrgID)
			}
			return &service.CustomerListResponse{
				Customers: []service.CustomerListItem{{Name: "Acme"}},
				Pagination: service.PaginationMeta{
					Page: 1, PerPage: 25, Total: 1, TotalPages: 1,
				},
			}, nil
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp service.CustomerListResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Customers) != 1 {
		t.Errorf("expected 1 customer, got %d", len(resp.Customers))
	}
}

func TestCustomerList_QueryParams(t *testing.T) {
	orgID := uuid.New()
	var captured repository.CustomerListParams
	mock := &mockCustomerService{
		listFn: func(ctx context.Context, params repository.CustomerListParams) (*service.CustomerListResponse, error) {
			captured = params
			return &service.CustomerListResponse{}, nil
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers?page=2&per_page=10&sort=mrr&order=desc&risk=high&search=acme&source=stripe", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if captured.Page != 2 {
		t.Errorf("expected page 2, got %d", captured.Page)
	}
	if captured.PerPage != 10 {
		t.Errorf("expected per_page 10, got %d", captured.PerPage)
	}
	if captured.Sort != "mrr" {
		t.Errorf("expected sort mrr, got %s", captured.Sort)
	}
	if captured.Order != "desc" {
		t.Errorf("expected order desc, got %s", captured.Order)
	}
	if captured.Risk != "high" {
		t.Errorf("expected risk high, got %s", captured.Risk)
	}
	if captured.Search != "acme" {
		t.Errorf("expected search acme, got %s", captured.Search)
	}
	if captured.Source != "stripe" {
		t.Errorf("expected source stripe, got %s", captured.Source)
	}
}

func TestCustomerList_ServiceError(t *testing.T) {
	orgID := uuid.New()
	mock := &mockCustomerService{
		listFn: func(ctx context.Context, params repository.CustomerListParams) (*service.CustomerListResponse, error) {
			return nil, &service.ValidationError{Field: "risk", Message: "invalid risk level"}
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestCustomerGetDetail_Unauthorized(t *testing.T) {
	h := NewCustomerHandler(&mockCustomerService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/"+uuid.New().String(), nil)
	req = withChiParam(req, "id", uuid.New().String())
	rr := httptest.NewRecorder()

	h.GetDetail(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestCustomerGetDetail_InvalidUUID(t *testing.T) {
	orgID := uuid.New()
	h := NewCustomerHandler(&mockCustomerService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/not-a-uuid", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.GetDetail(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCustomerGetDetail_NotFound(t *testing.T) {
	orgID := uuid.New()
	customerID := uuid.New()
	mock := &mockCustomerService{
		getDetailFn: func(ctx context.Context, cID, oID uuid.UUID) (*service.CustomerDetail, error) {
			return nil, &service.NotFoundError{Resource: "customer", Message: "customer not found"}
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/"+customerID.String(), nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", customerID.String())
	rr := httptest.NewRecorder()

	h.GetDetail(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestCustomerGetDetail_Success(t *testing.T) {
	orgID := uuid.New()
	customerID := uuid.New()
	mock := &mockCustomerService{
		getDetailFn: func(ctx context.Context, cID, oID uuid.UUID) (*service.CustomerDetail, error) {
			if cID != customerID {
				t.Errorf("expected customerID %s, got %s", customerID, cID)
			}
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			return &service.CustomerDetail{
				Customer: service.CustomerInfo{ID: customerID, Name: "Acme"},
			}, nil
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/"+customerID.String(), nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", customerID.String())
	rr := httptest.NewRecorder()

	h.GetDetail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestCustomerListEvents_Unauthorized(t *testing.T) {
	h := NewCustomerHandler(&mockCustomerService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/"+uuid.New().String()+"/events", nil)
	req = withChiParam(req, "id", uuid.New().String())
	rr := httptest.NewRecorder()

	h.ListEvents(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestCustomerListEvents_InvalidUUID(t *testing.T) {
	orgID := uuid.New()
	h := NewCustomerHandler(&mockCustomerService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/bad/events", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", "bad")
	rr := httptest.NewRecorder()

	h.ListEvents(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCustomerListEvents_Success(t *testing.T) {
	orgID := uuid.New()
	customerID := uuid.New()
	var captured repository.EventListParams
	mock := &mockCustomerService{
		listEventsFn: func(ctx context.Context, params repository.EventListParams) (*service.EventListResponse, error) {
			captured = params
			return &service.EventListResponse{
				Events:     []*service.EventInfo{{EventType: "login"}},
				Pagination: service.PaginationMeta{Page: 1, PerPage: 25, Total: 1, TotalPages: 1},
			}, nil
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/customers/%s/events?page=3&per_page=5&type=login&from=2024-01-01T00:00:00Z&to=2024-12-31T23:59:59Z", customerID), nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	req = withChiParam(req, "id", customerID.String())
	rr := httptest.NewRecorder()

	h.ListEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if captured.CustomerID != customerID {
		t.Errorf("expected customerID %s, got %s", customerID, captured.CustomerID)
	}
	if captured.OrgID != orgID {
		t.Errorf("expected orgID %s, got %s", orgID, captured.OrgID)
	}
	if captured.Page != 3 {
		t.Errorf("expected page 3, got %d", captured.Page)
	}
	if captured.PerPage != 5 {
		t.Errorf("expected per_page 5, got %d", captured.PerPage)
	}
	if captured.EventType != "login" {
		t.Errorf("expected type login, got %s", captured.EventType)
	}
	if captured.From.IsZero() {
		t.Error("expected from to be set")
	}
	if captured.To.IsZero() {
		t.Error("expected to to be set")
	}
}

func TestCustomerListNotes_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	customerID := uuid.New()
	mock := &mockCustomerService{
		listNotesFn: func(ctx context.Context, cID, oID, actorID uuid.UUID, actorRole string) (*service.CustomerNotesResponse, error) {
			if cID != customerID {
				t.Errorf("expected customerID %s, got %s", customerID, cID)
			}
			if oID != orgID {
				t.Errorf("expected orgID %s, got %s", orgID, oID)
			}
			if actorID != userID {
				t.Errorf("expected userID %s, got %s", userID, actorID)
			}
			if actorRole != "member" {
				t.Errorf("expected role member, got %s", actorRole)
			}
			return &service.CustomerNotesResponse{Notes: []service.CustomerNoteResponse{{Content: "**hello**"}}}, nil
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/"+customerID.String()+"/notes", nil)
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	ctx = auth.WithRole(ctx, "member")
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", customerID.String())
	rr := httptest.NewRecorder()

	h.ListNotes(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestCustomerCreateNote_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	customerID := uuid.New()
	mock := &mockCustomerService{
		createNoteFn: func(ctx context.Context, cID, oID, uID uuid.UUID, req service.CustomerNoteRequest) (*service.CustomerNoteResponse, error) {
			if cID != customerID || oID != orgID || uID != userID {
				t.Fatalf("unexpected ids: customer=%s org=%s user=%s", cID, oID, uID)
			}
			if req.Content != "New note" {
				t.Errorf("expected content New note, got %q", req.Content)
			}
			return &service.CustomerNoteResponse{ID: uuid.New(), Content: req.Content}, nil
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers/"+customerID.String()+"/notes", strings.NewReader(`{"content":"New note"}`))
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", customerID.String())
	rr := httptest.NewRecorder()

	h.CreateNote(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestCustomerUpdateNote_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	customerID := uuid.New()
	noteID := uuid.New()
	mock := &mockCustomerService{
		updateNoteFn: func(ctx context.Context, cID, nID, oID, uID uuid.UUID, actorRole string, req service.CustomerNoteRequest) (*service.CustomerNoteResponse, error) {
			if cID != customerID || nID != noteID || oID != orgID || uID != userID {
				t.Fatalf("unexpected ids: customer=%s note=%s org=%s user=%s", cID, nID, oID, uID)
			}
			if actorRole != "admin" {
				t.Errorf("expected role admin, got %s", actorRole)
			}
			return &service.CustomerNoteResponse{ID: nID, Content: req.Content}, nil
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/customers/"+customerID.String()+"/notes/"+noteID.String(), strings.NewReader(`{"content":"Updated"}`))
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	ctx = auth.WithRole(ctx, "admin")
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", customerID.String())
	req = withChiParam(req, "noteID", noteID.String())
	rr := httptest.NewRecorder()

	h.UpdateNote(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestCustomerDeleteNote_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	customerID := uuid.New()
	noteID := uuid.New()
	mock := &mockCustomerService{
		deleteNoteFn: func(ctx context.Context, cID, nID, oID, uID uuid.UUID, actorRole string) error {
			if cID != customerID || nID != noteID || oID != orgID || uID != userID {
				t.Fatalf("unexpected ids: customer=%s note=%s org=%s user=%s", cID, nID, oID, uID)
			}
			if actorRole != "owner" {
				t.Errorf("expected role owner, got %s", actorRole)
			}
			return nil
		},
	}

	h := NewCustomerHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/customers/"+customerID.String()+"/notes/"+noteID.String(), nil)
	ctx := auth.WithOrgID(req.Context(), orgID)
	ctx = auth.WithUserID(ctx, userID)
	ctx = auth.WithRole(ctx, "owner")
	req = req.WithContext(ctx)
	req = withChiParam(req, "id", customerID.String())
	req = withChiParam(req, "noteID", noteID.String())
	rr := httptest.NewRecorder()

	h.DeleteNote(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}
