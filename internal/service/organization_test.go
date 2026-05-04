package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestOrganizationUpdateCurrentRejectsUnknownIndustry(t *testing.T) {
	svc := NewOrganizationService(nil, nil)

	_, err := svc.UpdateCurrent(context.Background(), uuid.New(), UpdateOrgRequest{
		Name:     "Acme",
		Industry: "Professional Services",
	})

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Field != "industry" {
		t.Fatalf("expected industry validation error, got %q", validationErr.Field)
	}
}
