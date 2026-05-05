import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";

interface BenchmarkOrgSettings {
  benchmarking_enabled?: boolean;
  company_size?: number;
}

interface BenchmarkSettingsProps {
  org: BenchmarkOrgSettings;
  industry: string;
  saving: boolean;
  setSaving: (saving: boolean) => void;
  onSaved: (org: BenchmarkOrgSettings) => void;
}

export default function BenchmarkSettings({
  org,
  industry,
  saving,
  setSaving,
  onSaved,
}: BenchmarkSettingsProps) {
  const toast = useToast();
  const enabled = Boolean(org.benchmarking_enabled);
  const companySize = org.company_size ?? 0;
  const toggleClassName = enabled
    ? "bg-emerald-500/15 text-emerald-300 ring-1 ring-emerald-400/30"
    : "bg-[var(--galdr-bg-subtle)] text-[var(--galdr-fg-muted)] ring-1 ring-[var(--galdr-border)]";

  function benchmarkUpdate(nextEnabled: boolean, nextCompanySize: number) {
    return {
      benchmarking_enabled: nextEnabled,
      industry,
      company_size: nextCompanySize,
    };
  }

  async function updateBenchmarking(nextEnabled: boolean) {
    if (nextEnabled && !industry) {
      toast.error("Choose an industry before joining benchmarks.");
      return;
    }

    setSaving(true);
    try {
      const { data } = await api.patch<BenchmarkOrgSettings>(
        "/organizations/current",
        benchmarkUpdate(nextEnabled, companySize),
      );
      const successMessage = nextEnabled
        ? "Benchmark sharing enabled."
        : "Benchmark sharing disabled and prior contributions deleted.";

      onSaved(data);
      toast.success(successMessage);
    } catch {
      toast.error("Failed to update benchmark sharing.");
    } finally {
      setSaving(false);
    }
  }

  async function updateCompanySize(value: string) {
    const nextSize = Number(value);
    if (!Number.isInteger(nextSize) || nextSize < 0) {
      toast.error("Company size must be 0 or greater.");
      return;
    }

    setSaving(true);
    try {
      const { data } = await api.patch<BenchmarkOrgSettings>(
        "/organizations/current",
        benchmarkUpdate(enabled, nextSize),
      );
      onSaved(data);
      toast.success("Benchmark company size updated.");
    } catch {
      toast.error("Failed to update company size.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="galdr-panel space-y-4 p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
            Benchmarking
          </h3>
          <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
            Join anonymized industry benchmarks only when you explicitly opt in.
          </p>
        </div>
        <button
          type="button"
          onClick={() => updateBenchmarking(!enabled)}
          disabled={saving}
          aria-pressed={enabled}
          className={`rounded-full px-4 py-2 text-sm font-medium transition-colors disabled:opacity-50 ${
            toggleClassName
          }`}
        >
          {enabled ? "Opt out" : "Opt in"}
        </button>
      </div>

      <div className="rounded-lg border border-[var(--galdr-border)] bg-[var(--galdr-bg-subtle)] p-3 text-sm text-[var(--galdr-fg-muted)]">
        <p className="font-medium text-[var(--galdr-fg)]">Shared data</p>
        <p className="mt-1">
          PulseScore shares anonymized, org-level aggregates only: industry,
          company-size bucket, customer-count bucket, average health score,
          average MRR, and churn-rate percentile inputs.
        </p>
        <p className="mt-2">
          Customer PII, customer IDs, names, emails, external IDs, and metadata
          are never shared. Opting out deletes your organization&apos;s stored
          benchmark contributions and excludes future aggregation runs.
        </p>
        <a
          href="/legal/benchmark-data-usage"
          className="mt-3 inline-block text-[var(--galdr-accent)] hover:underline"
        >
          Benchmark data usage terms
        </a>
      </div>

      <label className="block text-sm font-medium text-[var(--galdr-fg-muted)]">
        Company size
        <input
          type="number"
          min="0"
          value={companySize}
          onChange={(event) => updateCompanySize(event.target.value)}
          disabled={saving}
          className="galdr-input mt-1 w-full px-3 py-2 text-sm disabled:opacity-50"
        />
      </label>
    </section>
  );
}
