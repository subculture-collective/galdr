package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	"github.com/onnwee/pulse-score/internal/service"
)

// customerServicer defines the methods the CustomerHandler needs.
type customerServicer interface {
	List(ctx context.Context, params repository.CustomerListParams) (*service.CustomerListResponse, error)
	GetDetail(ctx context.Context, customerID, orgID uuid.UUID) (*service.CustomerDetail, error)
	ListEvents(ctx context.Context, params repository.EventListParams) (*service.EventListResponse, error)
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

// integrationServicer defines the methods the IntegrationHandler needs.
type integrationServicer interface {
	List(ctx context.Context, orgID uuid.UUID) ([]service.IntegrationSummary, error)
	GetStatus(ctx context.Context, orgID uuid.UUID, provider string) (*service.IntegrationStatus, error)
	TriggerSync(ctx context.Context, orgID uuid.UUID, provider string) error
	Disconnect(ctx context.Context, orgID uuid.UUID, provider string) error
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

// organizationServicer defines the methods the OrganizationHandler needs.
type organizationServicer interface {
	GetCurrent(ctx context.Context, orgID uuid.UUID) (*service.OrgDetailResponse, error)
	UpdateCurrent(ctx context.Context, orgID uuid.UUID, req service.UpdateOrgRequest) (*service.OrgDetailResponse, error)
	Create(ctx context.Context, userID uuid.UUID, req service.CreateOrgRequest) (*service.OrgResponse, error)
}
