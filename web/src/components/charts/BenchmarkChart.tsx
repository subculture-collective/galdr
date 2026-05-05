import React from "react";

void React;

export interface BenchmarkPercentiles {
  p25: number;
  p50: number;
  p75: number;
}

export interface BenchmarkMetric {
  key: string;
  label: string;
  unit: "score" | "currency" | "percent" | "count";
  yourValue: number | null;
  percentile: number | null;
  benchmarks: BenchmarkPercentiles | null;
  sampleCount: number;
}

interface BenchmarkChartProps {
  metric: BenchmarkMetric;
}

function formatMetricValue(value: number | null, unit: BenchmarkMetric["unit"]) {
  if (value === null) return "N/A";
  if (unit === "currency") {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
      maximumFractionDigits: 0,
    }).format(value / 100);
  }
  if (unit === "percent") return `${Math.round(value * 100)}%`;
  if (unit === "score") return Math.round(value).toString();
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 1 }).format(
    value,
  );
}

function percentileLabel(percentile: number | null) {
  if (percentile === null) return "Percentile unavailable";
  return `${Math.round(percentile)}th percentile`;
}

function benchmarkRows(metric: BenchmarkMetric) {
  const rows = [
    { label: "Your value", value: metric.yourValue, tone: "org" },
    { label: "P25", value: metric.benchmarks?.p25 ?? null, tone: "peer" },
    { label: "P50", value: metric.benchmarks?.p50 ?? null, tone: "peer" },
    { label: "P75", value: metric.benchmarks?.p75 ?? null, tone: "peer" },
  ];
  const max = Math.max(...rows.map((row) => row.value ?? 0), 1);
  return rows.map((row) => ({
    ...row,
    width: row.value === null ? 0 : Math.max((row.value / max) * 100, 4),
  }));
}

export default function BenchmarkChart({ metric }: BenchmarkChartProps) {
  if (!metric.benchmarks || metric.sampleCount === 0) {
    return (
      <section className="galdr-card p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              {metric.label}
            </h3>
            <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
              Benchmark data unavailable
            </p>
          </div>
          <span className="rounded-full border border-[var(--galdr-border)] px-3 py-1 text-xs text-[var(--galdr-fg-muted)]">
            n=0
          </span>
        </div>
      </section>
    );
  }

  const rows = benchmarkRows(metric);

  return (
    <section className="galdr-card overflow-hidden p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
            {metric.label}
          </h3>
          <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
            Your value vs anonymized peer percentiles
          </p>
        </div>
        <div className="text-right">
          <p className="text-sm font-semibold text-[var(--galdr-fg)]">
            {percentileLabel(metric.percentile)}
          </p>
          <p className="text-xs text-[var(--galdr-fg-muted)]">
            n={metric.sampleCount}
          </p>
        </div>
      </div>

      <div className="mt-5 space-y-3">
        {rows.map((row) => (
          <div key={row.label}>
            <div className="mb-1 flex items-center justify-between text-xs">
              <span className="font-medium text-[var(--galdr-fg-muted)]">
                {row.label}
              </span>
              <span className="font-semibold text-[var(--galdr-fg)]">
                {formatMetricValue(row.value, metric.unit)}
              </span>
            </div>
            <div className="h-3 overflow-hidden rounded-full bg-[color-mix(in_srgb,var(--galdr-fg-muted)_14%,transparent)]">
              <div
                className={
                  row.tone === "org"
                    ? "h-full rounded-full bg-[var(--chart-series-primary)]"
                    : "h-full rounded-full bg-[color-mix(in_srgb,var(--chart-series-primary)_45%,var(--galdr-fg-muted))]"
                }
                style={{ width: `${row.width}%` }}
              />
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
