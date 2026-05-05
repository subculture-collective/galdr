package service

import (
	"context"
	"testing"
)

func TestAuthRegisterRejectsUnknownIndustry(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, nil, nil, 0, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:     "owner@example.com",
		Password:  "StrongPass123",
		FirstName: "Ada",
		LastName:  "Lovelace",
		OrgName:   "Acme",
		Industry:  "Professional Services",
	})

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Field != "industry" {
		t.Fatalf("expected industry validation error, got %q", validationErr.Field)
	}
}
