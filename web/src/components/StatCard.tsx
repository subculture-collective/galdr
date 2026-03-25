import { TrendingUp, TrendingDown, Minus } from "lucide-react";
import type { ReactNode } from "react";

interface StatCardProps {
  title: string;
  value: string | number;
  trend?: number;
  trendDirection?: "up" | "down" | "neutral";
  icon?: ReactNode;
}

const trendLabels = {
  up: "increased",
  down: "decreased",
  neutral: "unchanged",
};

export default function StatCard({
  title,
  value,
  trend,
  trendDirection = "neutral",
  icon,
}: StatCardProps) {
  const trendColor =
    trendDirection === "up"
      ? "text-[var(--galdr-success)]"
      : trendDirection === "down"
        ? "text-[var(--galdr-danger)]"
        : "text-[var(--galdr-fg-muted)]";

  const TrendIcon =
    trendDirection === "up"
      ? TrendingUp
      : trendDirection === "down"
        ? TrendingDown
        : Minus;

  return (
    <div className="galdr-card p-6">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-[var(--galdr-fg-muted)]">
          {title}
        </p>
        {icon && (
          <span className="text-[var(--galdr-fg-muted)]" aria-hidden="true">
            {icon}
          </span>
        )}
      </div>
      <p className="mt-2 text-3xl font-bold text-[var(--galdr-fg)]">{value}</p>
      {trend !== undefined && (
        <div className={`mt-2 flex items-center gap-1 text-sm ${trendColor}`}>
          <TrendIcon className="h-4 w-4" aria-hidden="true" />
          <span>{Math.abs(trend)}%</span>
          <span className="text-[var(--galdr-fg-muted)]">vs last period</span>
          <span className="sr-only">{trendLabels[trendDirection]}</span>
        </div>
      )}
    </div>
  );
}
