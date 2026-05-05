import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  ArrowLeft,
  CheckCircle2,
  ExternalLink,
  PlugZap,
  ShieldCheck,
  Star,
} from "lucide-react";
import api from "@/lib/api";
import EmptyState from "@/components/EmptyState";
import { useToast } from "@/contexts/ToastContext";
import type { MarketplaceConnector } from "./MarketplacePage";

export interface ConnectorDetailPageViewProps {
  connector: MarketplaceConnector | null;
  loading: boolean;
  error: string;
  installing: boolean;
  showInstallConfirm: boolean;
  onOpenInstall: () => void;
  onCloseInstall: () => void;
  onConfirmInstall: () => void;
  onRetry: () => void;
}

function formatLabel(value: string) {
  return value
    .split(/[\s_-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
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
      return formatLabel(type);
  }
}

function ratingLabel(connector: MarketplaceConnector) {
  if (typeof connector.rating !== "number" || connector.rating <= 0) {
    return "No ratings yet";
  }
  return `${connector.rating.toFixed(1)} rating`;
}

function installCountLabel(connector: MarketplaceConnector) {
  const count = connector.install_count ?? 0;
  return `${count.toLocaleString()} ${count === 1 ? "install" : "installs"}`;
}

function connectorCategories(connector: MarketplaceConnector) {
  return connector.manifest.categories?.length
    ? connector.manifest.categories
    : ["uncategorized"];
}

function DeveloperValue({
  developer,
}: {
  developer: MarketplaceConnector["manifest"]["developer"];
}) {
  if (!developer?.name) {
    return null;
  }

  if (!developer.website) {
    return developer.name;
  }

  return (
    <a
      className="galdr-link inline-flex items-center gap-1"
      href={developer.website}
    >
      {developer.name} <ExternalLink className="h-3.5 w-3.5" />
    </a>
  );
}

export function ConnectorDetailPageView({
  connector,
  loading,
  error,
  installing,
  showInstallConfirm,
  onOpenInstall,
  onCloseInstall,
  onConfirmInstall,
  onRetry,
}: ConnectorDetailPageViewProps) {
  if (loading) {
    return <div className="galdr-card h-96 animate-pulse p-6" />;
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

  if (!connector) {
    return (
      <div className="galdr-card">
        <EmptyState
          icon={<PlugZap className="h-12 w-12" />}
          title="Connector not found"
          description="This marketplace connector is not published or no longer exists."
        />
      </div>
    );
  }

  const categories = connectorCategories(connector);
  const screenshots = connector.manifest.screenshots ?? [];
  const developer = connector.manifest.developer;

  return (
    <div className="space-y-6">
      <a
        href="/marketplace"
        className="galdr-link inline-flex items-center gap-2 text-sm font-medium"
      >
        <ArrowLeft className="h-4 w-4" /> Back to marketplace
      </a>

      <section className="galdr-card overflow-hidden p-6 sm:p-8">
        <div className="grid gap-6 lg:grid-cols-[1fr_280px]">
          <div>
            <div className="flex flex-wrap items-center gap-2">
              {categories.map((category) => (
                <span key={category} className="galdr-pill px-2.5 py-1 text-xs">
                  {formatLabel(category)}
                </span>
              ))}
              <span className="galdr-pill px-2.5 py-1 text-xs">
                v{connector.version}
              </span>
            </div>
            <h1 className="mt-5 text-3xl font-bold tracking-tight text-[var(--galdr-fg)] sm:text-4xl">
              {connector.name}
            </h1>
            <p className="mt-4 max-w-3xl text-base leading-7 text-[var(--galdr-fg-muted)]">
              {connector.manifest.description || connector.description}
            </p>
            <div className="mt-5 flex flex-wrap gap-4 text-sm text-[var(--galdr-fg-muted)]">
              <span className="inline-flex items-center gap-2">
                <Star className="h-4 w-4 text-[var(--galdr-at-risk)]" />
                {ratingLabel(connector)}
              </span>
              <span>{installCountLabel(connector)}</span>
              <span>{formatLabel(connector.status)}</span>
            </div>
          </div>

          <aside className="rounded-2xl border border-[var(--galdr-border)] bg-[color:rgb(11_11_18_/_0.44)] p-5">
            <div className="flex h-14 w-14 items-center justify-center rounded-2xl border border-[color:rgb(34_211_238_/_0.26)] bg-[color:rgb(34_211_238_/_0.11)] text-[var(--galdr-accent-2)]">
              {connector.manifest.icon_url ? (
                <img
                  src={connector.manifest.icon_url}
                  alt={`${connector.name} icon`}
                  className="h-9 w-9 rounded-xl object-cover"
                />
              ) : (
                <PlugZap className="h-7 w-7" />
              )}
            </div>
            <dl className="mt-5 grid gap-3 text-sm">
              <div>
                <dt className="text-[var(--galdr-fg-muted)]">Authentication</dt>
                <dd className="font-medium text-[var(--galdr-fg)]">
                  {authLabel(connector.manifest.auth.type)}
                </dd>
              </div>
              <div>
                <dt className="text-[var(--galdr-fg-muted)]">Default sync</dt>
                <dd className="font-medium text-[var(--galdr-fg)]">
                  {formatLabel(connector.manifest.sync.default_mode)}
                </dd>
              </div>
              {developer?.name && (
                <div>
                  <dt className="text-[var(--galdr-fg-muted)]">Developer</dt>
                  <dd className="font-medium text-[var(--galdr-fg)]">
                    <DeveloperValue developer={developer} />
                  </dd>
                </div>
              )}
            </dl>
            <button
              onClick={onOpenInstall}
              className="galdr-button-primary mt-6 inline-flex w-full items-center justify-center px-4 py-2.5 text-sm font-medium"
            >
              Install connector
            </button>
          </aside>
        </div>
      </section>

      <section className="grid gap-6 lg:grid-cols-[1fr_340px]">
        <div className="galdr-card p-5 sm:p-6">
          <h2 className="text-xl font-semibold text-[var(--galdr-fg)]">
            Sync resources
          </h2>
          <div className="mt-4 grid gap-3">
            {connector.manifest.sync.resources.map((resource) => (
              <div
                key={resource.name}
                className="rounded-xl border border-[var(--galdr-border)] p-4"
              >
                <div className="flex items-center justify-between gap-3">
                  <h3 className="font-medium text-[var(--galdr-fg)]">
                    {formatLabel(resource.name)}
                  </h3>
                  {resource.required && (
                    <span className="galdr-pill px-2 py-0.5 text-xs">
                      Required
                    </span>
                  )}
                </div>
                {resource.description && (
                  <p className="mt-2 text-sm leading-6 text-[var(--galdr-fg-muted)]">
                    {resource.description}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>

        <div className="galdr-card p-5 sm:p-6">
          <h2 className="text-xl font-semibold text-[var(--galdr-fg)]">
            Install readiness
          </h2>
          <div className="mt-4 grid gap-3 text-sm text-[var(--galdr-fg-muted)]">
            <span className="inline-flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-[var(--galdr-success)]" />
              Reviewed marketplace connector
            </span>
            {connector.manifest.sync.supported_modes.map((mode) => (
              <span key={mode} className="inline-flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4 text-[var(--galdr-success)]" />
                {formatLabel(mode)} sync supported
              </span>
            ))}
          </div>
        </div>
      </section>

      <section className="galdr-card p-5 sm:p-6">
        <h2 className="text-xl font-semibold text-[var(--galdr-fg)]">
          Screenshots
        </h2>
        {screenshots.length === 0 ? (
          <p className="mt-3 text-sm text-[var(--galdr-fg-muted)]">
            No screenshots published for this connector yet.
          </p>
        ) : (
          <div className="mt-4 grid gap-4 md:grid-cols-2">
            {screenshots.map((screenshot) => (
              <div
                key={screenshot}
                className="overflow-hidden rounded-2xl border border-[var(--galdr-border)] bg-[color:rgb(11_11_18_/_0.44)]"
              >
                <img
                  src={screenshot}
                  alt={`${connector.name} screenshot`}
                  className="h-56 w-full object-cover"
                />
              </div>
            ))}
          </div>
        )}
      </section>

      {showInstallConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4">
          <div className="galdr-card w-full max-w-lg p-6 shadow-2xl">
            <div className="galdr-kicker px-3 py-1">Install connector</div>
            <h2 className="mt-4 text-2xl font-semibold text-[var(--galdr-fg)]">
              Install {connector.name}
            </h2>
            <p className="mt-3 text-sm leading-6 text-[var(--galdr-fg-muted)]">
              PulseScore will provision this connector for your organization and
              then open integration configuration.
            </p>
            <div className="mt-5 rounded-xl border border-[var(--galdr-border)] bg-[color:rgb(11_11_18_/_0.44)] p-4 text-sm text-[var(--galdr-fg-muted)]">
              {authLabel(connector.manifest.auth.type)} ·{" "}
              {formatLabel(connector.manifest.sync.default_mode)} sync ·{" "}
              {connector.manifest.sync.resources.length} resources
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={onCloseInstall}
                className="galdr-button-secondary px-4 py-2 text-sm font-medium"
              >
                Cancel
              </button>
              <button
                onClick={onConfirmInstall}
                disabled={installing}
                className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-70"
              >
                {installing ? "Installing" : "Install"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function ConnectorDetailPage() {
  const { id } = useParams();
  const [connector, setConnector] = useState<MarketplaceConnector | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [installing, setInstalling] = useState(false);
  const [showInstallConfirm, setShowInstallConfirm] = useState(false);
  const toast = useToast();
  const navigate = useNavigate();

  const fetchConnector = useCallback(async () => {
    if (!id) {
      setConnector(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const { data } = await api.get<MarketplaceConnector>(
        `/marketplace/connectors/${encodeURIComponent(id)}`,
      );
      setConnector(data);
      setError("");
    } catch {
      setError("Failed to load marketplace connector.");
      toast.error("Failed to load marketplace connector");
    } finally {
      setLoading(false);
    }
  }, [id, toast]);

  useEffect(() => {
    fetchConnector();
  }, [fetchConnector]);

  async function handleConfirmInstall() {
    if (!connector) return;
    setInstalling(true);
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
      setInstalling(false);
    }
  }

  return (
    <ConnectorDetailPageView
      connector={connector}
      loading={loading}
      error={error}
      installing={installing}
      showInstallConfirm={showInstallConfirm}
      onOpenInstall={() => setShowInstallConfirm(true)}
      onCloseInstall={() => setShowInstallConfirm(false)}
      onConfirmInstall={handleConfirmInstall}
      onRetry={fetchConnector}
    />
  );
}
