import { useState } from "react";
import { Loader2 } from "lucide-react";

import { useToast } from "@/contexts/ToastContext";
import { billingApi, type PlanChangeResponse } from "@/lib/api";
import {
  billingPlans,
  type BillingPlanDefinition,
  type BillingCycle,
} from "@/lib/billingPlans";

interface PlanChangeDialogProps {
  plan: BillingPlanDefinition;
  cycle: BillingCycle;
  currentTier: string;
  currentCycle: BillingCycle;
  onClose: () => void;
  onChanged: () => Promise<void>;
}

function money(cents: number): string {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: "USD",
  }).format(cents / 100);
}

function limitLabel(limit: number): string {
  return limit < 0 ? "unlimited" : String(limit);
}

function effectiveLabel(response: PlanChangeResponse | null): string {
  if (!response) return "Immediately after Stripe confirms checkout";
  if (response.status === "active") return "Immediately after confirmation";
  if (!response.effective_at_period_end) return "Immediately after checkout";
  if (!response.effective_at) return "At current period end";
  return new Date(response.effective_at).toLocaleDateString();
}

function successMessage(status: PlanChangeResponse["status"]): string {
  if (status === "active") return "Plan upgraded immediately.";
  return "Plan downgrade scheduled for period end.";
}

function planRank(tier: string): number {
  if (tier === "scale") return 2;
  if (tier === "growth") return 1;
  return 0;
}

function planPrice(plan: BillingPlanDefinition, cycle: BillingCycle): number {
  return cycle === "monthly" ? plan.monthlyPrice : plan.annualPrice;
}

function planPriceCents(plan: BillingPlanDefinition, cycle: BillingCycle): number {
  return planPrice(plan, cycle) * 100;
}

const featureLabels: Record<string, string> = {
  playbooks: "Automated playbooks",
  ai_insights: "AI-powered insights",
};

function featureFlagsForTier(tier: string): Record<string, boolean> {
  return {
    playbooks: tier === "growth" || tier === "scale",
    ai_insights: tier === "scale",
  };
}

function featureChanges(
  current: Record<string, boolean>,
  target: Record<string, boolean>,
): string[] {
  const changes: string[] = [];
  for (const key of Object.keys(featureLabels)) {
    if (current[key] === target[key]) continue;
    changes.push(`${target[key] ? "Gain" : "Lose"} ${featureLabels[key]}`);
  }
  return changes.length > 0 ? changes : ["No feature access changes"];
}

function isDowngradeChange(
  currentPlan: BillingPlanDefinition | undefined,
  currentTier: string,
  currentCycle: BillingCycle,
  targetPlan: BillingPlanDefinition,
  targetCycle: BillingCycle,
): boolean {
  if (planRank(targetPlan.tier) < planRank(currentTier)) return true;
  if (!currentPlan || targetPlan.tier !== currentTier) return false;
  return planPrice(targetPlan, targetCycle) < planPrice(currentPlan, currentCycle);
}

function estimatedProrationCents(
  currentPlan: BillingPlanDefinition | undefined,
  currentCycle: BillingCycle,
  targetPlan: BillingPlanDefinition,
  targetCycle: BillingCycle,
): number {
  if (!currentPlan) return planPriceCents(targetPlan, targetCycle);
  return Math.max(
    0,
    planPriceCents(targetPlan, targetCycle) -
      planPriceCents(currentPlan, currentCycle),
  );
}

export default function PlanChangeDialog({
  plan,
  cycle,
  currentTier,
  currentCycle,
  onClose,
  onChanged,
}: PlanChangeDialogProps) {
  const [submitting, setSubmitting] = useState(false);
  const [response, setResponse] = useState<PlanChangeResponse | null>(null);
  const toast = useToast();

  const price = cycle === "monthly" ? plan.monthlyPrice : plan.annualPrice;
  const currentPlan = billingPlans.find(
    (candidate) => candidate.tier === currentTier,
  );
  const isDowngrade = isDowngradeChange(
    currentPlan,
    currentTier,
    currentCycle,
    plan,
    cycle,
  );
  const prorationEstimate = estimatedProrationCents(
    currentPlan,
    currentCycle,
    plan,
    cycle,
  );
  const billingImpact = isDowngrade
    ? "No immediate credit. New lower limits apply at renewal."
    : `Estimated proration: ${response ? money(response.proration_cents) : money(prorationEstimate)}`;
  const featureImpact = featureChanges(
    response?.features.current ?? featureFlagsForTier(currentTier),
    response?.features.target ?? featureFlagsForTier(plan.tier),
  );

  async function confirmChange() {
    setSubmitting(true);
    try {
      const { data } = await billingApi.changePlan({ tier: plan.tier, cycle });
      setResponse(data);

      if (data.checkout_url) {
        window.location.href = data.checkout_url;
        return;
      }

      toast.success(successMessage(data.status));
      await onChanged();
    } catch {
      toast.error("Unable to change plan. Please try again.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="plan-change-title"
    >
      <div className="w-full max-w-lg rounded-2xl border border-[var(--galdr-border)] bg-[var(--galdr-surface)] p-5 shadow-2xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--galdr-fg-muted)]">
              Confirm plan change
            </p>
            <h3
              id="plan-change-title"
              className="mt-1 text-xl font-semibold text-[var(--galdr-fg)]"
            >
              Switch to {plan.name}
            </h3>
          </div>
          <button
            onClick={onClose}
            className="rounded-md px-2 py-1 text-sm text-[var(--galdr-fg-muted)] hover:text-[var(--galdr-fg)]"
          >
            Close
          </button>
        </div>

        <div className="mt-5 space-y-3 text-sm text-[var(--galdr-fg-muted)]">
          <div className="rounded-xl border border-[var(--galdr-border)] p-4">
            <p className="font-medium text-[var(--galdr-fg)]">
              ${price}/{cycle === "monthly" ? "mo" : "yr"}
            </p>
            <p className="mt-1">{plan.description}</p>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-xl bg-[color-mix(in_srgb,var(--galdr-surface-soft)_85%,black_15%)] p-3">
              <p className="text-xs uppercase tracking-[0.16em]">
                Billing impact
              </p>
              <p className="mt-2 text-[var(--galdr-fg)]">{billingImpact}</p>
            </div>
            <div className="rounded-xl bg-[color-mix(in_srgb,var(--galdr-surface-soft)_85%,black_15%)] p-3">
              <p className="text-xs uppercase tracking-[0.16em]">Effective</p>
              <p className="mt-2 text-[var(--galdr-fg)]">
                {effectiveLabel(response)}
              </p>
            </div>
          </div>

          <div className="rounded-xl border border-[var(--galdr-border)] p-4">
            <p className="font-medium text-[var(--galdr-fg)]">Limit impact</p>
            <div className="mt-2 grid gap-2 sm:grid-cols-2">
              <p>Customers: {plan.limits.customers}</p>
              <p>Integrations: {plan.limits.integrations}</p>
              {response && (
                <>
                  <p>
                    Customer limit:{" "}
                    {limitLabel(response.limits.current.customer_limit)} -&gt;{" "}
                    {limitLabel(response.limits.target.customer_limit)}
                  </p>
                  <p>
                    Integration limit:{" "}
                    {limitLabel(response.limits.current.integration_limit)}{" "}
                    -&gt; {limitLabel(response.limits.target.integration_limit)}
                  </p>
                </>
              )}
            </div>
          </div>

          <div className="rounded-xl border border-[var(--galdr-border)] p-4">
            <p className="font-medium text-[var(--galdr-fg)]">Feature impact</p>
            <ul className="mt-2 space-y-1">
              {featureImpact.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>

          <p className="text-xs">
            Downgrades never delete data; after renewal, PulseScore blocks new
            customers or integrations beyond the new limit.
          </p>
        </div>

        <div className="mt-6 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="galdr-button-secondary px-4 py-2 text-sm font-medium"
          >
            Keep current plan
          </button>
          <button
            onClick={confirmChange}
            disabled={submitting}
            className="galdr-button-primary inline-flex items-center gap-2 px-4 py-2 text-sm font-semibold disabled:opacity-60"
          >
            {submitting && <Loader2 className="h-4 w-4 animate-spin" />}
            Confirm change
          </button>
        </div>
      </div>
    </div>
  );
}
