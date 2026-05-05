import { useEffect, useState, type FormEvent } from "react";
import { Loader2 } from "lucide-react";
import { useToast } from "@/contexts/ToastContext";
import {
  teamApi,
  type TeamInvitation,
  type TeamMember,
  type TeamRole,
} from "@/lib/api";
import { TeamSettingsView } from "./TeamSettingsView";

type InviteRole = Exclude<TeamRole, "owner">;

function getMemberID(member: TeamMember) {
  return member.user_id;
}

export default function TeamTab() {
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [pendingInvitations, setPendingInvitations] = useState<TeamInvitation[]>(
    [],
  );
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<InviteRole>("member");
  const [loading, setLoading] = useState(true);
  const [savingInvite, setSavingInvite] = useState(false);
  const [busyID, setBusyID] = useState<string | null>(null);
  const toast = useToast();

  async function loadTeam() {
    const [memberRes, invitationRes] = await Promise.all([
      teamApi.listMembers(),
      teamApi.listInvitations(),
    ]);
    setMembers(memberRes.data.members ?? []);
    setPendingInvitations(invitationRes.data ?? []);
  }

  useEffect(() => {
    async function load() {
      try {
        await loadTeam();
      } catch {
        toast.error("Failed to load team settings");
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  async function handleInviteSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingInvite(true);
    try {
      const { data } = await teamApi.createInvitation({
        email: inviteEmail.trim(),
        role: inviteRole,
      });
      setPendingInvitations((prev) => [data, ...prev]);
      setInviteEmail("");
      toast.success("Invitation sent");
    } catch {
      toast.error("Failed to send invitation");
    } finally {
      setSavingInvite(false);
    }
  }

  async function handleRoleChange(member: TeamMember, role: TeamRole) {
    const userID = getMemberID(member);
    setBusyID(userID);
    try {
      await teamApi.updateRole(userID, role);
      setMembers((prev) =>
        prev.map((item) =>
          getMemberID(item) === userID ? { ...item, role } : item,
        ),
      );
      toast.success("Role updated");
    } catch {
      toast.error("Failed to update role");
    } finally {
      setBusyID(null);
    }
  }

  async function handleRemoveMember(member: TeamMember) {
    const name = `${member.first_name} ${member.last_name}`.trim() || member.email;
    if (!window.confirm(`Remove ${name} from this organization?`)) return;

    const userID = getMemberID(member);
    setBusyID(userID);
    try {
      await teamApi.removeMember(userID);
      setMembers((prev) => prev.filter((item) => getMemberID(item) !== userID));
      toast.success("Member removed");
    } catch {
      toast.error("Failed to remove member");
    } finally {
      setBusyID(null);
    }
  }

  async function handleResendInvitation(invitation: TeamInvitation) {
    setBusyID(invitation.id);
    try {
      await teamApi.revokeInvitation(invitation.id);
      const { data } = await teamApi.createInvitation({
        email: invitation.email,
        role: invitation.role as InviteRole,
      });
      setPendingInvitations((prev) =>
        prev.map((item) => (item.id === invitation.id ? data : item)),
      );
      toast.success("Invitation resent");
    } catch {
      toast.error("Failed to resend invitation");
    } finally {
      setBusyID(null);
    }
  }

  async function handleRevokeInvitation(invitation: TeamInvitation) {
    setBusyID(invitation.id);
    try {
      await teamApi.revokeInvitation(invitation.id);
      setPendingInvitations((prev) =>
        prev.filter((item) => item.id !== invitation.id),
      );
      toast.success("Invitation revoked");
    } catch {
      toast.error("Failed to revoke invitation");
    } finally {
      setBusyID(null);
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--galdr-fg-muted)]" />
      </div>
    );
  }

  return (
    <TeamSettingsView
      members={members}
      pendingInvitations={pendingInvitations}
      inviteEmail={inviteEmail}
      inviteRole={inviteRole}
      busyID={busyID}
      savingInvite={savingInvite}
      onInviteEmailChange={setInviteEmail}
      onInviteRoleChange={setInviteRole}
      onInviteSubmit={handleInviteSubmit}
      onRoleChange={handleRoleChange}
      onRemoveMember={handleRemoveMember}
      onResendInvitation={handleResendInvitation}
      onRevokeInvitation={handleRevokeInvitation}
    />
  );
}
