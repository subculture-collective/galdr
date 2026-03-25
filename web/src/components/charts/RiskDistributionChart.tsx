import { useEffect, useState } from "react";
import {
  PieChart,
  Pie,
  Cell,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import ChartSkeleton from "@/components/skeletons/ChartSkeleton";
import EmptyState from "@/components/EmptyState";
import { PieChart as PieChartIcon } from "lucide-react";
import {
  getScoreDistributionCached,
  type ScoreDistributionBucket,
} from "@/lib/scoreDistribution";

interface RiskSegment {
  name: string;
  value: number;
  color: string;
}

function aggregateRisk(buckets: ScoreDistributionBucket[]): RiskSegment[] {
  let healthy = 0;
  let atRisk = 0;
  let critical = 0;

  for (const b of buckets) {
    if (b.min_score >= 70) {
      healthy += b.count;
    } else if (b.min_score >= 40) {
      atRisk += b.count;
    } else {
      critical += b.count;
    }
  }

  return [
    { name: "Healthy", value: healthy, color: "var(--chart-risk-healthy)" },
    { name: "At Risk", value: atRisk, color: "var(--chart-risk-at-risk)" },
    { name: "Critical", value: critical, color: "var(--chart-risk-critical)" },
  ].filter((s) => s.value > 0);
}

export default function RiskDistributionChart() {
  const [segments, setSegments] = useState<RiskSegment[]>([]);
  const [loading, setLoading] = useState(true);
  const [empty, setEmpty] = useState(false);

  useEffect(() => {
    async function fetch() {
      try {
        const buckets = await getScoreDistributionCached();
        if (buckets.length > 0) {
          const agg = aggregateRisk(buckets);
          if (agg.length > 0) {
            setSegments(agg);
          } else {
            setEmpty(true);
          }
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
          icon={<PieChartIcon className="h-12 w-12" />}
          title="No risk data yet"
          description="Risk distribution will appear once customers are scored."
        />
      </div>
    );
  }

  const total = segments.reduce((sum, s) => sum + s.value, 0);

  return (
    <div className="galdr-card p-6">
      <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
        Risk Distribution
      </h3>
      <ResponsiveContainer width="100%" height={280}>
        <PieChart>
          <Pie
            data={segments}
            cx="50%"
            cy="50%"
            innerRadius={60}
            outerRadius={100}
            dataKey="value"
            label={({ name, value }) =>
              `${name}: ${value} (${Math.round((value / total) * 100)}%)`
            }
          >
            {segments.map((s) => (
              <Cell key={s.name} fill={s.color} />
            ))}
          </Pie>
          <Tooltip
            contentStyle={{
              backgroundColor: "var(--chart-tooltip-bg)",
              borderColor: "var(--chart-tooltip-border)",
              borderRadius: 8,
              color: "var(--chart-tooltip-text)",
            }}
            formatter={(value) => [
              `${value} (${Math.round(((value as number) / total) * 100)}%)`,
              "Customers",
            ]}
          />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
