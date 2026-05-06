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
	if nestedString(t, seenRequest, "trigger_event", "type") != "score_drop" {
		t.Fatalf("expected trigger payload, got %+v", seenRequest)
	}
	if nestedString(t, seenRequest, "playbook", "id") != playbookID.String() {
		t.Fatalf("expected playbook summary, got %+v", seenRequest["playbook"])
	}
	if nestedString(t, seenRequest, "customer", "email") != "billing@acme.test" {
		t.Fatalf("expected customer summary, got %+v", seenRequest["customer"])
	}
}

func TestWebhookActionRetriesTransientFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < defaultWebhookAttempts {
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
	if attempts != defaultWebhookAttempts || result.Attempts != defaultWebhookAttempts || result.StatusCode != http.StatusNoContent {
		t.Fatalf("expected third attempt success, attempts=%d result=%+v", attempts, result)
	}
}

func TestWebhookActionRetriesRequestErrors(t *testing.T) {
	attempts := 0
	requestErr := errors.New("dial webhook: connection refused")
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, requestErr
		}),
	}

	executor := NewWebhookActionExecutor(WebhookActionExecutorConfig{
		HTTPClient:  client,
		RetryDelays: []time.Duration{0, 0},
	})
	result, err := executor.Execute(context.Background(), minimalWebhookRequest("https://example.test/hook"))
	if err == nil {
		t.Fatal("expected request failure")
	}
	if !errors.Is(err, requestErr) {
		t.Fatalf("expected original request error, got %v", err)
	}
	if attempts != defaultWebhookAttempts || result.Attempts != defaultWebhookAttempts || result.Error != requestErr.Error() {
		t.Fatalf("expected recorded third-attempt request failure, attempts=%d result=%+v", attempts, result)
	}
}

func TestWebhookActionRejectsNonHTTPSURL(t *testing.T) {
	executor := NewWebhookActionExecutor(WebhookActionExecutorConfig{})
	_, err := executor.Execute(context.Background(), minimalWebhookRequest("http://example.test/hook"))
	if err == nil || !errors.Is(err, ErrWebhookURLMustBeHTTPS) {
		t.Fatalf("expected HTTPS validation error, got %v", err)
	}
}

func TestWebhookActionRejectsNonWebhookAction(t *testing.T) {
	req := minimalWebhookRequest("https://example.test/hook")
	req.Action.ActionType = repository.PlaybookActionSendEmail

	executor := NewWebhookActionExecutor(WebhookActionExecutorConfig{})
	_, err := executor.Execute(context.Background(), req)
	if err == nil {
		t.Fatal("expected non-webhook action error")
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

func nestedString(t *testing.T, values map[string]any, objectKey, valueKey string) string {
	t.Helper()
	nested, ok := values[objectKey].(map[string]any)
	if !ok {
		t.Fatalf("expected %q to be an object, got %+v", objectKey, values[objectKey])
	}
	value, ok := nested[valueKey].(string)
	if !ok {
		t.Fatalf("expected %q.%q to be a string, got %+v", objectKey, valueKey, nested[valueKey])
	}
	return value
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
