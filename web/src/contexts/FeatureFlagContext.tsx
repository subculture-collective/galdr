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

interface FeatureFlagContextValue {
  loading: boolean;
  error: string | null;
  subscription: BillingSubscriptionResponse | null;
  hasFeature: (featureName: string) => boolean;
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

  const getLimit = useCallback(
    (limitName: LimitName) => {
      if (!subscription) return null;
      switch (limitName) {
        case LIMIT_CUSTOMER:
          return subscription.usage.customers.limit;
        case LIMIT_INTEGRATION:
          return subscription.usage.integrations.limit;
        case LIMIT_TEAM_MEMBER:
          return subscription.usage.team_members.limit;
      }
    },
    [subscription],
  );

  const value = useMemo<FeatureFlagContextValue>(
    () => ({
      loading,
      error,
      subscription,
      hasFeature,
      getLimit,
      refresh,
    }),
    [loading, error, subscription, hasFeature, getLimit, refresh],
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

export function useFeatureFlag(featureName: string): boolean {
  return useFeatureFlags().hasFeature(featureName);
}

export function useFeatureLimit(limitName: LimitName): number | null {
  return useFeatureFlags().getLimit(limitName);
}
