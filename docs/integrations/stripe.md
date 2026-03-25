# Stripe Integration Guide

This guide explains how to connect your Stripe account to PulseScore, what data is synced, which permissions are required, and how Stripe data feeds into customer health scores.

---

## Prerequisites

Before connecting Stripe you will need:

- A PulseScore account with **admin** or **owner** role (required to manage integrations).
- A Stripe account that has at least one of the following: customers, subscriptions, or payment history.
- Your Stripe account must be in **live mode** or **test mode** — both are supported. Test-mode connections are labelled **(test)** in the PulseScore UI.

---

## Connecting Stripe

### Step 1 — Open the Integrations settings

1. Log in to PulseScore.
2. Click **Settings** in the left navigation bar.
3. Click **Integrations** in the Settings sub-menu.

```
[Screenshot placeholder: Settings → Integrations page showing available integration tiles]
```

---

### Step 2 — Start the OAuth flow

1. Locate the **Stripe** tile on the Integrations page.
2. Click **Connect Stripe**.
3. PulseScore redirects you to the Stripe OAuth authorization page (`https://connect.stripe.com`).

> 💡 **Tip:** If nothing happens when you click **Connect Stripe**, check that your browser is not blocking the redirect. PulseScore navigates away from the current page rather than opening a pop-up window.

```
[Screenshot placeholder: Stripe tile with "Connect Stripe" button highlighted]
```

---

### Step 3 — Authorize access in Stripe

1. If prompted, log in to Stripe.
2. Select the Stripe account you want to connect to PulseScore.
3. Review the permissions summary (read-only access — see [Permissions](#permissions) below).
4. Click **Allow access**.

```
[Screenshot placeholder: Stripe OAuth consent screen with "Allow access" button]
```

---

### Step 4 — Confirm the connection

After authorization, Stripe redirects you back to PulseScore.

- A green ✓ badge appears on the Stripe tile: **Connected**.
- The initial data sync starts automatically in the background.

```
[Screenshot placeholder: Integrations page with Stripe tile showing "Connected" badge]
```

---

### Step 5 — Wait for the initial sync

PulseScore imports your Stripe customers, subscriptions, and recent payment history.

1. A progress banner appears at the top of the screen: *"Syncing Stripe data…"*
2. The sync typically completes in **60–120 seconds** for most accounts.
3. The banner updates to *"Sync complete — X customers imported."* when finished.

> ⏱ **Large accounts:** Accounts with thousands of customers may take 2–5 minutes for the initial sync. Subsequent syncs are incremental and much faster.

```
[Screenshot placeholder: Dashboard with "Syncing Stripe data…" progress banner]
```

---

## Permissions

PulseScore requests **read-only** access to your Stripe account using the `read_only` OAuth scope. This means:

| Permission | Granted |
|---|---|
| Read customers | ✅ Yes |
| Read subscriptions | ✅ Yes |
| Read charges / payments | ✅ Yes |
| Read invoices | ✅ Yes |
| Create or modify charges | ❌ No |
| Issue refunds | ❌ No |
| Manage subscriptions | ❌ No |
| Access payouts or banking data | ❌ No |

PulseScore cannot and will never initiate any financial transactions on your behalf.

---

## Data synced

The following Stripe objects are imported into PulseScore during every sync.

### Customers

Each Stripe customer is mapped to a PulseScore **Customer** record.

| Stripe field | PulseScore field | Notes |
|---|---|---|
| `id` | `external_id` | Used to match records on re-sync |
| `email` | `email` | |
| `name` | `name` | |
| `currency` | `currency` | |
| `created` | `first_seen_at` | |
| `metadata` | `metadata` | Full metadata map preserved |

### Subscriptions

Each Stripe subscription is stored and linked to its customer.

| Stripe field | PulseScore field | Notes |
|---|---|---|
| `id` | `stripe_subscription_id` | |
| `status` | `status` | `active`, `canceled`, `past_due`, etc. |
| Items → price → product name | `plan_name` | |
| Items → price → `unit_amount × quantity` | `amount_cents` | Monthly recurring amount |
| Items → price → `recurring.interval` | `interval` | `month` or `year` |
| `current_period_start/end` | `current_period_start/end` | |
| `canceled_at` | `canceled_at` | |

### Payments (charges)

Stripe charges from the last **90 days** are synced by default.

| Stripe field | PulseScore field | Notes |
|---|---|---|
| `id` | `stripe_payment_id` | |
| `amount` | `amount_cents` | |
| `currency` | `currency` | |
| `status` / `paid` | `status` | `succeeded`, `failed`, or `pending` |
| `failure_code` | `failure_code` | Populated for failed charges |
| `failure_message` | `failure_message` | Human-readable failure reason |
| `created` | `paid_at` | |

> **Note:** Only charges associated with a known Stripe customer are synced. Anonymous charges are skipped.

### Sync frequency

| Sync type | Trigger | Coverage |
|---|---|---|
| Initial full sync | Immediately after OAuth connection | All customers, all subscriptions, last 90 days of payments |
| Incremental sync | On incoming Stripe webhook events | New/updated objects only |
| Manual re-sync | *Settings → Integrations → Stripe → Retry sync* | Full re-import |

---

## How Stripe data maps to health scores

PulseScore computes a **health score (0–100)** for each customer. Several score factors are derived directly from Stripe data.

### Score factors

| Factor | Source data | What it measures | Default weight |
|---|---|---|---|
| **Payment recency** | Latest successful charge date | How recently the customer made a successful payment | 25% |
| **MRR trend** | Subscription `amount_cents` over time | Whether monthly recurring revenue is growing, stable, or shrinking | 25% |
| **Failed payments** | Charges with `status = failed` | Frequency of payment failures in the last 30 days | 20% |
| **Churn risk** | Subscription `status` | Penalty applied for `past_due`, `unpaid`, or `canceled` subscriptions | part of failed payments / MRR factors |

### Risk levels

Scores are mapped to color-coded risk levels:

| Score range | Risk level | Color |
|---|---|---|
| 70–100 | Healthy | 🟢 Green |
| 40–69 | At risk | 🟡 Yellow |
| 0–39 | Critical | 🔴 Red |

> **Tip:** Score weights and thresholds are configurable per organization. See *Settings → Scoring → Configuration* or the [API Reference](../api-reference.md#put-scoringconfig).

### Score calculation example

Consider a customer with these Stripe signals:

- Last successful payment: **3 days ago** → payment recency score high
- Subscription status: **active** at \$500/month for 12 months → MRR stable
- Failed payments in last 30 days: **0** → no penalty

This customer would receive a **high health score** (🟢 green), indicating low churn risk.

Now consider a customer where:

- Last successful payment: **45 days ago**
- Subscription status: **past_due**
- Failed payments in last 30 days: **3**

This customer would receive a **low health score** (🔴 red), triggering churn-risk alerts.

---

## Webhook events

In addition to periodic syncs, PulseScore listens to Stripe webhook events in real time to keep data up to date. The following Stripe event types are processed:

| Stripe event | PulseScore action |
|---|---|
| `customer.created` / `customer.updated` | Upsert customer record |
| `customer.subscription.created` / `updated` | Upsert subscription |
| `customer.subscription.deleted` | Mark subscription canceled |
| `charge.succeeded` | Record successful payment |
| `charge.failed` | Record failed payment; emit `payment.failed` event |
| `invoice.payment_succeeded` | Update customer last-seen date |
| `invoice.payment_failed` | Trigger score recalculation |

Webhook events are verified using your **Stripe webhook signing secret** before being processed.

---

## Disconnecting Stripe

To remove the Stripe integration:

1. Go to *Settings → Integrations*.
2. Click the **⋮** menu on the Stripe tile.
3. Select **Disconnect**.
4. Confirm the disconnection in the dialog.

> ⚠️ **Warning:** Disconnecting Stripe stops all future syncs. Existing customer records and scores are retained but will no longer be updated from Stripe data.

---

## Troubleshooting

### "Connect Stripe" button does nothing

- **Browser redirect blocked:** PulseScore navigates away from the current page to open Stripe OAuth. Ensure no browser extension is intercepting the navigation.
- **Already connected:** If Stripe was connected in a previous session, disconnect it first (*Settings → Integrations → ⋮ → Disconnect*) then reconnect.

### OAuth error after clicking "Allow access"

| Error message | Likely cause | Solution |
|---|---|---|
| `invalid_client` | Incorrect Stripe Client ID configured | Contact support at support@pulsescore.app |
| `access_denied` | User clicked "Cancel" or denied access | Restart the OAuth flow and click **Allow access** |
| `redirect_uri_mismatch` | OAuth redirect URL mismatch | Contact support — this is a configuration issue on the PulseScore side |

### Sync completed but 0 customers imported

- Confirm you authorized the **correct** Stripe account. The connected account ID is shown in *Settings → Integrations → Stripe*.
- Stripe **test-mode** accounts are supported but must have customers in the Stripe dashboard. An empty test account will import 0 customers.
- Verify your Stripe account has customers in the [Stripe Dashboard → Customers](https://dashboard.stripe.com/customers).

### Some customers are missing

- Only customers **with at least one charge** in the last 90 days are guaranteed to appear. Customers with no transaction history are synced but may appear without scores.
- Check that those customers exist in your connected Stripe account (live mode vs. test mode).

### Sync is taking longer than 5 minutes

1. Navigate to *Settings → Integrations → Stripe* and check the **Last sync** status.
2. Click **Retry sync** to trigger a fresh import.
3. If the sync status shows an error, note the error message and contact support at support@pulsescore.app.

### Health scores show "N/A" after sync

- Scores require at least **one billing event** per customer (a subscription, charge, or invoice) to be computed.
- Customers imported without any Stripe transactions will receive scores once billing activity arrives via webhook or a subsequent sync.

### Webhook events are not being received

1. Confirm the Stripe webhook is configured in your [Stripe Dashboard → Developers → Webhooks](https://dashboard.stripe.com/webhooks).
2. The webhook endpoint URL should be `https://pulsescore.app/api/v1/webhooks/stripe`.
3. Ensure the webhook is configured to send the event types listed in the [Webhook events](#webhook-events) table above.
4. Verify the **Webhook signing secret** is correctly configured in PulseScore (*Settings → Integrations → Stripe → Webhook secret*).

---

## Getting help

| Channel | Details |
|---|---|
| **In-app chat** | Click the **?** icon in the bottom-right corner |
| **Email support** | support@pulsescore.app |
| **Status page** | https://status.pulsescore.app |
