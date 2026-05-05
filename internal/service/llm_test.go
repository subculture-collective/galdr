package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeLLMProvider struct {
	name          string
	completions   []LLMProviderResponse
	errors        []error
	calls         int
	countTokensFn func(string) int
}

func (f *fakeLLMProvider) Complete(ctx context.Context, req LLMProviderRequest) (*LLMProviderResponse, error) {
	f.calls++
	if len(f.errors) >= f.calls && f.errors[f.calls-1] != nil {
		return nil, f.errors[f.calls-1]
	}
	if len(f.completions) >= f.calls {
		res := f.completions[f.calls-1]
		return &res, nil
	}
	return &LLMProviderResponse{Text: "ok", InputTokens: 10, OutputTokens: 5}, nil
}

func TestLLMServiceEnforcesPerOrgRequestLimit(t *testing.T) {
	orgID := uuid.New()
	svc := NewLLMService(&fakeLLMProvider{}, nil, LLMServiceConfig{
		RequestsPerMinute: 1,
		MaxTokensPerDay:   10_000,
	})

	_, err := svc.Complete(context.Background(), LLMCompletionRequest{OrgID: orgID, Prompt: "first"})
	if err != nil {
		t.Fatalf("expected first request to pass, got %v", err)
	}
	_, err = svc.Complete(context.Background(), LLMCompletionRequest{OrgID: orgID, Prompt: "second"})
	if !errors.Is(err, ErrLLMRateLimited) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}

func TestLLMServiceEnforcesDailyTokenBudget(t *testing.T) {
	orgID := uuid.New()
	svc := NewLLMService(&fakeLLMProvider{countTokensFn: func(string) int { return 90 }}, nil, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   100,
	})

	_, err := svc.Complete(context.Background(), LLMCompletionRequest{OrgID: orgID, Prompt: "too much", MaxTokens: 20})
	if !errors.Is(err, ErrLLMTokenBudgetExceeded) {
		t.Fatalf("expected token budget error, got %v", err)
	}
}

func TestLLMServiceDoesNotSpendRequestLimitWhenTokenBudgetFails(t *testing.T) {
	orgID := uuid.New()
	provider := &fakeLLMProvider{countTokensFn: func(text string) int {
		if text == "too much" {
			return 90
		}
		return 1
	}}
	svc := NewLLMService(provider, nil, LLMServiceConfig{
		RequestsPerMinute: 1,
		MaxTokensPerDay:   100,
	})

	_, err := svc.Complete(context.Background(), LLMCompletionRequest{OrgID: orgID, Prompt: "too much", MaxTokens: 20})
	if !errors.Is(err, ErrLLMTokenBudgetExceeded) {
		t.Fatalf("expected token budget error, got %v", err)
	}

	_, err = svc.Complete(context.Background(), LLMCompletionRequest{OrgID: orgID, Prompt: "small", MaxTokens: 1})
	if err != nil {
		t.Fatalf("expected small request to keep request limit after budget failure, got %v", err)
	}
}

func TestLLMServiceRetriesProviderRateLimits(t *testing.T) {
	orgID := uuid.New()
	provider := &fakeLLMProvider{
		errors: []error{
			&LLMProviderError{StatusCode: http.StatusTooManyRequests, Message: "slow down"},
			nil,
		},
		completions: []LLMProviderResponse{
			{},
			{Text: "retried", InputTokens: 10, OutputTokens: 2},
		},
	}
	svc := NewLLMService(provider, nil, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   10_000,
		RetryDelays:       []time.Duration{0},
	})

	res, err := svc.Complete(context.Background(), LLMCompletionRequest{OrgID: orgID, Prompt: "retry"})
	if err != nil {
		t.Fatalf("expected retry success, got %v", err)
	}
	assertRetriedResponse(t, res, provider)
}

func TestLLMServiceRetriesProviderRateLimitsUsingRetryAfter(t *testing.T) {
	orgID := uuid.New()
	provider := &fakeLLMProvider{
		errors: []error{
			&LLMProviderError{StatusCode: http.StatusTooManyRequests, Message: "slow down", RetryAfter: time.Millisecond},
			nil,
		},
		completions: []LLMProviderResponse{
			{},
			{Text: "retried", InputTokens: 10, OutputTokens: 2},
		},
	}
	svc := NewLLMService(provider, nil, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   10_000,
		RetryDelays:       []time.Duration{time.Hour},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	res, err := svc.Complete(ctx, LLMCompletionRequest{OrgID: orgID, Prompt: "retry"})
	if err != nil {
		t.Fatalf("expected retry-after success, got %v", err)
	}
	assertRetriedResponse(t, res, provider)
}

func assertRetriedResponse(t *testing.T, res *LLMCompletionResponse, provider *fakeLLMProvider) {
	t.Helper()
	if res.Text != "retried" || provider.calls != 2 {
		t.Fatalf("expected retried response and two calls, got res=%+v calls=%d", res, provider.calls)
	}
}

func TestLLMServiceFallsBackToAlternateProvider(t *testing.T) {
	orgID := uuid.New()
	primary := &fakeLLMProvider{
		name:   "primary",
		errors: []error{errors.New("provider unavailable")},
	}
	fallback := &fakeLLMProvider{
		name: "fallback",
		completions: []LLMProviderResponse{{
			Text:         "fallback insight",
			InputTokens:  8,
			OutputTokens: 3,
		}},
	}
	svc := NewLLMService(primary, nil, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   10_000,
		FallbackProviders: []LLMProvider{fallback},
	})

	res, err := svc.Complete(context.Background(), LLMCompletionRequest{OrgID: orgID, Prompt: "fallback"})
	if err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}
	if res.Text != "fallback insight" || res.Provider != "fallback" {
		t.Fatalf("expected fallback response/provider, got %+v", res)
	}
	if primary.calls != 1 || fallback.calls != 1 {
		t.Fatalf("expected primary and fallback each called once, got primary=%d fallback=%d", primary.calls, fallback.calls)
	}
}

func TestOpenAIProviderCompleteUsesMockedAPI(t *testing.T) {
	var seenAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuthorization = r.Header.Get("Authorization")
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("expected /chat/completions, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"role": "assistant", "content": "Analyze churn risk."}}],
			"usage": {"prompt_tokens": 12, "completion_tokens": 6, "total_tokens": 18}
		}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIProviderConfig{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})

	res, err := provider.Complete(context.Background(), LLMProviderRequest{Prompt: "summarize", MaxTokens: 50})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if seenAuthorization != "Bearer sk-test" {
		t.Fatalf("expected auth header, got %q", seenAuthorization)
	}
	if res.Text != "Analyze churn risk." || res.InputTokens != 12 || res.OutputTokens != 6 {
		t.Fatalf("unexpected response: %+v", res)
	}
}

func (f *fakeLLMProvider) CountTokens(text string) int {
	if f.countTokensFn != nil {
		return f.countTokensFn(text)
	}
	return len(text)
}

func (f *fakeLLMProvider) Name() string {
	if f.name != "" {
		return f.name
	}
	return "fake"
}

type recordingLLMUsageTracker struct {
	usages []LLMUsage
}

func (r *recordingLLMUsageTracker) TrackLLMUsage(ctx context.Context, usage LLMUsage) error {
	r.usages = append(r.usages, usage)
	return nil
}

func TestLLMServiceRendersTemplateAndTracksCost(t *testing.T) {
	orgID := uuid.New()
	provider := &fakeLLMProvider{
		name: "fake-model",
		completions: []LLMProviderResponse{{
			Text:         "Call Acme before renewal.",
			InputTokens:  1000,
			OutputTokens: 500,
		}},
	}
	tracker := &recordingLLMUsageTracker{}
	svc := NewLLMService(provider, tracker, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   10_000,
		Templates: map[string]string{
			"customer_health": "Summarize {{.CustomerName}} at {{.RiskLevel}} risk.",
		},
	})

	res, err := svc.Complete(context.Background(), LLMCompletionRequest{
		OrgID:        orgID,
		TemplateName: "customer_health",
		TemplateData: map[string]any{"CustomerName": "Acme", "RiskLevel": "red"},
		MaxTokens:    500,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Text != "Call Acme before renewal." {
		t.Fatalf("expected provider text, got %q", res.Text)
	}
	if res.Prompt != "Summarize Acme at red risk." {
		t.Fatalf("expected rendered prompt, got %q", res.Prompt)
	}
	if res.CostUSD <= 0 {
		t.Fatalf("expected cost to be calculated, got %f", res.CostUSD)
	}
	if len(tracker.usages) != 1 {
		t.Fatalf("expected usage to be tracked once, got %d", len(tracker.usages))
	}
	if tracker.usages[0].OrgID != orgID || tracker.usages[0].Provider != "fake-model" {
		t.Fatalf("usage not tracked with org/provider: %+v", tracker.usages[0])
	}
}
