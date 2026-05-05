import { useEffect, useState } from "react";
import api, { onboardingApi } from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import { useNavigate } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { ORGANIZATION_INDUSTRIES } from "@/lib/industries";

interface Organization {
  id: string;
  name: string;
  slug: string;
  industry: string;
  benchmarking_enabled?: boolean;
  plan?: string;
}

export default function OrganizationTab() {
  const [org, setOrg] = useState<Organization | null>(null);
  const [name, setName] = useState("");
  const [industry, setIndustry] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);
  const toast = useToast();
  const navigate = useNavigate();

  useEffect(() => {
    async function fetchOrganization() {
      try {
        const { data } = await api.get<Organization>("/organizations/current");
        setOrg(data);
        setName(data.name);
        setIndustry(data.industry ?? "");
      } catch {
        toast.error("Failed to load organization info");
      } finally {
        setLoading(false);
      }
    }
    fetchOrganization();
  }, [toast]);

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--galdr-fg-muted)]" />
      </div>
    );
  }

  if (!org) return null;

  const industryRequiredForBenchmarking = org.benchmarking_enabled && !industry;

  async function saveOrganization() {
    const trimmedName = name.trim();
    if (!trimmedName) {
      toast.error("Organization name is required.");
      return;
    }
    if (industryRequiredForBenchmarking) {
      toast.error("Industry is required for benchmarking.");
      return;
    }

    setSaving(true);
    try {
      const { data } = await api.patch<Organization>("/organizations/current", {
        name: trimmedName,
        industry,
      });
      setOrg(data);
      setName(data.name);
      setIndustry(data.industry ?? "");
      toast.success("Organization updated.");
    } catch {
      toast.error("Failed to update organization.");
    } finally {
      setSaving(false);
    }
  }

  async function rerunOnboarding() {
    setResetting(true);
    try {
      await onboardingApi.reset();
      toast.success("Onboarding reset. Let’s run setup again.");
      navigate("/onboarding?step=welcome");
    } catch {
      toast.error("Failed to reset onboarding.");
    } finally {
      setResetting(false);
    }
  }

  return (
    <div className="max-w-lg space-y-4">
      <div className="space-y-4">
        <label className="block text-sm font-medium text-[var(--galdr-fg-muted)]">
          Organization Name
        </label>
        <input
          value={name}
          onChange={(event) => setName(event.target.value)}
          className="galdr-input mt-1 w-full px-3 py-2 text-sm"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-[var(--galdr-fg-muted)]">
          Industry
        </label>
        <select
          value={industry}
          onChange={(event) => setIndustry(event.target.value)}
          className="galdr-input mt-1 w-full px-3 py-2 text-sm"
        >
          <option value="">Select an industry</option>
          {ORGANIZATION_INDUSTRIES.map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
        <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
          Used to segment anonymized benchmarks by peer industry.
        </p>
      </div>
      <div>
        <button
          onClick={saveOrganization}
          disabled={saving || !name.trim() || industryRequiredForBenchmarking}
          className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
        >
          {saving ? "Saving..." : "Save organization"}
        </button>
      </div>
      <div>
        <label className="block text-sm font-medium text-[var(--galdr-fg-muted)]">
          Slug
        </label>
        <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">{org.slug}</p>
      </div>
      {org.plan && (
        <div>
          <label className="block text-sm font-medium text-[var(--galdr-fg-muted)]">
            Plan
          </label>
          <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
            {org.plan}
          </p>
        </div>
      )}

      <div className="galdr-panel p-4">
        <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
          Onboarding
        </h3>
        <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
          Need to revisit setup? You can restart the onboarding wizard anytime.
        </p>
        <button
          onClick={rerunOnboarding}
          disabled={resetting}
          className="galdr-button-primary mt-3 px-4 py-2 text-sm font-medium disabled:opacity-50"
        >
          {resetting ? "Resetting..." : "Re-run onboarding"}
        </button>
      </div>
    </div>
  );
}
