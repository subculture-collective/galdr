package config

import (
	"os"
	"testing"
	"time"
)

func clearEnv() {
	for _, key := range []string{
		"PORT", "HOST", "ENVIRONMENT", "DATABASE_URL",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS",
		"DB_MAX_CONN_LIFETIME", "DB_HEALTH_CHECK_SEC",
		"READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT",
		"CORS_ALLOWED_ORIGINS", "RATE_LIMIT_RPM", "JWT_SECRET",
		"STRIPE_BILLING_SECRET_KEY", "STRIPE_BILLING_PUBLISHABLE_KEY",
		"STRIPE_BILLING_WEBHOOK_SECRET", "STRIPE_BILLING_PORTAL_RETURN_URL",
		"STRIPE_BILLING_PRICE_GROWTH_MONTHLY", "STRIPE_BILLING_PRICE_GROWTH_ANNUAL",
		"STRIPE_BILLING_PRICE_SCALE_MONTHLY", "STRIPE_BILLING_PRICE_SCALE_ANNUAL",
		"OPENAI_API_KEY", "OPENAI_MODEL", "OPENAI_MAX_TOKENS",
		"OPENAI_REQUESTS_PER_MINUTE", "OPENAI_MAX_TOKENS_PER_DAY",
		"BENCHMARK_CONTRIBUTION_INTERVAL_HR",
	} {
		os.Unsetenv(key)
	}
}

func setProductionCoreEnv() {
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("DATABASE_URL", "postgres://prod-db")
	os.Setenv("JWT_SECRET", "prod-super-secret")
}

func setProductionBillingStripeEnv() {
	os.Setenv("STRIPE_BILLING_SECRET_KEY", "sk_live_123")
	os.Setenv("STRIPE_BILLING_PUBLISHABLE_KEY", "pk_live_123")
	os.Setenv("STRIPE_BILLING_WEBHOOK_SECRET", "whsec_live_123")
	os.Setenv("STRIPE_BILLING_PRICE_GROWTH_MONTHLY", "price_growth_monthly")
	os.Setenv("STRIPE_BILLING_PRICE_GROWTH_ANNUAL", "price_growth_annual")
	os.Setenv("STRIPE_BILLING_PRICE_SCALE_MONTHLY", "price_scale_monthly")
	os.Setenv("STRIPE_BILLING_PRICE_SCALE_ANNUAL", "price_scale_annual")
}

func TestLoadOpenAIFromEnv(t *testing.T) {
	clearEnv()
	os.Setenv("OPENAI_API_KEY", "sk-test")
	os.Setenv("OPENAI_MODEL", "gpt-4o-mini")
	os.Setenv("OPENAI_MAX_TOKENS", "750")
	os.Setenv("OPENAI_REQUESTS_PER_MINUTE", "30")
	os.Setenv("OPENAI_MAX_TOKENS_PER_DAY", "25000")
	defer clearEnv()

	cfg := Load()

	if cfg.OpenAI.APIKey != "sk-test" {
		t.Errorf("expected OpenAI API key to load from env")
	}
	if cfg.OpenAI.Model != "gpt-4o-mini" {
		t.Errorf("expected OpenAI model to load from env, got %s", cfg.OpenAI.Model)
	}
	if cfg.OpenAI.MaxTokens != 750 {
		t.Errorf("expected OpenAI max tokens 750, got %d", cfg.OpenAI.MaxTokens)
	}
	if cfg.OpenAI.RequestsPerMinute != 30 {
		t.Errorf("expected OpenAI rpm 30, got %d", cfg.OpenAI.RequestsPerMinute)
	}
	if cfg.OpenAI.MaxTokensPerDay != 25000 {
		t.Errorf("expected OpenAI daily tokens 25000, got %d", cfg.OpenAI.MaxTokensPerDay)
	}
}

func TestLoadBenchmarkContributionIntervalFromEnv(t *testing.T) {
	clearEnv()
	os.Setenv("BENCHMARK_CONTRIBUTION_INTERVAL_HR", "12")
	defer clearEnv()

	cfg := Load()

	if cfg.Benchmark.ContributionIntervalHr != 12 {
		t.Errorf("expected benchmark contribution interval 12h, got %d", cfg.Benchmark.ContributionIntervalHr)
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv()

	cfg := Load()

	if cfg.Server.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if cfg.Server.Environment != "development" {
		t.Errorf("expected default environment development, got %s", cfg.Server.Environment)
	}
	if cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("expected read timeout 5s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 10*time.Second {
		t.Errorf("expected write timeout 10s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("expected max open conns 25, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 5 {
		t.Errorf("expected max idle conns 5, got %d", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.MaxConnLifetime != 3600 {
		t.Errorf("expected max conn lifetime 3600, got %d", cfg.Database.MaxConnLifetime)
	}
	if cfg.Database.HealthCheckSec != 30 {
		t.Errorf("expected health check sec 30, got %d", cfg.Database.HealthCheckSec)
	}
	if cfg.Rate.RequestsPerMinute != 100 {
		t.Errorf("expected rate limit 100, got %d", cfg.Rate.RequestsPerMinute)
	}
	if cfg.Benchmark.ContributionIntervalHr != 24 {
		t.Errorf("expected benchmark contribution interval 24h, got %d", cfg.Benchmark.ContributionIntervalHr)
	}
	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "http://localhost:5173" {
		t.Errorf("expected default CORS origin, got %v", cfg.CORS.AllowedOrigins)
	}
}

func TestLoadFromEnv(t *testing.T) {
	clearEnv()
	os.Setenv("PORT", "3000")
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com, https://app.example.com")
	os.Setenv("RATE_LIMIT_RPM", "200")
	defer clearEnv()

	cfg := Load()

	if cfg.Server.Port != "3000" {
		t.Errorf("expected port 3000, got %s", cfg.Server.Port)
	}
	if cfg.Server.Environment != "production" {
		t.Errorf("expected environment production, got %s", cfg.Server.Environment)
	}
	if len(cfg.CORS.AllowedOrigins) != 2 {
		t.Errorf("expected 2 CORS origins, got %d", len(cfg.CORS.AllowedOrigins))
	}
	if cfg.Rate.RequestsPerMinute != 200 {
		t.Errorf("expected rate limit 200, got %d", cfg.Rate.RequestsPerMinute)
	}
}

func TestValidateProduction(t *testing.T) {
	clearEnv()
	os.Setenv("ENVIRONMENT", "production")
	os.Setenv("DATABASE_URL", "")
	defer clearEnv()

	cfg := Load()
	// In production with empty DATABASE_URL env var, it will use fallback
	// Explicitly clear it to test validation
	cfg.Database.URL = ""

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing DATABASE_URL in production")
	}
}

func TestValidateDevelopment(t *testing.T) {
	clearEnv()
	cfg := Load()

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no validation error in development, got %v", err)
	}
}

func TestLoadBillingStripeFromEnv(t *testing.T) {
	clearEnv()
	os.Setenv("STRIPE_BILLING_SECRET_KEY", "sk_test_123")
	os.Setenv("STRIPE_BILLING_PUBLISHABLE_KEY", "pk_test_123")
	os.Setenv("STRIPE_BILLING_WEBHOOK_SECRET", "whsec_123")
	os.Setenv("STRIPE_BILLING_PORTAL_RETURN_URL", "https://app.example.com/settings/billing")
	os.Setenv("STRIPE_BILLING_PRICE_GROWTH_MONTHLY", "price_growth_monthly")
	os.Setenv("STRIPE_BILLING_PRICE_GROWTH_ANNUAL", "price_growth_annual")
	os.Setenv("STRIPE_BILLING_PRICE_SCALE_MONTHLY", "price_scale_monthly")
	os.Setenv("STRIPE_BILLING_PRICE_SCALE_ANNUAL", "price_scale_annual")
	defer clearEnv()

	cfg := Load()

	if cfg.BillingStripe.SecretKey != "sk_test_123" {
		t.Errorf("expected billing secret key to load from env")
	}
	if cfg.BillingStripe.PublishableKey != "pk_test_123" {
		t.Errorf("expected billing publishable key to load from env")
	}
	if cfg.BillingStripe.WebhookSecret != "whsec_123" {
		t.Errorf("expected billing webhook secret to load from env")
	}
	if cfg.BillingStripe.PriceGrowthMonthly != "price_growth_monthly" {
		t.Errorf("expected growth monthly price id to load from env")
	}
	if cfg.BillingStripe.PriceScaleAnnual != "price_scale_annual" {
		t.Errorf("expected scale annual price id to load from env")
	}
}

func TestValidateProductionRequiresBillingStripeConfig(t *testing.T) {
	clearEnv()
	setProductionCoreEnv()
	defer clearEnv()

	cfg := Load()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when billing stripe config is missing in production")
	}
}

func TestValidateProductionRequiresOpenAIAPIKey(t *testing.T) {
	clearEnv()
	setProductionCoreEnv()
	setProductionBillingStripeEnv()
	defer clearEnv()

	cfg := Load()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when OPENAI_API_KEY is missing in production")
	}
}

func TestValidateProductionWithBillingStripeConfig(t *testing.T) {
	clearEnv()
	setProductionCoreEnv()
	setProductionBillingStripeEnv()
	os.Setenv("OPENAI_API_KEY", "sk_live_123")
	defer clearEnv()

	cfg := Load()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected production validation to pass, got %v", err)
	}
}

func TestIsProd(t *testing.T) {
	clearEnv()
	cfg := Load()
	if cfg.IsProd() {
		t.Error("expected IsProd false in development")
	}

	os.Setenv("ENVIRONMENT", "production")
	defer clearEnv()
	cfg = Load()
	if !cfg.IsProd() {
		t.Error("expected IsProd true in production")
	}
}
