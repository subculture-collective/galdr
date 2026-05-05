export type BillingCycle = "monthly" | "annual";
export type PlanTier = "free" | "growth" | "scale";

export interface BillingPlanDefinition {
  tier: PlanTier;
  name: string;
  description: string;
  monthlyPrice: number;
  annualPrice: number;
  limits: {
    customers: string;
    integrations: string;
    team: string;
    dashboard: string;
    features: string;
  };
  featured?: boolean;
}

export const billingPlans: BillingPlanDefinition[] = [
  {
    tier: "free",
    name: "Free",
    monthlyPrice: 0,
    annualPrice: 0,
    description: "Best for evaluating PulseScore with a small portfolio.",
    limits: {
      customers: "Up to 10 customers",
      integrations: "1 integration",
      team: "1 team member",
      dashboard: "Basic dashboard",
      features: "Email alerts included",
    },
  },
  {
    tier: "growth",
    name: "Growth",
    monthlyPrice: 49,
    annualPrice: 490,
    description: "For fast-moving teams managing churn at scale.",
    featured: true,
    limits: {
      customers: "Up to 500 customers",
      integrations: "Up to 3 integrations",
      team: "Up to 5 team members",
      dashboard: "Full dashboard",
      features: "Basic playbooks and alerts",
    },
  },
  {
    tier: "scale",
    name: "Scale",
    monthlyPrice: 149,
    annualPrice: 1490,
    description: "For mature revenue teams with complex customer motion.",
    limits: {
      customers: "Unlimited customers",
      integrations: "Unlimited integrations",
      team: "Unlimited team members",
      dashboard: "Full dashboard",
      features: "All playbooks, AI insights, and benchmarks",
    },
  },
];

export function savingsBadge(plan: BillingPlanDefinition): string {
  const monthlyAnnualized = plan.monthlyPrice * 12;
  const delta = monthlyAnnualized - plan.annualPrice;
  return delta > 0 ? `Save $${delta}/yr` : "Annual billing";
}
