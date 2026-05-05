package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSalesforceClientQueriesAccountsContactsOpportunities(t *testing.T) {
	var queries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/services/data/v59.0/query") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token_123" {
			t.Fatalf("unexpected auth header %q", got)
		}
		query := r.URL.Query().Get("q")
		queries = append(queries, query)

		var records []map[string]any
		switch {
		case strings.Contains(query, "FROM Account"):
			records = []map[string]any{{"Id": "001", "Name": "Acme"}}
		case strings.Contains(query, "FROM Contact"):
			records = []map[string]any{{"Id": "003", "Email": "buyer@acme.test", "AccountId": "001"}}
		case strings.Contains(query, "FROM Opportunity"):
			records = []map[string]any{{"Id": "006", "Name": "Renewal", "StageName": "Proposal", "AccountId": "001"}}
		default:
			t.Fatalf("unexpected query %q", query)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"done": true, "records": records})
	}))
	defer server.Close()

	client := NewSalesforceClient()
	access := SalesforceAccess{AccessToken: "token_123", InstanceURL: server.URL}
	ctx := context.Background()

	accounts, err := client.ListAccounts(ctx, access, nil)
	if err != nil || len(accounts) != 1 || accounts[0].ID != "001" {
		t.Fatalf("unexpected accounts %#v err %v", accounts, err)
	}
	contacts, err := client.ListContacts(ctx, access, nil)
	if err != nil || len(contacts) != 1 || contacts[0].Email != "buyer@acme.test" {
		t.Fatalf("unexpected contacts %#v err %v", contacts, err)
	}
	opps, err := client.ListOpportunities(ctx, access, nil)
	if err != nil || len(opps) != 1 || opps[0].StageName != "Proposal" {
		t.Fatalf("unexpected opportunities %#v err %v", opps, err)
	}

	if len(queries) != 3 {
		t.Fatalf("expected 3 queries, got %d", len(queries))
	}
}
