import { useEffect, useState } from "react";
import { Code2, Copy, FlaskConical, Link2, Plus, Webhook } from "lucide-react";
import { useToast } from "@/contexts/ToastContext";
import {
  webhooksApi,
  type WebhookConfiguration,
  type WebhookFieldMapping,
} from "@/lib/webhooks";

const emptyMapping = (): WebhookFieldMapping => ({
  source_path: "",
  target_field: "",
});

const defaultPayload = JSON.stringify(
  {
    user: { email: "founder@acme.test" },
    company: { name: "Acme Corp" },
    account: { mrr: 12900 },
    event: "customer.updated",
  },
  null,
  2,
);

type ExamplePayload = {
  name: string;
  payload: Record<string, unknown>;
};

const examplePayloads: ExamplePayload[] = [
  {
    name: "Customer.io",
    payload: {
      user: { email: "founder@acme.test", id: "cio_123" },
      company: { name: "Acme Corp" },
      account: { mrr: 12900 },
      event: "customer.updated",
    },
  },
  {
    name: "Zapier",
    payload: {
      zapier_hook_id: "hook_123",
      contact: { email: "founder@acme.test", name: "Avery Stone" },
      account: { company: "Acme Corp", monthly_revenue: 12900 },
      event_type: "contact.updated",
    },
  },
  {
    name: "PostHog",
    payload: {
      event: "feature_used",
      distinct_id: "user_123",
      properties: {
        email: "founder@acme.test",
        company_name: "Acme Corp",
      },
      timestamp: "2026-05-05T18:00:00Z",
    },
  },
];

const targetFields = [
  "email",
  "name",
  "company_name",
  "mrr_cents",
  "external_id",
  "event_type",
  "occurred_at",
];

const webhookStatusStyles: Record<WebhookConfiguration["status"], string> = {
  active:
    "border-[color:rgb(52_211_153_/_0.35)] bg-[color:rgb(52_211_153_/_0.14)] text-[var(--galdr-success)]",
  error:
    "border-[color:rgb(244_63_94_/_0.35)] bg-[color:rgb(244_63_94_/_0.14)] text-[var(--galdr-danger)]",
  paused:
    "border-[color:rgb(245_158_11_/_0.35)] bg-[color:rgb(245_158_11_/_0.14)] text-[var(--galdr-at-risk)]",
};

function getCleanMappings(mappings: WebhookFieldMapping[]) {
  return mappings
    .map((mapping) => ({
      source_path: mapping.source_path.trim(),
      target_field: mapping.target_field.trim(),
    }))
    .filter((mapping) => mapping.source_path && mapping.target_field);
}

function parseSamplePayload(value: string) {
  const parsed = JSON.parse(value) as unknown;
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("payload must be an object");
  }
  return parsed as Record<string, unknown>;
}

function formatMapping(mapping: WebhookFieldMapping) {
  return `${mapping.source_path} -> ${mapping.target_field}`;
}

export default function WebhookConfig() {
  const [webhooks, setWebhooks] = useState<WebhookConfiguration[]>([]);
  const [name, setName] = useState("");
  const [mappings, setMappings] = useState<WebhookFieldMapping[]>([
    emptyMapping(),
  ]);
  const [samplePayload, setSamplePayload] = useState(defaultPayload);
  const [mappedResult, setMappedResult] = useState<Record<
    string,
    unknown
  > | null>(null);
  const [latestCreated, setLatestCreated] =
    useState<WebhookConfiguration | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const toast = useToast();

  useEffect(() => {
    async function fetchWebhooks() {
      try {
        const { data } = await webhooksApi.list();
        setWebhooks(data.webhooks ?? []);
      } catch {
        toast.error("Failed to load webhook configurations.");
      } finally {
        setLoading(false);
      }
    }

    fetchWebhooks();
  }, [toast]);

  function updateMapping(index: number, patch: Partial<WebhookFieldMapping>) {
    setMappings((current) =>
      current.map((mapping, mappingIndex) =>
        mappingIndex === index ? { ...mapping, ...patch } : mapping,
      ),
    );
  }

  async function copyValue(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      toast.success("Copied to clipboard.");
    } catch {
      toast.error("Copy failed.");
    }
  }

  async function handleCreate() {
    const cleanMappings = getCleanMappings(mappings);
    if (!name.trim()) {
      toast.error("Webhook name is required.");
      return;
    }
    if (cleanMappings.length === 0) {
      toast.error("Add at least one complete field mapping.");
      return;
    }
    setSaving(true);
    try {
      const { data } = await webhooksApi.create({
        name: name.trim(),
        mappings: cleanMappings,
      });
      setLatestCreated(data.webhook);
      setWebhooks((current) => [data.webhook, ...current]);
      setName("");
      toast.success("Webhook URL and secret generated.");
    } catch {
      toast.error("Failed to create webhook.");
    } finally {
      setSaving(false);
    }
  }

  async function handleTest() {
    const cleanMappings = getCleanMappings(mappings);
    if (cleanMappings.length === 0) {
      toast.error("Add at least one complete field mapping.");
      return;
    }
    let parsedPayload: Record<string, unknown>;
    try {
      parsedPayload = parseSamplePayload(samplePayload);
    } catch {
      toast.error("Sample payload must be valid JSON object.");
      return;
    }

    setTesting(true);
    try {
      const { data } = await webhooksApi.testMapping({
        mappings: cleanMappings,
        sample_payload: parsedPayload,
      });
      setMappedResult(data.mapped_result);
      toast.success("Mapping validated.");
    } catch {
      toast.error("Failed to validate mapping.");
    } finally {
      setTesting(false);
    }
  }

  function loadExamplePayload(payload: Record<string, unknown>) {
    setSamplePayload(JSON.stringify(payload, null, 2));
    setMappedResult(null);
  }

  return (
    <section className="space-y-6">
      <div className="galdr-card overflow-hidden p-0">
        <div className="border-b border-[var(--galdr-border)] bg-[radial-gradient(circle_at_top_left,rgb(34_211_238_/_0.18),transparent_34%),linear-gradient(135deg,rgb(139_92_246_/_0.12),transparent)] p-6">
          <span className="galdr-kicker px-3 py-1">Generic receiver</span>
          <div className="mt-4 flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
            <div>
              <h3 className="text-xl font-semibold text-[var(--galdr-fg)]">
                Webhook Configuration
              </h3>
              <p className="mt-2 max-w-2xl text-sm text-[var(--galdr-fg-muted)]">
                Generate a signed endpoint, map incoming JSON paths to
                PulseScore customer fields, then test the payload before wiring
                automation tools.
              </p>
            </div>
            <Webhook className="hidden h-12 w-12 text-[var(--galdr-accent-2)] md:block" />
          </div>
        </div>

        <div className="grid gap-6 p-6 lg:grid-cols-[1.05fr_0.95fr]">
          <div className="space-y-5">
            <label className="block text-sm font-medium text-[var(--galdr-fg)]">
              Webhook name
              <input
                value={name}
                onChange={(event) => setName(event.target.value)}
                className="galdr-input mt-2 w-full px-3 py-2 text-sm"
                placeholder="Customer.io product events"
              />
            </label>

            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div>
                  <h4 className="text-sm font-semibold text-[var(--galdr-fg)]">
                    Field mapping builder
                  </h4>
                  <p className="text-xs text-[var(--galdr-fg-muted)]">
                    Use dotted paths such as <code>user.email</code> or
                    <code> account.mrr</code>.
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() =>
                    setMappings((current) => [...current, emptyMapping()])
                  }
                  className="galdr-button-secondary inline-flex items-center gap-2 px-3 py-2 text-xs font-medium"
                >
                  <Plus className="h-3.5 w-3.5" />
                  Add mapping
                </button>
              </div>

              {mappings.map((mapping, index) => (
                <div
                  key={index}
                  className="grid gap-3 rounded-2xl border border-[var(--galdr-border)] bg-[color:rgb(0_0_0_/_0.12)] p-3 md:grid-cols-[1fr_auto_1fr] md:items-end"
                >
                  <label className="block text-xs font-medium text-[var(--galdr-fg-muted)]">
                    Source path {index + 1}
                    <input
                      value={mapping.source_path}
                      onChange={(event) =>
                        updateMapping(index, {
                          source_path: event.target.value,
                        })
                      }
                      className="galdr-input mt-1 w-full px-3 py-2 text-sm font-mono"
                      placeholder="user.email"
                    />
                  </label>
                  <span className="hidden pb-2 text-[var(--galdr-accent-2)] md:block">
                    -&gt;
                  </span>
                  <label className="block text-xs font-medium text-[var(--galdr-fg-muted)]">
                    Target field {index + 1}
                    <input
                      value={mapping.target_field}
                      onChange={(event) =>
                        updateMapping(index, {
                          target_field: event.target.value,
                        })
                      }
                      list="webhook-target-fields"
                      className="galdr-input mt-1 w-full px-3 py-2 text-sm font-mono"
                      placeholder="email"
                    />
                  </label>
                </div>
              ))}
              <datalist id="webhook-target-fields">
                {targetFields.map((field) => (
                  <option key={field} value={field} />
                ))}
              </datalist>
            </div>

            <button
              type="button"
              onClick={handleCreate}
              disabled={saving}
              className="galdr-button-primary inline-flex items-center gap-2 px-4 py-2 text-sm font-medium disabled:opacity-50"
            >
              <Link2 className="h-4 w-4" />
              {saving ? "Creating..." : "Create webhook"}
            </button>
          </div>

          <div className="space-y-4">
            <div className="galdr-panel p-4">
              <h4 className="flex items-center gap-2 text-sm font-semibold text-[var(--galdr-fg)]">
                <Code2 className="h-4 w-4 text-[var(--galdr-accent-2)]" />
                Test sender
              </h4>
              <div className="mt-3 flex flex-wrap gap-2">
                {examplePayloads.map((example) => (
                  <button
                    key={example.name}
                    type="button"
                    onClick={() => loadExamplePayload(example.payload)}
                    className="galdr-pill px-2.5 py-1 text-xs font-medium text-[var(--galdr-fg)] transition hover:border-[var(--galdr-accent-2)]"
                  >
                    Use {example.name} sample
                  </button>
                ))}
              </div>
              <label className="mt-3 block text-xs font-medium text-[var(--galdr-fg-muted)]">
                Sample payload
                <textarea
                  value={samplePayload}
                  onChange={(event) => setSamplePayload(event.target.value)}
                  rows={9}
                  className="galdr-input mt-1 w-full px-3 py-2 font-mono text-xs"
                />
              </label>
              <button
                type="button"
                onClick={handleTest}
                disabled={testing}
                className="galdr-button-secondary mt-3 inline-flex items-center gap-2 px-3 py-2 text-xs font-medium disabled:opacity-50"
              >
                <FlaskConical className="h-3.5 w-3.5" />
                {testing ? "Testing..." : "Test mapping"}
              </button>
              {mappedResult && (
                <pre
                  aria-label="Mapped result"
                  className="mt-3 overflow-auto rounded-xl border border-[var(--galdr-border)] bg-[color:rgb(0_0_0_/_0.22)] p-3 text-xs text-[var(--galdr-fg)]"
                >
                  {JSON.stringify(mappedResult, null, 2)}
                </pre>
              )}
            </div>

            {latestCreated && (
              <div className="galdr-alert-success p-4 text-sm">
                <p className="font-semibold">Webhook generated</p>
                <SecretRow
                  label="URL"
                  value={latestCreated.url}
                  onCopy={copyValue}
                />
                <SecretRow
                  label="Secret"
                  value={latestCreated.secret}
                  onCopy={copyValue}
                />
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="galdr-card p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              Configured webhooks
            </h3>
            <p className="text-xs text-[var(--galdr-fg-muted)]">
              Endpoint status, last event, and accepted event volume.
            </p>
          </div>
        </div>

        {loading ? (
          <p className="mt-4 text-sm text-[var(--galdr-fg-muted)]">
            Loading webhooks...
          </p>
        ) : webhooks.length === 0 ? (
          <p className="galdr-panel mt-4 p-4 text-sm text-[var(--galdr-fg-muted)]">
            No webhooks configured yet.
          </p>
        ) : (
          <div className="mt-4 grid gap-3">
            {webhooks.map((webhook) => (
              <article
                key={webhook.id}
                className="rounded-2xl border border-[var(--galdr-border)] bg-[color:rgb(0_0_0_/_0.12)] p-4"
              >
                <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <h4 className="text-sm font-semibold text-[var(--galdr-fg)]">
                        {webhook.name}
                      </h4>
                      <StatusBadge status={webhook.status} />
                    </div>
                    <p className="mt-2 break-all font-mono text-xs text-[var(--galdr-fg-muted)]">
                      {webhook.url}
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => copyValue(webhook.url)}
                    className="galdr-button-secondary inline-flex items-center gap-2 px-3 py-2 text-xs font-medium"
                  >
                    <Copy className="h-3.5 w-3.5" />
                    Copy URL
                  </button>
                </div>
                <div className="mt-3 flex flex-wrap gap-2 text-xs text-[var(--galdr-fg-muted)]">
                  <span className="galdr-pill px-2.5 py-1">
                    {webhook.event_count} events
                  </span>
                  <span className="galdr-pill px-2.5 py-1">
                    Last event: {formatDate(webhook.last_event_at)}
                  </span>
                  <span className="galdr-pill px-2.5 py-1">
                    {webhook.mappings.length} mappings
                  </span>
                </div>
                <WebhookMappingList mappings={webhook.mappings} />
              </article>
            ))}
          </div>
        )}
      </div>

      <ExamplePayloadReference examples={examplePayloads} />
    </section>
  );
}

function SecretRow({
  label,
  value,
  onCopy,
}: {
  label: string;
  value: string;
  onCopy: (value: string) => void;
}) {
  return (
    <div className="mt-2 flex items-center justify-between gap-3 rounded-xl bg-[color:rgb(0_0_0_/_0.18)] px-3 py-2">
      <p className="min-w-0 break-all">
        <span className="font-medium">{label}:</span>{" "}
        <span className="font-mono">{value}</span>
      </p>
      <button
        type="button"
        onClick={() => onCopy(value)}
        className="galdr-button-secondary shrink-0 px-2 py-1 text-xs font-medium"
      >
        Copy
      </button>
    </div>
  );
}

function StatusBadge({ status }: { status: WebhookConfiguration["status"] }) {
  const label = status === "active" ? "Active" : status;

  return (
    <span
      className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium capitalize ${webhookStatusStyles[status]}`}
    >
      {label}
    </span>
  );
}

function WebhookMappingList({ mappings }: { mappings: WebhookFieldMapping[] }) {
  if (mappings.length === 0) return null;

  return (
    <div className="mt-3 flex flex-wrap gap-2">
      {mappings.map((mapping, index) => {
        const label = formatMapping(mapping);

        return (
          <span
            key={`${label}:${index}`}
            className="rounded-full border border-[var(--galdr-border)] bg-[color:rgb(0_0_0_/_0.18)] px-2.5 py-1 font-mono text-xs text-[var(--galdr-fg-muted)]"
          >
            {label}
          </span>
        );
      })}
    </div>
  );
}

function ExamplePayloadReference({ examples }: { examples: ExamplePayload[] }) {
  return (
    <div className="galdr-alert-info p-4 text-sm">
      <p className="font-semibold">Example payloads</p>
      <p className="mt-1">
        Customer.io and Zapier payloads usually map <code>user.email</code> to
        <code> email</code>, <code>company.name</code> to
        <code> company_name</code>, and <code>account.mrr</code> to
        <code> mrr_cents</code>.
      </p>
      <div className="mt-4 grid gap-3 lg:grid-cols-3">
        {examples.map((example) => (
          <div
            key={example.name}
            className="rounded-2xl border border-[var(--galdr-border)] bg-[color:rgb(0_0_0_/_0.14)] p-3"
          >
            <p className="text-xs font-semibold text-[var(--galdr-fg)]">
              {example.name} example
            </p>
            <pre className="mt-2 max-h-52 overflow-auto rounded-xl bg-[color:rgb(0_0_0_/_0.2)] p-3 text-xs text-[var(--galdr-fg-muted)]">
              {JSON.stringify(example.payload, null, 2)}
            </pre>
          </div>
        ))}
      </div>
    </div>
  );
}

function formatDate(value: string | null) {
  if (!value) return "Never";
  return new Date(value).toLocaleString();
}
