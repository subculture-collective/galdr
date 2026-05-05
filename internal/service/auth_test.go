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

	assertIndustryValidationError(t, err, industryValidationMessage)
}

func TestAuthRegisterRequiresIndustry(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, nil, nil, 0, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:     "owner@example.com",
		Password:  "StrongPass123",
		FirstName: "Ada",
		LastName:  "Lovelace",
		OrgName:   "Acme",
		Industry:  " ",
	})

	assertIndustryValidationError(t, err, industryRequiredMessage)
}
