import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { billingApi, type BillingSubscriptionResponse } from "@/lib/api";
import { useAuth } from "@/contexts/AuthContext";

export const FEATURE_BASIC_DASHBOARD = "basic_dashboard";
export const FEATURE_FULL_DASHBOARD = "full_dashboard";
export const FEATURE_EMAIL_ALERTS = "email_alerts";
export const FEATURE_PLAYBOOKS = "playbooks";
export const FEATURE_AI_INSIGHTS = "ai_insights";
export const FEATURE_BENCHMARKS = "benchmarks";

export const LIMIT_CUSTOMER = "customer_limit";
export const LIMIT_INTEGRATION = "integration_limit";
export const LIMIT_TEAM_MEMBER = "team_member_limit";

type LimitName =
  | typeof LIMIT_CUSTOMER
  | typeof LIMIT_INTEGRATION
  | typeof LIMIT_TEAM_MEMBER;

interface LimitUsage {
  current: number | null;
  limit: number | null;
}

export interface FeatureFlagDecision {
  allowed: boolean;
  limit: number | null;
  current: number | null;
  recommendedTier: string | null;
  currentPlan: string | null;
}

interface FeatureFlagContextValue {
  loading: boolean;
  error: string | null;
  subscription: BillingSubscriptionResponse | null;
  hasFeature: (featureName: string) => boolean;
  getFeatureDecision: (featureName: string) => FeatureFlagDecision;
  getLimit: (limitName: LimitName) => number | null;
  refresh: () => Promise<void>;
}

const FeatureFlagContext = createContext<FeatureFlagContextValue | null>(null);

export function FeatureFlagProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, organization } = useAuth();
  const [subscription, setSubscription] =
    useState<BillingSubscriptionResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!isAuthenticated || !organization) {
      setSubscription(null);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const { data } = await billingApi.getSubscription();
      setSubscription(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load features");
      setSubscription(null);
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated, organization]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const hasFeature = useCallback(
    (featureName: string) => Boolean(subscription?.features[featureName]),
    [subscription],
  );

  const getLimitUsage = useCallback(
    (limitName: LimitName): LimitUsage => {
      if (!subscription) return { current: null, limit: null };
      switch (limitName) {
        case LIMIT_CUSTOMER:
          return usageMetricToLimitUsage(subscription.usage.customers);
        case LIMIT_INTEGRATION:
          return usageMetricToLimitUsage(subscription.usage.integrations);
        case LIMIT_TEAM_MEMBER:
          return usageMetricToLimitUsage(subscription.usage.team_members);
      }
    },
    [subscription],
  );

  const getLimit = useCallback(
    (limitName: LimitName) => {
      return getLimitUsage(limitName).limit;
    },
    [getLimitUsage],
  );

  const getFeatureDecision = useCallback(
    (featureName: string): FeatureFlagDecision => {
      if (!subscription) {
        return {
          allowed: false,
          limit: null,
          current: null,
          recommendedTier: "growth",
          currentPlan: null,
        };
      }

      if (isLimitName(featureName)) {
        const usage = getLimitUsage(featureName);
        const allowed =
          usage.limit === null ||
          usage.current === null ||
          usage.limit === -1 ||
          usage.current < usage.limit;
        return {
          allowed,
          limit: usage.limit,
          current: usage.current,
          recommendedTier: allowed ? null : nextTier(subscription.tier),
          currentPlan: subscription.tier,
        };
      }

      const allowed = Boolean(subscription.features[featureName]);
      return {
        allowed,
        limit: null,
        current: null,
        recommendedTier: allowed
          ? null
          : tierForFeature(featureName, subscription.tier),
        currentPlan: subscription.tier,
      };
    },
    [getLimitUsage, subscription],
  );

  const value = useMemo<FeatureFlagContextValue>(
    () => ({
      loading,
      error,
      subscription,
      hasFeature,
      getFeatureDecision,
      getLimit,
      refresh,
    }),
    [loading, error, subscription, hasFeature, getFeatureDecision, getLimit, refresh],
  );

  return (
    <FeatureFlagContext.Provider value={value}>
      {children}
    </FeatureFlagContext.Provider>
  );
}

export function useFeatureFlags(): FeatureFlagContextValue {
  const ctx = useContext(FeatureFlagContext);
  if (!ctx) {
    throw new Error("useFeatureFlags must be used within FeatureFlagProvider");
  }
  return ctx;
}

export function useFeatureFlag(featureName: string): FeatureFlagDecision {
  return useFeatureFlags().getFeatureDecision(featureName);
}

export function useFeatureLimit(limitName: LimitName): number | null {
  return useFeatureFlags().getLimit(limitName);
}

function isLimitName(value: string): value is LimitName {
  return (
    value === LIMIT_CUSTOMER ||
    value === LIMIT_INTEGRATION ||
    value === LIMIT_TEAM_MEMBER
  );
}

function nextTier(tier: string) {
  if (tier === "growth") return "scale";
  return "growth";
}

function tierForFeature(featureName: string, currentTier: string) {
  if (
    featureName === FEATURE_AI_INSIGHTS ||
    featureName === FEATURE_BENCHMARKS
  ) {
    return "scale";
  }
  return nextTier(currentTier);
}

function usageMetricToLimitUsage(metric: {
  used: number;
  limit: number;
}): LimitUsage {
  return { current: metric.used, limit: metric.limit };
}
