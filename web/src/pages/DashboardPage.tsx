import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import {
  FEATURE_FULL_DASHBOARD,
  useFeatureFlag,
} from "@/contexts/FeatureFlagContext";
import StatCard from "@/components/StatCard";
import UpgradePrompt from "@/components/UpgradePrompt";
import CardSkeleton from "@/components/skeletons/CardSkeleton";
import { Users, AlertTriangle, DollarSign, Activity } from "lucide-react";
import { formatCurrency } from "@/lib/format";

const ScoreDistributionChart = lazy(
  () => import("@/components/charts/ScoreDistributionChart"),
);
const MRRTrendChart = lazy(() => import("@/components/charts/MRRTrendChart"));
const RiskDistributionChart = lazy(
  () => import("@/components/charts/RiskDistributionChart"),
);
const AtRiskCustomersTable = lazy(
  () => import("@/components/AtRiskCustomersTable"),
);

interface DashboardSummary {
  total_customers: number;
  at_risk_customers: number;
  total_mrr: number;
  average_health_score: number;
}

function ChartPanelFallback() {
  return <CardSkeleton />;
}

function TablePanelFallback() {
  return (
    <div className="galdr-card p-6">
      <div className="h-4 w-40 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_35%,transparent)]" />
      <div className="mt-4 space-y-2">
        {[...Array(4)].map((_, idx) => (
          <div
            key={idx}
            className="h-10 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_25%,transparent)]"
          />
        ))}
      </div>
    </div>
  );
}

export default function DashboardPage() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const toast = useToast();
  const fullDashboard = useFeatureFlag(FEATURE_FULL_DASHBOARD);

  const fetchSummary = useCallback(async () => {
    try {
      const { data } = await api.get<DashboardSummary>("/dashboard/summary");
      setSummary(data);
      setError(false);
    } catch {
      setError(true);
      toast.error("Failed to load dashboard summary");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    fetchSummary();
    const interval = setInterval(fetchSummary, 5 * 60 * 1000);
    return () => clearInterval(interval);
  }, [fetchSummary]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-[var(--galdr-fg)]">Dashboard</h1>

      {/* Stat cards */}
      {loading ? (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <CardSkeleton key={i} />
          ))}
        </div>
      ) : error ? (
        <div role="alert" className="galdr-alert-danger p-6 text-center">
          <p className="text-sm">Failed to load dashboard data.</p>
          <button
            onClick={fetchSummary}
            className="galdr-link mt-2 text-sm font-medium"
          >
            Retry
          </button>
        </div>
      ) : summary ? (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
          <StatCard
            title="Total Customers"
            value={summary.total_customers}
            icon={<Users className="h-5 w-5" />}
          />
          <StatCard
            title="At-Risk Customers"
            value={summary.at_risk_customers}
            icon={<AlertTriangle className="h-5 w-5" />}
          />
          <StatCard
            title="Total MRR"
            value={formatCurrency(summary.total_mrr)}
            icon={<DollarSign className="h-5 w-5" />}
          />
          <StatCard
            title="Avg Health Score"
            value={Math.round(summary.average_health_score)}
            icon={<Activity className="h-5 w-5" />}
          />
        </div>
      ) : null}

      {fullDashboard.allowed ? (
        <>
          {/* Charts */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <Suspense fallback={<ChartPanelFallback />}>
              <ScoreDistributionChart />
            </Suspense>
            <Suspense fallback={<ChartPanelFallback />}>
              <MRRTrendChart />
            </Suspense>
          </div>

          {/* Risk overview */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <Suspense fallback={<ChartPanelFallback />}>
              <RiskDistributionChart />
            </Suspense>
            <div className="lg:col-span-2">
              <Suspense fallback={<TablePanelFallback />}>
                <AtRiskCustomersTable />
              </Suspense>
            </div>
          </div>
        </>
      ) : (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <UpgradePrompt
            featureName="Full dashboard"
            recommendedTier={fullDashboard.recommendedTier}
            description="Unlock score distribution, MRR trend, risk breakdown, and at-risk customer views."
            className="lg:col-span-2"
          />
        </div>
      )}
    </div>
  );
}
