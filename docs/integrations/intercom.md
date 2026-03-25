# Intercom Integration Guide

This guide explains how to connect your Intercom account to PulseScore, what data is synced, which permissions are required, and how Intercom data feeds into customer health scores.

---

## Prerequisites

Before connecting Intercom you will need:

- A PulseScore account with **admin** or **owner** role (required to manage integrations).
- An Intercom workspace with at least one of the following: contacts, conversations, or companies.
- Your Intercom account must be on a **Starter** plan or higher. The free developer workspace is supported for testing purposes.

---

## Connecting Intercom

### Step 1 — Open the Integrations settings

1. Log in to PulseScore.
2. Click **Settings** in the left navigation bar.
3. Click **Integrations** in the Settings sub-menu.

```
[Screenshot placeholder: Settings → Integrations page showing available integration tiles]
```

---

### Step 2 — Start the OAuth flow

1. Locate the **Intercom** tile on the Integrations page.
2. Click **Connect Intercom**.
3. PulseScore redirects you to the Intercom OAuth authorization page (`https://app.intercom.com/oauth`).

> 💡 **Tip:** If nothing happens when you click **Connect Intercom**, check that your browser is not blocking the redirect. PulseScore navigates away from the current page rather than opening a pop-up window.

```
[Screenshot placeholder: Intercom tile with "Connect Intercom" button highlighted]
```

---

### Step 3 — Authorize access in Intercom

1. If prompted, log in to Intercom.
2. Select the Intercom workspace you want to connect to PulseScore.
3. Review the permissions summary (read-only access — see [Permissions](#permissions) below).
4. Click **Authorize access**.

```
[Screenshot placeholder: Intercom OAuth consent screen with "Authorize access" button]
```

---

### Step 4 — Confirm the connection

After authorization, Intercom redirects you back to PulseScore.

- A green ✓ badge appears on the Intercom tile: **Connected**.
- The initial data sync starts automatically in the background.

```
[Screenshot placeholder: Integrations page with Intercom tile showing "Connected" badge]
```

---

### Step 5 — Wait for the initial sync

PulseScore imports your Intercom contacts and conversations.

1. A progress banner appears at the top of the screen: *"Syncing Intercom data…"*
2. The sync typically completes in **60–120 seconds** for most workspaces.
3. The banner updates to *"Sync complete — X contacts imported."* when finished.

> ⏱ **Large workspaces:** Workspaces with a high volume of conversations may take 3–8 minutes for the initial sync. Subsequent syncs are incremental and much faster.

```
[Screenshot placeholder: Dashboard with "Syncing Intercom data…" progress banner]
```

---

## Permissions

PulseScore requests **read-only** access to your Intercom workspace using the following OAuth scopes:

| Scope | Purpose |
|---|---|
| `read_contacts` | Read contact records and attributes |
| `read_conversations` | Read conversation history and metadata |
| `read_companies` | Read company records and attributes |
| `read_tags` | Read tags applied to contacts and conversations |

| Permission | Granted |
|---|---|
| Read contacts | ✅ Yes |
| Read conversations | ✅ Yes |
| Read companies | ✅ Yes |
| Read tags | ✅ Yes |
| Create or modify contacts | ❌ No |
| Send messages or reply to conversations | ❌ No |
| Delete contacts or conversations | ❌ No |
| Access billing or team inbox settings | ❌ No |

PulseScore cannot and will never send messages or modify any data in your Intercom workspace.

---

## Data synced

The following Intercom objects are imported into PulseScore during every sync.

### Contacts

Each Intercom contact is mapped to a PulseScore **Customer** record.

| Intercom field | PulseScore field | Notes |
|---|---|---|
| `id` | `external_id` | Used to match records on re-sync |
| `email` | `email` | |
| `name` | `name` | |
| `role` | `contact_role` | `user` or `lead` |
| `created_at` | `first_seen_at` | |
| `last_seen_at` | `last_seen_at` | Last time contact was active |
| `last_replied_at` | `last_replied_at` | Last time contact replied in a conversation |
| `tags.data` | `tags` | Array of tag names |
| `custom_attributes` | `metadata` | Full custom attributes map preserved |

### Conversations

Each Intercom conversation is stored and linked to its associated contact.

| Intercom field | PulseScore field | Notes |
|---|---|---|
| `id` | `external_id` | |
| `state` | `status` | `open`, `closed`, or `snoozed` |
| `created_at` | `created_at` | |
| `updated_at` | `updated_at` | |
| `statistics.time_to_first_response_secs` | `time_to_first_response_secs` | Support response speed |
| `statistics.time_to_last_close_secs` | `time_to_resolution_secs` | Total time to close conversation |
| `statistics.count_reopens` | `reopen_count` | How many times conversation was re-opened |
| `statistics.count_assignments` | `assignment_count` | How many times conversation was reassigned |
| `tags.data` | `tags` | Array of tag names |

> **Note:** Only conversations associated with a known Intercom contact are synced. Conversations without a linked contact are skipped.

### Sync frequency

| Sync type | Trigger | Coverage |
|---|---|---|
| Initial full sync | Immediately after OAuth connection | All contacts and last 90 days of conversations |
| Incremental sync | On incoming Intercom webhook events | New/updated objects only |
| Manual re-sync | *Settings → Integrations → Intercom → Retry sync* | Full re-import |

---

## How Intercom data maps to health scores

PulseScore computes a **health score (0–100)** for each customer. Several score factors are derived directly from Intercom data.

### Score factors

| Factor | Source data | What it measures | Default weight |
|---|---|---|---|
| **Support recency** | `last_seen_at` and latest conversation `updated_at` | How recently the customer interacted with support | 20% |
| **Open ticket volume** | Count of conversations with `state = open` | Number of unresolved support issues | 25% |
| **Resolution time** | `time_to_resolution_secs` across recent conversations | Average speed at which issues are resolved | 20% |
| **Reopen rate** | `reopen_count` across recent conversations | Frequency with which issues require re-opening, indicating poor first resolution | 20% |
| **Response rate** | Contact `last_replied_at` relative to latest conversation | Whether the customer is actively engaging with support responses | 15% |

### Risk levels

Scores are mapped to color-coded risk levels:

| Score range | Risk level | Color |
|---|---|---|
| 70–100 | Healthy | 🟢 Green |
| 40–69 | At risk | 🟡 Yellow |
| 0–39 | Critical | 🔴 Red |

> **Tip:** Score weights and thresholds are configurable per organization. See *Settings → Scoring → Configuration* or the [API Reference](../api-reference.md#put-scoringconfig).

### Score calculation example

Consider a customer with these Intercom signals:

- Last seen: **1 day ago** → support recency score high
- Open tickets: **0** → no penalty for unresolved issues
- Average resolution time: **4 hours** → resolution time score high
- Reopen count in last 30 days: **0** → no reopen penalty

This customer would receive a **high health score** (🟢 green), indicating strong support satisfaction.

Now consider a customer where:

- Last seen: **30 days ago**
- Open tickets: **4** (multiple unresolved conversations)
- Average resolution time: **72 hours**
- Reopen count in last 30 days: **3**

This customer would receive a **low health score** (🔴 red), triggering support-escalation alerts.

---

## Webhook events

In addition to periodic syncs, PulseScore listens to Intercom webhook events in real time to keep data up to date. The following Intercom event types are processed:

| Intercom event | PulseScore action |
|---|---|
| `contact.created` | Create new customer record |
| `contact.updated` | Update contact fields and recalculate score |
| `contact.deleted` | Mark customer record as deleted |
| `conversation.created` | Create new conversation linked to customer |
| `conversation.updated` | Update conversation fields and recalculate score |
| `conversation.closed` | Mark conversation closed; trigger score recalculation |
| `conversation.reopened` | Increment reopen count; apply reopen penalty |
| `conversation.assigned` | Record assignment change |

Webhook events are verified using your **Intercom webhook secret** before being processed.

---

## Disconnecting Intercom

To remove the Intercom integration:

1. Go to *Settings → Integrations*.
2. Click the **⋮** menu on the Intercom tile.
3. Select **Disconnect**.
4. Confirm the disconnection in the dialog.

> ⚠️ **Warning:** Disconnecting Intercom stops all future syncs. Existing customer records and scores are retained but will no longer be updated from Intercom data.

---

## Troubleshooting

### "Connect Intercom" button does nothing

- **Browser redirect blocked:** PulseScore navigates away from the current page to open Intercom OAuth. Ensure no browser extension is intercepting the navigation.
- **Already connected:** If Intercom was connected in a previous session, disconnect it first (*Settings → Integrations → ⋮ → Disconnect*) then reconnect.

### OAuth error after clicking "Authorize access"

| Error message | Likely cause | Solution |
|---|---|---|
| `invalid_client` | Incorrect Intercom Client ID configured | Contact support at support@pulsescore.app |
| `access_denied` | User clicked "Cancel" or denied access | Restart the OAuth flow and click **Authorize access** |
| `redirect_uri_mismatch` | OAuth redirect URL mismatch | Contact support — this is a configuration issue on the PulseScore side |

### Sync completed but 0 contacts imported

- Confirm you authorized the **correct** Intercom workspace. The connected workspace ID is shown in *Settings → Integrations → Intercom*.
- Verify your Intercom workspace has contacts in the [Intercom Contacts](https://app.intercom.com/contacts) view.
- Contacts without an `email` address are skipped — PulseScore requires a valid email to create a customer record. Check if your contacts are stored as anonymous visitors.

### Some contacts are missing

- PulseScore syncs all contacts with a valid email. Anonymous visitors (contacts with no email) are not synced.
- If specific contacts are missing, check whether they have been deleted or merged in Intercom.

### Sync is taking longer than 8 minutes

1. Navigate to *Settings → Integrations → Intercom* and check the **Last sync** status.
2. Click **Retry sync** to trigger a fresh import.
3. If the sync status shows an error, note the error message and contact support at support@pulsescore.app.

### Health scores show "N/A" after sync

- Scores require at least **one conversation** per contact to be computed.
- Contacts imported without any conversation history will receive scores once support activity arrives via webhook or a subsequent sync.

### Webhook events are not being received

1. Confirm the Intercom webhook is configured in your [Intercom Developer Hub](https://developers.intercom.com/).
2. The webhook endpoint URL should be `https://pulsescore.app/api/v1/webhooks/intercom`.
3. Ensure subscriptions are enabled for the event types listed in the [Webhook events](#webhook-events) table above.
4. Verify the **Webhook secret** is correctly configured in PulseScore (*Settings → Integrations → Intercom → Webhook secret*).

---

## Getting help

| Channel | Details |
|---|---|
| **In-app chat** | Click the **?** icon in the bottom-right corner |
| **Email support** | support@pulsescore.app |
| **Status page** | https://status.pulsescore.app |
