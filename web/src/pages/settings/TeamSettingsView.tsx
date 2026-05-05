import React, { type FormEvent } from "react";
import { MailPlus, RefreshCw, Trash2, UserMinus, Users } from "lucide-react";

void React;

export type TeamRole = "owner" | "admin" | "member";

export interface TeamMember {
  user_id: string;
  email: string;
  first_name: string;
  last_name: string;
  avatar_url?: string;
  role: TeamRole | string;
  joined_at: string;
}

export interface TeamInvitation {
  id: string;
  email: string;
  role: Exclude<TeamRole, "owner"> | string;
  status: string;
  expires_at: string;
  created_at: string;
}

interface TeamSettingsViewProps {
  members: TeamMember[];
  pendingInvitations: TeamInvitation[];
  inviteEmail: string;
  inviteRole: Exclude<TeamRole, "owner">;
  busyID: string | null;
  savingInvite: boolean;
  onInviteEmailChange: (value: string) => void;
  onInviteRoleChange: (value: Exclude<TeamRole, "owner">) => void;
  onInviteSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onRoleChange: (member: TeamMember, role: TeamRole) => void;
  onRemoveMember: (member: TeamMember) => void;
  onResendInvitation: (invitation: TeamInvitation) => void;
  onRevokeInvitation: (invitation: TeamInvitation) => void;
}

const ROLE_OPTIONS: TeamRole[] = ["owner", "admin", "member"];
const INVITE_ROLE_OPTIONS: Exclude<TeamRole, "owner">[] = ["member", "admin"];

function displayName(member: TeamMember) {
  const name = `${member.first_name} ${member.last_name}`.trim();
  return name || member.email;
}

function memberID(member: TeamMember) {
  return member.user_id;
}

function formatDate(value: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(new Date(value));
}

function rolePill(role: string) {
  const cls =
    role === "owner"
      ? "border-amber-400/40 bg-amber-400/10 text-amber-200"
      : role === "admin"
        ? "border-cyan-400/40 bg-cyan-400/10 text-cyan-200"
        : "border-[var(--galdr-border)] bg-[color:rgb(255_255_255_/_0.04)] text-[var(--galdr-fg-muted)]";

  return (
    <span
      className={`inline-flex rounded-full border px-2.5 py-0.5 text-xs font-medium capitalize ${cls}`}
    >
      {role}
    </span>
  );
}

export function TeamSettingsView({
  members,
  pendingInvitations,
  inviteEmail,
  inviteRole,
  busyID,
  savingInvite,
  onInviteEmailChange,
  onInviteRoleChange,
  onInviteSubmit,
  onRoleChange,
  onRemoveMember,
  onResendInvitation,
  onRevokeInvitation,
}: TeamSettingsViewProps) {
  return (
    <div className="space-y-6">
      <section className="galdr-card p-5">
        <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div>
            <div className="flex items-center gap-2 text-[var(--galdr-fg)]">
              <MailPlus className="h-5 w-5 text-[var(--galdr-accent)]" />
              <h2 className="text-lg font-semibold">Invite team member</h2>
            </div>
            <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
              Send a join link and assign the right access level.
            </p>
          </div>
        </div>

        <form
          onSubmit={onInviteSubmit}
          className="mt-5 grid gap-3 md:grid-cols-[minmax(0,1fr)_180px_auto]"
        >
          <label className="sr-only" htmlFor="team-invite-email">
            Email address
          </label>
          <input
            id="team-invite-email"
            className="galdr-input px-3 py-2 text-sm"
            type="email"
            required
            value={inviteEmail}
            onChange={(event) => onInviteEmailChange(event.target.value)}
            placeholder="teammate@example.com"
          />

          <label className="sr-only" htmlFor="team-invite-role">
            Invite role
          </label>
          <select
            id="team-invite-role"
            className="galdr-input px-3 py-2 text-sm capitalize"
            value={inviteRole}
            onChange={(event) =>
              onInviteRoleChange(event.target.value as Exclude<TeamRole, "owner">)
            }
          >
            {INVITE_ROLE_OPTIONS.map((role) => (
              <option key={role} value={role}>
                {role}
              </option>
            ))}
          </select>

          <button
            type="submit"
            disabled={savingInvite}
            className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
          >
            {savingInvite ? "Sending..." : "Send invite"}
          </button>
        </form>
      </section>

      <section className="galdr-card overflow-hidden">
        <div className="flex items-center justify-between border-b border-[var(--galdr-border)] px-5 py-4">
          <div className="flex items-center gap-2">
            <Users className="h-5 w-5 text-[var(--galdr-accent)]" />
            <h2 className="text-lg font-semibold text-[var(--galdr-fg)]">
              Active members
            </h2>
          </div>
          <span className="text-sm text-[var(--galdr-fg-muted)]">
            {members.length} total
          </span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-[color:rgb(31_31_46_/_0.72)] text-xs uppercase text-[var(--galdr-fg-muted)]">
              <tr>
                <th className="px-5 py-3">Name</th>
                <th className="px-5 py-3">Email</th>
                <th className="px-5 py-3">Role</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Joined</th>
                <th className="px-5 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {members.map((member) => {
                const name = displayName(member);
                return (
                  <tr
                    key={memberID(member)}
                    className="border-t border-[var(--galdr-border)]/70"
                  >
                    <td className="px-5 py-4 font-medium text-[var(--galdr-fg)]">
                      {name}
                    </td>
                    <td className="px-5 py-4 text-[var(--galdr-fg-muted)]">
                      {member.email}
                    </td>
                    <td className="px-5 py-4">
                      <label className="sr-only" htmlFor={`role-${memberID(member)}`}>
                        Change role for {name}
                      </label>
                      {member.role === "owner" ? (
                        rolePill(member.role)
                      ) : (
                        <select
                          id={`role-${memberID(member)}`}
                          aria-label={`Change role for ${name}`}
                          className="galdr-input px-2 py-1 text-xs capitalize"
                          value={member.role}
                          disabled={busyID === memberID(member)}
                          onChange={(event) =>
                            onRoleChange(member, event.target.value as TeamRole)
                          }
                        >
                          {ROLE_OPTIONS.filter((role) => role !== "owner").map(
                            (role) => (
                              <option key={role} value={role}>
                                {role}
                              </option>
                            ),
                          )}
                        </select>
                      )}
                    </td>
                    <td className="px-5 py-4">
                      <span className="inline-flex rounded-full border border-emerald-400/40 bg-emerald-400/10 px-2.5 py-0.5 text-xs font-medium text-emerald-200">
                        active
                      </span>
                    </td>
                    <td className="px-5 py-4 text-[var(--galdr-fg-muted)]">
                      {formatDate(member.joined_at)}
                    </td>
                    <td className="px-5 py-4 text-right">
                      <button
                        type="button"
                        aria-label={`Remove ${name}`}
                        disabled={member.role === "owner" || busyID === memberID(member)}
                        onClick={() => onRemoveMember(member)}
                        className="galdr-button-secondary inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium disabled:cursor-not-allowed disabled:opacity-40"
                      >
                        <UserMinus className="h-3.5 w-3.5" />
                        Remove
                      </button>
                    </td>
                  </tr>
                );
              })}
              {members.length === 0 && (
                <tr>
                  <td
                    colSpan={6}
                    className="px-5 py-8 text-center text-[var(--galdr-fg-muted)]"
                  >
                    No active members found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      <section className="galdr-card overflow-hidden">
        <div className="border-b border-[var(--galdr-border)] px-5 py-4">
          <h2 className="text-lg font-semibold text-[var(--galdr-fg)]">
            Pending invites
          </h2>
          <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
            Resend stale links or revoke invites that should no longer be valid.
          </p>
        </div>

        <div className="divide-y divide-[var(--galdr-border)]/70">
          {pendingInvitations.map((invitation) => (
            <div
              key={invitation.id}
              className="flex flex-col gap-3 px-5 py-4 md:flex-row md:items-center md:justify-between"
            >
              <div>
                <div className="font-medium text-[var(--galdr-fg)]">
                  {invitation.email}
                </div>
                <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-[var(--galdr-fg-muted)]">
                  {rolePill(invitation.role)}
                  <span>Status: {invitation.status}</span>
                  <span>Expires {formatDate(invitation.expires_at)}</span>
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  aria-label={`Resend invite to ${invitation.email}`}
                  disabled={busyID === invitation.id}
                  onClick={() => onResendInvitation(invitation)}
                  className="galdr-button-secondary inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium disabled:opacity-50"
                >
                  <RefreshCw className="h-3.5 w-3.5" />
                  Resend
                </button>
                <button
                  type="button"
                  aria-label={`Revoke invite to ${invitation.email}`}
                  disabled={busyID === invitation.id}
                  onClick={() => onRevokeInvitation(invitation)}
                  className="galdr-button-secondary inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium text-red-200 disabled:opacity-50"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  Revoke
                </button>
              </div>
            </div>
          ))}
          {pendingInvitations.length === 0 && (
            <div className="px-5 py-8 text-center text-sm text-[var(--galdr-fg-muted)]">
              No pending invitations.
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
