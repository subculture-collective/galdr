import { useCallback, useEffect, useState } from "react";
import { Plus, UserRound, X } from "lucide-react";
import api from "@/lib/api";
import { relativeTime } from "@/lib/format";
import { useToast } from "@/contexts/ToastContext";

export interface CustomerAssignmentUser {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
}

export interface CustomerAssignment {
  customer_id: string;
  user_id: string;
  assignee: CustomerAssignmentUser;
  assigned_at: string;
  assigned_by: string;
}

interface CustomerAssignmentsResponse {
  assignments: CustomerAssignment[];
}

interface Member {
  user_id: string;
  email: string;
  first_name: string;
  last_name: string;
  avatar_url?: string;
}

interface AccountAssignmentProps {
  customerId: string;
  initialAssignments?: CustomerAssignment[];
}

const EMPTY_ASSIGNMENTS: CustomerAssignment[] = [];

export default function AccountAssignment({
  customerId,
  initialAssignments = EMPTY_ASSIGNMENTS,
}: AccountAssignmentProps) {
  const [assignments, setAssignments] =
    useState<CustomerAssignment[]>(initialAssignments);
  const [members, setMembers] = useState<Member[]>([]);
  const [selectedUserId, setSelectedUserId] = useState("");
  const [saving, setSaving] = useState(false);
  const toast = useToast();

  const loadAssignments = useCallback(async () => {
    try {
      const { data } = await api.get<CustomerAssignmentsResponse>(
        `/customers/${customerId}/assignments`,
      );
      setAssignments(data.assignments);
    } catch {
      toast.error("Failed to load account assignments");
    }
  }, [customerId, toast]);

  useEffect(() => {
    setAssignments(initialAssignments);
  }, [initialAssignments]);

  useEffect(() => {
    async function loadMembers() {
      try {
        const { data } = await api.get<{ members: Member[] }>("/members");
        setMembers(data.members);
      } catch {
        toast.error("Failed to load team members");
      }
    }
    loadMembers();
    loadAssignments();
  }, [loadAssignments, toast]);

  async function assignUser() {
    if (!selectedUserId) return;
    setSaving(true);
    try {
      const { data } = await api.post<CustomerAssignment>(
        `/customers/${customerId}/assignments`,
        { user_id: selectedUserId },
      );
      setAssignments((current) => [
        data,
        ...current.filter((item) => item.user_id !== data.user_id),
      ]);
      setSelectedUserId("");
      toast.success("Assignee added");
    } catch {
      toast.error("Failed to assign customer");
    } finally {
      setSaving(false);
    }
  }

  async function unassignUser(userId: string) {
    setSaving(true);
    try {
      await api.delete(`/customers/${customerId}/assignments/${userId}`);
      setAssignments((current) =>
        current.filter((item) => item.user_id !== userId),
      );
      toast.success("Assignee removed");
    } catch {
      toast.error("Failed to remove assignee");
    } finally {
      setSaving(false);
    }
  }

  const assignedIDs = new Set(assignments.map((item) => item.user_id));
  const availableMembers = members.filter(
    (member) => !assignedIDs.has(member.user_id),
  );

  return (
    <section className="galdr-card p-6">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-medium text-[var(--galdr-fg)]">
            Account assignment
          </h3>
          <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
            Assign team members who own this customer relationship.
          </p>
        </div>
        <div className="flex gap-2">
          <select
            value={selectedUserId}
            onChange={(event) => setSelectedUserId(event.target.value)}
            className="galdr-input px-3 py-2 text-sm"
          >
            <option value="">Select member</option>
            {availableMembers.map((member) => (
              <option key={member.user_id} value={member.user_id}>
                {memberName(member)}
              </option>
            ))}
          </select>
          <button
            onClick={assignUser}
            disabled={saving || !selectedUserId}
            className="galdr-button-primary disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Plus className="h-4 w-4" /> Assign
          </button>
        </div>
      </div>

      {assignments.length === 0 ? (
        <div className="rounded-xl border border-dashed border-[var(--galdr-border)] p-5 text-sm text-[var(--galdr-fg-muted)]">
          No assignees yet.
        </div>
      ) : (
        <div className="flex flex-wrap gap-3">
          {assignments.map((assignment) => (
            <div
              key={assignment.user_id}
              className="flex items-center gap-3 rounded-2xl border border-[var(--galdr-border)] bg-[var(--galdr-surface-soft)] px-3 py-2"
            >
              <Avatar user={assignment.assignee} />
              <div>
                <p className="text-sm font-medium text-[var(--galdr-fg)]">
                  {assignment.assignee.name || assignment.assignee.email}
                </p>
                <p className="text-xs text-[var(--galdr-fg-muted)]">
                  Assigned {relativeTime(assignment.assigned_at)}
                </p>
              </div>
              <button
                onClick={() => unassignUser(assignment.user_id)}
                disabled={saving}
                className="rounded-full p-1 text-[var(--galdr-fg-muted)] transition hover:bg-[var(--galdr-surface)] hover:text-[var(--galdr-danger)] disabled:opacity-50"
                aria-label="Remove assignee"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function Avatar({ user }: { user: CustomerAssignmentUser }) {
  if (user.avatar_url) {
    return (
      <img
        src={user.avatar_url}
        alt=""
        className="h-9 w-9 rounded-full object-cover"
      />
    );
  }
  return (
    <div className="flex h-9 w-9 items-center justify-center rounded-full bg-[color:rgb(139_92_246_/_0.18)] text-xs font-semibold text-[var(--galdr-accent)]">
      {initials(user.name || user.email) || <UserRound className="h-4 w-4" />}
    </div>
  );
}

function memberName(member: Member) {
  const name = `${member.first_name ?? ""} ${member.last_name ?? ""}`.trim();
  return name || member.email;
}

function initials(value: string) {
  return value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("");
}
