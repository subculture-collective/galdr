package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/service/prompts"
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

func TestLLMServiceEnforcesGlobalRequestLimit(t *testing.T) {
	svc := NewLLMService(&fakeLLMProvider{}, nil, LLMServiceConfig{
		RequestsPerMinute:       10,
		GlobalRequestsPerMinute: 1,
		MaxTokensPerDay:         10_000,
	})

	_, err := svc.Complete(context.Background(), LLMCompletionRequest{OrgID: uuid.New(), Prompt: "first"})
	if err != nil {
		t.Fatalf("expected first request to pass, got %v", err)
	}
	_, err = svc.Complete(context.Background(), LLMCompletionRequest{OrgID: uuid.New(), Prompt: "second"})
	if !errors.Is(err, ErrLLMRateLimited) {
		t.Fatalf("expected global rate limit error, got %v", err)
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

func TestOpenAIProviderRateLimitErrorIncludesRetryMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"Rate limit reached for gpt-4o-mini."}}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIProviderConfig{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})

	_, err := provider.Complete(context.Background(), LLMProviderRequest{Prompt: "summarize", MaxTokens: 50})
	var providerErr *LLMProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected LLMProviderError, got %v", err)
	}
	if providerErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 status, got %d", providerErr.StatusCode)
	}
	if providerErr.Message != "Rate limit reached for gpt-4o-mini." {
		t.Fatalf("expected parsed OpenAI error message, got %q", providerErr.Message)
	}
	if providerErr.RetryAfter != 2*time.Second {
		t.Fatalf("expected retry-after 2s, got %v", providerErr.RetryAfter)
	}
}

func TestOpenAIProviderCountsCompactStructuredPrompts(t *testing.T) {
	provider := NewOpenAIProvider(OpenAIProviderConfig{APIKey: "sk-test"})

	if tokens := provider.CountTokens(`{"signals":[{"name":"failed_payments"}]}`); tokens <= 0 {
		t.Fatalf("expected compact structured prompt to count tokens, got %d", tokens)
	}
}

func TestNewOpenAILLMServiceUsesProviderConfigMaxTokens(t *testing.T) {
	var seenRequest openAIChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&seenRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"role": "assistant", "content": "Configured."}}],
			"usage": {"prompt_tokens": 4, "completion_tokens": 2, "total_tokens": 6}
		}`))
	}))
	defer server.Close()

	svc := NewOpenAILLMService(OpenAIProviderConfig{
		APIKey:    "sk-test",
		Model:     "gpt-test",
		BaseURL:   server.URL,
		MaxTokens: 77,
	}, nil, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   10_000,
	})

	_, err := svc.Complete(context.Background(), LLMCompletionRequest{OrgID: uuid.New(), Prompt: "configured"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if seenRequest.Model != "gpt-test" || seenRequest.MaxTokens != 77 {
		t.Fatalf("expected OpenAI config model/max tokens, got %+v", seenRequest)
	}
}

func TestOpenAIProviderRateLimitErrorUsesRetryAfterMilliseconds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After-Ms", "1500")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"Rate limit reached for gpt-4o-mini."}}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIProviderConfig{
		APIKey:  "sk-test",
		BaseURL: server.URL,
	})

	_, err := provider.Complete(context.Background(), LLMProviderRequest{Prompt: "summarize", MaxTokens: 50})
	var providerErr *LLMProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected LLMProviderError, got %v", err)
	}
	if providerErr.RetryAfter != 1500*time.Millisecond {
		t.Fatalf("expected retry-after-ms 1500ms, got %v", providerErr.RetryAfter)
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

type fakeLLMUsageGate struct {
	checkCalls int
	records    []LLMUsage
	checkErr   error
	warnings   []LLMBudgetWarning
}

func (f *fakeLLMUsageGate) CheckLLMUsage(ctx context.Context, orgID uuid.UUID, estimatedCostUSD float64, manualRegeneration bool) error {
	f.checkCalls++
	return f.checkErr
}

func (f *fakeLLMUsageGate) RecordLLMUsage(ctx context.Context, usage LLMUsage) (*LLMUsageSummary, error) {
	f.records = append(f.records, usage)
	return &LLMUsageSummary{BudgetWarning: true, MonthlyCostUSD: 4.25, BudgetUSD: 5}, nil
}

func (f *fakeLLMUsageGate) NotifyLLMBudgetWarning(ctx context.Context, warning LLMBudgetWarning) error {
	f.warnings = append(f.warnings, warning)
	return nil
}

func TestLLMServiceBlocksWhenTierBudgetExceeded(t *testing.T) {
	orgID := uuid.New()
	gate := &fakeLLMUsageGate{checkErr: ErrLLMBudgetExceeded}
	provider := &fakeLLMProvider{}
	svc := NewLLMService(provider, nil, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   10_000,
		UsageGate:         gate,
	})

	_, err := svc.Complete(context.Background(), LLMCompletionRequest{OrgID: orgID, Prompt: "budget"})
	if !errors.Is(err, ErrLLMBudgetExceeded) {
		t.Fatalf("expected budget error, got %v", err)
	}
	if provider.calls != 0 {
		t.Fatalf("expected provider not to be called after budget block, got %d", provider.calls)
	}
}

func TestLLMServiceRecordsUsageAndWarnsAtBudgetThreshold(t *testing.T) {
	orgID := uuid.New()
	gate := &fakeLLMUsageGate{}
	provider := &fakeLLMProvider{name: "gpt-4o-mini", completions: []LLMProviderResponse{{Text: "ok", InputTokens: 1000, OutputTokens: 500}}}
	svc := NewLLMService(provider, nil, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   10_000,
		UsageGate:         gate,
	})

	_, err := svc.Complete(context.Background(), LLMCompletionRequest{OrgID: orgID, Prompt: "warn", RequestType: "customer_summary"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(gate.records) != 1 {
		t.Fatalf("expected usage record, got %d", len(gate.records))
	}
	if gate.records[0].RequestType != "customer_summary" || gate.records[0].Model != "gpt-4o-mini" {
		t.Fatalf("expected request type/model on usage, got %+v", gate.records[0])
	}
	if len(gate.warnings) != 1 || gate.warnings[0].OrgID != orgID {
		t.Fatalf("expected one budget warning, got %+v", gate.warnings)
	}
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

func TestLLMServiceRendersBundledPromptTemplate(t *testing.T) {
	orgID := uuid.New()
	templates, err := prompts.Templates()
	if err != nil {
		t.Fatalf("expected bundled templates to load, got %v", err)
	}
	svc := NewLLMService(newBundledTemplateProvider(), nil, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   10_000,
		Templates:         templates,
	})

	res, err := svc.Complete(context.Background(), bundledTemplateRequest(orgID))
	if err != nil {
		t.Fatalf("expected bundled prompt to render, got %v", err)
	}
	assertRenderedBundledPrompt(t, res.Prompt)
}

func TestLLMServiceLoadsBundledTemplatesByDefault(t *testing.T) {
	orgID := uuid.New()
	svc := NewLLMService(newBundledTemplateProvider(), nil, LLMServiceConfig{
		RequestsPerMinute: 10,
		MaxTokensPerDay:   10_000,
	})

	res, err := svc.Complete(context.Background(), bundledTemplateRequest(orgID))
	if err != nil {
		t.Fatalf("expected default bundled prompt to render, got %v", err)
	}
	assertRenderedBundledPrompt(t, res.Prompt)
}

func newBundledTemplateProvider() *fakeLLMProvider {
	return &fakeLLMProvider{
		completions: []LLMProviderResponse{{
			Text:         `{"insight_type":"summary"}`,
			InputTokens:  120,
			OutputTokens: 30,
		}},
	}
}

func bundledTemplateRequest(orgID uuid.UUID) LLMCompletionRequest {
	return LLMCompletionRequest{
		OrgID:        orgID,
		TemplateName: string(prompts.SummaryTemplate),
		TemplateData: prompts.CustomerAnalysisData{
			Customer: prompts.CustomerSnapshot{
				Name:     "Acme",
				MRRCents: 125000,
				Currency: "usd",
			},
			HealthScore: prompts.HealthScoreSnapshot{
				OverallScore:  38,
				RiskLevel:     "red",
				ScoreChange7d: -12,
				Factors: []prompts.ScoreFactor{{
					Name:        "failed_payments",
					Score:       0.2,
					Explanation: "two recent failures",
				}},
			},
		},
		MaxTokens: 100,
	}
}

func assertRenderedBundledPrompt(t *testing.T, prompt string) {
	t.Helper()
	if !strings.Contains(prompt, "Customer: Acme") || !strings.Contains(prompt, "USD 1,250.00") {
		t.Fatalf("expected rendered bundled prompt with customer data, got %q", prompt)
	}
}
