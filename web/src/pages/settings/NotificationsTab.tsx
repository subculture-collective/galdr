import { useEffect, useMemo, useState } from "react";
import {
  notificationPreferencesApi,
  alertsApi,
  type NotificationPreference,
  type AlertRule,
} from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import { Loader2, Bell, Mail, BookOpen } from "lucide-react";

export default function NotificationsTab() {
  const [prefs, setPrefs] = useState<NotificationPreference | null>(null);
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const toast = useToast();
  const mutedSet = useMemo(
    () => new Set(prefs?.muted_rule_ids ?? []),
    [prefs?.muted_rule_ids],
  );

  useEffect(() => {
    async function load() {
      try {
        const [prefRes, rulesRes] = await Promise.all([
          notificationPreferencesApi.get(),
          alertsApi.listRules(),
        ]);
        setPrefs(prefRes.data);
        setRules(rulesRes.data.rules ?? []);
      } catch {
        toast.error("Failed to load notification preferences");
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  async function toggleField(
    field: keyof Pick<
      NotificationPreference,
      "email_enabled" | "in_app_enabled" | "digest_enabled"
    >,
  ) {
    if (!prefs) return;
    setSaving(true);
    try {
      const { data } = await notificationPreferencesApi.update({
        [field]: !prefs[field],
      });
      setPrefs(data);
      toast.success("Preferences updated");
    } catch {
      toast.error("Failed to update preferences");
    } finally {
      setSaving(false);
    }
  }

  async function updateDigestFrequency(freq: string) {
    setSaving(true);
    try {
      const { data } = await notificationPreferencesApi.update({
        digest_frequency: freq,
      });
      setPrefs(data);
      toast.success("Digest frequency updated");
    } catch {
      toast.error("Failed to update preferences");
    } finally {
      setSaving(false);
    }
  }

  async function toggleMuteRule(ruleId: string) {
    if (!prefs) return;
    setSaving(true);

    const currentlyMuted = mutedSet.has(ruleId);
    const muted = prefs.muted_rule_ids ?? [];
    const newMuted = currentlyMuted
      ? muted.filter((id) => id !== ruleId)
      : [...muted, ruleId];

    try {
      const { data } = await notificationPreferencesApi.update({
        muted_rule_ids: newMuted,
      });
      setPrefs(data);
      toast.success(currentlyMuted ? "Rule unmuted" : "Rule muted");
    } catch {
      toast.error("Failed to update preferences");
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--galdr-accent)]" />
      </div>
    );
  }

  if (!prefs) return null;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-[var(--galdr-fg)]">
          Notification Preferences
        </h2>
        <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
          Control how and when you receive alert notifications.
        </p>
      </div>

      {/* Channel toggles */}
      <div className="galdr-card p-6">
        <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
          Channels
        </h3>
        <div className="space-y-4">
          <Toggle
            icon={<Mail className="h-5 w-5" />}
            label="Email notifications"
            description="Receive alert notifications via email"
            enabled={prefs.email_enabled}
            saving={saving}
            onToggle={() => toggleField("email_enabled")}
          />
          <Toggle
            icon={<Bell className="h-5 w-5" />}
            label="In-app notifications"
            description="Show notifications in the app"
            enabled={prefs.in_app_enabled}
            saving={saving}
            onToggle={() => toggleField("in_app_enabled")}
          />
        </div>
      </div>

      {/* Digest settings */}
      <div className="galdr-card p-6">
        <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
          Digest
        </h3>
        <div className="space-y-4">
          <Toggle
            icon={<BookOpen className="h-5 w-5" />}
            label="Email digest"
            description="Receive a periodic summary of alerts"
            enabled={prefs.digest_enabled}
            saving={saving}
            onToggle={() => toggleField("digest_enabled")}
          />
          {prefs.digest_enabled && (
            <div className="ml-10">
              <label className="text-sm text-[var(--galdr-fg-muted)]">
                Frequency
              </label>
              <select
                value={prefs.digest_frequency}
                onChange={(e) => updateDigestFrequency(e.target.value)}
                disabled={saving}
                className="galdr-input mt-1 block w-40 px-3 py-2 text-sm"
              >
                <option value="daily">Daily</option>
                <option value="weekly">Weekly</option>
              </select>
            </div>
          )}
        </div>
      </div>

      {/* Muted rules */}
      {rules.length > 0 && (
        <div className="galdr-card p-6">
          <h3 className="mb-2 text-sm font-medium text-[var(--galdr-fg)]">
            Mute individual rules
          </h3>
          <p className="mb-4 text-xs text-[var(--galdr-fg-muted)]">
            Muted rules will not trigger notifications for you.
          </p>
          <div className="space-y-3">
            {rules.map((rule) => {
              const isMuted = mutedSet.has(rule.id);
              return (
                <div
                  key={rule.id}
                  className="flex items-center justify-between"
                >
                  <div>
                    <p className="text-sm font-medium text-[var(--galdr-fg)]">
                      {rule.name}
                    </p>
                    <p className="text-xs text-[var(--galdr-fg-muted)]">
                      {rule.trigger_type}
                    </p>
                  </div>
                  <button
                    onClick={() => toggleMuteRule(rule.id)}
                    disabled={saving}
                    className={`rounded px-3 py-1 text-xs font-medium transition-colors ${
                      isMuted
                        ? "galdr-pill hover:border-[color:rgb(139_92_246_/_0.3)] hover:text-[var(--galdr-fg)]"
                        : "galdr-button-danger-outline"
                    }`}
                  >
                    {isMuted ? "Unmute" : "Mute"}
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

function Toggle({
  icon,
  label,
  description,
  enabled,
  saving,
  onToggle,
}: {
  icon: React.ReactNode;
  label: string;
  description: string;
  enabled: boolean;
  saving: boolean;
  onToggle: () => void;
}) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <span className="text-[var(--galdr-fg-muted)]">{icon}</span>
        <div>
          <p className="text-sm font-medium text-[var(--galdr-fg)]">{label}</p>
          <p className="text-xs text-[var(--galdr-fg-muted)]">{description}</p>
        </div>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={enabled}
        disabled={saving}
        onClick={onToggle}
        className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-[var(--galdr-accent)] focus:ring-offset-2 focus:ring-offset-[var(--galdr-bg)] disabled:opacity-50 ${
          enabled ? "bg-[var(--galdr-accent)]" : "bg-[var(--galdr-border)]"
        }`}
      >
        <span
          className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-[var(--galdr-fg)] shadow ring-0 transition duration-200 ease-in-out ${
            enabled ? "translate-x-5" : "translate-x-0"
          }`}
        />
      </button>
    </div>
  );
}
