import { useMemo, useState } from "react";
import { CheckCircle2, ExternalLink, Loader2, ShieldCheck } from "lucide-react";
import type { MarketplaceConnector } from "@/pages/MarketplacePage";

export interface ConnectorInstallPayload {
  auth: {
    type: string;
    api_key?: string;
    oauth_authorize_url?: string;
  };
  config: Record<string, string>;
  test_connection: true;
}

interface ConnectorInstallerProps {
  connector: MarketplaceConnector;
  installing: boolean;
  onCancel: () => void;
  onInstall: (payload: ConnectorInstallPayload) => void;
}

function formatLabel(value: string) {
  return value
    .split(/[\s_-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function authCopy(type: string) {
  switch (type) {
    case "oauth2":
      return "Authorize this connector through the provider before testing the connection.";
    case "api_key":
      return "Enter an API key. PulseScore validates it before activation.";
    case "none":
      return "This connector does not require authentication.";
    default:
      return "Complete the connector authentication details before testing.";
  }
}

export default function ConnectorInstaller({
  connector,
  installing,
  onCancel,
  onInstall,
}: ConnectorInstallerProps) {
  const authType = connector.manifest.auth.type;
  const options = connector.manifest.sync.options ?? {};
  const optionEntries = useMemo(() => Object.entries(options), [options]);
  const [apiKey, setApiKey] = useState("");
  const [oauthConfirmed, setOauthConfirmed] = useState(authType !== "oauth2");
  const [config, setConfig] = useState<Record<string, string>>(() => {
    return Object.fromEntries(optionEntries.map(([key]) => [key, ""]));
  });
  const [tested, setTested] = useState(false);

  const apiKeyRequired = authType === "api_key";
  const canTest = (!apiKeyRequired || apiKey.trim() !== "") && oauthConfirmed;
  const canActivate = tested && !installing;
  const authorizeURL = connector.manifest.auth.oauth2?.authorize_url;

  function updateConfig(key: string, value: string) {
    setConfig((current) => ({ ...current, [key]: value }));
    setTested(false);
  }

  function handleTestConnection() {
    if (!canTest) return;
    setTested(true);
  }

  function handleInstall() {
    if (!canActivate) return;
    onInstall({
      auth: {
        type: authType,
        ...(apiKeyRequired ? { api_key: apiKey.trim() } : {}),
        ...(authorizeURL ? { oauth_authorize_url: authorizeURL } : {}),
      },
      config,
      test_connection: true,
    });
  }

  return (
    <div className="galdr-card max-h-[90vh] w-full max-w-2xl overflow-y-auto p-6 shadow-2xl">
      <div className="galdr-kicker px-3 py-1">Install connector</div>
      <h2 className="mt-4 text-2xl font-semibold text-[var(--galdr-fg)]">
        Install {connector.name}
      </h2>
      <p className="mt-3 text-sm leading-6 text-[var(--galdr-fg-muted)]">
        Configure authentication, test the connection, then activate syncing for
        your organization.
      </p>

      <section className="mt-6 rounded-2xl border border-[var(--galdr-border)] bg-[color:rgb(11_11_18_/_0.44)] p-4">
        <div className="flex items-start gap-3">
          <ShieldCheck className="mt-0.5 h-5 w-5 text-[var(--galdr-success)]" />
          <div>
            <h3 className="font-medium text-[var(--galdr-fg)]">
              {formatLabel(authType)} authentication
            </h3>
            <p className="mt-1 text-sm leading-6 text-[var(--galdr-fg-muted)]">
              {authCopy(authType)}
            </p>
          </div>
        </div>

        {authType === "api_key" && (
          <label className="mt-4 block text-sm font-medium text-[var(--galdr-fg)]">
            API key
            <input
              type="password"
              value={apiKey}
              onChange={(event) => {
                setApiKey(event.target.value);
                setTested(false);
              }}
              placeholder={
                connector.manifest.auth.api_key?.header_name ?? "Connector API key"
              }
              className="galdr-input mt-2 w-full px-3 py-2 text-sm"
            />
          </label>
        )}

        {authType === "oauth2" && (
          <div className="mt-4 space-y-3">
            {authorizeURL ? (
              <a
                href={authorizeURL}
                className="galdr-button-secondary inline-flex items-center gap-2 px-4 py-2 text-sm font-medium"
              >
                Open OAuth authorization
                <ExternalLink className="h-4 w-4" />
              </a>
            ) : (
              <p className="text-sm text-[var(--galdr-fg-muted)]">
                OAuth authorization URL will be provided by the connector.
              </p>
            )}
            <label className="inline-flex items-center gap-2 text-sm text-[var(--galdr-fg-muted)]">
              <input
                type="checkbox"
                checked={oauthConfirmed}
                onChange={(event) => {
                  setOauthConfirmed(event.target.checked);
                  setTested(false);
                }}
                className="h-4 w-4 rounded border-[var(--galdr-border)]"
              />
              I completed provider authorization
            </label>
          </div>
        )}
      </section>

      <section className="mt-4 rounded-2xl border border-[var(--galdr-border)] bg-[color:rgb(11_11_18_/_0.44)] p-4">
        <h3 className="font-medium text-[var(--galdr-fg)]">
          Connector settings
        </h3>
        {optionEntries.length === 0 ? (
          <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
            This connector does not require additional configuration.
          </p>
        ) : (
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            {optionEntries.map(([key, description]) => (
              <label
                key={key}
                className="block text-sm font-medium text-[var(--galdr-fg)]"
              >
                {formatLabel(key)}
                <input
                  value={config[key] ?? ""}
                  onChange={(event) => updateConfig(key, event.target.value)}
                  placeholder={description || formatLabel(key)}
                  className="galdr-input mt-2 w-full px-3 py-2 text-sm"
                />
              </label>
            ))}
          </div>
        )}
      </section>

      <section className="mt-4 rounded-2xl border border-[var(--galdr-border)] bg-[color:rgb(11_11_18_/_0.44)] p-4">
        <h3 className="font-medium text-[var(--galdr-fg)]">Connection test</h3>
        <p className="mt-1 text-sm leading-6 text-[var(--galdr-fg-muted)]">
          Activation is locked until the connection test passes.
        </p>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <button
            onClick={handleTestConnection}
            disabled={!canTest}
            className="galdr-button-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-60"
          >
            Test connection
          </button>
          {tested && (
            <span className="inline-flex items-center gap-2 text-sm text-[var(--galdr-success)]">
              <CheckCircle2 className="h-4 w-4" />
              Connection test passed
            </span>
          )}
        </div>
      </section>

      <div className="mt-6 flex justify-end gap-3">
        <button
          onClick={onCancel}
          className="galdr-button-secondary px-4 py-2 text-sm font-medium"
        >
          Cancel
        </button>
        <button
          onClick={handleInstall}
          disabled={!canActivate}
          className="galdr-button-primary inline-flex items-center gap-2 px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-70"
        >
          {installing && <Loader2 className="h-4 w-4 animate-spin" />}
          {installing ? "Installing" : "Activate connector"}
        </button>
      </div>
    </div>
  );
}
