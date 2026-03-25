import { useNavigate } from "react-router-dom";
import HealthScoreBadge from "@/components/HealthScoreBadge";
import {
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  ShieldCheck,
  ShieldAlert,
  ShieldX,
} from "lucide-react";
import { formatCurrency, relativeTime } from "@/lib/format";

const riskConfig = {
  green: {
    label: "Healthy",
    Icon: ShieldCheck,
    className:
      "border border-[color:rgb(52_211_153_/_0.35)] bg-[color:rgb(52_211_153_/_0.14)] text-[var(--galdr-success)]",
  },
  yellow: {
    label: "At Risk",
    Icon: ShieldAlert,
    className:
      "border border-[color:rgb(245_158_11_/_0.35)] bg-[color:rgb(245_158_11_/_0.14)] text-[var(--galdr-at-risk)]",
  },
  red: {
    label: "Critical",
    Icon: ShieldX,
    className:
      "border border-[color:rgb(244_63_94_/_0.35)] bg-[color:rgb(244_63_94_/_0.14)] text-[var(--galdr-danger)]",
  },
} as const;

export interface Customer {
  id: string;
  name: string;
  email: string;
  company: string;
  mrr: number;
  health_score: number;
  risk_level: "green" | "yellow" | "red";
  last_seen_at: string;
}

interface CustomerTableProps {
  customers: Customer[];
  sort: string;
  order: "asc" | "desc";
  onSort: (field: string) => void;
}

const columns = [
  { key: "name", label: "Name" },
  { key: "company", label: "Company" },
  { key: "mrr", label: "MRR" },
  { key: "health_score", label: "Health Score" },
  { key: "risk_level", label: "Risk" },
  { key: "last_seen_at", label: "Last Seen" },
];

function SortIcon({
  field,
  sort,
  order,
}: {
  field: string;
  sort: string;
  order: string;
}) {
  if (sort !== field)
    return <ChevronsUpDown className="h-3 w-3 text-[var(--galdr-fg-muted)]" />;
  return order === "asc" ? (
    <ChevronUp className="h-3 w-3" />
  ) : (
    <ChevronDown className="h-3 w-3" />
  );
}

export default function CustomerTable({
  customers,
  sort,
  order,
  onSort,
}: CustomerTableProps) {
  const navigate = useNavigate();

  return (
    <div className="galdr-card overflow-x-auto">
      <table className="w-full text-left text-sm">
        <thead className="border-b border-[var(--galdr-border)] bg-[color:rgb(31_31_46_/_0.72)] text-xs uppercase text-[var(--galdr-fg-muted)]">
          <tr>
            {columns.map((col) => (
              <th
                key={col.key}
                className="cursor-pointer px-6 py-3 transition-colors hover:text-[var(--galdr-fg)]"
                onClick={() => onSort(col.key)}
              >
                <span className="flex items-center gap-1">
                  {col.label}
                  <SortIcon field={col.key} sort={sort} order={order} />
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {customers.map((c) => (
            <tr
              key={c.id}
              onClick={() => navigate(`/customers/${c.id}`)}
              className="cursor-pointer border-b border-[var(--galdr-border)]/70 transition-colors hover:bg-[color:rgb(139_92_246_/_0.08)]"
            >
              <td className="px-6 py-4 font-medium text-[var(--galdr-fg)]">
                {c.name}
                <div className="text-xs text-[var(--galdr-fg-muted)]">
                  {c.email}
                </div>
              </td>
              <td className="px-6 py-4 text-[var(--galdr-fg)]">{c.company}</td>
              <td className="px-6 py-4 text-[var(--galdr-fg)]">
                {formatCurrency(c.mrr)}
              </td>
              <td className="px-6 py-4">
                <HealthScoreBadge
                  score={c.health_score}
                  riskLevel={c.risk_level}
                  size="sm"
                />
              </td>
              <td className="px-6 py-4">
                {(() => {
                  const config = riskConfig[c.risk_level];
                  const Icon = config.Icon;
                  return (
                    <span
                      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${config.className}`}
                    >
                      <Icon className="h-3 w-3" />
                      {config.label}
                    </span>
                  );
                })()}
              </td>
              <td className="px-6 py-4 text-[var(--galdr-fg-muted)]">
                {relativeTime(c.last_seen_at)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
