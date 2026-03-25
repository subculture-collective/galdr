import { useCallback, useEffect, useMemo, useState } from "react";
import { ExternalLink, Loader2 } from "lucide-react";

import { useToast } from "@/contexts/ToastContext";
import { useCheckout } from "@/hooks/useCheckout";
import { billingApi, type BillingSubscriptionResponse } from "@/lib/api";
import {
  billingPlans,
  type BillingCycle,
  type PlanTier,
} from "@/lib/billingPlans";

interface SubscriptionManagerProps {
  checkoutState?: "success" | "cancelled" | null;
}

function normalizeTier(value?: string): PlanTier {
  const normalized = value?.toLowerCase();
  if (normalized === "growth" || normalized === "scale") return normalized;
  return "free";
}

function formatRenewalDate(date: string | null): string {
  if (!date) return "—";
  const parsed = new Date(date);
  if (Number.isNaN(parsed.getTime())) return "—";
  return parsed.toLocaleDateString();
}

function formatLimit(limit: number): string {
  return limit < 0 ? "∞" : String(limit);
}

function usagePercent(used: number, limit: number): number {
  if (limit <= 0) return 0;
  return Math.min(100, Math.round((used / limit) * 100));
}

export default function SubscriptionManager({
  checkoutState,
}: SubscriptionManagerProps) {
  const [subscription, setSubscription] =
    useState<BillingSubscriptionResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [cancelling, setCancelling] = useState(false);
  const [openingPortal, setOpeningPortal] = useState(false);

  const { loading: checkoutLoading, startCheckout } = useCheckout();
  const toast = useToast();

  const currentTier = normalizeTier(subscription?.tier);
  const cycle = (subscription?.billing_cycle ?? "monthly") as BillingCycle;

  const fetchSubscription = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await billingApi.getSubscription();
      setSubscription(data);
    } catch {
      toast.error("Failed to load subscription details");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void fetchSubscription();
  }, [fetchSubscription]);

  useEffect(() => {
    if (checkoutState === "success") {
      toast.success("Checkout complete. Subscription is refreshing...");
      void fetchSubscription();
    }
    if (checkoutState === "cancelled") {
      toast.info("Checkout cancelled. No changes were made.");
    }
  }, [checkoutState, fetchSubscription, toast]);

  const recommendedPlans = useMemo(
    () => billingPlans.filter((plan) => plan.tier !== "free"),
    [],
  );

  async function handleOpenPortal() {
    setOpeningPortal(true);
    try {
      const { data } = await billingApi.createPortalSession();
      window.location.href = data.url;
    } catch {
      toast.error("Unable to open Stripe customer portal right now.");
    } finally {
      setOpeningPortal(false);
    }
  }

  async function handleCancelAtPeriodEnd() {
    if (
      !window.confirm(
        "Cancel this subscription at the end of the current billing period?",
      )
    ) {
      return;
    }

    setCancelling(true);
    try {
      await billingApi.cancelAtPeriodEnd();
      toast.success("Your subscription will cancel at period end.");
      await fetchSubscription();
    } catch {
      toast.error("Failed to schedule cancellation. Please try again.");
    } finally {
      setCancelling(false);
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-10">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--galdr-fg-muted)]" />
      </div>
    );
  }

  if (!subscription) {
    return null;
  }

  return (
    <div className="space-y-6">
      <section className="galdr-panel p-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h3 className="text-lg font-semibold text-[var(--galdr-fg)]">
              {currentTier[0].toUpperCase() + currentTier.slice(1)} plan
            </h3>
            <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
              Status:{" "}
              <span className="font-medium text-[var(--galdr-fg)]">
                {subscription.status}
              </span>
              {" · "}
              Cycle:{" "}
              <span className="font-medium text-[var(--galdr-fg)]">
                {cycle}
              </span>
              {" · "}
              Renewal:{" "}
              <span className="font-medium text-[var(--galdr-fg)]">
                {formatRenewalDate(subscription.renewal_date)}
              </span>
            </p>
            {subscription.cancel_at_period_end && (
              <p className="mt-2 text-xs font-medium text-[var(--galdr-at-risk)]">
                This subscription is scheduled to cancel at period end.
              </p>
            )}
          </div>

          <div className="flex flex-wrap gap-2">
            <button
              onClick={handleOpenPortal}
              disabled={openingPortal}
              className="galdr-button-secondary inline-flex items-center gap-1 px-3 py-2 text-sm font-medium disabled:opacity-60"
            >
              {openingPortal ? "Opening..." : "Open customer portal"}
              <ExternalLink className="h-4 w-4" />
            </button>
            {currentTier !== "free" && !subscription.cancel_at_period_end && (
              <button
                onClick={handleCancelAtPeriodEnd}
                disabled={cancelling}
                className="galdr-button-danger-outline px-3 py-2 text-sm font-medium disabled:opacity-60"
              >
                {cancelling ? "Cancelling..." : "Cancel at period end"}
              </button>
            )}
          </div>
        </div>
      </section>

      <section className="galdr-panel p-5">
        <h4 className="text-sm font-semibold text-[var(--galdr-fg)]">Usage</h4>
        <div className="mt-4 space-y-4">
          {[
            {
              label: "Customers",
              used: subscription.usage.customers.used,
              limit: subscription.usage.customers.limit,
            },
            {
              label: "Integrations",
              used: subscription.usage.integrations.used,
              limit: subscription.usage.integrations.limit,
            },
          ].map((item) => (
            <div key={item.label}>
              <div className="mb-1 flex items-center justify-between text-sm text-[var(--galdr-fg-muted)]">
                <span>{item.label}</span>
                <span>
                  {item.used} / {formatLimit(item.limit)}
                </span>
              </div>
              <div className="h-2 rounded-full bg-[color-mix(in_srgb,var(--galdr-fg-muted)_30%,transparent)]">
                <div
                  className="h-2 rounded-full bg-[var(--galdr-accent)]"
                  style={{ width: `${usagePercent(item.used, item.limit)}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="galdr-panel p-5">
        <h4 className="text-sm font-semibold text-[var(--galdr-fg)]">
          Change plan
        </h4>
        <div className="mt-3 grid gap-3 md:grid-cols-2">
          {recommendedPlans.map((plan) => {
            const isCurrent = plan.tier === currentTier;
            return (
              <div
                key={plan.tier}
                className={`rounded-lg border p-4 ${
                  isCurrent
                    ? "border-[color:rgb(52_211_153_/_0.45)] bg-[color:rgb(52_211_153_/_0.12)]"
                    : "border-[var(--galdr-border)] bg-[color-mix(in_srgb,var(--galdr-surface-soft)_80%,black_20%)]"
                }`}
              >
                <div className="flex items-start justify-between">
                  <div>
                    <p className="font-semibold text-[var(--galdr-fg)]">
                      {plan.name}
                    </p>
                    <p className="text-xs text-[var(--galdr-fg-muted)]">
                      {plan.description}
                    </p>
                  </div>
                  <p className="text-sm font-medium text-[var(--galdr-fg-muted)]">
                    $
                    {cycle === "monthly" ? plan.monthlyPrice : plan.annualPrice}
                    /{cycle === "monthly" ? "mo" : "yr"}
                  </p>
                </div>
                <button
                  disabled={checkoutLoading || isCurrent}
                  onClick={() => startCheckout({ tier: plan.tier, cycle })}
                  className="galdr-button-primary mt-3 w-full px-3 py-2 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {isCurrent
                    ? "Current plan"
                    : checkoutLoading
                      ? "Redirecting..."
                      : `Switch to ${plan.name}`}
                </button>
              </div>
            );
          })}
        </div>
      </section>
    </div>
  );
}
