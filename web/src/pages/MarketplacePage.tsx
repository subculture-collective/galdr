import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Search,
  SlidersHorizontal,
  Star,
  PlugZap,
  ShieldCheck,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import api from "@/lib/api";
import EmptyState from "@/components/EmptyState";
import { useToast } from "@/contexts/ToastContext";

export interface MarketplaceManifestResource {
  name: string;
  description?: string;
  required?: boolean;
}

export interface MarketplaceManifest {
  id: string;
  name: string;
  version: string;
  description: string;
  icon_url?: string;
  categories?: string[];
  auth: { type: string };
  sync: {
    supported_modes: string[];
    default_mode: string;
    resources: MarketplaceManifestResource[];
  };
  screenshots?: string[];
  developer?: {
    name?: string;
    website?: string;
  };
}

export interface MarketplaceConnector {
  id: string;
  version: string;
  developer_id: string;
  name: string;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
  published_at?: string;
  rating?: number;
  install_count?: number;
  manifest: MarketplaceManifest;
}

interface MarketplaceResponse {
  connectors: MarketplaceConnector[];
}

export interface MarketplacePageViewProps {
  connectors: MarketplaceConnector[];
  loading: boolean;
  error: string;
  search: string;
  category: string;
  status: string;
  installingId: string | null;
  selectedInstall: MarketplaceConnector | null;
  onSearchChange: (value: string) => void;
  onCategoryChange: (value: string) => void;
  onStatusChange: (value: string) => void;
  onOpenInstall: (connector: MarketplaceConnector) => void;
  onCloseInstall: () => void;
  onConfirmInstall: (connector: MarketplaceConnector) => void;
  onRetry: () => void;
}

interface MarketplaceResultsProps {
  visibleConnectors: MarketplaceConnector[];
  loading: boolean;
  error: string;
  onRetry: () => void;
  onOpenInstall: (connector: MarketplaceConnector) => void;
}

function authLabel(type: string) {
  switch (type) {
    case "oauth2":
      return "OAuth 2";
    case "api_key":
      return "API key";
    case "none":
      return "No auth";
    default:
      return type;
  }
}

function formatCategory(category: string) {
  return category
    .split(/[\s_-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function connectorCategories(connector: MarketplaceConnector) {
  return connector.manifest.categories?.length
    ? connector.manifest.categories
    : ["uncategorized"];
}

function connectorRating(connector: MarketplaceConnector) {
  if (typeof connector.rating !== "number" || connector.rating <= 0) {
    return "No ratings yet";
  }
  return `${connector.rating.toFixed(1)} rating`;
}

function connectorInstallCount(connector: MarketplaceConnector) {
  const count = connector.install_count ?? 0;
  return `${count.toLocaleString()} ${count === 1 ? "install" : "installs"}`;
}

function filterConnectors(
  connectors: MarketplaceConnector[],
  search: string,
  category: string,
  status: string,
) {
  const normalizedSearch = search.trim().toLowerCase();
  return connectors.filter((connector) => {
    const categories = connectorCategories(connector);
    const matchesCategory =
      category === "all" || categories.some((item) => item === category);
    const matchesStatus = status === "all" || connector.status === status;
    const searchTarget = [
      connector.name,
      connector.description,
      connector.manifest.name,
      connector.manifest.description,
      ...categories,
    ]
      .join(" ")
      .toLowerCase();
    return (
      matchesCategory &&
      matchesStatus &&
      (!normalizedSearch || searchTarget.includes(normalizedSearch))
    );
  });
}

function MarketplaceResults({
  visibleConnectors,
  loading,
  error,
  onRetry,
  onOpenInstall,
}: MarketplaceResultsProps) {
  if (loading) {
    return (
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {[...Array(6)].map((_, index) => (
          <div key={index} className="galdr-card h-64 animate-pulse p-5" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div role="alert" className="galdr-alert-danger p-6 text-center">
        <p className="text-sm">{error}</p>
        <button
          onClick={onRetry}
          className="galdr-link mt-2 text-sm font-medium"
        >
          Retry
        </button>
      </div>
    );
  }

  if (visibleConnectors.length === 0) {
    return (
      <div className="galdr-card">
        <EmptyState
          icon={<PlugZap className="h-12 w-12" />}
          title="No connectors found"
          description="Try a different search or category filter."
        />
      </div>
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {visibleConnectors.map((connector) => (
        <article
          key={`${connector.id}:${connector.version}`}
          className="galdr-card flex min-h-72 flex-col p-5"
        >
          <div className="flex items-start justify-between gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-2xl border border-[color:rgb(34_211_238_/_0.26)] bg-[color:rgb(34_211_238_/_0.11)] text-[var(--galdr-accent-2)]">
              {connector.manifest.icon_url ? (
                <img
                  src={connector.manifest.icon_url}
                  alt={`${connector.name} icon`}
                  className="h-8 w-8 rounded-xl object-cover"
                />
              ) : (
                <PlugZap className="h-6 w-6" />
              )}
            </div>
            <span className="galdr-pill px-2.5 py-1 text-xs font-medium">
              v{connector.version}
            </span>
          </div>
          <div className="mt-4 flex-1">
            <h2 className="text-xl font-semibold text-[var(--galdr-fg)]">
              <a
                className="galdr-link"
                href={`/marketplace/connectors/${connector.id}`}
              >
                {connector.name}
              </a>
            </h2>
            <p className="mt-2 line-clamp-3 text-sm leading-6 text-[var(--galdr-fg-muted)]">
              {connector.description}
            </p>
            <div className="mt-4 flex flex-wrap gap-2">
              {connectorCategories(connector).map((item) => (
                <span key={item} className="galdr-pill px-2.5 py-1 text-xs">
                  {formatCategory(item)}
                </span>
              ))}
            </div>
          </div>
          <div className="mt-5 grid gap-2 text-xs text-[var(--galdr-fg-muted)]">
            <span className="inline-flex items-center gap-2">
              <Star className="h-4 w-4 text-[var(--galdr-at-risk)]" />
              {connectorRating(connector)} · {connectorInstallCount(connector)}
            </span>
            <span className="inline-flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-[var(--galdr-success)]" />
              {authLabel(connector.manifest.auth.type)} ·{" "}
              {connector.manifest.sync.resources.length} resources
            </span>
            <span>Status: {formatCategory(connector.status)}</span>
          </div>
          <div className="mt-5 flex gap-2">
            <a
              href={`/marketplace/connectors/${connector.id}`}
              className="galdr-button-secondary inline-flex flex-1 items-center justify-center px-3 py-2 text-sm font-medium"
            >
              Details
            </a>
            <button
              onClick={() => onOpenInstall(connector)}
              className="galdr-button-primary inline-flex flex-1 items-center justify-center px-3 py-2 text-sm font-medium"
            >
              Install
            </button>
          </div>
        </article>
      ))}
    </div>
  );
}

export function MarketplacePageView({
  connectors,
  loading,
  error,
  search,
  category,
  status,
  installingId,
  selectedInstall,
  onSearchChange,
  onCategoryChange,
  onStatusChange,
  onOpenInstall,
  onCloseInstall,
  onConfirmInstall,
  onRetry,
}: MarketplacePageViewProps) {
  const categories = useMemo(() => {
    return Array.from(new Set(connectors.flatMap(connectorCategories))).sort();
  }, [connectors]);
  const statuses = useMemo(() => {
    return Array.from(
      new Set(connectors.map((connector) => connector.status)),
    ).sort();
  }, [connectors]);
  const visibleConnectors = filterConnectors(
    connectors,
    search,
    category,
    status,
  );

  return (
    <div className="space-y-6">
      <section className="galdr-card overflow-hidden p-6 sm:p-8">
        <div className="relative">
          <div className="absolute right-0 top-0 hidden h-32 w-32 rounded-full bg-[color:rgb(34_211_238_/_0.14)] blur-3xl sm:block" />
          <div className="galdr-kicker px-3 py-1">Connector discovery</div>
          <div className="mt-4 max-w-3xl">
            <h1 className="text-3xl font-bold tracking-tight text-[var(--galdr-fg)] sm:text-4xl">
              Integration Marketplace
            </h1>
            <p className="mt-3 text-sm leading-6 text-[var(--galdr-fg-muted)] sm:text-base">
              Find vetted community connectors, inspect their sync resources,
              then install them into PulseScore without waiting on core product
              releases.
            </p>
          </div>
        </div>
      </section>

      <section className="galdr-card p-4 sm:p-5">
        <div className="grid gap-3 lg:grid-cols-[1fr_180px_180px]">
          <label className="relative block">
            <span className="sr-only">Search connectors</span>
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--galdr-fg-muted)]" />
            <input
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder="Search connectors"
              className="galdr-input w-full py-2.5 pl-10 pr-3 text-sm"
            />
          </label>
          <label className="relative block">
            <span className="sr-only">Filter by category</span>
            <SlidersHorizontal className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--galdr-fg-muted)]" />
            <select
              value={category}
              onChange={(event) => onCategoryChange(event.target.value)}
              className="galdr-input w-full py-2.5 pl-10 pr-3 text-sm"
            >
              <option value="all">All categories</option>
              {categories.map((item) => (
                <option key={item} value={item}>
                  {formatCategory(item)}
                </option>
              ))}
            </select>
          </label>
          <label className="block">
            <span className="sr-only">Filter by status</span>
            <select
              value={status}
              onChange={(event) => onStatusChange(event.target.value)}
              className="galdr-input w-full px-3 py-2.5 text-sm"
            >
              <option value="all">All statuses</option>
              {statuses.map((item) => (
                <option key={item} value={item}>
                  {formatCategory(item)}
                </option>
              ))}
            </select>
          </label>
        </div>
      </section>

      <MarketplaceResults
        visibleConnectors={visibleConnectors}
        loading={loading}
        error={error}
        onRetry={onRetry}
        onOpenInstall={onOpenInstall}
      />

      {selectedInstall && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
          <div className="galdr-card w-full max-w-lg p-6 shadow-2xl">
            <div className="galdr-kicker px-3 py-1">Install connector</div>
            <h2 className="mt-4 text-2xl font-semibold text-[var(--galdr-fg)]">
              Install {selectedInstall.name}
            </h2>
            <p className="mt-3 text-sm leading-6 text-[var(--galdr-fg-muted)]">
              PulseScore will install this connector for your organization and
              redirect you to connector configuration.
            </p>
            <div className="mt-5 rounded-xl border border-[var(--galdr-border)] bg-[color:rgb(11_11_18_/_0.44)] p-4 text-sm text-[var(--galdr-fg-muted)]">
              Auth: {authLabel(selectedInstall.manifest.auth.type)} · Default
              sync: {selectedInstall.manifest.sync.default_mode}
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={onCloseInstall}
                className="galdr-button-secondary px-4 py-2 text-sm font-medium"
              >
                Cancel
              </button>
              <button
                onClick={() => onConfirmInstall(selectedInstall)}
                disabled={installingId === selectedInstall.id}
                className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-70"
              >
                {installingId === selectedInstall.id ? "Installing" : "Install"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function MarketplacePage() {
  const [connectors, setConnectors] = useState<MarketplaceConnector[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("all");
  const [status, setStatus] = useState("all");
  const [selectedInstall, setSelectedInstall] =
    useState<MarketplaceConnector | null>(null);
  const [installingId, setInstallingId] = useState<string | null>(null);
  const toast = useToast();
  const navigate = useNavigate();

  const fetchConnectors = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await api.get<MarketplaceResponse>(
        "/marketplace/connectors",
      );
      setConnectors(data.connectors ?? []);
      setError("");
    } catch {
      setError("Failed to load marketplace connectors.");
      toast.error("Failed to load marketplace connectors");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    fetchConnectors();
  }, [fetchConnectors]);

  async function handleConfirmInstall(connector: MarketplaceConnector) {
    setInstallingId(connector.id);
    try {
      await api.post(
        `/marketplace/connectors/${encodeURIComponent(connector.id)}/install`,
        {},
      );
      toast.success(`${connector.name} installed`);
      navigate(
        `/settings/integrations?connector=${encodeURIComponent(connector.id)}&installed=1`,
      );
    } catch {
      toast.error("Failed to install connector");
    } finally {
      setInstallingId(null);
    }
  }

  return (
    <MarketplacePageView
      connectors={connectors}
      loading={loading}
      error={error}
      search={search}
      category={category}
      status={status}
      installingId={installingId}
      selectedInstall={selectedInstall}
      onSearchChange={setSearch}
      onCategoryChange={setCategory}
      onStatusChange={setStatus}
      onOpenInstall={setSelectedInstall}
      onCloseInstall={() => setSelectedInstall(null)}
      onConfirmInstall={handleConfirmInstall}
      onRetry={fetchConnectors}
    />
  );
}
