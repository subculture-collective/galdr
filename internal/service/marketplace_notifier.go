package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

type marketplaceDeveloperRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*repository.User, error)
}

type MarketplaceEmailNotifier struct {
	users marketplaceDeveloperRepository
	email EmailService
}

func NewMarketplaceEmailNotifier(users marketplaceDeveloperRepository, email EmailService) *MarketplaceEmailNotifier {
	return &MarketplaceEmailNotifier{users: users, email: email}
}

func (n *MarketplaceEmailNotifier) NotifyConnectorStatusChange(ctx context.Context, connector *repository.MarketplaceConnector, status string) error {
	if n == nil || n.users == nil || n.email == nil || connector == nil || connector.DeveloperID == uuid.Nil {
		return nil
	}
	developer, err := n.users.GetByID(ctx, connector.DeveloperID)
	if err != nil {
		return err
	}
	if developer == nil || strings.TrimSpace(developer.Email) == "" {
		return nil
	}

	subject := fmt.Sprintf("PulseScore connector %s is %s", connector.Name, strings.ReplaceAll(status, "_", " "))
	text := fmt.Sprintf("Your connector submission %s@%s is now %s.", connector.ID, connector.Version, strings.ReplaceAll(status, "_", " "))
	_, err = n.email.SendEmail(ctx, SendEmailParams{
		To:       developer.Email,
		Subject:  subject,
		TextBody: text,
		HTMLBody: "<p>" + text + "</p>",
	})
	return err
}
