package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/onnwee/pulse-score/internal/repository"
)

func TestOrganizationUpdateCurrentRejectsUnknownIndustry(t *testing.T) {
	svc := NewOrganizationService(nil, nil, nil)
	industry := "Professional Services"

	_, err := svc.UpdateCurrent(context.Background(), uuid.New(), UpdateOrgRequest{
		Industry: &industry,
	})

	assertIndustryValidationError(t, err, industryValidationMessage)
}

func TestOrganizationUpdateCurrentRequiresIndustryForBenchmarking(t *testing.T) {
	svc := NewOrganizationService(nil, nil, nil)
	enabled := true
	industry := " "

	_, err := svc.UpdateCurrent(context.Background(), uuid.New(), UpdateOrgRequest{
		BenchmarkingEnabled: &enabled,
		Industry:            &industry,
	})

	assertIndustryValidationError(t, err, benchmarkIndustryRequiredMessage)
}

func TestOrganizationUpdateCurrentEnablesBenchmarkingWithIndustry(t *testing.T) {
	orgID := uuid.New()
	enabled := true
	industry := "SaaS"
	companySize := 42
	orgs := &fakeOrganizationStore{
		org: &repository.Organization{
			ID:       orgID,
			Name:     "Acme",
			Slug:     "acme",
			Industry: "",
		},
	}
	svc := NewOrganizationService(nil, orgs, nil)

	resp, err := svc.UpdateCurrent(context.Background(), orgID, UpdateOrgRequest{
		BenchmarkingEnabled: &enabled,
		Industry:            &industry,
		CompanySize:         &companySize,
	})

	if err != nil {
		t.Fatalf("update current failed: %v", err)
	}
	if !resp.BenchmarkingEnabled {
		t.Fatal("expected benchmarking enabled after opt-in")
	}
	if resp.Industry != industry {
		t.Fatalf("expected industry %q, got %q", industry, resp.Industry)
	}
	if resp.CompanySize != companySize {
		t.Fatalf("expected company size %d, got %d", companySize, resp.CompanySize)
	}
}

func TestOrganizationUpdateCurrentDeletesBenchmarkContributionsOnOptOut(t *testing.T) {
	orgID := uuid.New()
	enabled := false
	orgs := &fakeOrganizationStore{
		org: &repository.Organization{
			ID:                  orgID,
			Name:                "Acme",
			Slug:                "acme",
			Industry:            "SaaS",
			BenchmarkingEnabled: true,
			CompanySize:         25,
		},
	}
	contributions := &fakeBenchmarkContributionDeleter{}
	svc := NewOrganizationService(nil, orgs, contributions)

	resp, err := svc.UpdateCurrent(context.Background(), orgID, UpdateOrgRequest{
		BenchmarkingEnabled: &enabled,
	})

	if err != nil {
		t.Fatalf("update current failed: %v", err)
	}
	if resp.BenchmarkingEnabled {
		t.Fatal("expected benchmarking disabled after opt-out")
	}
	if contributions.deletedOrgID != orgID {
		t.Fatalf("expected benchmark contributions deleted for %s, got %s", orgID, contributions.deletedOrgID)
	}
}

func TestOrganizationCreateRejectsUnknownIndustry(t *testing.T) {
	svc := NewOrganizationService(nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), CreateOrgRequest{
		Name:     "Acme",
		Industry: "Professional Services",
	})

	assertIndustryValidationError(t, err, industryValidationMessage)
}

func TestOrganizationCreateRequiresIndustry(t *testing.T) {
	svc := NewOrganizationService(nil, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), CreateOrgRequest{
		Name:     "Acme",
		Industry: " ",
	})

	assertIndustryValidationError(t, err, industryRequiredMessage)
}

func assertIndustryValidationError(t *testing.T, err error, message string) {
	t.Helper()

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Field != "industry" {
		t.Fatalf("expected industry validation error, got %q", validationErr.Field)
	}
	if validationErr.Message != message {
		t.Fatalf("expected %q validation message, got %q", message, validationErr.Message)
	}
}

type fakeOrganizationStore struct {
	org *repository.Organization
}

func (f *fakeOrganizationStore) GetByID(ctx context.Context, id uuid.UUID) (*repository.Organization, error) {
	return f.org, nil
}

func (f *fakeOrganizationStore) GetWithStats(ctx context.Context, id uuid.UUID) (*repository.OrganizationWithStats, error) {
	return &repository.OrganizationWithStats{Organization: *f.org}, nil
}

func (f *fakeOrganizationStore) Update(ctx context.Context, orgID uuid.UUID, name, slug, industry string) error {
	f.org.Name = name
	f.org.Slug = slug
	f.org.Industry = industry
	return nil
}

func (f *fakeOrganizationStore) UpdateBenchmarkSettings(ctx context.Context, orgID uuid.UUID, benchmarkingEnabled bool, industry string, companySize int) error {
	f.org.BenchmarkingEnabled = benchmarkingEnabled
	f.org.Industry = industry
	f.org.CompanySize = companySize
	return nil
}

func (f *fakeOrganizationStore) SlugExists(ctx context.Context, slug string) (bool, error) {
	return false, nil
}

func (f *fakeOrganizationStore) Create(ctx context.Context, tx pgx.Tx, org *repository.Organization) error {
	return nil
}

func (f *fakeOrganizationStore) AddMember(ctx context.Context, tx pgx.Tx, userID, orgID uuid.UUID, role string) error {
	return nil
}

type fakeBenchmarkContributionDeleter struct {
	deletedOrgID uuid.UUID
}

func (f *fakeBenchmarkContributionDeleter) DeleteContributionsByOrg(ctx context.Context, orgID uuid.UUID) error {
	f.deletedOrgID = orgID
	return nil
}
