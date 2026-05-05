package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/repository"
)

func TestSalesforceSyncMapsContactsAndOpportunities(t *testing.T) {
	orgID := uuid.New()
	customerID := uuid.New()
	api := &fakeSalesforceAPI{
		accounts: []SalesforceAccount{{ID: "001", Name: "Acme Corp", Industry: "SaaS"}},
		contacts: []SalesforceContact{{ID: "003", Email: "buyer@acme.test", FirstName: "Bea", LastName: "Buyer", AccountID: "001"}},
		opps:     []SalesforceOpportunity{{ID: "006", Name: "Renewal", StageName: "Proposal", Amount: 1200.50, AccountID: "001", LastModifiedDate: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}},
	}
	customers := &fakeSalesforceCustomers{id: customerID}
	events := &fakeSalesforceEvents{}
	svc := NewSalesforceSyncService(&fakeSalesforceAccess{}, api, customers, events)

	result, err := svc.Sync(ctxBackground(), orgID, nil)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if result.Contacts.Current != 1 || result.Accounts.Current != 1 || result.Opportunities.Current != 1 {
		t.Fatalf("unexpected result %#v", result)
	}
	if len(customers.upserts) != 1 {
		t.Fatalf("expected one customer upsert, got %d", len(customers.upserts))
	}
	customer := customers.upserts[0]
	if customer.Source != "salesforce" || customer.ExternalID != "003" || customer.Email != "buyer@acme.test" || customer.CompanyName != "Acme Corp" {
		t.Fatalf("unexpected customer %#v", customer)
	}
	if len(events.upserts) != 1 {
		t.Fatalf("expected one event upsert, got %d", len(events.upserts))
	}
	event := events.upserts[0]
	if event.CustomerID != customerID || event.EventType != "opportunity_stage_change" || event.ExternalEventID != "opportunity_006_Proposal" {
		t.Fatalf("unexpected event %#v", event)
	}
	if event.Data["amount_cents"] != 120050 {
		t.Fatalf("unexpected amount cents %#v", event.Data["amount_cents"])
	}
}

func ctxBackground() context.Context { return context.Background() }

type fakeSalesforceAccess struct{}

func (f *fakeSalesforceAccess) GetAccess(context.Context, uuid.UUID) (*SalesforceAccess, error) {
	return &SalesforceAccess{AccessToken: "token", InstanceURL: "https://example.my.salesforce.com"}, nil
}

type fakeSalesforceAPI struct {
	accounts []SalesforceAccount
	contacts []SalesforceContact
	opps     []SalesforceOpportunity
}

func (f *fakeSalesforceAPI) ListAccounts(context.Context, SalesforceAccess, *time.Time) ([]SalesforceAccount, error) {
	return f.accounts, nil
}

func (f *fakeSalesforceAPI) ListContacts(context.Context, SalesforceAccess, *time.Time) ([]SalesforceContact, error) {
	return f.contacts, nil
}

func (f *fakeSalesforceAPI) ListOpportunities(context.Context, SalesforceAccess, *time.Time) ([]SalesforceOpportunity, error) {
	return f.opps, nil
}

type fakeSalesforceCustomers struct {
	id      uuid.UUID
	upserts []*repository.Customer
}

func (f *fakeSalesforceCustomers) UpsertByExternal(_ context.Context, customer *repository.Customer) error {
	customer.ID = f.id
	f.upserts = append(f.upserts, customer)
	return nil
}

type fakeSalesforceEvents struct {
	upserts []*repository.CustomerEvent
}

func (f *fakeSalesforceEvents) Upsert(_ context.Context, event *repository.CustomerEvent) error {
	f.upserts = append(f.upserts, event)
	return nil
}
