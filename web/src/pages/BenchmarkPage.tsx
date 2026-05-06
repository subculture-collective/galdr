import { useCallback, useEffect, useState } from "react";
import { BarChart3, LockKeyhole, RefreshCw } from "lucide-react";
import BenchmarkChart, {
  type BenchmarkMetric,
} from "@/components/charts/BenchmarkChart";
import EmptyState from "@/components/EmptyState";
import ChartSkeleton from "@/components/skeletons/ChartSkeleton";
import api, {
  benchmarksApi,
  type BenchmarkMetricResponse,
  type BenchmarksResponse,
} from "@/lib/api";
import { ORGANIZATION_INDUSTRIES } from "@/lib/industries";

const COMPANY_SIZES = ["1-10", "11-50", "51-200", "201-1000", "1000+"];

const METRIC_LABELS: Record<string, { label: string; unit: BenchmarkMetric["unit"] }> = {
  health_score: { label: "Avg health score", unit: "score" },
  mrr_per_customer: { label: "MRR/customer", unit: "currency" },
  churn_rate: { label: "Churn rate", unit: "percent" },
  integration_usage: { label: "Integration count", unit: "count" },
};

interface OrganizationBenchmarkSettings {
  industry?: string;
  company_size?: number | string;
  benchmarking_enabled?: boolean;
}

interface BenchmarkPageViewProps {
  industry: string;
  size: string;
  participating: boolean;
  loading: boolean;
  error: boolean;
  metrics: BenchmarkMetric[];
  calloutPercentile: number | null;
  onIndustryChange: (industry: string) => void;
  onSizeChange: (size: string) => void;
  onRetry: () => void;
}

function defaultSize(size?: number | string) {
  if (typeof size === "string" && COMPANY_SIZES.includes(size)) {
    return size;
  }
  if (typeof size !== "number") {
    return "51-200";
  }
  if (size <= 10) {
    return "1-10";
  }
  if (size <= 50) {
    return "11-50";
  }
  if (size <= 200) {
    return "51-200";
  }
  if (size <= 1000) {
    return "201-1000";
  }
  return "1000+";
}

function normalizeMetric(metric: BenchmarkMetricResponse): BenchmarkMetric {
  const fallback = METRIC_LABELS[metric.key] ?? {
    label: metric.label,
    unit: metric.unit,
  };
  return {
    key: metric.key,
    label: metric.label || fallback.label,
    unit: metric.unit || fallback.unit,
    yourValue: metric.your_value,
    percentile: metric.percentile,
    benchmarks: metric.benchmarks,
    sampleCount: metric.sample_count,
  };
}

function averageMetricPercentile(metrics: BenchmarkMetric[]) {
  const percentiles = metrics
    .map((metric) => metric.percentile)
    .filter((value): value is number => value !== null);
  if (percentiles.length === 0) {
    return null;
  }
  return Math.round(
    percentiles.reduce((sum, value) => sum + value, 0) / percentiles.length,
  );
}

export function BenchmarkPageView({
  industry,
  size,
  participating,
  loading,
  error,
  metrics,
  calloutPercentile,
  onIndustryChange,
  onSizeChange,
  onRetry,
}: BenchmarkPageViewProps) {
  let metricContent;
  if (loading) {
    metricContent = (
      <div className="grid gap-4 lg:grid-cols-2">
        {[...Array(4)].map((_, index) => (
          <ChartSkeleton key={index} />
        ))}
      </div>
    );
  } else if (error) {
    metricContent = (
      <div role="alert" className="galdr-alert-danger p-6 text-center">
        <p className="text-sm">Failed to load benchmark data.</p>
        <button
          onClick={onRetry}
          className="galdr-link mt-2 inline-flex items-center gap-2 text-sm font-medium"
        >
          <RefreshCw className="h-4 w-4" /> Retry
        </button>
      </div>
    );
  } else if (metrics.length === 0) {
    metricContent = (
      <div className="galdr-card p-6">
        <EmptyState
          icon={<BarChart3 className="h-12 w-12" />}
          title="No benchmark data yet"
          description="Choose another peer segment or wait for enough opted-in organizations to produce a safe aggregate."
        />
      </div>
    );
  } else {
    metricContent = (
      <div className="grid gap-4 lg:grid-cols-2">
        {metrics.map((metric) => (
          <BenchmarkChart key={metric.key} metric={metric} />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="galdr-card overflow-hidden p-6">
        <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.28em] text-[var(--galdr-fg-muted)]">
              Anonymized Benchmarking
            </p>
            <h1 className="mt-2 text-2xl font-bold text-[var(--galdr-fg)]">
              Benchmark comparison
            </h1>
            <p className="mt-2 max-w-2xl text-sm text-[var(--galdr-fg-muted)]">
              Compare your customer health against opt-in peer aggregates by
              industry and company size. No customer-level data leaves its org.
            </p>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <label className="text-xs font-medium text-[var(--galdr-fg-muted)]">
              Industry
              <select
                value={industry}
                onChange={(event) => onIndustryChange(event.target.value)}
                className="galdr-input mt-1 w-full px-3 py-2 text-sm"
              >
                {ORGANIZATION_INDUSTRIES.map((option) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
              </select>
            </label>
            <label className="text-xs font-medium text-[var(--galdr-fg-muted)]">
              Company size
              <select
                value={size}
                onChange={(event) => onSizeChange(event.target.value)}
                className="galdr-input mt-1 w-full px-3 py-2 text-sm"
              >
                {COMPANY_SIZES.map((option) => (
                  <option key={option} value={option}>
                    {option} customers
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>
      </div>

      {!participating && (
        <div className="galdr-alert-warning flex flex-col gap-3 p-5 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex gap-3">
            <LockKeyhole className="mt-0.5 h-5 w-5 shrink-0" />
            <div>
              <h2 className="text-sm font-semibold">Opt in to benchmarking</h2>
              <p className="mt-1 text-sm">
                Enable anonymized benchmarking in organization settings to unlock
                peer comparisons and contribute privacy-safe aggregates.
              </p>
            </div>
          </div>
          <a
            href="/settings/organization"
            className="galdr-button-primary px-4 py-2 text-sm font-medium"
          >
            Review settings
          </a>
        </div>
      )}

      {calloutPercentile !== null && (
        <section className="galdr-card p-6">
          <p className="text-sm text-[var(--galdr-fg-muted)]">Peer position</p>
          <p className="mt-2 text-3xl font-bold text-[var(--galdr-fg)]">
            You are at the {calloutPercentile}th percentile
          </p>
        </section>
      )}

      {metricContent}
    </div>
  );
}

export default function BenchmarkPage() {
  const [industry, setIndustry] = useState("SaaS");
  const [size, setSize] = useState("51-200");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [participating, setParticipating] = useState(true);
  const [data, setData] = useState<BenchmarksResponse | null>(null);

  useEffect(() => {
    async function fetchDefaults() {
      try {
        const { data: org } = await api.get<OrganizationBenchmarkSettings>(
          "/organizations/current",
        );
        if (org.industry) setIndustry(org.industry);
        setSize(defaultSize(org.company_size));
        setParticipating(org.benchmarking_enabled !== false);
      } catch {
        setParticipating(true);
      }
    }
    void fetchDefaults();
  }, []);

  const fetchBenchmarks = useCallback(async () => {
    setLoading(true);
    setError(false);
    try {
      const { data: response } = await benchmarksApi.compare({ industry, size });
      setData(response);
      setParticipating(response.participating);
    } catch {
      setError(true);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [industry, size]);

  useEffect(() => {
    void fetchBenchmarks();
  }, [fetchBenchmarks]);

  const metrics = data?.metrics.map(normalizeMetric) ?? [];
  const calloutPercentile = data?.percentile ?? averageMetricPercentile(metrics);

  return (
    <BenchmarkPageView
      industry={industry}
      size={size}
      participating={participating}
      loading={loading}
      error={error}
      metrics={metrics}
      calloutPercentile={calloutPercentile}
      onIndustryChange={setIndustry}
      onSizeChange={setSize}
      onRetry={fetchBenchmarks}
    />
  );
}
