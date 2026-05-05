package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

type salesforceAccessProvider interface {
	GetAccess(context.Context, uuid.UUID) (*SalesforceAccess, error)
}

type salesforceAPI interface {
	ListAccounts(context.Context, SalesforceAccess, *time.Time) ([]SalesforceAccount, error)
	ListContacts(context.Context, SalesforceAccess, *time.Time) ([]SalesforceContact, error)
	ListOpportunities(context.Context, SalesforceAccess, *time.Time) ([]SalesforceOpportunity, error)
}

type salesforceCustomerStore interface {
	UpsertByExternal(context.Context, *repository.Customer) error
}

type salesforceEventStore interface {
	Upsert(context.Context, *repository.CustomerEvent) error
}

// SalesforceSyncService syncs Salesforce CRM data into PulseScore customers and events.
type SalesforceSyncService struct {
	oauth     salesforceAccessProvider
	client    salesforceAPI
	customers salesforceCustomerStore
	events    salesforceEventStore
}

// NewSalesforceSyncService creates a Salesforce sync service.
func NewSalesforceSyncService(oauth salesforceAccessProvider, client salesforceAPI, customers salesforceCustomerStore, events salesforceEventStore) *SalesforceSyncService {
	return &SalesforceSyncService{oauth: oauth, client: client, customers: customers, events: events}
}

// SalesforceSyncResult summarizes a Salesforce sync run.
type SalesforceSyncResult struct {
	Accounts      *SyncProgress
	Contacts      *SyncProgress
	Opportunities *SyncProgress
	Duration      time.Duration
	Errors        []string
}

// Sync fetches Salesforce accounts, contacts, and opportunities.
func (s *SalesforceSyncService) Sync(ctx context.Context, orgID uuid.UUID, since *time.Time) (*SalesforceSyncResult, error) {
	if s.oauth == nil || s.client == nil || s.customers == nil || s.events == nil {
		return nil, &ValidationError{Field: providerSalesforce, Message: "Salesforce sync service is not configured"}
	}
	start := time.Now()
	access, err := s.oauth.GetAccess(ctx, orgID)
	if err != nil {
		return nil, err
	}

	accounts, err := s.client.ListAccounts(ctx, *access, since)
	if err != nil {
		return nil, fmt.Errorf("list salesforce accounts: %w", err)
	}
	contacts, err := s.client.ListContacts(ctx, *access, since)
	if err != nil {
		return nil, fmt.Errorf("list salesforce contacts: %w", err)
	}
	opportunities, err := s.client.ListOpportunities(ctx, *access, since)
	if err != nil {
		return nil, fmt.Errorf("list salesforce opportunities: %w", err)
	}

	accountByID := make(map[string]SalesforceAccount, len(accounts))
	for _, account := range accounts {
		accountByID[account.ID] = account
	}
	customerByAccount := make(map[string]uuid.UUID)
	result := &SalesforceSyncResult{
		Accounts:      &SyncProgress{Step: "accounts", Total: len(accounts), Current: len(accounts)},
		Contacts:      &SyncProgress{Step: "contacts", Total: len(contacts)},
		Opportunities: &SyncProgress{Step: "opportunities", Total: len(opportunities)},
	}

	for _, contact := range contacts {
		if strings.TrimSpace(contact.Email) == "" {
			result.Contacts.Errors++
			continue
		}
		customer := salesforceContactToCustomer(orgID, contact, accountByID[contact.AccountID])
		if err := s.customers.UpsertByExternal(ctx, customer); err != nil {
			result.Contacts.Errors++
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.Contacts.Current++
		if contact.AccountID != "" && customer.ID != uuid.Nil {
			customerByAccount[contact.AccountID] = customer.ID
		}
	}

	for _, opp := range opportunities {
		customerID := customerByAccount[opp.AccountID]
		if customerID == uuid.Nil {
			result.Opportunities.Errors++
			continue
		}
		if err := s.events.Upsert(ctx, salesforceOpportunityToEvent(orgID, customerID, opp)); err != nil {
			result.Opportunities.Errors++
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.Opportunities.Current++
	}

	result.Duration = time.Since(start)
	return result, nil
}

func salesforceContactToCustomer(orgID uuid.UUID, contact SalesforceContact, account SalesforceAccount) *repository.Customer {
	name := strings.TrimSpace(strings.TrimSpace(contact.FirstName) + " " + strings.TrimSpace(contact.LastName))
	lastSeen := contact.LastModifiedDate
	metadata := map[string]any{
		"salesforce": map[string]any{
			"account_id":          contact.AccountID,
			"account_name":        account.Name,
			"industry":            account.Industry,
			"website":             account.Website,
			"number_of_employees": account.NumberOfEmployees,
			"annual_revenue_cents": int(math.Round(account.AnnualRevenue * 100)),
		},
	}
	return &repository.Customer{
		OrgID:       orgID,
		ExternalID:  contact.ID,
		Source:      providerSalesforce,
		Email:       contact.Email,
		Name:        name,
		CompanyName: account.Name,
		Currency:    "usd",
		FirstSeenAt: &lastSeen,
		LastSeenAt:  &lastSeen,
		Metadata:    metadata,
	}
}

func salesforceOpportunityToEvent(orgID, customerID uuid.UUID, opp SalesforceOpportunity) *repository.CustomerEvent {
	occurredAt := opp.LastModifiedDate
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return &repository.CustomerEvent{
		OrgID:           orgID,
		CustomerID:      customerID,
		EventType:       "opportunity_stage_change",
		Source:          providerSalesforce,
		ExternalEventID: "opportunity_" + opp.ID + "_" + opp.StageName,
		OccurredAt:      occurredAt,
		Data: map[string]any{
			"opportunity_id": opp.ID,
			"name":           opp.Name,
			"stage":          opp.StageName,
			"amount_cents":   int(math.Round(opp.Amount * 100)),
			"close_date":     opp.CloseDate,
			"account_id":     opp.AccountID,
		},
	}
}
