# HubSpot Integration Guide

This guide explains how to connect your HubSpot account to PulseScore, what data is synced, which permissions are required, and how HubSpot data feeds into customer health scores.

---

## Prerequisites

Before connecting HubSpot you will need:

- A PulseScore account with **admin** or **owner** role (required to manage integrations).
- A HubSpot account with at least one of the following: contacts, deals, or companies.
- Your HubSpot account must be in a **Professional** or **Enterprise** tier to access the full API scope required by PulseScore. Starter accounts are supported but some deal and company fields may be unavailable.

---

## Connecting HubSpot

### Step 1 — Open the Integrations settings

1. Log in to PulseScore.
2. Click **Settings** in the left navigation bar.
3. Click **Integrations** in the Settings sub-menu.

```
[Screenshot placeholder: Settings → Integrations page showing available integration tiles]
```

---

### Step 2 — Start the OAuth flow

1. Locate the **HubSpot** tile on the Integrations page.
2. Click **Connect HubSpot**.
3. PulseScore redirects you to the HubSpot OAuth authorization page (`https://app.hubspot.com/oauth`).

> 💡 **Tip:** If nothing happens when you click **Connect HubSpot**, check that your browser is not blocking the redirect. PulseScore navigates away from the current page rather than opening a pop-up window.

```
[Screenshot placeholder: HubSpot tile with "Connect HubSpot" button highlighted]
```

---

### Step 3 — Authorize access in HubSpot

1. If prompted, log in to HubSpot.
2. Select the HubSpot account (portal) you want to connect to PulseScore.
3. Review the permissions summary (read-only access — see [Permissions](#permissions) below).
4. Click **Grant access**.

```
[Screenshot placeholder: HubSpot OAuth consent screen with "Grant access" button]
```

---

### Step 4 — Confirm the connection

After authorization, HubSpot redirects you back to PulseScore.

- A green ✓ badge appears on the HubSpot tile: **Connected**.
- The initial data sync starts automatically in the background.

```
[Screenshot placeholder: Integrations page with HubSpot tile showing "Connected" badge]
```

---

### Step 5 — Wait for the initial sync

PulseScore imports your HubSpot contacts, deals, and companies.

1. A progress banner appears at the top of the screen: *"Syncing HubSpot data…"*
2. The sync typically completes in **60–120 seconds** for most accounts.
3. The banner updates to *"Sync complete — X contacts imported."* when finished.

> ⏱ **Large accounts:** Portals with tens of thousands of contacts may take 3–8 minutes for the initial sync. Subsequent syncs are incremental and much faster.

```
[Screenshot placeholder: Dashboard with "Syncing HubSpot data…" progress banner]
```

---

## Permissions

PulseScore requests **read-only** access to your HubSpot account using the following OAuth scopes:

| Scope | Purpose |
|---|---|
| `crm.objects.contacts.read` | Read contact records and properties |
| `crm.objects.deals.read` | Read deal records and pipeline stages |
| `crm.objects.companies.read` | Read company records and properties |
| `crm.objects.owners.read` | Read owner (rep) assignments |

| Permission | Granted |
|---|---|
| Read contacts | ✅ Yes |
| Read deals | ✅ Yes |
| Read companies | ✅ Yes |
| Read owner assignments | ✅ Yes |
| Create or modify contacts | ❌ No |
| Create or modify deals | ❌ No |
| Send emails or enroll contacts in sequences | ❌ No |
| Access billing or subscription data | ❌ No |

PulseScore cannot and will never write to or modify any data in your HubSpot portal.

---

## Data synced

The following HubSpot objects are imported into PulseScore during every sync.

### Contacts

Each HubSpot contact is mapped to a PulseScore **Customer** record.

| HubSpot field | PulseScore field | Notes |
|---|---|---|
| `hs_object_id` | `external_id` | Used to match records on re-sync |
| `email` | `email` | |
| `firstname` + `lastname` | `name` | Concatenated |
| `company` | `company_name` | |
| `hs_lead_status` | `lead_status` | e.g. `NEW`, `OPEN`, `IN_PROGRESS` |
| `hs_last_sales_activity_date` | `last_activity_at` | |
| `createdate` | `first_seen_at` | |
| `lifecyclestage` | `lifecycle_stage` | e.g. `lead`, `customer`, `evangelist` |

### Deals

Each HubSpot deal is stored and linked to its associated contact and company.

| HubSpot field | PulseScore field | Notes |
|---|---|---|
| `hs_object_id` | `external_id` | |
| `dealname` | `name` | |
| `dealstage` | `stage` | Pipeline stage key |
| `amount` | `amount_cents` | Converted to cents |
| `closedate` | `closed_at` | |
| `hs_deal_stage_probability` | `close_probability` | 0–1 float |
| `pipeline` | `pipeline` | Pipeline name |

### Companies

Each HubSpot company is stored and linked to associated contacts and deals.

| HubSpot field | PulseScore field | Notes |
|---|---|---|
| `hs_object_id` | `external_id` | |
| `name` | `name` | |
| `domain` | `domain` | |
| `industry` | `industry` | |
| `numberofemployees` | `employee_count` | |
| `annualrevenue` | `annual_revenue_cents` | Converted to cents |
| `hs_last_sales_activity_date` | `last_activity_at` | |

### Sync frequency

| Sync type | Trigger | Coverage |
|---|---|---|
| Initial full sync | Immediately after OAuth connection | All contacts, deals, and companies |
| Incremental sync | On incoming HubSpot webhook events | New/updated objects only |
| Manual re-sync | *Settings → Integrations → HubSpot → Retry sync* | Full re-import |

---

## How HubSpot data maps to health scores

PulseScore computes a **health score (0–100)** for each customer. Several score factors are derived directly from HubSpot data.

### Score factors

| Factor | Source data | What it measures | Default weight |
|---|---|---|---|
| **Engagement recency** | `hs_last_sales_activity_date` (contacts and companies) | How recently the customer had a logged sales activity | 25% |
| **Deal momentum** | Deal `dealstage` and `hs_deal_stage_probability` | Whether open deals are progressing toward close | 20% |
| **Lifecycle stage** | Contact `lifecyclestage` | How advanced the contact is in the customer journey | 15% |
| **Activity frequency** | Count of deal and contact updates in the last 30 days | Cadence of engagement and CRM activity | 20% |
| **Deal value** | Deal `amount_cents` relative to account average | Size signal for revenue-weighted scoring | 20% |

### Risk levels

Scores are mapped to color-coded risk levels:

| Score range | Risk level | Color |
|---|---|---|
| 70–100 | Healthy | 🟢 Green |
| 40–69 | At risk | 🟡 Yellow |
| 0–39 | Critical | 🔴 Red |

> **Tip:** Score weights and thresholds are configurable per organization. See *Settings → Scoring → Configuration* or the [API Reference](../api-reference.md#put-scoringconfig).

### Score calculation example

Consider a customer with these HubSpot signals:

- Last sales activity: **2 days ago** → engagement recency score high
- Open deal at **Proposal** stage with 60% close probability → deal momentum positive
- Lifecycle stage: **customer** → lifecycle bonus applied
- 8 contact/deal updates in the last 30 days → activity frequency high

This customer would receive a **high health score** (🟢 green), indicating strong engagement.

Now consider a customer where:

- Last sales activity: **60 days ago**
- No open deals
- Lifecycle stage: **lead** (has not converted)
- 0 updates in the last 30 days

This customer would receive a **low health score** (🔴 red), triggering re-engagement alerts.

---

## Webhook events

In addition to periodic syncs, PulseScore listens to HubSpot webhook events in real time to keep data up to date. The following HubSpot event types are processed:

| HubSpot event | PulseScore action |
|---|---|
| `contact.creation` | Create new customer record |
| `contact.propertyChange` | Update contact fields and recalculate score |
| `contact.deletion` | Mark customer record as deleted |
| `deal.creation` | Create new deal linked to customer |
| `deal.propertyChange` | Update deal fields and recalculate score |
| `deal.deletion` | Remove deal from customer record |
| `company.creation` | Create new company record |
| `company.propertyChange` | Update company fields |

Webhook events are verified using your **HubSpot webhook client secret** before being processed.

---

## Disconnecting HubSpot

To remove the HubSpot integration:

1. Go to *Settings → Integrations*.
2. Click the **⋮** menu on the HubSpot tile.
3. Select **Disconnect**.
4. Confirm the disconnection in the dialog.

> ⚠️ **Warning:** Disconnecting HubSpot stops all future syncs. Existing customer records and scores are retained but will no longer be updated from HubSpot data.

---

## Troubleshooting

### "Connect HubSpot" button does nothing

- **Browser redirect blocked:** PulseScore navigates away from the current page to open HubSpot OAuth. Ensure no browser extension is intercepting the navigation.
- **Already connected:** If HubSpot was connected in a previous session, disconnect it first (*Settings → Integrations → ⋮ → Disconnect*) then reconnect.

### OAuth error after clicking "Grant access"

| Error message | Likely cause | Solution |
|---|---|---|
| `invalid_client` | Incorrect HubSpot Client ID configured | Contact support at support@pulsescore.app |
| `access_denied` | User clicked "Cancel" or denied access | Restart the OAuth flow and click **Grant access** |
| `redirect_uri_mismatch` | OAuth redirect URL mismatch | Contact support — this is a configuration issue on the PulseScore side |

### Sync completed but 0 contacts imported

- Confirm you authorized the **correct** HubSpot portal. The connected portal ID is shown in *Settings → Integrations → HubSpot*.
- Verify your HubSpot portal has contacts in the [HubSpot CRM](https://app.hubspot.com/contacts).
- Check that your HubSpot account tier grants API access. Some **Free** tier limitations may restrict the scopes PulseScore requires.

### Some contacts are missing

- PulseScore syncs all contacts. If specific contacts are missing, check whether they have been deleted or merged in HubSpot.
- Contacts without an `email` property are skipped — PulseScore requires a valid email address to create a customer record.

### Sync is taking longer than 8 minutes

1. Navigate to *Settings → Integrations → HubSpot* and check the **Last sync** status.
2. Click **Retry sync** to trigger a fresh import.
3. If the sync status shows an error, note the error message and contact support at support@pulsescore.app.

### Health scores show "N/A" after sync

- Scores require at least **one recorded sales activity or deal** per contact to be computed.
- Contacts imported without any associated deals or logged activities will receive scores once engagement data arrives via webhook or a subsequent sync.

### Webhook events are not being received

1. Confirm the HubSpot webhook subscription is active in your [HubSpot Developer account](https://developers.hubspot.com/).
2. The webhook endpoint URL should be `https://pulsescore.app/api/v1/webhooks/hubspot`.
3. Ensure subscriptions are enabled for the event types listed in the [Webhook events](#webhook-events) table above.
4. Verify the **Webhook client secret** is correctly configured in PulseScore (*Settings → Integrations → HubSpot → Webhook secret*).

---

## Getting help

| Channel | Details |
|---|---|
| **In-app chat** | Click the **?** icon in the bottom-right corner |
| **Email support** | support@pulsescore.app |
| **Status page** | https://status.pulsescore.app |
