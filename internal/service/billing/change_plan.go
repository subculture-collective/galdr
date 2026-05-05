package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
	core "github.com/onnwee/pulse-score/internal/service"
)

type planChangeSubscriptionReader interface {
	GetByOrg(ctx context.Context, orgID uuid.UUID) (*repository.OrgSubscription, error)
	UpsertByOrg(ctx context.Context, sub *repository.OrgSubscription) error
}

const (
	planChangeActionUpgrade   = "upgrade"
	planChangeActionDowngrade = "downgrade"

	planChangeStatusCheckoutRequired = "checkout_required"
	planChangeStatusScheduled        = "scheduled"
	planChangeStatusActive           = "active"
)

// ChangePlanRequest is the requested target plan and billing cycle.
type ChangePlanRequest struct {
	Tier   string `json:"tier"`
	Cycle  string `json:"cycle"`
	Annual bool   `json:"annual"`
}

// PlanChangeLimitsImpact explains limit changes shown in confirmation UI.
type PlanChangeLimitsImpact struct {
	Current planmodel.UsageLimits `json:"current"`
	Target  planmodel.UsageLimits `json:"target"`
}

// PlanChangeFeaturesImpact explains feature changes shown in confirmation UI.
type PlanChangeFeaturesImpact struct {
	Current planmodel.FeatureFlags `json:"current"`
	Target  planmodel.FeatureFlags `json:"target"`
}

// ChangePlanResponse describes upgrade checkout or scheduled downgrade result.
type ChangePlanResponse struct {
	Action               string                    `json:"action"`
	Status               string                    `json:"status"`
	CurrentTier          string                    `json:"current_tier"`
	CurrentCycle         string                    `json:"current_cycle"`
	TargetTier           string                    `json:"target_tier"`
	TargetCycle          string                    `json:"target_cycle"`
	CheckoutURL          string                    `json:"checkout_url,omitempty"`
	EffectiveAt          *time.Time                `json:"effective_at,omitempty"`
	EffectiveAtPeriodEnd bool                      `json:"effective_at_period_end"`
	ProrationCents       int                       `json:"proration_cents"`
	Limits               PlanChangeLimitsImpact    `json:"limits"`
	Features             PlanChangeFeaturesImpact  `json:"features"`
}

// ChangePlanService coordinates paid plan upgrades and scheduled downgrades.
type ChangePlanService struct {
	stripeSecretKey string
	checkout        *CheckoutService
	subscriptions   planChangeSubscriptionReader
	catalog         *planmodel.Catalog
	httpClient      *http.Client
}

func NewChangePlanService(stripeSecretKey string, checkout *CheckoutService, subscriptions planChangeSubscriptionReader, catalog *planmodel.Catalog) *ChangePlanService {
	return &ChangePlanService{
		stripeSecretKey: strings.TrimSpace(stripeSecretKey),
		checkout:        checkout,
		subscriptions:   subscriptions,
		catalog:         catalog,
		httpClient:      http.DefaultClient,
	}
}

func (s *ChangePlanService) ChangePlan(ctx context.Context, orgID, userID uuid.UUID, req ChangePlanRequest) (*ChangePlanResponse, error) {
	targetTier := planmodel.NormalizeTier(req.Tier)
	targetCycle := normalizeBillingCycle(req.Cycle, req.Annual)

	if targetTier == planmodel.TierFree {
		return nil, &core.ValidationError{Field: "tier", Message: "use cancellation to return to the free plan at period end"}
	}

	sub, err := s.subscriptions.GetByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	resp, err := buildPlanChangeImpact(s.catalog, sub, targetTier, targetCycle)
	if err != nil {
		return nil, err
	}

	switch resp.Action {
	case planChangeActionUpgrade:
		if !hasPaidStripeSubscription(sub) {
			checkoutResp, err := s.checkout.CreateCheckoutSession(ctx, orgID, userID, CreateCheckoutSessionRequest{Tier: string(targetTier), Cycle: string(targetCycle)})
			if err != nil {
				return nil, err
			}
			resp.CheckoutURL = checkoutResp.URL
		} else {
			if err := s.applyUpgradeNow(ctx, sub, targetTier, targetCycle); err != nil {
				return nil, err
			}
			resp.Status = planChangeStatusActive
		}
	case planChangeActionDowngrade:
		effectiveAt, err := s.scheduleDowngrade(ctx, sub, targetTier, targetCycle)
		if err != nil {
			return nil, err
		}
		resp.EffectiveAt = effectiveAt
	}

	return resp, nil
}

func (s *ChangePlanService) applyUpgradeNow(ctx context.Context, sub *repository.OrgSubscription, targetTier planmodel.Tier, targetCycle planmodel.BillingCycle) error {
	if s.stripeSecretKey == "" {
		return &core.ValidationError{Field: "billing", Message: "stripe billing is not configured"}
	}

	annual := targetCycle == planmodel.BillingCycleAnnual
	priceID, err := s.catalog.GetPriceID(string(targetTier), annual)
	if err != nil {
		return &core.ValidationError{Field: "tier", Message: err.Error()}
	}

	stripeSub, err := s.fetchStripeSubscription(ctx, sub.StripeSubscriptionID)
	if err != nil {
		return err
	}
	if stripeSub.ItemID == "" {
		return &core.ValidationError{Field: "subscription", Message: "subscription has no billable item"}
	}

	updated, err := s.updateSubscriptionForUpgrade(ctx, stripeSub.ID, stripeSub.ItemID, priceID, targetTier, targetCycle)
	if err != nil {
		return err
	}

	sub.PlanTier = string(targetTier)
	sub.BillingCycle = string(targetCycle)
	sub.CancelAtPeriodEnd = false
	if updated.Status != "" {
		sub.Status = updated.Status
	}
	if updated.CurrentPeriodStart > 0 {
		t := time.Unix(updated.CurrentPeriodStart, 0)
		sub.CurrentPeriodStart = &t
	}
	if updated.CurrentPeriodEnd > 0 {
		t := time.Unix(updated.CurrentPeriodEnd, 0)
		sub.CurrentPeriodEnd = &t
	}

	if err := s.subscriptions.UpsertByOrg(ctx, sub); err != nil {
		return fmt.Errorf("persist upgraded subscription: %w", err)
	}
	return nil
}

func buildPlanChangeImpact(catalog *planmodel.Catalog, sub *repository.OrgSubscription, targetTier planmodel.Tier, targetCycle planmodel.BillingCycle) (*ChangePlanResponse, error) {
	return buildPlanChangeImpactAt(catalog, sub, targetTier, targetCycle, time.Now())
}

func buildPlanChangeImpactAt(catalog *planmodel.Catalog, sub *repository.OrgSubscription, targetTier planmodel.Tier, targetCycle planmodel.BillingCycle, now time.Time) (*ChangePlanResponse, error) {
	currentTier := planmodel.TierFree
	currentCycle := planmodel.BillingCycleMonthly
	var effectiveAt *time.Time
	if sub != nil {
		currentTier = planmodel.NormalizeTier(sub.PlanTier)
		if strings.EqualFold(sub.BillingCycle, string(planmodel.BillingCycleAnnual)) {
			currentCycle = planmodel.BillingCycleAnnual
		}
		effectiveAt = sub.CurrentPeriodEnd
	}

	currentPlan, ok := catalog.GetPlanByTier(string(currentTier))
	if !ok {
		return nil, &core.ValidationError{Field: "current_tier", Message: "current plan is invalid"}
	}
	targetPlan, ok := catalog.GetPlanByTier(string(targetTier))
	if !ok || targetTier == planmodel.TierFree {
		return nil, &core.ValidationError{Field: "tier", Message: "target plan is invalid"}
	}

	currentRank := planRank(currentTier)
	targetRank := planRank(targetTier)

	if targetRank == currentRank && targetCycle == currentCycle {
		return nil, &core.ValidationError{Field: "tier", Message: "target plan matches current plan"}
	}

	isDowngrade := targetRank < currentRank
	if targetRank == currentRank {
		isDowngrade = planPriceCents(targetPlan, targetCycle) < planPriceCents(currentPlan, currentCycle)
	}

	action := planChangeActionUpgrade
	status := planChangeStatusCheckoutRequired
	effectiveAtPeriodEnd := false
	if isDowngrade {
		action = planChangeActionDowngrade
		status = planChangeStatusScheduled
		effectiveAtPeriodEnd = true
	}

	return &ChangePlanResponse{
		Action:               action,
		Status:               status,
		CurrentTier:          string(currentTier),
		CurrentCycle:         string(currentCycle),
		TargetTier:           string(targetTier),
		TargetCycle:          string(targetCycle),
		EffectiveAt:          effectiveAt,
		EffectiveAtPeriodEnd: effectiveAtPeriodEnd,
		ProrationCents:       estimateProrationCents(currentPlan, currentCycle, targetPlan, targetCycle, action, sub, now),
		Limits: PlanChangeLimitsImpact{
			Current: currentPlan.Limits,
			Target:  targetPlan.Limits,
		},
		Features: PlanChangeFeaturesImpact{
			Current: currentPlan.Features,
			Target:  targetPlan.Features,
		},
	}, nil
}

func (s *ChangePlanService) scheduleDowngrade(ctx context.Context, sub *repository.OrgSubscription, targetTier planmodel.Tier, targetCycle planmodel.BillingCycle) (*time.Time, error) {
	if s.stripeSecretKey == "" {
		return nil, &core.ValidationError{Field: "billing", Message: "stripe billing is not configured"}
	}
	if sub == nil || strings.TrimSpace(sub.StripeSubscriptionID) == "" {
		return nil, &core.NotFoundError{Resource: "subscription", Message: "no active Stripe subscription found"}
	}

	annual := targetCycle == planmodel.BillingCycleAnnual
	priceID, err := s.catalog.GetPriceID(string(targetTier), annual)
	if err != nil {
		return nil, &core.ValidationError{Field: "tier", Message: err.Error()}
	}

	stripeSub, err := s.fetchStripeSubscription(ctx, sub.StripeSubscriptionID)
	if err != nil {
		return nil, err
	}
	if stripeSub.ItemPriceID == "" {
		return nil, &core.ValidationError{Field: "subscription", Message: "subscription has no billable item"}
	}

	scheduleID := stripeSub.ScheduleID
	if scheduleID == "" {
		scheduleID, err = s.createScheduleFromSubscription(ctx, sub.StripeSubscriptionID)
		if err != nil {
			return nil, err
		}
	}

	if err := s.updateScheduleForDowngrade(ctx, scheduleID, stripeSub, priceID, targetTier, targetCycle); err != nil {
		return nil, err
	}

	if stripeSub.CurrentPeriodEnd == 0 {
		return nil, nil
	}
	effectiveAt := time.Unix(stripeSub.CurrentPeriodEnd, 0)
	return &effectiveAt, nil
}

type stripeSubscriptionSnapshot struct {
	ID                 string
	ScheduleID         string
	CurrentPeriodStart int64
	CurrentPeriodEnd   int64
	ItemID             string
	ItemPriceID        string
	ItemQuantity       int64
}

func (s *ChangePlanService) fetchStripeSubscription(ctx context.Context, subscriptionID string) (*stripeSubscriptionSnapshot, error) {
	var payload struct {
		ID                 string `json:"id"`
		Schedule           string `json:"schedule"`
		CurrentPeriodStart int64  `json:"current_period_start"`
		CurrentPeriodEnd   int64  `json:"current_period_end"`
		Items              struct {
			Data []struct {
				ID       string `json:"id"`
				Quantity int64 `json:"quantity"`
				Price    struct {
					ID string `json:"id"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	}

	if err := s.stripeGet(ctx, "/v1/subscriptions/"+url.PathEscape(subscriptionID), &payload); err != nil {
		return nil, fmt.Errorf("get stripe subscription: %w", err)
	}

	snap := &stripeSubscriptionSnapshot{
		ID:                 payload.ID,
		ScheduleID:         payload.Schedule,
		CurrentPeriodStart: payload.CurrentPeriodStart,
		CurrentPeriodEnd:   payload.CurrentPeriodEnd,
		ItemQuantity:       1,
	}
	if len(payload.Items.Data) > 0 {
		snap.ItemID = payload.Items.Data[0].ID
		snap.ItemPriceID = payload.Items.Data[0].Price.ID
		if payload.Items.Data[0].Quantity > 0 {
			snap.ItemQuantity = payload.Items.Data[0].Quantity
		}
	}

	return snap, nil
}

func (s *ChangePlanService) createScheduleFromSubscription(ctx context.Context, subscriptionID string) (string, error) {
	values := url.Values{}
	values.Set("from_subscription", subscriptionID)

	var payload struct {
		ID string `json:"id"`
	}
	if err := s.stripePostForm(ctx, "/v1/subscription_schedules", values, &payload); err != nil {
		return "", fmt.Errorf("create subscription schedule: %w", err)
	}
	return payload.ID, nil
}

func (s *ChangePlanService) updateScheduleForDowngrade(ctx context.Context, scheduleID string, sub *stripeSubscriptionSnapshot, targetPriceID string, targetTier planmodel.Tier, targetCycle planmodel.BillingCycle) error {
	values := url.Values{}
	values.Set("end_behavior", "release")
	values.Set("phases[0][start_date]", fmt.Sprintf("%d", sub.CurrentPeriodStart))
	values.Set("phases[0][end_date]", fmt.Sprintf("%d", sub.CurrentPeriodEnd))
	values.Set("phases[0][items][0][price]", sub.ItemPriceID)
	values.Set("phases[0][items][0][quantity]", fmt.Sprintf("%d", sub.ItemQuantity))
	values.Set("phases[1][start_date]", fmt.Sprintf("%d", sub.CurrentPeriodEnd))
	values.Set("phases[1][items][0][price]", targetPriceID)
	values.Set("phases[1][items][0][quantity]", fmt.Sprintf("%d", sub.ItemQuantity))
	values.Set("phases[1][metadata][tier]", string(targetTier))
	values.Set("phases[1][metadata][cycle]", string(targetCycle))

	var payload struct {
		ID string `json:"id"`
	}
	if err := s.stripePostForm(ctx, "/v1/subscription_schedules/"+url.PathEscape(scheduleID), values, &payload); err != nil {
		return fmt.Errorf("schedule downgrade: %w", err)
	}
	return nil
}

type stripeSubscriptionUpdateResult struct {
	Status             string
	CurrentPeriodStart int64
	CurrentPeriodEnd   int64
}

func (s *ChangePlanService) updateSubscriptionForUpgrade(ctx context.Context, subscriptionID, itemID, targetPriceID string, targetTier planmodel.Tier, targetCycle planmodel.BillingCycle) (*stripeSubscriptionUpdateResult, error) {
	values := url.Values{}
	values.Set("items[0][id]", itemID)
	values.Set("items[0][price]", targetPriceID)
	values.Set("proration_behavior", "create_prorations")
	values.Set("metadata[tier]", string(targetTier))
	values.Set("metadata[cycle]", string(targetCycle))

	var payload struct {
		Status             string `json:"status"`
		CurrentPeriodStart int64  `json:"current_period_start"`
		CurrentPeriodEnd   int64  `json:"current_period_end"`
	}
	if err := s.stripePostForm(ctx, "/v1/subscriptions/"+url.PathEscape(subscriptionID), values, &payload); err != nil {
		return nil, fmt.Errorf("upgrade subscription: %w", err)
	}

	return &stripeSubscriptionUpdateResult{
		Status:             payload.Status,
		CurrentPeriodStart: payload.CurrentPeriodStart,
		CurrentPeriodEnd:   payload.CurrentPeriodEnd,
	}, nil
}

func (s *ChangePlanService) stripeGet(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.stripe.com"+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.stripeSecretKey, "")
	return s.doStripeRequest(req, dest)
}

func (s *ChangePlanService) stripePostForm(ctx context.Context, path string, values url.Values, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.stripe.com"+path, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.stripeSecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.doStripeRequest(req, dest)
}

func (s *ChangePlanService) doStripeRequest(req *http.Request, dest any) error {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var stripeErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&stripeErr)
		if stripeErr.Error.Message == "" {
			stripeErr.Error.Message = resp.Status
		}
		return &core.ValidationError{Field: "stripe", Message: stripeErr.Error.Message}
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

func normalizeBillingCycle(cycle string, annual bool) planmodel.BillingCycle {
	if annual || strings.EqualFold(strings.TrimSpace(cycle), string(planmodel.BillingCycleAnnual)) {
		return planmodel.BillingCycleAnnual
	}
	return planmodel.BillingCycleMonthly
}

func hasPaidStripeSubscription(sub *repository.OrgSubscription) bool {
	if sub == nil || strings.TrimSpace(sub.StripeSubscriptionID) == "" {
		return false
	}
	return planRank(planmodel.NormalizeTier(sub.PlanTier)) > planRank(planmodel.TierFree)
}

func planRank(tier planmodel.Tier) int {
	switch tier {
	case planmodel.TierScale:
		return 2
	case planmodel.TierGrowth:
		return 1
	default:
		return 0
	}
}

func estimateProrationCents(current planmodel.Plan, currentCycle planmodel.BillingCycle, target planmodel.Plan, targetCycle planmodel.BillingCycle, action string, sub *repository.OrgSubscription, now time.Time) int {
	if action != planChangeActionUpgrade {
		return 0
	}

	delta := planPriceCents(target, targetCycle) - planPriceCents(current, currentCycle)
	if delta < 0 {
		return 0
	}

	return prorateCentsForRemainingPeriod(delta, sub, now)
}

func prorateCentsForRemainingPeriod(delta int, sub *repository.OrgSubscription, now time.Time) int {
	if sub == nil || sub.CurrentPeriodStart == nil || sub.CurrentPeriodEnd == nil {
		return delta
	}

	periodDuration := sub.CurrentPeriodEnd.Sub(*sub.CurrentPeriodStart)
	remainingPeriod := sub.CurrentPeriodEnd.Sub(now)
	if periodDuration <= 0 || remainingPeriod <= 0 {
		return 0
	}
	if remainingPeriod >= periodDuration {
		return delta
	}

	return int(float64(delta) * remainingPeriod.Seconds() / periodDuration.Seconds())
}

func planPriceCents(plan planmodel.Plan, cycle planmodel.BillingCycle) int {
	if cycle == planmodel.BillingCycleAnnual {
		return plan.AnnualPriceCents
	}
	return plan.MonthlyPriceCents
}
