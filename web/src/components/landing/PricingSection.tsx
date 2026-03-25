import { useEffect, useMemo, useState } from "react";
import { Check, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";

import { useAuth } from "@/contexts/AuthContext";
import { useCheckout } from "@/hooks/useCheckout";
import { billingApi } from "@/lib/api";
import {
  billingPlans,
  savingsBadge,
  type BillingCycle,
  type PlanTier,
} from "@/lib/billingPlans";

interface PricingSectionProps {
  showStandaloneHeader?: boolean;
}

function normalizeTier(value?: string | null): PlanTier | null {
  if (!value) return null;
  const tier = value.trim().toLowerCase();
  if (tier === "free" || tier === "growth" || tier === "scale") {
    return tier;
  }
  return null;
}

export default function PricingSection({
  showStandaloneHeader = false,
}: PricingSectionProps) {
  const [cycle, setCycle] = useState<BillingCycle>("monthly");
  const { isAuthenticated, organization } = useAuth();
  const { loading, startCheckout } = useCheckout();
  const [currentTier, setCurrentTier] = useState<PlanTier | null>(
    normalizeTier(organization?.plan),
  );

  useEffect(() => {
    let cancelled = false;

    async function loadSubscription() {
      if (!isAuthenticated) {
        setCurrentTier(null);
        return;
      }

      setCurrentTier(normalizeTier(organization?.plan));

      try {
        const { data } = await billingApi.getSubscription();
        if (!cancelled) {
          setCurrentTier(normalizeTier(data.tier));
        }
      } catch {
        // Keep fallback from auth payload if subscription endpoint is unavailable.
      }
    }

    void loadSubscription();
    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, organization?.plan]);

  const annualSavingsText = useMemo(() => {
    const growthPlan =
      billingPlans.find((plan) => plan.tier === "growth") ?? billingPlans[0];
    return savingsBadge(growthPlan);
  }, []);

  return (
    <section id="pricing" className="px-6 py-16 sm:px-10 lg:px-14 lg:py-24">
      <div className="mx-auto max-w-7xl">
        {showStandaloneHeader && (
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-[var(--galdr-fg)] sm:text-4xl">
              Choose the right PulseScore plan
            </h1>
            <p className="mt-2 text-[var(--galdr-fg-muted)]">
              Start on Free and upgrade when your customer health workflow
              scales.
            </p>
          </div>
        )}

        <div className="flex flex-col items-start justify-between gap-6 md:flex-row md:items-end">
          <div className="max-w-3xl">
            <h2 className="text-3xl font-bold tracking-tight text-[var(--galdr-fg)] sm:text-4xl">
              Pricing that fits the gap between spreadsheets and $10K tools.
            </h2>
            <p className="mt-3 text-[var(--galdr-fg-muted)]">
              Transparent tiers. No setup fees. Upgrade when your customer base
              grows.
            </p>
          </div>

          <div className="galdr-panel inline-flex items-center p-1">
            <button
              onClick={() => setCycle("monthly")}
              className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${
                cycle === "monthly"
                  ? "galdr-button-secondary text-[var(--galdr-fg)]"
                  : "text-[var(--galdr-fg-muted)] hover:text-[var(--galdr-fg)]"
              }`}
            >
              Monthly
            </button>
            <button
              onClick={() => setCycle("annual")}
              className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${
                cycle === "annual"
                  ? "galdr-button-secondary text-[var(--galdr-fg)]"
                  : "text-[var(--galdr-fg-muted)] hover:text-[var(--galdr-fg)]"
              }`}
            >
              Annual
            </button>
            {cycle === "annual" && (
              <span className="galdr-alert-success ml-2 inline-flex items-center rounded-full px-2 py-1 text-xs font-semibold">
                <Sparkles className="mr-1 h-3 w-3" />
                {annualSavingsText}
              </span>
            )}
          </div>
        </div>

        <div className="mt-10 grid grid-cols-1 gap-4 lg:grid-cols-3">
          {billingPlans.map((plan) => {
            const isFree = plan.monthlyPrice === 0;
            const displayPrice =
              cycle === "monthly" ? plan.monthlyPrice : plan.annualPrice;
            const period = cycle === "monthly" ? "/mo" : "/yr";
            const isCurrentPlan = currentTier === plan.tier;

            const ctaLabel = isCurrentPlan
              ? "Current plan"
              : isFree
                ? isAuthenticated
                  ? "Stay on Free"
                  : "Get Started Free"
                : isAuthenticated
                  ? `Choose ${plan.name}`
                  : `Start ${plan.name}`;

            return (
              <article
                key={plan.tier}
                className={`relative rounded-2xl p-6 ${
                  isCurrentPlan
                    ? "galdr-card border-[color:rgb(52_211_153_/_0.52)]"
                    : plan.featured
                      ? "galdr-card border-[color-mix(in_srgb,var(--galdr-accent)_55%,var(--galdr-border)_45%)]"
                      : "galdr-panel"
                }`}
              >
                {isCurrentPlan && (
                  <span className="galdr-alert-success absolute -top-3 left-4 rounded-full px-3 py-1 text-xs font-semibold">
                    Current plan
                  </span>
                )}
                {plan.featured && (
                  <span className="absolute -top-3 right-4 rounded-full bg-[var(--galdr-accent)] px-3 py-1 text-xs font-semibold text-white">
                    Most popular
                  </span>
                )}

                <h3 className="text-xl font-bold text-[var(--galdr-fg)]">
                  {plan.name}
                </h3>
                <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
                  {plan.description}
                </p>

                <div className="mt-5 flex items-baseline gap-1">
                  <span className="text-4xl font-extrabold tracking-tight text-[var(--galdr-fg)]">
                    ${displayPrice}
                  </span>
                  <span className="text-sm text-[var(--galdr-fg-muted)]">
                    {period}
                  </span>
                </div>
                {isFree && (
                  <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
                    No credit card required
                  </p>
                )}

                {isAuthenticated && !isFree && !isCurrentPlan ? (
                  <button
                    onClick={() => startCheckout({ tier: plan.tier, cycle })}
                    disabled={loading}
                    className={`mt-5 inline-flex w-full items-center justify-center rounded-xl px-4 py-2.5 text-sm font-semibold transition disabled:cursor-not-allowed disabled:opacity-60 ${
                      plan.featured
                        ? "galdr-button-primary"
                        : "galdr-button-secondary"
                    }`}
                  >
                    {loading ? "Redirecting..." : ctaLabel}
                  </button>
                ) : (
                  <Link
                    to={
                      isAuthenticated
                        ? "/dashboard"
                        : `/register?plan=${plan.tier}`
                    }
                    className={`mt-5 inline-flex w-full items-center justify-center rounded-xl px-4 py-2.5 text-sm font-semibold transition ${
                      plan.featured
                        ? "galdr-button-primary"
                        : "galdr-button-secondary"
                    }`}
                  >
                    {ctaLabel}
                  </Link>
                )}

                <ul className="mt-5 space-y-2 text-sm text-[var(--galdr-fg-muted)]">
                  {Object.values(plan.limits).map((item) => (
                    <li key={item} className="flex items-start gap-2">
                      <Check className="mt-0.5 h-4 w-4 shrink-0 text-[var(--galdr-accent)]" />
                      {item}
                    </li>
                  ))}
                </ul>
              </article>
            );
          })}
        </div>
      </div>
    </section>
  );
}
