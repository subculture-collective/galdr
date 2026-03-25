import { useEffect, useState } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ReferenceArea,
  ResponsiveContainer,
} from "recharts";
import api from "@/lib/api";
import ChartSkeleton from "@/components/skeletons/ChartSkeleton";
import EmptyState from "@/components/EmptyState";
import { TrendingUp } from "lucide-react";

interface ScorePoint {
  date: string;
  score: number;
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

export default function ScoreHistoryChart({
  customerId,
}: {
  customerId: string;
}) {
  const [data, setData] = useState<ScorePoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [empty, setEmpty] = useState(false);

  useEffect(() => {
    async function fetch() {
      try {
        const { data: res } = await api.get(
          `/customers/${customerId}/score-history`,
        );
        const points = res.history ?? res.data ?? res ?? [];
        if (Array.isArray(points) && points.length > 0) {
          setData(points);
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
  }, [customerId]);

  if (loading) return <ChartSkeleton />;

  if (empty) {
    return (
      <div className="galdr-card p-6">
        <EmptyState
          icon={<TrendingUp className="h-12 w-12" />}
          title="No score history"
          description="Score history will appear as health scores are calculated over time."
        />
      </div>
    );
  }

  return (
    <div className="galdr-card p-6">
      <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
        Health Score History
      </h3>
      <ResponsiveContainer width="100%" height={280}>
        <LineChart data={data}>
          <ReferenceArea
            y1={70}
            y2={100}
            fill="var(--chart-risk-healthy)"
            fillOpacity={0.08}
          />
          <ReferenceArea
            y1={40}
            y2={70}
            fill="var(--chart-risk-at-risk)"
            fillOpacity={0.08}
          />
          <ReferenceArea
            y1={0}
            y2={40}
            fill="var(--chart-risk-critical)"
            fillOpacity={0.08}
          />
          <XAxis
            dataKey="date"
            tickFormatter={formatDate}
            tick={{ fontSize: 12 }}
            stroke="var(--chart-axis-stroke)"
          />
          <YAxis
            domain={[0, 100]}
            tick={{ fontSize: 12 }}
            stroke="var(--chart-axis-stroke)"
          />
          <Tooltip
            formatter={(value) => [value, "Score"]}
            labelFormatter={(label) => formatDate(String(label))}
            contentStyle={{
              backgroundColor: "var(--chart-tooltip-bg)",
              borderColor: "var(--chart-tooltip-border)",
              borderRadius: 8,
              color: "var(--chart-tooltip-text)",
            }}
          />
          <Line
            type="monotone"
            dataKey="score"
            stroke="var(--chart-series-primary)"
            strokeWidth={2}
            dot={{ r: data.length <= 10 ? 4 : 2 }}
            activeDot={{ r: 6 }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
