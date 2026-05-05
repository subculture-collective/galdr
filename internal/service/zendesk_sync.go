package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// ZendeskSyncService handles syncing Zendesk data.
type ZendeskSyncService struct {
	oauthSvc  *ZendeskOAuthService
	client    *ZendeskClient
	users     *repository.ZendeskUserRepository
	tickets   *repository.ZendeskTicketRepository
	customers *repository.CustomerRepository
	events    *repository.CustomerEventRepository
}

type zendeskUserPageFetcher func(ctx context.Context, accessToken, subdomain, pageURL string) (*ZendeskUserListResponse, error)
type zendeskTicketPageFetcher func(ctx context.Context, accessToken, subdomain, pageURL string) (*ZendeskTicketListResponse, error)

// NewZendeskSyncService creates a new ZendeskSyncService.
func NewZendeskSyncService(oauthSvc *ZendeskOAuthService, client *ZendeskClient, users *repository.ZendeskUserRepository, tickets *repository.ZendeskTicketRepository, customers *repository.CustomerRepository, events *repository.CustomerEventRepository) *ZendeskSyncService {
	return &ZendeskSyncService{oauthSvc: oauthSvc, client: client, users: users, tickets: tickets, customers: customers, events: events}
}

// SyncUsers fetches all Zendesk users and maps them to customers by email.
func (s *ZendeskSyncService) SyncUsers(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	return s.syncUsers(ctx, orgID, "zendesk_users", "list users", true, s.client.ListUsers)
}

// SyncUsersSince fetches Zendesk users updated since time.
func (s *ZendeskSyncService) SyncUsersSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	fetch := func(ctx context.Context, accessToken, subdomain, pageURL string) (*ZendeskUserListResponse, error) {
		return s.client.ListUsersUpdatedSince(ctx, accessToken, subdomain, since, pageURL)
	}
	return s.syncUsers(ctx, orgID, "zendesk_users_incremental", "list users since", false, fetch)
}

func (s *ZendeskSyncService) syncUsers(ctx context.Context, orgID uuid.UUID, step, listErrorPrefix string, logCompletion bool, fetchPage zendeskUserPageFetcher) (*SyncProgress, error) {
	accessToken, subdomain, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	progress := &SyncProgress{Step: step}
	pageURL := ""
	for {
		resp, err := fetchPage(ctx, accessToken, subdomain, pageURL)
		if err != nil {
			return progress, fmt.Errorf("%s: %w", listErrorPrefix, err)
		}
		for _, user := range resp.Users {
			progress.Total++
			if err := s.upsertUserAndCustomer(ctx, orgID, user); err != nil {
				progress.Errors++
				continue
			}
			progress.Current++
		}
		if resp.NextPage == "" || resp.EndOfStream {
			break
		}
		pageURL = resp.NextPage
	}
	if logCompletion {
		slog.Info("zendesk user sync complete", "org_id", orgID, "total", progress.Total, "synced", progress.Current, "errors", progress.Errors)
	}
	return progress, nil
}

func (s *ZendeskSyncService) upsertUserAndCustomer(ctx context.Context, orgID uuid.UUID, user ZendeskAPIUser) error {
	zUser := &repository.ZendeskUser{OrgID: orgID, ZendeskUserID: zendeskInt64ID(user.ID), Email: user.Email, Name: user.Name, Role: user.Role, Metadata: map[string]any{"active": user.Active}}
	if err := s.users.Upsert(ctx, zUser); err != nil {
		return err
	}

	var customerID uuid.UUID
	if user.Email != "" {
		existing, err := s.customers.GetByEmail(ctx, orgID, user.Email)
		if err != nil {
			return err
		}
		if existing != nil {
			customerID = existing.ID
		}
	}
	if customerID == uuid.Nil {
		now := time.Now()
		customer := &repository.Customer{OrgID: orgID, ExternalID: zendeskInt64ID(user.ID), Source: zendeskProvider, Email: user.Email, Name: user.Name, FirstSeenAt: &now, LastSeenAt: &now, Metadata: map[string]any{"zendesk": map[string]any{"role": user.Role, "active": user.Active}}}
		if err := s.customers.UpsertByExternal(ctx, customer); err != nil {
			return err
		}
		customerID = customer.ID
	} else if err := s.customers.UpdateCompanyAndMetadata(ctx, customerID, "", map[string]any{"zendesk": map[string]any{"user_id": zendeskInt64ID(user.ID), "role": user.Role, "active": user.Active}}); err != nil {
		return err
	}

	return s.users.LinkCustomer(ctx, zUser.ID, customerID)
}

// SyncTickets fetches all Zendesk tickets and emits ticket events.
func (s *ZendeskSyncService) SyncTickets(ctx context.Context, orgID uuid.UUID) (*SyncProgress, error) {
	return s.syncTickets(ctx, orgID, "zendesk_tickets", "list tickets", true, true, s.client.ListTickets)
}

// SyncTicketsSince fetches Zendesk tickets updated since time.
func (s *ZendeskSyncService) SyncTicketsSince(ctx context.Context, orgID uuid.UUID, since time.Time) (*SyncProgress, error) {
	fetch := func(ctx context.Context, accessToken, subdomain, pageURL string) (*ZendeskTicketListResponse, error) {
		return s.client.ListTicketsUpdatedSince(ctx, accessToken, subdomain, since, pageURL)
	}
	return s.syncTickets(ctx, orgID, "zendesk_tickets_incremental", "list tickets since", false, false, fetch)
}

func (s *ZendeskSyncService) syncTickets(ctx context.Context, orgID uuid.UUID, step, listErrorPrefix string, emitEvents bool, logCompletion bool, fetchPage zendeskTicketPageFetcher) (*SyncProgress, error) {
	accessToken, subdomain, err := s.oauthSvc.GetAccessToken(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	progress := &SyncProgress{Step: step}
	pageURL := ""
	for {
		resp, err := fetchPage(ctx, accessToken, subdomain, pageURL)
		if err != nil {
			return progress, fmt.Errorf("%s: %w", listErrorPrefix, err)
		}
		for _, ticket := range resp.Tickets {
			progress.Total++
			if err := s.upsertTicket(ctx, orgID, ticket, emitEvents); err != nil {
				progress.Errors++
				continue
			}
			progress.Current++
		}
		if resp.NextPage == "" || resp.EndOfStream {
			break
		}
		pageURL = resp.NextPage
	}
	if logCompletion {
		slog.Info("zendesk ticket sync complete", "org_id", orgID, "total", progress.Total, "synced", progress.Current, "errors", progress.Errors)
	}
	return progress, nil
}

func (s *ZendeskSyncService) upsertTicket(ctx context.Context, orgID uuid.UUID, ticket ZendeskAPITicket, emitEvent bool) error {
	user, err := s.users.GetByZendeskID(ctx, orgID, zendeskInt64ID(ticket.RequesterID))
	if err != nil {
		return err
	}
	var customerID *uuid.UUID
	if user != nil && user.CustomerID != nil {
		customerID = user.CustomerID
	}
	zTicket := &repository.ZendeskTicket{OrgID: orgID, CustomerID: customerID, ZendeskTicketID: zendeskInt64ID(ticket.ID), ZendeskUserID: zendeskInt64ID(ticket.RequesterID), Subject: ticket.Subject, Status: ticket.Status, Priority: ticket.Priority, Type: ticket.Type, CreatedAtRemote: &ticket.CreatedAt, UpdatedAtRemote: &ticket.UpdatedAt, SolvedAt: ticket.SolvedAt, Metadata: map[string]any{"submitter_id": zendeskInt64ID(ticket.SubmitterID)}}
	if err := s.tickets.Upsert(ctx, zTicket); err != nil {
		return err
	}
	if emitEvent && customerID != nil {
		return s.events.Upsert(ctx, &repository.CustomerEvent{OrgID: orgID, CustomerID: *customerID, EventType: zendeskTicketEventType(ticket), Source: zendeskProvider, ExternalEventID: "ticket_" + zendeskInt64ID(ticket.ID), OccurredAt: ticket.UpdatedAt, Data: map[string]any{"ticket_id": zendeskInt64ID(ticket.ID), "status": ticket.Status, "priority": ticket.Priority, "subject": ticket.Subject}})
	}
	return nil
}

func zendeskTicketEventType(ticket ZendeskAPITicket) string {
	if ticket.Status == "solved" || ticket.Status == "closed" {
		return "ticket.solved"
	}
	if ticket.CreatedAt.Equal(ticket.UpdatedAt) {
		return "ticket.created"
	}
	return "ticket.updated"
}
