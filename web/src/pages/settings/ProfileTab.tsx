import { useEffect, useState } from "react";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import { Loader2 } from "lucide-react";

interface UserProfile {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
}

export default function ProfileTab() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const toast = useToast();

  useEffect(() => {
    async function fetch() {
      try {
        const { data } = await api.get<UserProfile>("/users/me");
        setProfile(data);
        setFirstName(data.first_name);
        setLastName(data.last_name);
      } catch {
        toast.error("Failed to load profile");
      } finally {
        setLoading(false);
      }
    }
    fetch();
  }, [toast]);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await api.patch("/users/me", {
        first_name: firstName,
        last_name: lastName,
      });
      toast.success("Profile updated");
    } catch {
      toast.error("Failed to update profile");
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

  if (!profile) return null;

  return (
    <form onSubmit={handleSave} className="max-w-lg space-y-4">
      <div>
        <label className="block text-sm font-medium text-[var(--galdr-fg-muted)]">
          Email
        </label>
        <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
          {profile.email}
        </p>
      </div>
      <div>
        <label
          htmlFor="firstName"
          className="block text-sm font-medium text-[var(--galdr-fg-muted)]"
        >
          First Name
        </label>
        <input
          id="firstName"
          type="text"
          value={firstName}
          onChange={(e) => setFirstName(e.target.value)}
          className="galdr-input mt-1 w-full px-3 py-2 text-sm"
        />
      </div>
      <div>
        <label
          htmlFor="lastName"
          className="block text-sm font-medium text-[var(--galdr-fg-muted)]"
        >
          Last Name
        </label>
        <input
          id="lastName"
          type="text"
          value={lastName}
          onChange={(e) => setLastName(e.target.value)}
          className="galdr-input mt-1 w-full px-3 py-2 text-sm"
        />
      </div>
      <button
        type="submit"
        disabled={saving}
        className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
      >
        {saving ? "Saving..." : "Save Changes"}
      </button>
    </form>
  );
}
