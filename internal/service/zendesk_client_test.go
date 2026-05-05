package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestZendeskClientListUsersUsesBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer zendesk-token" {
			t.Fatalf("Authorization header = %q, want Bearer zendesk-token", got)
		}
		if err := json.NewEncoder(w).Encode(ZendeskUserListResponse{Users: []ZendeskAPIUser{{ID: 42, Email: "user@example.com", Name: "Zendesk User", Role: "end-user"}}}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := &ZendeskClient{client: server.Client(), limiter: rate.NewLimiter(rate.Inf, 1)}
	resp, err := client.ListUsers(context.Background(), "zendesk-token", "ignored", server.URL)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(resp.Users) != 1 || resp.Users[0].Email != "user@example.com" {
		t.Fatalf("users = %#v", resp.Users)
	}
}

func TestZendeskClientListTicketsParsesSolvedAt(t *testing.T) {
	solvedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(ZendeskTicketListResponse{Tickets: []ZendeskAPITicket{{ID: 99, RequesterID: 42, Subject: "Help", Status: "solved", SolvedAt: &solvedAt}}}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := &ZendeskClient{client: server.Client(), limiter: rate.NewLimiter(rate.Inf, 1)}
	resp, err := client.ListTickets(context.Background(), "zendesk-token", "ignored", server.URL)
	if err != nil {
		t.Fatalf("ListTickets() error = %v", err)
	}
	if len(resp.Tickets) != 1 || resp.Tickets[0].SolvedAt == nil || !resp.Tickets[0].SolvedAt.Equal(solvedAt) {
		t.Fatalf("tickets = %#v", resp.Tickets)
	}
}
