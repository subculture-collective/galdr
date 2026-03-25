import { useEffect, useState } from "react";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import { Loader2 } from "lucide-react";

interface Member {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
}

export default function TeamTab() {
  const [members, setMembers] = useState<Member[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();

  useEffect(() => {
    async function fetch() {
      try {
        const { data } = await api.get<{ members: Member[] }>("/members");
        setMembers(data.members ?? []);
      } catch {
        toast.error("Failed to load team members");
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

  return (
    <div className="space-y-4">
      <div className="galdr-card overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-[var(--galdr-border)] bg-[color:rgb(31_31_46_/_0.72)] text-xs uppercase text-[var(--galdr-fg-muted)]">
            <tr>
              <th className="px-6 py-3">Name</th>
              <th className="px-6 py-3">Email</th>
              <th className="px-6 py-3">Role</th>
            </tr>
          </thead>
          <tbody>
            {members.map((m) => (
              <tr
                key={m.id}
                className="border-b border-[var(--galdr-border)]/70"
              >
                <td className="px-6 py-4 font-medium text-[var(--galdr-fg)]">
                  {m.first_name} {m.last_name}
                </td>
                <td className="px-6 py-4 text-[var(--galdr-fg-muted)]">
                  {m.email}
                </td>
                <td className="px-6 py-4">
                  <span className="galdr-pill inline-flex px-2.5 py-0.5 text-xs font-medium capitalize">
                    {m.role}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
