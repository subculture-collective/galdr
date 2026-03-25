import { useEffect, useState } from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Cell,
  LabelList,
} from "recharts";
import ChartSkeleton from "@/components/skeletons/ChartSkeleton";
import EmptyState from "@/components/EmptyState";
import { BarChart3 } from "lucide-react";
import {
  getScoreDistributionCached,
  type ScoreDistributionBucket,
} from "@/lib/scoreDistribution";

function getBarColor(minScore: number): string {
  if (minScore >= 70) return "var(--chart-risk-healthy)";
  if (minScore >= 40) return "var(--chart-risk-at-risk)";
  return "var(--chart-risk-critical)";
}

export default function ScoreDistributionChart() {
  const [data, setData] = useState<ScoreDistributionBucket[]>([]);
  const [loading, setLoading] = useState(true);
  const [empty, setEmpty] = useState(false);

  useEffect(() => {
    async function fetch() {
      try {
        const buckets = await getScoreDistributionCached();
        if (buckets.length > 0) {
          setData(buckets);
          setEmpty(false);
        } else {
          setEmpty(true);
        }
      } catch {
        setEmpty(true);
      } finally {
        setLoading(false);
      }
    }
    fetch();
  }, []);

  if (loading) return <ChartSkeleton />;

  if (empty) {
    return (
      <div className="galdr-card p-6">
        <EmptyState
          icon={<BarChart3 className="h-12 w-12" />}
          title="No score data yet"
          description="Health scores will appear here once customers are scored."
        />
      </div>
    );
  }

  const total = data.reduce((sum, d) => sum + d.count, 0);

  return (
    <div className="galdr-card p-6">
      <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
        Health Score Distribution
      </h3>
      <ResponsiveContainer width="100%" height={280}>
        <BarChart data={data}>
          <XAxis
            dataKey="range"
            tick={{ fontSize: 12 }}
            stroke="var(--chart-axis-stroke)"
          />
          <YAxis
            tick={{ fontSize: 12 }}
            stroke="var(--chart-axis-stroke)"
            allowDecimals={false}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: "var(--chart-tooltip-bg)",
              borderColor: "var(--chart-tooltip-border)",
              borderRadius: 8,
              color: "var(--chart-tooltip-text)",
            }}
            formatter={(value) => [value, "Customers"]}
          />
          <Bar dataKey="count" radius={[4, 4, 0, 0]}>
            {data.map((entry, index) => (
              <Cell
                key={entry.range}
                fill={getBarColor(entry.min_score ?? index * 10)}
              />
            ))}
            <LabelList
              dataKey="count"
              position="top"
              formatter={(value: unknown) => {
                const v = value as number;
                return total > 0
                  ? `${v} (${Math.round((v / total) * 100)}%)`
                  : `${v}`;
              }}
              style={{ fontSize: 11, fill: "var(--chart-tooltip-text)" }}
            />
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
