import { useEffect, useState } from "react";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import { Loader2 } from "lucide-react";

interface ScoringConfig {
  id: string;
  payment_weight: number;
  usage_weight: number;
  support_weight: number;
  engagement_weight: number;
  red_threshold: number;
  yellow_threshold: number;
}

export default function ScoringTab() {
  const [config, setConfig] = useState<ScoringConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const toast = useToast();

  useEffect(() => {
    async function fetch() {
      try {
        const { data } = await api.get<ScoringConfig>("/scoring/config");
        setConfig(data);
      } catch {
        // May not exist yet
        setConfig(null);
      } finally {
        setLoading(false);
      }
    }
    fetch();
  }, [toast]);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    if (!config) return;
    setSaving(true);
    try {
      await api.put("/scoring/config", config);
      toast.success("Scoring configuration saved");
    } catch {
      toast.error("Failed to save scoring configuration");
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--galdr-fg-muted)]" />
      </div>
    );
  }

  if (!config) {
    return (
      <p className="py-8 text-center text-sm text-[var(--galdr-fg-muted)]">
        No scoring configuration found. Scoring will use default weights.
      </p>
    );
  }

  function updateField(field: keyof ScoringConfig, value: number) {
    setConfig((prev) => (prev ? { ...prev, [field]: value } : prev));
  }

  const weightFields: { key: keyof ScoringConfig; label: string }[] = [
    { key: "payment_weight", label: "Payment Weight" },
    { key: "usage_weight", label: "Usage Weight" },
    { key: "support_weight", label: "Support Weight" },
    { key: "engagement_weight", label: "Engagement Weight" },
  ];

  return (
    <form onSubmit={handleSave} className="max-w-lg space-y-6">
      <div className="space-y-4">
        <h3 className="text-sm font-medium text-[var(--galdr-fg)]">
          Score Weights
        </h3>
        {weightFields.map(({ key, label }) => (
          <div key={key}>
            <label className="block text-sm text-[var(--galdr-fg-muted)]">
              {label}
            </label>
            <input
              type="number"
              min={0}
              max={100}
              value={config[key] as number}
              onChange={(e) =>
                updateField(key, parseInt(e.target.value, 10) || 0)
              }
              className="galdr-input mt-1 w-24 px-3 py-2 text-sm"
            />
          </div>
        ))}
      </div>

      <div className="space-y-4">
        <h3 className="text-sm font-medium text-[var(--galdr-fg)]">
          Risk Thresholds
        </h3>
        <div>
          <label className="block text-sm text-[var(--galdr-fg-muted)]">
            Red Threshold (below this = critical)
          </label>
          <input
            type="number"
            min={0}
            max={100}
            value={config.red_threshold}
            onChange={(e) =>
              updateField("red_threshold", parseInt(e.target.value, 10) || 0)
            }
            className="galdr-input mt-1 w-24 px-3 py-2 text-sm"
          />
        </div>
        <div>
          <label className="block text-sm text-[var(--galdr-fg-muted)]">
            Yellow Threshold (below this = at-risk)
          </label>
          <input
            type="number"
            min={0}
            max={100}
            value={config.yellow_threshold}
            onChange={(e) =>
              updateField("yellow_threshold", parseInt(e.target.value, 10) || 0)
            }
            className="galdr-input mt-1 w-24 px-3 py-2 text-sm"
          />
        </div>
      </div>

      <button
        type="submit"
        disabled={saving}
        className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
      >
        {saving ? "Saving..." : "Save Configuration"}
      </button>
    </form>
  );
}
