package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/onnwee/pulse-score/internal/repository"
)

// CreateOrgRequest holds the input for creating an organization.
type CreateOrgRequest struct {
	Name     string `json:"name"`
	Industry string `json:"industry"`
}

var allowedIndustries = map[string]struct{}{
	"SaaS":        {},
	"E-commerce":  {},
	"Fintech":     {},
	"Healthcare":  {},
	"Education":   {},
	"Media":       {},
	"Marketplace": {},
	"Agency":      {},
	"Other":       {},
}

const industryValidationMessage = "industry must be one of the predefined options"
const industryRequiredMessage = "industry is required"
const benchmarkIndustryRequiredMessage = "industry is required for benchmark participation"

func validateIndustry(industry string) (string, error) {
	industry = strings.TrimSpace(industry)
	if _, ok := allowedIndustries[industry]; industry != "" && !ok {
		return "", &ValidationError{Field: "industry", Message: industryValidationMessage}
	}
	return industry, nil
}

func validateRequiredIndustry(industry string) (string, error) {
	industry, err := validateIndustry(industry)
	if err != nil {
		return "", err
	}
	if industry == "" {
		return "", &ValidationError{Field: "industry", Message: industryRequiredMessage}
	}
	return industry, nil
}

func validateBenchmarkIndustry(benchmarkingEnabled bool, industry string) error {
	if benchmarkingEnabled && strings.TrimSpace(industry) == "" {
		return &ValidationError{Field: "industry", Message: benchmarkIndustryRequiredMessage}
	}
	return nil
}

// OrgResponse is the response for organization operations.
type OrgResponse struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	Industry            string    `json:"industry"`
	Plan                string    `json:"plan"`
	BenchmarkingEnabled bool      `json:"benchmarking_enabled"`
	CompanySize         int       `json:"company_size"`
}

// OrganizationService handles organization logic.
type OrganizationService struct {
	pool *pgxpool.Pool
	orgs organizationStore
	benchmarkContributions benchmarkContributionDeleter
}

type organizationStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*repository.Organization, error)
	GetWithStats(ctx context.Context, orgID uuid.UUID) (*repository.OrganizationWithStats, error)
	Update(ctx context.Context, orgID uuid.UUID, name, slug, industry string) error
	UpdateBenchmarkSettings(ctx context.Context, orgID uuid.UUID, benchmarkingEnabled bool, industry string, companySize int) error
	SlugExists(ctx context.Context, slug string) (bool, error)
	Create(ctx context.Context, tx pgx.Tx, org *repository.Organization) error
	AddMember(ctx context.Context, tx pgx.Tx, userID, orgID uuid.UUID, role string) error
}

type benchmarkContributionDeleter interface {
	DeleteContributionsByOrg(ctx context.Context, orgID uuid.UUID) error
}

// NewOrganizationService creates a new OrganizationService.
func NewOrganizationService(pool *pgxpool.Pool, orgs organizationStore, benchmarkContributions ...benchmarkContributionDeleter) *OrganizationService {
	var deleter benchmarkContributionDeleter
	if len(benchmarkContributions) > 0 {
		deleter = benchmarkContributions[0]
	}
	return &OrganizationService{pool: pool, orgs: orgs, benchmarkContributions: deleter}
}

// UpdateOrgRequest holds input for updating an organization.
type UpdateOrgRequest struct {
	Name                *string `json:"name"`
	BenchmarkingEnabled *bool   `json:"benchmarking_enabled"`
	Industry            *string `json:"industry"`
	CompanySize         *int    `json:"company_size"`
}

// OrgDetailResponse is the response for organization detail.
type OrgDetailResponse struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	Industry            string    `json:"industry"`
	Plan                string    `json:"plan"`
	BenchmarkingEnabled bool      `json:"benchmarking_enabled"`
	CompanySize         int       `json:"company_size"`
	MemberCount         int       `json:"member_count"`
	CustomerCount       int       `json:"customer_count"`
}

// GetCurrent returns the current org with stats.
func (s *OrganizationService) GetCurrent(ctx context.Context, orgID uuid.UUID) (*OrgDetailResponse, error) {
	org, err := s.orgs.GetWithStats(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}
	if org == nil {
		return nil, &NotFoundError{Resource: "organization", Message: "organization not found"}
	}

	return &OrgDetailResponse{
		ID:                  org.ID,
		Name:                org.Name,
		Slug:                org.Slug,
		Industry:            org.Industry,
		Plan:                org.Plan,
		BenchmarkingEnabled: org.BenchmarkingEnabled,
		CompanySize:         org.CompanySize,
		MemberCount:         org.MemberCount,
		CustomerCount:       org.CustomerCount,
	}, nil
}

// UpdateCurrent updates the current org settings.
func (s *OrganizationService) UpdateCurrent(ctx context.Context, orgID uuid.UUID, req UpdateOrgRequest) (*OrgDetailResponse, error) {
	var requestedIndustry string
	if req.Industry != nil {
		var err error
		requestedIndustry, err = validateIndustry(*req.Industry)
		if err != nil {
			return nil, err
		}
	}
	if req.BenchmarkingEnabled != nil && *req.BenchmarkingEnabled && req.Industry != nil {
		if err := validateBenchmarkIndustry(true, requestedIndustry); err != nil {
			return nil, err
		}
	}

	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}
	if org == nil {
		return nil, &NotFoundError{Resource: "organization", Message: "organization not found"}
	}

	name := org.Name
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, &ValidationError{Field: "name", Message: "organization name is required"}
		}
	}
	industry := org.Industry
	if req.Industry != nil {
		industry = requestedIndustry
	}

	slug, err := s.uniqueSlug(ctx, name, org.Slug)
	if err != nil {
		return nil, err
	}

	benchmarkingEnabled := org.BenchmarkingEnabled
	if req.BenchmarkingEnabled != nil {
		benchmarkingEnabled = *req.BenchmarkingEnabled
	}
	companySize := org.CompanySize
	if req.CompanySize != nil {
		if *req.CompanySize < 0 {
			return nil, &ValidationError{Field: "company_size", Message: "company size must be greater than or equal to 0"}
		}
		companySize = *req.CompanySize
	}
	if err := validateBenchmarkIndustry(benchmarkingEnabled, industry); err != nil {
		return nil, err
	}

	if err := s.orgs.Update(ctx, orgID, name, slug, industry); err != nil {
		return nil, fmt.Errorf("update org: %w", err)
	}
	if err := s.orgs.UpdateBenchmarkSettings(ctx, orgID, benchmarkingEnabled, industry, companySize); err != nil {
		return nil, fmt.Errorf("update org benchmark settings: %w", err)
	}
	if org.BenchmarkingEnabled && !benchmarkingEnabled && s.benchmarkContributions != nil {
		if err := s.benchmarkContributions.DeleteContributionsByOrg(ctx, orgID); err != nil {
			return nil, fmt.Errorf("delete benchmark contributions on opt-out: %w", err)
		}
	}

	return s.GetCurrent(ctx, orgID)
}

// Create creates a new organization and assigns the caller as owner.
func (s *OrganizationService) Create(ctx context.Context, userID uuid.UUID, req CreateOrgRequest) (*OrgResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, &ValidationError{Field: "name", Message: "organization name is required"}
	}
	industry, err := validateRequiredIndustry(req.Industry)
	if err != nil {
		return nil, err
	}

	slug, err := s.uniqueSlug(ctx, name, "")
	if err != nil {
		return nil, err
	}

	org := &repository.Organization{
		Name:     name,
		Slug:     slug,
		Industry: industry,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.orgs.Create(ctx, tx, org); err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}

	if err := s.orgs.AddMember(ctx, tx, userID, org.ID, "owner"); err != nil {
		return nil, fmt.Errorf("add owner: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &OrgResponse{
		ID:                  org.ID,
		Name:                org.Name,
		Slug:                org.Slug,
		Industry:            org.Industry,
		Plan:                "free",
		BenchmarkingEnabled: false,
		CompanySize:         0,
	}, nil
}

func (s *OrganizationService) uniqueSlug(ctx context.Context, name, currentSlug string) (string, error) {
	slug := generateSlug(name)
	baseSlug := slug
	for i := 1; ; i++ {
		exists, err := s.orgs.SlugExists(ctx, slug)
		if err != nil {
			return "", fmt.Errorf("check slug: %w", err)
		}
		if !exists || currentSlug == slug {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}
}
