package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestOrganizationUpdateCurrentRejectsUnknownIndustry(t *testing.T) {
	svc := NewOrganizationService(nil, nil)
	industry := "Professional Services"

	_, err := svc.UpdateCurrent(context.Background(), uuid.New(), UpdateOrgRequest{
		Industry: &industry,
	})

	assertIndustryValidationError(t, err)
}

func TestOrganizationCreateRejectsUnknownIndustry(t *testing.T) {
	svc := NewOrganizationService(nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), CreateOrgRequest{
		Name:     "Acme",
		Industry: "Professional Services",
	})

	assertIndustryValidationError(t, err)
}

func assertIndustryValidationError(t *testing.T, err error) {
	t.Helper()

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Field != "industry" {
		t.Fatalf("expected industry validation error, got %q", validationErr.Field)
	}
}
