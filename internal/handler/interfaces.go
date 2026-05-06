package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

// customerServicer defines the methods the CustomerHandler needs.
type customerServicer interface {
	List(ctx context.Context, params repository.CustomerListParams) (*service.CustomerListResponse, error)
	GetDetail(ctx context.Context, customerID, orgID uuid.UUID) (*service.CustomerDetail, error)
	GetChurnPrediction(ctx context.Context, customerID, orgID uuid.UUID) (*repository.ChurnPrediction, error)
	ListEvents(ctx context.Context, params repository.EventListParams) (*service.EventListResponse, error)
	ListAssignments(ctx context.Context, customerID, orgID uuid.UUID) (*service.CustomerAssignmentsResponse, error)
	AssignCustomer(ctx context.Context, customerID, orgID, assigneeID, assignedBy uuid.UUID) (*service.CustomerAssignmentResponse, error)
	UnassignCustomer(ctx context.Context, customerID, orgID, assigneeID uuid.UUID) error
	ListNotes(ctx context.Context, customerID, orgID, actorID uuid.UUID, actorRole string) (*service.CustomerNotesResponse, error)
	CreateNote(ctx context.Context, customerID, orgID, userID uuid.UUID, req service.CustomerNoteRequest) (*service.CustomerNoteResponse, error)
	UpdateNote(ctx context.Context, customerID, noteID, orgID, userID uuid.UUID, actorRole string, req service.CustomerNoteRequest) (*service.CustomerNoteResponse, error)
	DeleteNote(ctx context.Context, customerID, noteID, orgID, userID uuid.UUID, actorRole string) error
}

// customerInsightServicer defines the methods CustomerHandler needs for AI insights.
type customerInsightServicer interface {
	GenerateCustomerInsight(ctx context.Context, orgID, customerID uuid.UUID, opts service.InsightGenerationOptions) (*service.CustomerInsightResponse, error)
	ListCustomerInsights(ctx context.Context, orgID, customerID uuid.UUID, limit int) ([]*repository.CustomerInsight, error)
}

// dashboardServicer defines the methods the DashboardHandler needs.
type dashboardServicer interface {
	GetSummary(ctx context.Context, orgID uuid.UUID) (*service.DashboardSummary, error)
	GetScoreDistribution(ctx context.Context, orgID uuid.UUID) (*service.ScoreDistributionResponse, error)
}

type benchmarkServicer interface {
	Compare(ctx context.Context, orgID uuid.UUID, industry, companySizeBucket string) (*service.BenchmarkComparisonResponse, error)
}

// integrationServicer defines the methods the IntegrationHandler needs.
type integrationServicer interface {
	List(ctx context.Context, orgID uuid.UUID) ([]service.IntegrationSummary, error)
	Connect(ctx context.Context, orgID uuid.UUID, provider string, req service.ConnectIntegrationRequest) (*connectorsdk.AuthResult, error)
	GetStatus(ctx context.Context, orgID uuid.UUID, provider string) (*service.IntegrationStatus, error)
	TriggerSync(ctx context.Context, orgID uuid.UUID, provider string) error
	Disconnect(ctx context.Context, orgID uuid.UUID, provider string) error
}

// genericWebhookServicer defines the methods the GenericWebhookHandler needs.
type genericWebhookServicer interface {
	Create(ctx context.Context, orgID uuid.UUID, req service.GenericWebhookConfigRequest) (*repository.GenericWebhookConfig, error)
	List(ctx context.Context, orgID uuid.UUID) ([]*repository.GenericWebhookConfig, error)
	Get(ctx context.Context, orgID, id uuid.UUID) (*repository.GenericWebhookConfig, error)
	Update(ctx context.Context, orgID, id uuid.UUID, req service.GenericWebhookConfigRequest) (*repository.GenericWebhookConfig, error)
	Delete(ctx context.Context, orgID, id uuid.UUID) error
	Process(ctx context.Context, orgSlug string, webhookID uuid.UUID, payload []byte, signature string) (*service.GenericWebhookProcessResult, error)
	Test(ctx context.Context, orgID uuid.UUID, req service.GenericWebhookTestRequest) (*service.GenericWebhookTestResult, error)
}

// marketplaceServicer defines the methods the MarketplaceHandler needs.
type marketplaceServicer interface {
	Register(ctx context.Context, developerID uuid.UUID, req service.RegisterConnectorRequest) (*repository.MarketplaceConnector, error)
	ListPublished(ctx context.Context) ([]*repository.MarketplaceConnector, error)
	GetPublished(ctx context.Context, id string) (*repository.MarketplaceConnector, error)
	Install(ctx context.Context, orgID uuid.UUID, id string, req service.InstallConnectorRequest) (*repository.ConnectorInstallation, error)
	Review(ctx context.Context, reviewerID uuid.UUID, id, version string, req service.ConnectorReviewRequest) (*repository.ConnectorReviewResult, error)
}

// memberServicer defines the methods the MemberHandler needs.
type memberServicer interface {
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]service.MemberResponse, error)
	UpdateRole(ctx context.Context, orgID, callerID, targetUserID uuid.UUID, req service.UpdateRoleRequest) error
	RemoveMember(ctx context.Context, orgID, callerID, targetUserID uuid.UUID) error
}

// alertRuleServicer defines the methods the AlertRuleHandler needs.
type alertRuleServicer interface {
	List(ctx context.Context, orgID uuid.UUID) ([]*repository.AlertRule, error)
	GetByID(ctx context.Context, id, orgID uuid.UUID) (*repository.AlertRule, error)
	Create(ctx context.Context, orgID, userID uuid.UUID, req service.CreateAlertRuleRequest) (*repository.AlertRule, error)
	Update(ctx context.Context, id, orgID uuid.UUID, req service.UpdateAlertRuleRequest) (*repository.AlertRule, error)
	Delete(ctx context.Context, id, orgID uuid.UUID) error
}

// savedViewServicer defines the methods the SavedViewHandler needs.
type savedViewServicer interface {
	List(ctx context.Context, orgID, userID uuid.UUID) ([]*repository.SavedView, error)
	GetByID(ctx context.Context, id, orgID, userID uuid.UUID) (*repository.SavedView, error)
	Create(ctx context.Context, orgID, userID uuid.UUID, req service.CreateSavedViewRequest) (*repository.SavedView, error)
	Update(ctx context.Context, id, orgID, userID uuid.UUID, req service.UpdateSavedViewRequest) (*repository.SavedView, error)
	Delete(ctx context.Context, id, orgID, userID uuid.UUID) error
}

// organizationServicer defines the methods the OrganizationHandler needs.
type organizationServicer interface {
	GetCurrent(ctx context.Context, orgID uuid.UUID) (*service.OrgDetailResponse, error)
	UpdateCurrent(ctx context.Context, orgID uuid.UUID, req service.UpdateOrgRequest) (*service.OrgDetailResponse, error)
	Create(ctx context.Context, userID uuid.UUID, req service.CreateOrgRequest) (*service.OrgResponse, error)
}
