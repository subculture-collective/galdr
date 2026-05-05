import React from "react";
import { renderToStaticMarkup } from "react-dom/server";

import { TeamSettingsView } from "./TeamSettingsView";

const noop = () => undefined;

function render() {
  return renderToStaticMarkup(
    React.createElement(TeamSettingsView, {
      members: [
        {
          user_id: "user-1",
          email: "ada@example.com",
          first_name: "Ada",
          last_name: "Lovelace",
          role: "owner",
          joined_at: "2026-01-01T00:00:00Z",
        },
        {
          user_id: "user-2",
          email: "grace@example.com",
          first_name: "Grace",
          last_name: "Hopper",
          role: "member",
          joined_at: "2026-01-02T00:00:00Z",
        },
      ],
      pendingInvitations: [
        {
          id: "invite-1",
          email: "pending@example.com",
          role: "admin",
          status: "pending",
          expires_at: "2026-02-01T00:00:00Z",
          created_at: "2026-01-01T00:00:00Z",
        },
      ],
      inviteEmail: "",
      inviteRole: "member",
      busyID: null,
      savingInvite: false,
      onInviteEmailChange: noop,
      onInviteRoleChange: noop,
      onInviteSubmit: noop,
      onRoleChange: noop,
      onRemoveMember: noop,
      onResendInvitation: noop,
      onRevokeInvitation: noop,
    }),
  );
}

function assertMatch(input: string, pattern: RegExp) {
  if (!pattern.test(input)) {
    throw new Error(`Expected ${pattern} to match ${input}`);
  }
}

const html = render();
assertMatch(html, /Invite team member/);
assertMatch(html, /ada@example\.com/);
assertMatch(html, /grace@example\.com/);
assertMatch(html, /Active members/);
assertMatch(html, /active/);
assertMatch(html, /Pending invites/);
assertMatch(html, /Status: pending/);
assertMatch(html, /pending@example\.com/);
assertMatch(html, /Change role for Grace Hopper/);
assertMatch(html, /Remove Grace Hopper/);
assertMatch(html, /Resend invite to pending@example\.com/);
assertMatch(html, /Revoke invite to pending@example\.com/);
