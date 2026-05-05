package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/repository"
)

func TestWebhookActionPostsPayloadWithSignature(t *testing.T) {
	secret := "whsec_test"
	playbookID := uuid.New()
	actionID := uuid.New()
	customerID := uuid.New()

	var seenRequest map[string]any
	var seenSignature string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected json content type, got %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-Custom") != "galdr" {
			t.Fatalf("expected configured header, got %q", r.Header.Get("X-Custom"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		seenSignature = r.Header.Get("X-PulseScore-Signature")
		if seenSignature != expectedWebhookSignature(secret, body) {
			t.Fatalf("expected valid signature, got %q", seenSignature)
		}
		if err := json.Unmarshal(body, &seenRequest); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	executor := NewWebhookActionExecutor(WebhookActionExecutorConfig{HTTPClient: server.Client()})
	result, err := executor.Execute(context.Background(), WebhookActionRequest{
		Playbook: &repository.Playbook{ID: playbookID, Name: "At-risk outreach", TriggerType: repository.PlaybookTriggerCustomerEvent},
		Action: &repository.PlaybookAction{
			ID:         actionID,
			ActionType: repository.PlaybookActionWebhook,
			ActionConfig: map[string]any{
				"url":                   server.URL,
				"headers":               map[string]any{"X-Custom": "galdr"},
				"include_customer_data": true,
				"signing_secret":        secret,
			},
		},
		Customer:     &repository.Customer{ID: customerID, Email: "billing@acme.test", Name: "Acme", CompanyName: "Acme Inc", MRRCents: 4200},
		TriggerEvent: map[string]any{"type": "score_drop", "score": float64(39)},
	})
	if err != nil {
		t.Fatalf("expected webhook success, got %v", err)
	}
	if result.StatusCode != http.StatusAccepted || result.Attempts != 1 || result.LatencyMS < 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if seenRequest["trigger_event"].(map[string]any)["type"] != "score_drop" {
		t.Fatalf("expected trigger payload, got %+v", seenRequest)
	}
	if seenRequest["playbook"].(map[string]any)["id"] != playbookID.String() {
		t.Fatalf("expected playbook summary, got %+v", seenRequest["playbook"])
	}
	if seenRequest["customer"].(map[string]any)["email"] != "billing@acme.test" {
		t.Fatalf("expected customer summary, got %+v", seenRequest["customer"])
	}
}

func TestWebhookActionRetriesTransientFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	executor := NewWebhookActionExecutor(WebhookActionExecutorConfig{
		HTTPClient:  server.Client(),
		RetryDelays: []time.Duration{0, 0},
	})
	result, err := executor.Execute(context.Background(), minimalWebhookRequest(server.URL))
	if err != nil {
		t.Fatalf("expected retry success, got %v", err)
	}
	if attempts != 3 || result.Attempts != 3 || result.StatusCode != http.StatusNoContent {
		t.Fatalf("expected third attempt success, attempts=%d result=%+v", attempts, result)
	}
}

func TestWebhookActionRejectsNonHTTPSURL(t *testing.T) {
	executor := NewWebhookActionExecutor(WebhookActionExecutorConfig{})
	_, err := executor.Execute(context.Background(), minimalWebhookRequest("http://example.test/hook"))
	if err == nil || !errors.Is(err, ErrWebhookURLMustBeHTTPS) {
		t.Fatalf("expected HTTPS validation error, got %v", err)
	}
}

func TestWebhookActionRecordsPermanentFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	executor := NewWebhookActionExecutor(WebhookActionExecutorConfig{HTTPClient: server.Client()})
	result, err := executor.Execute(context.Background(), minimalWebhookRequest(server.URL))
	if err == nil {
		t.Fatal("expected webhook failure")
	}
	if attempts != 1 || result.StatusCode != http.StatusBadRequest || result.Attempts != 1 || result.Error == "" {
		t.Fatalf("expected recorded one-attempt failure, attempts=%d result=%+v", attempts, result)
	}
}

func minimalWebhookRequest(url string) WebhookActionRequest {
	return WebhookActionRequest{
		Playbook: &repository.Playbook{ID: uuid.New(), Name: "At-risk outreach", TriggerType: repository.PlaybookTriggerCustomerEvent},
		Action: &repository.PlaybookAction{
			ID:         uuid.New(),
			ActionType: repository.PlaybookActionWebhook,
			ActionConfig: map[string]any{
				"url":            url,
				"signing_secret": "whsec_test",
			},
		},
		TriggerEvent: map[string]any{"type": "score_drop"},
	}
}

func expectedWebhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
