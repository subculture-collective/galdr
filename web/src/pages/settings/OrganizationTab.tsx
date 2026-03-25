import { useEffect, useState } from "react";
import api, { onboardingApi } from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import { useNavigate } from "react-router-dom";
import { Loader2 } from "lucide-react";

interface Organization {
  id: string;
  name: string;
  slug: string;
  plan?: string;
}

export default function OrganizationTab() {
  const [org, setOrg] = useState<Organization | null>(null);
  const [loading, setLoading] = useState(true);
  const [resetting, setResetting] = useState(false);
  const toast = useToast();
  const navigate = useNavigate();

  useEffect(() => {
    async function fetch() {
      try {
        const { data } = await api.get<Organization>("/organizations/current");
        setOrg(data);
      } catch {
        toast.error("Failed to load organization info");
      } finally {
        setLoading(false);
      }
    }
    fetch();
  }, [toast]);

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--galdr-fg-muted)]" />
      </div>
    );
  }

  if (!org) return null;

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
      <div>
        <label className="block text-sm font-medium text-[var(--galdr-fg-muted)]">
          Organization Name
        </label>
        <p className="mt-1 text-sm text-[var(--galdr-fg)]">{org.name}</p>
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
