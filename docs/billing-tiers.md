# Billing Tier Specifications

PulseScore uses plan definitions in `internal/billing/plans.go` as the source of truth for tier limits and feature access. API responses expose the same limits and feature flags through `GET /api/v1/billing/subscription`.

## Usage Limits

`-1` in code means unlimited and is rendered as `unlimited` in product copy.

| Tier | Customers | Integrations | Team members |
| --- | ---: | ---: | ---: |
| Free | 10 | 1 | 1 |
| Growth | 500 | 3 | 5 |
| Scale | unlimited | unlimited | unlimited |

## Feature Matrix

| Feature flag | Free | Growth | Scale |
| --- | --- | --- | --- |
| `basic_dashboard` | Yes | Yes | Yes |
| `full_dashboard` | No | Yes | Yes |
| `email_alerts` | Yes | Yes | Yes |
| `playbooks` | No | Basic playbooks | All playbooks |
| `ai_insights` | No | No | Yes |
| `benchmarks` | No | No | Yes |

## Access Controls

Feature access should be checked through `Catalog.HasFeature(tier, featureName)` for catalog-level lookups or `LimitsService.CanAccess(ctx, orgID, featureName)` for org-aware service checks.

Customer and integration limits are enforced through `LimitsService.CheckCustomerLimit` and `LimitsService.CheckIntegrationLimit`. Team-member enforcement should use `UsageLimits.TeamMemberLimit` when team gating is wired into member management.

## Tier Intent

Free is for initial evaluation: basic dashboard, email alerts, one integration, and a small customer portfolio.

Growth is for active customer-success teams: larger customer portfolio, three integrations, full dashboard, alerts, basic playbooks, and up to five team members.

Scale is for mature revenue teams: unlimited customers, unlimited integrations, unlimited team, all playbooks, AI insights, and benchmarking.

Enterprise is a future tier for custom limits, SSO, and dedicated support. It is documented here for roadmap context only and is not implemented in code yet.
