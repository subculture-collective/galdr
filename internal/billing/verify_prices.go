package billing

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/stripe/stripe-go/v81"
	stripeprice "github.com/stripe/stripe-go/v81/price"
)

// VerifyConfiguredPrices validates that configured Stripe price IDs exist and include required metadata.
// Required metadata keys: tier, customer_limit, integration_limit, team_member_limit.
func VerifyConfiguredPrices(ctx context.Context, stripeSecretKey string, catalog *Catalog) error {
	if strings.TrimSpace(stripeSecretKey) == "" {
		return fmt.Errorf("stripe billing secret key is required for price verification")
	}
	if catalog == nil {
		return fmt.Errorf("billing catalog is required for price verification")
	}

	client := stripeprice.Client{B: stripe.GetBackend(stripe.APIBackend), Key: stripeSecretKey}

	for _, tier := range []Tier{TierGrowth, TierScale} {
		plan, ok := catalog.GetPlanByTier(string(tier))
		if !ok {
			return fmt.Errorf("plan not found for tier %s", tier)
		}

		for _, cycle := range []BillingCycle{BillingCycleMonthly, BillingCycleAnnual} {
			priceID := plan.StripeMonthlyPriceID
			if cycle == BillingCycleAnnual {
				priceID = plan.StripeAnnualPriceID
			}
			if priceID == "" {
				return fmt.Errorf("missing configured price id for tier=%s cycle=%s", tier, cycle)
			}

			p, err := client.Get(priceID, nil)
			if err != nil {
				return fmt.Errorf("fetch stripe price %s: %w", priceID, err)
			}

			if got := strings.ToLower(strings.TrimSpace(p.Metadata["tier"])); got != string(tier) {
				return fmt.Errorf("price %s metadata tier mismatch: expected=%s got=%s", priceID, tier, got)
			}

			if !limitMetadataMatches(p.Metadata["customer_limit"], plan.Limits.CustomerLimit) {
				return fmt.Errorf("price %s metadata customer_limit mismatch for tier %s", priceID, tier)
			}
			if !limitMetadataMatches(p.Metadata["integration_limit"], plan.Limits.IntegrationLimit) {
				return fmt.Errorf("price %s metadata integration_limit mismatch for tier %s", priceID, tier)
			}
			if !limitMetadataMatches(p.Metadata["team_member_limit"], plan.Limits.TeamMemberLimit) {
				return fmt.Errorf("price %s metadata team_member_limit mismatch for tier %s", priceID, tier)
			}
		}
	}

	_ = ctx // kept for future request scoping if Stripe SDK adds context support to this client path
	return nil
}

func limitMetadataMatches(raw string, expected int) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if expected == Unlimited {
		return raw == "unlimited" || raw == "-1"
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return false
	}
	return parsed == expected
}
