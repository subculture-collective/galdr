package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	billingcatalog "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/config"
	"github.com/onnwee/pulse-score/internal/database"
	"github.com/onnwee/pulse-score/internal/handler"
	"github.com/onnwee/pulse-score/internal/middleware"
	"github.com/onnwee/pulse-score/internal/ml"
	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
	billingsvc "github.com/onnwee/pulse-score/internal/service/billing"
	"github.com/onnwee/pulse-score/internal/service/scoring"
)

const (
	corsMaxAgeSeconds                = 300
	connectionMonitorIntervalSeconds = 60
)

func main() {
	initLogger()

	cfg := loadAndValidateConfig()
	pool := openDatabase(cfg)
	if pool != nil {
		defer pool.P.Close()
	}

	jwtMgr := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	r := newRouter(cfg, pool, jwtMgr)
	srv := newHTTPServer(cfg, r)
	runServerWithGracefulShutdown(srv, cfg.Server.Environment)
}

func initLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
}

func loadAndValidateConfig() *config.Config {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	return cfg
}

func openDatabase(cfg *config.Config) *database.Pool {
	if cfg.Database.URL == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := database.NewPool(ctx, database.PoolConfig{
		URL:               cfg.Database.URL,
		MaxConns:          int32(cfg.Database.MaxOpenConns),
		MinConns:          int32(cfg.Database.MaxIdleConns),
		MaxConnLifetime:   time.Duration(cfg.Database.MaxConnLifetime) * time.Second,
		HealthCheckPeriod: time.Duration(cfg.Database.HealthCheckSec) * time.Second,
	})
	if err != nil {
		slog.Warn("database not reachable at startup", "error", err)
		return nil
	}

	return &database.Pool{P: dbPool}
}

func newRouter(cfg *config.Config, pool *database.Pool, jwtMgr *auth.JWTManager) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Internal-Analytics-Token", "X-Request-ID", "X-Organization-ID", "X-PulseScore-Signature"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           corsMaxAgeSeconds,
	}))
	r.Use(httprate.LimitByIP(cfg.Rate.RequestsPerMinute, time.Minute))

	health := handler.NewHealthHandler(pool)
	r.Get("/healthz", health.Liveness)
	r.Get("/readyz", health.Readiness)

	registerAPIRoutes(r, cfg, pool, jwtMgr)
	return r
}

func registerAPIRoutes(r *chi.Mux, cfg *config.Config, pool *database.Pool, jwtMgr *auth.JWTManager) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"pong"}`))
		})

		// Auth routes (public)
		if pool != nil {
			userRepo := repository.NewUserRepository(pool.P)
			orgRepo := repository.NewOrganizationRepository(pool.P)
			orgSubRepo := repository.NewOrgSubscriptionRepository(pool.P)
			refreshTokenRepo := repository.NewRefreshTokenRepository(pool.P)
			invitationRepo := repository.NewInvitationRepository(pool.P)
			passwordResetRepo := repository.NewPasswordResetRepository(pool.P)
			billingWebhookEventRepo := repository.NewBillingWebhookEventRepository(pool.P)
			featureOverrideRepo := repository.NewFeatureOverrideRepository(pool.P)
			usageSnapshotRepo := repository.NewUsageSnapshotRepository(pool.P)

			emailSvc := service.NewSendGridEmailService(service.SendGridConfig{
				APIKey:      cfg.SendGrid.APIKey,
				FromEmail:   cfg.SendGrid.FromEmail,
				FrontendURL: cfg.SendGrid.FrontendURL,
				DevMode:     !cfg.IsProd(),
			})

			authSvc := service.NewAuthService(pool.P, userRepo, orgRepo, refreshTokenRepo, passwordResetRepo, jwtMgr, cfg.JWT.RefreshTTL, emailSvc)
			authHandler := handler.NewAuthHandler(authSvc)

			invitationSvc := service.NewInvitationService(pool.P, invitationRepo, orgRepo, userRepo, emailSvc, jwtMgr)
			invitationHandler := handler.NewInvitationHandler(invitationSvc)

			// Stripe integration repositories
			connRepo := repository.NewIntegrationConnectionRepository(pool.P)
			marketplaceRepo := repository.NewMarketplaceRepository(pool.P)
			customerRepo := repository.NewCustomerRepository(pool.P)
			customerNoteRepo := repository.NewCustomerNoteRepository(pool.P)
			subRepo := repository.NewStripeSubscriptionRepository(pool.P)
			paymentRepo := repository.NewStripePaymentRepository(pool.P)
			eventRepo := repository.NewCustomerEventRepository(pool.P)
			playbookRepo := repository.NewPlaybookRepository(pool.P)
			savedViewRepo := repository.NewSavedViewRepository(pool.P)
			genericWebhookConfigRepo := repository.NewGenericWebhookConfigRepository(pool.P)

			// HubSpot/Intercom/Zendesk repositories
			hubspotContactRepo := repository.NewHubSpotContactRepository(pool.P)
			hubspotDealRepo := repository.NewHubSpotDealRepository(pool.P)
			hubspotCompanyRepo := repository.NewHubSpotCompanyRepository(pool.P)
			intercomContactRepo := repository.NewIntercomContactRepository(pool.P)
			intercomConversationRepo := repository.NewIntercomConversationRepository(pool.P)
			zendeskUserRepo := repository.NewZendeskUserRepository(pool.P)
			zendeskTicketRepo := repository.NewZendeskTicketRepository(pool.P)

			// Onboarding repositories
			onboardingStatusRepo := repository.NewOnboardingStatusRepository(pool.P)
			onboardingEventRepo := repository.NewOnboardingEventRepository(pool.P)

			// Stripe services
			planCatalog := billingcatalog.NewCatalog(billingcatalog.PriceConfig{
				GrowthMonthly: cfg.BillingStripe.PriceGrowthMonthly,
				GrowthAnnual:  cfg.BillingStripe.PriceGrowthAnnual,
				ScaleMonthly:  cfg.BillingStripe.PriceScaleMonthly,
				ScaleAnnual:   cfg.BillingStripe.PriceScaleAnnual,
			})

			if cfg.IsProd() {
				if err := billingcatalog.VerifyConfiguredPrices(context.Background(), cfg.BillingStripe.SecretKey, planCatalog); err != nil {
					slog.Error("invalid Stripe billing price configuration", "error", err)
					os.Exit(1)
				}
			}

			stripeOAuthSvc := service.NewStripeOAuthService(service.StripeOAuthConfig{
				ClientID:         cfg.Stripe.ClientID,
				SecretKey:        cfg.Stripe.SecretKey,
				OAuthRedirectURL: cfg.Stripe.OAuthRedirectURL,
				EncryptionKey:    cfg.Stripe.EncryptionKey,
			}, connRepo)

			hubspotOAuthSvc := service.NewHubSpotOAuthService(service.HubSpotOAuthConfig{
				ClientID:         cfg.HubSpot.ClientID,
				ClientSecret:     cfg.HubSpot.ClientSecret,
				OAuthRedirectURL: cfg.HubSpot.OAuthRedirectURL,
				EncryptionKey:    cfg.HubSpot.EncryptionKey,
			}, connRepo)

			intercomOAuthSvc := service.NewIntercomOAuthService(service.IntercomOAuthConfig{
				ClientID:         cfg.Intercom.ClientID,
				ClientSecret:     cfg.Intercom.ClientSecret,
				OAuthRedirectURL: cfg.Intercom.OAuthRedirectURL,
				EncryptionKey:    cfg.Intercom.EncryptionKey,
			}, connRepo)

			zendeskOAuthSvc := service.NewZendeskOAuthService(service.ZendeskOAuthConfig{
				ClientID:         cfg.Zendesk.ClientID,
				ClientSecret:     cfg.Zendesk.ClientSecret,
				OAuthRedirectURL: cfg.Zendesk.OAuthRedirectURL,
				EncryptionKey:    cfg.Zendesk.EncryptionKey,
			}, connRepo)

			salesforceOAuthSvc := service.NewSalesforceOAuthService(service.SalesforceOAuthConfig{
				ClientID:         cfg.Salesforce.ClientID,
				ClientSecret:     cfg.Salesforce.ClientSecret,
				OAuthRedirectURL: cfg.Salesforce.OAuthRedirectURL,
				EncryptionKey:    cfg.Salesforce.EncryptionKey,
				LoginURL:         cfg.Salesforce.LoginURL,
			}, connRepo)

			hubspotClient := service.NewHubSpotClient()
			intercomClient := service.NewIntercomClient()
			zendeskClient := service.NewZendeskClient()
			salesforceClient := service.NewSalesforceClient()
			posthogClient := service.NewPostHogClient("", nil)

			stripeSyncSvc := service.NewStripeSyncService(
				customerRepo, subRepo, paymentRepo, eventRepo,
				stripeOAuthSvc, cfg.Stripe.PaymentSyncDays,
			)

			mergeSvc := service.NewCustomerMergeService(customerRepo, hubspotContactRepo)

			hubspotSyncSvc := service.NewHubSpotSyncService(
				hubspotOAuthSvc,
				hubspotClient,
				hubspotContactRepo,
				hubspotDealRepo,
				hubspotCompanyRepo,
				customerRepo,
				eventRepo,
			)

			intercomSyncSvc := service.NewIntercomSyncService(
				intercomOAuthSvc,
				intercomClient,
				intercomContactRepo,
				intercomConversationRepo,
				customerRepo,
				eventRepo,
			)

			zendeskSyncSvc := service.NewZendeskSyncService(
				zendeskOAuthSvc,
				zendeskClient,
				zendeskUserRepo,
				zendeskTicketRepo,
				customerRepo,
				eventRepo,
			)

			salesforceSyncSvc := service.NewSalesforceSyncService(
				salesforceOAuthSvc,
				salesforceClient,
				customerRepo,
				eventRepo,
			)

			posthogSvc := service.NewPostHogService(
				service.PostHogConfig{EncryptionKey: cfg.PostHog.EncryptionKey},
				connRepo,
				posthogClient,
				customerRepo,
				eventRepo,
			)

			mrrSvc := service.NewMRRService(customerRepo, subRepo, eventRepo)
			paymentHealthSvc := service.NewPaymentHealthService(paymentRepo, eventRepo, customerRepo)
			paymentRecencySvc := service.NewPaymentRecencyService(paymentRepo, subRepo)

			syncOrchestrator := service.NewSyncOrchestratorService(connRepo, stripeSyncSvc, mrrSvc)
			hubspotSyncOrchestrator := service.NewHubSpotSyncOrchestratorService(connRepo, hubspotSyncSvc, mergeSvc)
			intercomSyncOrchestrator := service.NewIntercomSyncOrchestratorService(connRepo, intercomSyncSvc, mergeSvc)
			zendeskSyncOrchestrator := service.NewZendeskSyncOrchestratorService(connRepo, zendeskSyncSvc, mergeSvc)
			salesforceSyncOrchestrator := service.NewSalesforceSyncOrchestratorService(connRepo, salesforceSyncSvc)
			connectorRegistry, err := service.NewIntegrationConnectorRegistry(
				stripeOAuthSvc,
				syncOrchestrator,
				hubspotOAuthSvc,
				hubspotSyncOrchestrator,
				intercomOAuthSvc,
				intercomSyncOrchestrator,
				zendeskOAuthSvc,
				zendeskSyncOrchestrator,
				salesforceOAuthSvc,
				salesforceSyncOrchestrator,
				posthogSvc,
			)
			if err != nil {
				slog.Error("failed to create connector registry", "error", err)
				os.Exit(1)
			}
			connectorSyncSvc := service.NewConnectorSyncService(connectorRegistry)

			stripeWebhookSvc := service.NewStripeWebhookService(
				cfg.Stripe.WebhookSecret,
				connRepo, customerRepo, subRepo, paymentRepo, eventRepo,
				mrrSvc, paymentHealthSvc,
			)

			billingSubscriptionSvc := billingsvc.NewSubscriptionService(
				orgSubRepo,
				orgRepo,
				customerRepo,
				connRepo,
				planCatalog,
			)
			billingSubscriptionSvc.SetFeatureOverrides(featureOverrideRepo)
			llmUsageStore := service.NewPostgresLLMUsageStore(pool.P)
			llmBudgetNotifier := service.NewPostgresLLMBudgetNotifier(pool.P)
			llmUsageSvc := service.NewLLMUsageService(llmUsageStore, billingSubscriptionSvc, llmBudgetNotifier)

			usageAnalyticsSvc := billingsvc.NewUsageService(billingsvc.UsageServiceDeps{
				Subscriptions: orgSubRepo,
				Organizations: orgRepo,
				Customers:     customerRepo,
				Integrations:  connRepo,
				Playbooks:     playbookRepo,
				Snapshots:     usageSnapshotRepo,
				Catalog:       planCatalog,
				Overrides:     featureOverrideRepo,
			})

			billingLimitsSvc := billingsvc.NewLimitsService(
				billingSubscriptionSvc,
				customerRepo,
				connRepo,
				connRepo,
				featureOverrideRepo,
				planCatalog,
			)

			billingCheckoutSvc := billingsvc.NewCheckoutService(
				cfg.BillingStripe.SecretKey,
				cfg.SendGrid.FrontendURL,
				orgRepo,
				planCatalog,
			)

			billingPortalSvc := billingsvc.NewPortalService(
				cfg.BillingStripe.SecretKey,
				cfg.BillingStripe.PortalReturnURL,
				cfg.SendGrid.FrontendURL,
				orgRepo,
				orgSubRepo,
			)

			billingPlanChangeSvc := billingsvc.NewChangePlanService(
				cfg.BillingStripe.SecretKey,
				billingCheckoutSvc,
				orgSubRepo,
				planCatalog,
			)

			billingWebhookSvc := billingsvc.NewWebhookService(
				cfg.BillingStripe.WebhookSecret,
				pool.P,
				orgRepo,
				orgSubRepo,
				billingWebhookEventRepo,
				planCatalog,
			)

			hubspotWebhookSvc := service.NewHubSpotWebhookService(
				cfg.HubSpot.WebhookSecret,
				hubspotSyncSvc,
				mergeSvc,
				connRepo,
				hubspotContactRepo,
				hubspotDealRepo,
				hubspotCompanyRepo,
				eventRepo,
			)

			intercomWebhookSvc := service.NewIntercomWebhookService(
				cfg.Intercom.WebhookSecret,
				intercomSyncSvc,
				mergeSvc,
				connRepo,
				intercomContactRepo,
				intercomConversationRepo,
				eventRepo,
			)

			genericWebhookSvc := service.NewGenericWebhookService(orgRepo, genericWebhookConfigRepo, customerRepo, eventRepo)
			genericWebhookHandler := handler.NewGenericWebhookHandler(genericWebhookSvc)

			onboardingSvc := service.NewOnboardingService(onboardingStatusRepo, onboardingEventRepo)

			// Health scoring engine
			scoringConfigRepo := repository.NewScoringConfigRepository(pool.P)
			healthScoreRepo := repository.NewHealthScoreRepository(pool.P)
			benchmarkRepo := repository.NewBenchmarkRepository(pool.P)
			benchmarkMetricsRepo := repository.NewBenchmarkMetricsRepository(customerRepo, healthScoreRepo, connRepo)
			customerFeatureRepo := repository.NewCustomerFeatureRepository(pool.P)
			customerInsightRepo := repository.NewCustomerInsightRepository(pool.P)
			churnPredictionRepo := repository.NewChurnPredictionRepository(pool.P)

			paymentRecencyFactor := scoring.NewPaymentRecencyFactor(paymentRecencySvc)
			mrrTrendFactor := scoring.NewMRRTrendFactor(customerRepo, eventRepo)
			failedPaymentsFactor := scoring.NewFailedPaymentsFactor(paymentHealthSvc, paymentRepo)
			supportTicketsFactor := scoring.NewSupportTicketsFactor(eventRepo)
			engagementFactor := scoring.NewEngagementFactor(eventRepo)

			scoreAggregator := scoring.NewScoreAggregator(
				[]scoring.ScoreFactor{
					paymentRecencyFactor,
					mrrTrendFactor,
					failedPaymentsFactor,
					supportTicketsFactor,
					engagementFactor,
				},
				scoringConfigRepo,
			)

			changeDetector := scoring.NewChangeDetector(eventRepo, cfg.Scoring.ChangeDelta)
			riskCategorizer := scoring.NewRiskCategorizer(healthScoreRepo)

			scoreScheduler := scoring.NewScoreScheduler(
				scoreAggregator, healthScoreRepo, customerRepo, connRepo, changeDetector,
				time.Duration(cfg.Scoring.RecalcIntervalMin)*time.Minute,
				cfg.Scoring.Workers,
			)

			scoringConfigSvc := scoring.NewConfigService(scoringConfigRepo, scoreScheduler)

			churnPredictionSvc := service.NewChurnPredictionService(service.ChurnPredictionDeps{
				Customers:   customerRepo,
				Features:    customerFeatureRepo,
				Store:       churnPredictionRepo,
				Connections: connRepo,
			})

			featurePipeline := ml.NewFeaturePipeline(ml.FeaturePipelineDeps{
				Customers:    customerRepo,
				HealthScores: healthScoreRepo,
				Payments:     paymentRepo,
				Events:       eventRepo,
				Store:        customerFeatureRepo,
				Connections:  connRepo,
				AfterBatch: func(ctx context.Context) error {
					return churnPredictionSvc.RunBatch(ctx)
				},
			})

			llmSvc := service.NewOpenAILLMService(
				service.OpenAIProviderConfig{APIKey: cfg.OpenAI.APIKey, Model: cfg.OpenAI.Model, MaxTokens: cfg.OpenAI.MaxTokens},
				nil,
				service.LLMServiceConfig{
					RequestsPerMinute: cfg.OpenAI.RequestsPerMinute,
					MaxTokensPerDay:   cfg.OpenAI.MaxTokensPerDay,
					DefaultMaxTokens:  cfg.OpenAI.MaxTokens,
				},
			)
			insightPipeline := service.NewInsightPipeline(service.InsightPipelineDeps{
				Customers:    customerRepo,
				HealthScores: healthScoreRepo,
				Events:       eventRepo,
				Insights:     customerInsightRepo,
				LLM:          llmSvc,
			})

			// Alert engine + scheduler
			alertRuleRepo := repository.NewAlertRuleRepository(pool.P)
			alertHistoryRepo := repository.NewAlertHistoryRepository(pool.P)
			emailTemplateSvc, err := service.NewEmailTemplateService()
			if err != nil {
				slog.Error("failed to initialize email templates", "error", err)
				os.Exit(1)
			}

			alertEngine := service.NewAlertEngine(
				alertRuleRepo, alertHistoryRepo, healthScoreRepo,
				customerRepo, eventRepo, cfg.Alert.DefaultCooldownHr,
			)

			notifPrefRepo := repository.NewNotificationPreferenceRepository(pool.P)
			notifPrefSvc := service.NewNotificationPreferenceService(notifPrefRepo)

			alertScheduler := service.NewAlertScheduler(
				service.AlertSchedulerDeps{
					Engine:       alertEngine,
					EmailService: emailSvc,
					Templates:    emailTemplateSvc,
					AlertHistory: alertHistoryRepo,
					AlertRules:   alertRuleRepo,
					UserRepo:     userRepo,
					NotifPrefSvc: notifPrefSvc,
				},
				cfg.Alert.EvalIntervalMin,
				cfg.SendGrid.FrontendURL,
			)

			// Wire in-app notifications into the alert scheduler
			notifRepo := repository.NewNotificationRepository(pool.P)
			notifSvc := service.NewNotificationService(notifRepo, userRepo, notifPrefSvc)
			alertScheduler.SetNotificationService(notifSvc)

			// Hook real-time alert evaluation into score recalculation
			scoreScheduler.SetAlertCallback(func(ctx context.Context, customerID, orgID uuid.UUID) {
				matches, err := alertEngine.EvaluateForCustomer(ctx, customerID, orgID)
				if err != nil {
					slog.Error("real-time alert eval error", "customer_id", customerID, "error", err)
					return
				}
				for _, match := range matches {
					alertScheduler.ProcessMatch(ctx, match)
				}
			})

			// Generate per-customer AI insights for meaningful score changes.
			scoreScheduler.SetInsightCallback(func(ctx context.Context, previous, current *repository.HealthScore) {
				if _, triggered, err := insightPipeline.GenerateForScoreChange(ctx, previous, current); err != nil {
					slog.Error("score-triggered insight generation error", "customer_id", current.CustomerID, "error", err)
				} else if triggered {
					slog.Info("score-triggered insight generated", "customer_id", current.CustomerID, "risk_level", current.RiskLevel)
				}
			})

			// Start background services
			bgCtx, bgCancel := context.WithCancel(context.Background())
			defer bgCancel()

			if cfg.Stripe.SyncIntervalMin > 0 {
				syncScheduler := service.NewSyncSchedulerService(
					connRepo,
					connectorSyncSvc,
					connectorRegistry,
					cfg.Stripe.SyncIntervalMin,
				)
				go syncScheduler.Start(bgCtx)
			}

			if cfg.Scoring.RecalcIntervalMin > 0 {
				go scoreScheduler.Start(bgCtx)
			}

			go featurePipeline.Start(bgCtx)

			connMonitor := service.NewConnectionMonitorService(
				connRepo,
				stripeOAuthSvc,
				hubspotOAuthSvc,
				hubspotClient,
				intercomOAuthSvc,
				intercomClient,
				connectionMonitorIntervalSeconds,
			)
			go connMonitor.Start(bgCtx)

			if cfg.Alert.EvalIntervalMin > 0 {
				go alertScheduler.Start(bgCtx)
			}

			if cfg.Benchmark.ContributionIntervalHr > 0 {
				benchmarkPipeline := service.NewBenchmarkPipeline(orgRepo, benchmarkMetricsRepo, benchmarkRepo, service.NewBenchmarkAnonymizer())
				benchmarkAggregation := service.NewBenchmarkAggregationService(benchmarkRepo, benchmarkRepo)
				benchmarkNotifications := service.NewBenchmarkInsightNotificationService(service.BenchmarkInsightNotificationDeps{
					Contributions: benchmarkRepo,
					Aggregates:     benchmarkRepo,
					Members:        orgRepo,
					Notifications:  notifRepo,
					Preferences:    notifPrefSvc,
					Emails:         emailSvc,
				})
				benchmarkWorkflow := service.NewBenchmarkWorkflow(benchmarkPipeline, benchmarkAggregation, benchmarkNotifications)
				benchmarkScheduler := service.NewBenchmarkScheduler(benchmarkWorkflow, time.Duration(cfg.Benchmark.ContributionIntervalHr)*time.Hour)
				go benchmarkScheduler.Start(bgCtx)

				benchmarkDigestScheduler := service.NewBenchmarkScheduler(
					service.BenchmarkRunnerFunc(benchmarkNotifications.SendWeeklyDigestFromLatest),
					7*24*time.Hour,
				)
				go benchmarkDigestScheduler.Start(bgCtx)
			}

			usageScheduler := billingsvc.NewUsageScheduler(orgRepo, usageAnalyticsSvc, 24*time.Hour)
			go usageScheduler.Start(bgCtx)

			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.Refresh)
			r.Post("/auth/password-reset/request", authHandler.RequestPasswordReset)
			r.Post("/auth/password-reset/complete", authHandler.CompletePasswordReset)

			// Invitation acceptance (public — no auth required)
			r.Post("/invitations/accept", invitationHandler.Accept)

			// Stripe webhook (public — verified by signature)
			webhookHandler := handler.NewWebhookStripeHandler(stripeWebhookSvc)
			r.Post("/webhooks/stripe", webhookHandler.HandleWebhook)

			// Stripe billing webhook (public — verified by signature)
			billingWebhookHandler := handler.NewWebhookStripeBillingHandler(billingWebhookSvc)
			r.Post("/webhooks/stripe-billing", billingWebhookHandler.HandleWebhook)

			// SendGrid webhook (public — for delivery tracking)
			sendgridWebhookHandler := handler.NewWebhookSendGridHandler(alertHistoryRepo)
			r.Post("/webhooks/sendgrid", sendgridWebhookHandler.HandleWebhook)

			// HubSpot webhook (public — verified by signature)
			hubspotWebhookHandler := handler.NewWebhookHubSpotHandler(hubspotWebhookSvc)
			r.Post("/webhooks/hubspot", hubspotWebhookHandler.HandleWebhook)

			// Intercom webhook (public — verified by signature)
			intercomWebhookHandler := handler.NewWebhookIntercomHandler(intercomWebhookSvc)
			r.Post("/webhooks/intercom", intercomWebhookHandler.HandleWebhook)

			// Generic webhook receiver (public — optionally verified by config secret)
			r.Post("/webhooks/generic/{org_slug}/{webhook_id}", genericWebhookHandler.Process)

			// Protected routes (JWT required)
			r.Group(func(r chi.Router) {
				r.Use(middleware.JWTAuth(jwtMgr))
				r.Use(middleware.TenantIsolation(orgRepo))
				r.Use(middleware.TrackAPIUsage(usageAnalyticsSvc))

				// Organization routes
				orgSvc := service.NewOrganizationService(pool.P, orgRepo, benchmarkRepo)
				orgHandler := handler.NewOrganizationHandler(orgSvc)
				r.Post("/organizations", orgHandler.Create)
				r.Get("/organizations/current", orgHandler.GetCurrent)
				r.Patch("/organizations/current", orgHandler.UpdateCurrent)

				billingHandler := handler.NewBillingHandler(
					billingCheckoutSvc,
					billingPortalSvc,
					billingSubscriptionSvc,
					usageAnalyticsSvc,
					billingPlanChangeSvc,
					llmUsageSvc,
				)
				r.Route("/billing", func(r chi.Router) {
					r.Get("/subscription", billingHandler.GetSubscription)
					r.Get("/usage", billingHandler.GetUsage)
					r.Get("/ai-usage", billingHandler.GetAIUsage)
					r.Group(func(r chi.Router) {
						r.Use(middleware.RequireInternalAnalyticsToken(cfg.Internal.AnalyticsToken))
						r.Use(middleware.RequireRole("owner"))
						r.Get("/usage/analytics", billingHandler.GetUsageAnalytics)
					})
					r.Group(func(r chi.Router) {
						r.Use(middleware.RequireRole("admin"))
						r.Post("/checkout", billingHandler.CreateCheckout)
						r.Post("/change-plan", billingHandler.ChangePlan)
						r.Post("/portal-session", billingHandler.CreatePortalSession)
						r.Post("/cancel", billingHandler.CancelSubscription)
					})
				})

				// User profile routes
				userSvc := service.NewUserService(userRepo, orgRepo)
				userHandler := handler.NewUserHandler(userSvc)
				r.Get("/users/me", userHandler.GetProfile)
				r.Patch("/users/me", userHandler.UpdateProfile)

				// Customer routes
				customerAssignmentRepo := repository.NewCustomerAssignmentRepository(pool.P)
				customerSvc := service.NewCustomerService(customerRepo, healthScoreRepo, subRepo, eventRepo, customerNoteRepo, customerAssignmentRepo, churnPredictionRepo)
				customerHandler := handler.NewCustomerHandlerWithInsights(customerSvc, insightPipeline)
				savedViewSvc := service.NewSavedViewService(savedViewRepo)
				savedViewHandler := handler.NewSavedViewHandler(savedViewSvc)
				r.Get("/customers", customerHandler.List)
				r.Get("/customers/saved-views", savedViewHandler.List)
				r.Post("/customers/saved-views", savedViewHandler.Create)
				r.Get("/customers/saved-views/{id}", savedViewHandler.Get)
				r.Patch("/customers/saved-views/{id}", savedViewHandler.Update)
				r.Delete("/customers/saved-views/{id}", savedViewHandler.Delete)
				r.Get("/customers/{id}", customerHandler.GetDetail)
				r.Get("/customers/{id}/churn-prediction", customerHandler.GetChurnPrediction)
				r.Get("/customers/{id}/events", customerHandler.ListEvents)
				r.Get("/customers/{id}/assignments", customerHandler.ListAssignments)
				r.Post("/customers/{id}/assignments", customerHandler.AssignCustomer)
				r.Delete("/customers/{id}/assignments/{userID}", customerHandler.UnassignCustomer)
				r.Get("/customers/{id}/notes", customerHandler.ListNotes)
				r.Post("/customers/{id}/notes", customerHandler.CreateNote)
				r.Put("/customers/{id}/notes/{noteID}", customerHandler.UpdateNote)
				r.Delete("/customers/{id}/notes/{noteID}", customerHandler.DeleteNote)
				r.Get("/customers/{id}/insights", customerHandler.ListInsights)
				r.Post("/customers/{id}/insights", customerHandler.GenerateInsight)

				// Dashboard routes
				dashboardSvc := service.NewDashboardService(customerRepo, healthScoreRepo)
				dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
				r.Get("/dashboard/summary", dashboardHandler.GetSummary)
				r.With(
					middleware.RequireFeature(billingLimitsSvc, billingcatalog.FeatureFullDashboard),
				).Get("/dashboard/score-distribution", dashboardHandler.GetScoreDistribution)

				// Integration management routes (admin+ required)
				integrationSvc := service.NewIntegrationService(connRepo, connectorSyncSvc, connectorRegistry)
				integrationHandler := handler.NewIntegrationHandler(integrationSvc)
				r.Route("/integrations", func(r chi.Router) {
					r.Get("/", integrationHandler.List)
					r.Route("/generic-webhooks", func(r chi.Router) {
						r.Use(middleware.RequireRole("admin"))
						r.Get("/", genericWebhookHandler.List)
						r.Post("/", genericWebhookHandler.Create)
						r.Post("/test", genericWebhookHandler.Test)
						r.Get("/{id}", genericWebhookHandler.Get)
						r.Patch("/{id}", genericWebhookHandler.Update)
						r.Delete("/{id}", genericWebhookHandler.Delete)
					})
					r.Route("/{provider}", func(r chi.Router) {
						r.Use(middleware.RequireRole("admin"))
						r.Post("/connect", integrationHandler.Connect)
						r.Get("/status", integrationHandler.GetStatus)
						r.Post("/sync", integrationHandler.TriggerSync)
						r.Delete("/", integrationHandler.Disconnect)
					})
				})

				// Connector marketplace routes
				marketplaceSvc := service.NewMarketplaceService(marketplaceRepo)
				marketplaceHandler := handler.NewMarketplaceHandler(marketplaceSvc)
				r.Route("/marketplace/connectors", func(r chi.Router) {
					r.Get("/", marketplaceHandler.ListPublished)
					r.Get("/{id}", marketplaceHandler.GetPublished)
					r.Group(func(r chi.Router) {
						r.Use(middleware.RequireRole("admin"))
						r.Post("/", marketplaceHandler.Register)
						r.Post("/{id}/versions/{version}/review", marketplaceHandler.Review)
						r.Post("/{id}/install", marketplaceHandler.Install)
					})
				})

				// Member management routes (admin+ required)
				memberSvc := service.NewMemberService(orgRepo)
				memberHandler := handler.NewMemberHandler(memberSvc)
				r.Get("/members", memberHandler.List)
				r.Route("/members/{id}", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.Patch("/role", memberHandler.UpdateRole)
					r.Delete("/", memberHandler.Remove)
				})

				// Invitation routes (admin+ required)
				r.Route("/invitations", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.Post("/", invitationHandler.Create)
					r.Get("/", invitationHandler.List)
					r.Delete("/{id}", invitationHandler.Revoke)
				})

				// Notification preferences routes
				notifPrefHandler := handler.NewNotificationPreferenceHandler(notifPrefSvc)
				r.Get("/notifications/preferences", notifPrefHandler.Get)
				r.Patch("/notifications/preferences", notifPrefHandler.Update)

				// Notification routes
				notifHandler := handler.NewNotificationHandler(notifSvc)
				r.Get("/notifications", notifHandler.List)
				r.Get("/notifications/unread-count", notifHandler.CountUnread)
				r.Post("/notifications/{id}/read", notifHandler.MarkRead)
				r.Post("/notifications/read-all", notifHandler.MarkAllRead)

				// Alert rule routes (admin+ required)
				alertRuleSvc := service.NewAlertRuleService(alertRuleRepo)
				alertRuleHandler := handler.NewAlertRuleHandler(alertRuleSvc)
				alertHistoryHandler := handler.NewAlertHistoryHandler(alertHistoryRepo)
				r.Route("/alerts/rules", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.Get("/", alertRuleHandler.List)
					r.Post("/", alertRuleHandler.Create)
					r.Get("/{id}", alertRuleHandler.Get)
					r.Patch("/{id}", alertRuleHandler.Update)
					r.Delete("/{id}", alertRuleHandler.Delete)
					r.Get("/{id}/history", alertHistoryHandler.ListByRule)
				})

				// Alert history routes
				r.Get("/alerts/history", alertHistoryHandler.List)
				r.Get("/alerts/stats", alertHistoryHandler.Stats)

				// Stripe integration routes (admin+ required)
				stripeHandler := handler.NewIntegrationStripeHandler(stripeOAuthSvc, connectorSyncSvc)
				r.Route("/integrations/stripe", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.With(middleware.RequireIntegrationLimit(billingLimitsSvc, "stripe")).Get("/connect", stripeHandler.Connect)
					r.Get("/callback", stripeHandler.Callback)
					r.Get("/status", stripeHandler.Status)
					r.Delete("/", stripeHandler.Disconnect)
					r.Post("/sync", stripeHandler.TriggerSync)
				})

				// HubSpot integration routes (admin+ required)
				hubspotHandler := handler.NewIntegrationHubSpotHandler(hubspotOAuthSvc, connectorSyncSvc)
				r.Route("/integrations/hubspot", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.With(middleware.RequireIntegrationLimit(billingLimitsSvc, "hubspot")).Get("/connect", hubspotHandler.Connect)
					r.Get("/callback", hubspotHandler.Callback)
					r.Get("/status", hubspotHandler.Status)
					r.Delete("/", hubspotHandler.Disconnect)
					r.Post("/sync", hubspotHandler.TriggerSync)
				})

				// Intercom integration routes (admin+ required)
				intercomHandler := handler.NewIntegrationIntercomHandler(intercomOAuthSvc, connectorSyncSvc)
				r.Route("/integrations/intercom", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.With(middleware.RequireIntegrationLimit(billingLimitsSvc, "intercom")).Get("/connect", intercomHandler.Connect)
					r.Get("/callback", intercomHandler.Callback)
					r.Get("/status", intercomHandler.Status)
					r.Delete("/", intercomHandler.Disconnect)
					r.Post("/sync", intercomHandler.TriggerSync)
				})

				// Zendesk integration routes (admin+ required)
				zendeskHandler := handler.NewIntegrationZendeskHandler(zendeskOAuthSvc, connectorSyncSvc)
				r.Route("/integrations/zendesk", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.With(middleware.RequireIntegrationLimit(billingLimitsSvc, "zendesk")).Get("/connect", zendeskHandler.Connect)
					r.Get("/callback", zendeskHandler.Callback)
					r.Get("/status", zendeskHandler.Status)
					r.Delete("/", zendeskHandler.Disconnect)
					r.Post("/sync", zendeskHandler.TriggerSync)
				})

				// Salesforce integration routes (admin+ required)
				salesforceHandler := handler.NewIntegrationSalesforceHandler(salesforceOAuthSvc, connectorSyncSvc)
				r.Route("/integrations/salesforce", func(r chi.Router) {
					r.Use(middleware.RequireRole("admin"))
					r.With(middleware.RequireIntegrationLimit(billingLimitsSvc, "salesforce")).Get("/connect", salesforceHandler.Connect)
					r.Get("/callback", salesforceHandler.Callback)
					r.Get("/status", salesforceHandler.Status)
					r.Delete("/", salesforceHandler.Disconnect)
					r.Post("/sync", salesforceHandler.TriggerSync)
				})

				// Onboarding routes
				onboardingHandler := handler.NewOnboardingHandler(onboardingSvc)
				r.Route("/onboarding", func(r chi.Router) {
					r.Get("/status", onboardingHandler.GetStatus)
					r.Patch("/status", onboardingHandler.UpdateStatus)
					r.Post("/complete", onboardingHandler.Complete)
					r.Post("/reset", onboardingHandler.Reset)
					r.Get("/analytics", onboardingHandler.Analytics)
				})

				// Health scoring routes
				scoringHandler := handler.NewScoringHandler(scoringConfigSvc, riskCategorizer, scoreScheduler)
				r.Route("/scoring", func(r chi.Router) {
					r.With(
						middleware.RequireFeature(billingLimitsSvc, billingcatalog.FeatureFullDashboard),
					).Get("/risk-distribution", scoringHandler.GetRiskDistribution)
					r.With(
						middleware.RequireFeature(billingLimitsSvc, billingcatalog.FeatureFullDashboard),
					).Get("/histogram", scoringHandler.GetScoreHistogram)
					r.Post("/customers/{id}/recalculate", scoringHandler.RecalculateCustomer)
					r.Route("/config", func(r chi.Router) {
						r.Use(middleware.RequireRole("admin"))
						r.Get("/", scoringHandler.GetConfig)
						r.Put("/", scoringHandler.UpdateConfig)
					})
				})
			})
		}
	})
}

func newHTTPServer(cfg *config.Config, handler http.Handler) *http.Server {
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

func runServerWithGracefulShutdown(srv *http.Server, environment string) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("starting PulseScore API", "addr", srv.Addr, "env", environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
