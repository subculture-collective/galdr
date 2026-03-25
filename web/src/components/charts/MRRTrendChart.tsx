import { useEffect, useState } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import api from "@/lib/api";
import ChartSkeleton from "@/components/skeletons/ChartSkeleton";
import EmptyState from "@/components/EmptyState";
import { TrendingUp } from "lucide-react";
import { formatCurrency } from "@/lib/format";

interface MRRPoint {
  date: string;
  mrr: number;
}

const ranges = ["7d", "30d", "90d", "1y"] as const;
type Range = (typeof ranges)[number];

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

export default function MRRTrendChart() {
  const [data, setData] = useState<MRRPoint[]>([]);
  const [range, setRange] = useState<Range>("30d");
  const [loading, setLoading] = useState(true);
  const [empty, setEmpty] = useState(false);

  useEffect(() => {
    async function fetch() {
      setLoading(true);
      try {
        const { data: res } = await api.get("/dashboard/mrr-trend", {
          params: { range },
        });
        const points = res.data ?? res.points ?? res ?? [];
        if (Array.isArray(points) && points.length > 0) {
          setData(points);
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
  }, [range]);

  if (loading) return <ChartSkeleton />;

  if (empty) {
    return (
      <div className="galdr-card p-6">
        <EmptyState
          icon={<TrendingUp className="h-12 w-12" />}
          title="No MRR data yet"
          description="MRR trends will appear once subscription data is synced."
        />
      </div>
    );
  }

  return (
    <div className="galdr-card p-6">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-medium text-[var(--galdr-fg)]">
            MRR Trend
          </h3>
          {data.length >= 2 &&
            (() => {
              const netChange = data[data.length - 1].mrr - data[0].mrr;
              const sign = netChange >= 0 ? "+" : "";
              return (
                <p className="text-xs text-[var(--galdr-fg-muted)]">
                  {sign}
                  {formatCurrency(netChange)} over period
                </p>
              );
            })()}
        </div>
        <div className="flex gap-1">
          {ranges.map((r) => (
            <button
              key={r}
              onClick={() => setRange(r)}
              aria-pressed={range === r}
              className={`rounded-md px-2 py-1 text-xs font-medium transition-colors ${
                range === r
                  ? "border border-[color:rgb(139_92_246_/_0.4)] bg-[color:rgb(139_92_246_/_0.2)] text-[var(--galdr-fg)]"
                  : "text-[var(--galdr-fg-muted)] hover:bg-[color:rgb(139_92_246_/_0.1)]"
              }`}
            >
              {r}
            </button>
          ))}
        </div>
      </div>
      <ResponsiveContainer width="100%" height={280}>
        <AreaChart data={data}>
          <defs>
            <linearGradient id="mrrGradient" x1="0" y1="0" x2="0" y2="1">
              <stop
                offset="0%"
                stopColor="var(--chart-series-primary)"
                stopOpacity={0.3}
              />
              <stop
                offset="100%"
                stopColor="var(--chart-series-primary)"
                stopOpacity={0}
              />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="date"
            tickFormatter={formatDate}
            tick={{ fontSize: 12 }}
            stroke="var(--chart-axis-stroke)"
          />
          <YAxis
            tickFormatter={(v) => formatCurrency(v)}
            tick={{ fontSize: 12 }}
            stroke="var(--chart-axis-stroke)"
          />
          <Tooltip
            formatter={(value) => [formatCurrency(value as number), "MRR"]}
            labelFormatter={(label) => formatDate(String(label))}
            contentStyle={{
              backgroundColor: "var(--chart-tooltip-bg)",
              borderColor: "var(--chart-tooltip-border)",
              borderRadius: 8,
              color: "var(--chart-tooltip-text)",
            }}
          />
          <Area
            type="monotone"
            dataKey="mrr"
            stroke="var(--chart-series-primary)"
            strokeWidth={2}
            fill="url(#mrrGradient)"
            activeDot={{ r: 6, strokeWidth: 2 }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
