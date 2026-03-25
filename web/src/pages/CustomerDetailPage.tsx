import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import HealthScoreBadge from "@/components/HealthScoreBadge";
import EventTimeline from "@/components/EventTimeline";
import ScoreHistoryChart from "@/components/charts/ScoreHistoryChart";
import ProfileSkeleton from "@/components/skeletons/ProfileSkeleton";
import { ChevronRight, Mail, Building, DollarSign, Clock } from "lucide-react";
import { formatCurrency, relativeTime } from "@/lib/format";

interface CustomerDetail {
  id: string;
  name: string;
  email: string;
  company: string;
  mrr: number;
  health_score: number;
  risk_level: "green" | "yellow" | "red";
  last_seen_at: string;
  source: string;
  subscriptions?: Subscription[];
  score_factors?: ScoreFactor[];
}

interface Subscription {
  id: string;
  plan_name: string;
  status: string;
  amount: number;
  interval: string;
}

interface ScoreFactor {
  name: string;
  score: number;
  weight: number;
}

type Tab = "overview" | "events" | "subscriptions";

export default function CustomerDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [customer, setCustomer] = useState<CustomerDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [activeTab, setActiveTab] = useState<Tab>("overview");
  const toast = useToast();

  const fetchCustomer = useCallback(async () => {
    try {
      const { data } = await api.get<CustomerDetail>(`/customers/${id}`);
      setCustomer(data);
    } catch (err: unknown) {
      if (err && typeof err === "object" && "response" in err) {
        const resp = err as { response?: { status?: number } };
        if (resp.response?.status === 404) {
          setNotFound(true);
          return;
        }
      }
      toast.error("Failed to load customer details");
    } finally {
      setLoading(false);
    }
  }, [id, toast]);

  useEffect(() => {
    fetchCustomer();
  }, [fetchCustomer]);

  if (loading) {
    return (
      <div className="space-y-6">
        <ProfileSkeleton />
      </div>
    );
  }

  if (notFound || !customer) {
    return (
      <div className="py-12 text-center">
        <h2 className="text-lg font-semibold text-[var(--galdr-fg)]">
          Customer not found
        </h2>
        <Link to="/customers" className="galdr-link mt-4 inline-block text-sm">
          Back to Customers
        </Link>
      </div>
    );
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: "overview", label: "Overview" },
    { key: "events", label: "Events" },
    { key: "subscriptions", label: "Subscriptions" },
  ];

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <nav className="flex items-center gap-1 text-sm text-[var(--galdr-fg-muted)]">
        <Link
          to="/customers"
          className="transition-colors hover:text-[var(--galdr-fg)]"
        >
          Customers
        </Link>
        <ChevronRight className="h-4 w-4" />
        <span className="text-[var(--galdr-fg)]">{customer.name}</span>
      </nav>

      {/* Header */}
      <div className="flex flex-wrap items-start gap-6">
        <HealthScoreBadge
          score={customer.health_score}
          riskLevel={customer.risk_level}
          size="lg"
        />
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-[var(--galdr-fg)]">
            {customer.name}
          </h1>
          <div className="mt-2 flex flex-wrap gap-4 text-sm text-[var(--galdr-fg-muted)]">
            <span className="flex items-center gap-1">
              <Mail className="h-4 w-4" /> {customer.email}
            </span>
            <span className="flex items-center gap-1">
              <Building className="h-4 w-4" /> {customer.company}
            </span>
            <span className="flex items-center gap-1">
              <DollarSign className="h-4 w-4" /> {formatCurrency(customer.mrr)}{" "}
              MRR
            </span>
            <span className="flex items-center gap-1">
              <Clock className="h-4 w-4" />{" "}
              {relativeTime(customer.last_seen_at)}
            </span>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-[var(--galdr-border)]">
        <nav className="flex gap-4">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`border-b-2 px-1 pb-3 text-sm font-medium transition-colors ${
                activeTab === tab.key
                  ? "border-[var(--galdr-accent)] text-[var(--galdr-accent)]"
                  : "border-transparent text-[var(--galdr-fg-muted)] hover:text-[var(--galdr-fg)]"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab content */}
      {activeTab === "overview" && (
        <div className="space-y-6">
          {/* Score factors */}
          {customer.score_factors && customer.score_factors.length > 0 && (
            <div className="galdr-card p-6">
              <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
                Score Factors
              </h3>
              <div className="space-y-3">
                {customer.score_factors.map((factor) => (
                  <div key={factor.name}>
                    <div className="mb-1 flex items-center justify-between text-sm">
                      <span className="text-[var(--galdr-fg-muted)]">
                        {factor.name}
                      </span>
                      <span className="font-medium text-[var(--galdr-fg)]">
                        {factor.score}
                      </span>
                    </div>
                    <div className="h-2 rounded-full bg-[var(--galdr-surface-soft)]">
                      <div
                        className="h-2 rounded-full bg-[var(--chart-series-primary)]"
                        style={{ width: `${factor.score}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Score history chart */}
          <ScoreHistoryChart customerId={customer.id} />
        </div>
      )}

      {activeTab === "events" && <EventTimeline customerId={customer.id} />}

      {activeTab === "subscriptions" && (
        <div className="space-y-4">
          {!customer.subscriptions || customer.subscriptions.length === 0 ? (
            <p className="py-8 text-center text-sm text-[var(--galdr-fg-muted)]">
              No active subscriptions
            </p>
          ) : (
            <div className="galdr-card overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-[var(--galdr-border)] bg-[color:rgb(31_31_46_/_0.72)] text-xs uppercase text-[var(--galdr-fg-muted)]">
                  <tr>
                    <th className="px-6 py-3">Plan</th>
                    <th className="px-6 py-3">Status</th>
                    <th className="px-6 py-3">Amount</th>
                    <th className="px-6 py-3">Interval</th>
                  </tr>
                </thead>
                <tbody>
                  {customer.subscriptions.map((sub) => (
                    <tr
                      key={sub.id}
                      className="border-b border-[var(--galdr-border)]/70"
                    >
                      <td className="px-6 py-4 font-medium text-[var(--galdr-fg)]">
                        {sub.plan_name}
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${
                            sub.status === "active"
                              ? "border border-[color:rgb(52_211_153_/_0.35)] bg-[color:rgb(52_211_153_/_0.14)] text-[var(--galdr-success)]"
                              : "galdr-pill"
                          }`}
                        >
                          {sub.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-[var(--galdr-fg)]">
                        {formatCurrency(sub.amount)}
                      </td>
                      <td className="px-6 py-4 text-[var(--galdr-fg)]">
                        {sub.interval}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
