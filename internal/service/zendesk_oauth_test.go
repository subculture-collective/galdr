package service

import (
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestZendeskConnectURLRequiresSubdomain(t *testing.T) {
	svc := NewZendeskOAuthService(ZendeskOAuthConfig{ClientID: "client"}, nil)

	_, err := svc.ConnectURL(uuid.New(), "")
	if err == nil {
		t.Fatal("expected missing subdomain error")
	}
}

func TestZendeskConnectURLRejectsHostInjection(t *testing.T) {
	svc := NewZendeskOAuthService(ZendeskOAuthConfig{ClientID: "client"}, nil)

	_, err := svc.ConnectURL(uuid.New(), "acme.zendesk.com.evil")
	if err == nil {
		t.Fatal("expected invalid subdomain error")
	}
}

func TestZendeskConnectURLSuccess(t *testing.T) {
	orgID := uuid.New()
	svc := NewZendeskOAuthService(ZendeskOAuthConfig{
		ClientID:         "test-client-id",
		OAuthRedirectURL: "http://localhost/callback",
	}, nil)

	raw, err := svc.ConnectURL(orgID, "https://Acme-Support.zendesk.com")
	if err != nil {
		t.Fatalf("ConnectURL error: %v", err)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if parsed.Host != "acme-support.zendesk.com" || parsed.Path != "/oauth/authorizations/new" {
		t.Fatalf("unexpected url: %s", raw)
	}
	if got := parsed.Query().Get("client_id"); got != "test-client-id" {
		t.Fatalf("expected client_id, got %q", got)
	}
	if got := parsed.Query().Get("scope"); got != "read" {
		t.Fatalf("expected read scope, got %q", got)
	}
	state := parsed.Query().Get("state")
	if !strings.HasPrefix(state, orgID.String()+":acme-support:") {
		t.Fatalf("unexpected state %q", state)
	}
}
