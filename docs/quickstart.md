# PulseScore Quickstart Guide

Welcome to PulseScore! This guide walks you through signing up, connecting Stripe, and viewing your first customer health scores — all in under 5 minutes.

---

## What you'll accomplish

By the end of this guide you will have:

- A PulseScore account and organization
- Stripe connected as your first data source
- Live customer health scores on your dashboard

---

## Step 1 — Sign up at pulsescore.app

1. Open [https://pulsescore.app](https://pulsescore.app) in your browser.
2. Click **Get started free** on the hero section.
3. Enter your **email address** and choose a **password**, then click **Create account**.
4. Check your inbox for a verification email and click the confirmation link.

> 💡 **Tip:** Use the email address tied to your company so your team-mates can join the same organization later.

```
[Screenshot placeholder: Sign-up form with email + password fields and "Create account" button]
```

**Expected result:** You are logged in and land on the onboarding wizard.

---

## Step 2 — Complete the onboarding wizard

The onboarding wizard collects the information PulseScore needs to set up your organization.

1. **Welcome step** — Read the brief overview, then click **Let's go**.
2. **Organization setup** — Enter your company name (e.g. *Acme Inc*). PulseScore generates a URL-friendly slug automatically. Click **Continue**.
3. **Invite teammates** *(optional)* — Enter colleagues' email addresses to invite them now, or click **Skip for now**.
4. Click **Finish setup** on the summary screen.

```
[Screenshot placeholder: Multi-step onboarding wizard showing the organization name field]
```

**Expected result:** Your organization is created and you are redirected to the integration connection screen.

---

## Step 3 — Connect your Stripe account

PulseScore pulls subscription, payment, and churn signals directly from Stripe.

1. On the **Connect your data sources** screen, click the **Stripe** tile.
2. Click **Connect Stripe** — you are redirected to Stripe's OAuth authorization page.
3. Log in to Stripe (if prompted) and select the Stripe account you want to connect.
4. Click **Allow access** to grant PulseScore read permissions.
5. You are redirected back to PulseScore. A green ✓ badge appears on the Stripe tile confirming the connection.

> 💡 **Tip:** PulseScore requests *read-only* access to your Stripe data. No charges or refunds can be made through this integration.

```
[Screenshot placeholder: Integration screen with Stripe tile highlighted and "Connect Stripe" button]
```

**Expected result:** The Stripe tile shows **Connected** and the initial sync begins automatically.

---

## Step 4 — Wait for the initial sync (~1 minute)

PulseScore imports your Stripe customers and subscription history and computes the first round of health scores.

1. A progress banner appears at the top of the screen: *"Syncing Stripe data…"*
2. Wait approximately **60 seconds**. You can stay on the page or navigate away — the sync continues in the background.
3. When the sync finishes the banner updates to *"Sync complete — X customers imported."*

> ⏱ **Large accounts:** If you have thousands of Stripe customers the initial sync may take 2–3 minutes. Subsequent syncs run incrementally and are much faster.

```
[Screenshot placeholder: Dashboard with "Syncing Stripe data…" progress banner]
```

**Expected result:** The sync banner disappears and customer rows appear in the dashboard table.

---

## Step 5 — View your dashboard

The dashboard gives you an instant snapshot of your customers' health.

1. Click **Dashboard** in the left navigation (or navigate to `/dashboard`).
2. The main table lists every customer with:
   - **Health score** (0–100, color-coded: 🟢 healthy · 🟡 at risk · 🔴 critical)
   - **MRR** pulled from Stripe
   - **Last active** date
   - **Trend** arrow (improving / stable / declining)
3. Use the **filter bar** to narrow results by health tier, MRR range, or date range.
4. Click a column header to sort the table.

```
[Screenshot placeholder: Dashboard table showing customer list with health score badges]
```

**Expected result:** You can see health scores for all imported Stripe customers.

---

## Step 6 — Explore a customer detail page

Drill into any customer to see the full breakdown behind their score.

1. Click any customer row in the dashboard table.
2. The **Customer detail page** opens and shows:
   - **Score history** chart (30-day trend)
   - **Signal breakdown** — individual factors that make up the score (payment history, subscription changes, activity, etc.)
   - **Recent events** — Stripe events (renewals, failed payments, upgrades, cancellations)
3. Hover over any signal bar to see a tooltip explaining what it measures and how it affects the score.

```
[Screenshot placeholder: Customer detail page with score history chart and signal breakdown]
```

**Expected result:** You understand *why* a customer has the health score they do.

---

## Step 7 — Set up your first alert

Alerts notify you when a customer's health drops below a threshold so you can act before they churn.

1. Click **Alerts** in the left navigation, then click **New alert rule**.
2. Fill in the rule form:
   | Field | Example value |
   |-------|---------------|
   | **Name** | At-risk customers |
   | **Condition** | Health score drops below **50** |
   | **Notify via** | Email |
   | **Recipients** | your@company.com |
3. Click **Save rule**.
4. The rule appears in the **Alert rules** list with status *Active*.

> 💡 **Tip:** Start with a threshold of **50** (the default "at risk" boundary) and tune it after you've watched scores for a week.

```
[Screenshot placeholder: Alert rule form with condition and notification fields]
```

**Expected result:** You receive an email notification the next time a customer's health score falls below your threshold.

---

## 🎉 You're all set!

You've completed the PulseScore quickstart. Here's what you accomplished:

- ✅ Created your account and organization
- ✅ Connected Stripe
- ✅ Viewed customer health scores on the dashboard
- ✅ Explored a customer detail page
- ✅ Created your first alert rule

### What to explore next

- **Connect HubSpot or Intercom** — Add CRM and support signals for richer, more accurate scores *(Settings → Integrations)*.
- **Invite your team** — Bring in your CS or sales team *(Settings → Team)*.
- **API access** — Embed scores in your own tooling. See the [API Reference](./api-reference.md).
- **Scoring methodology** — Understand how scores are calculated. See the [Scoring Methodology](./scoring-methodology.md).

---

## Troubleshooting

### "Connect Stripe" button does nothing

- **Pop-up blocker:** PulseScore opens Stripe OAuth in the same window. Ensure your browser isn't blocking navigation away from the page.
- **Already connected:** If Stripe was connected during a previous session, disconnect it first under *Settings → Integrations* and reconnect.

### The sync is taking longer than 5 minutes

- Check your Stripe account has active customers — a sandbox/test-mode account with no data will show 0 customers immediately.
- Navigate to *Settings → Integrations → Stripe* and click **Retry sync**.
- If the problem persists, contact support at support@pulsescore.app.

### I see 0 customers after the sync

- Confirm you authorized the *correct* Stripe account (live mode vs. test mode).
- Stripe test-mode accounts are supported but will be labelled **(test)** next to the integration name.
- Navigate to *Settings → Integrations → Stripe* and verify the connected account ID matches your Stripe dashboard.

### My health scores all show "N/A"

- Scores require at least **one billing event** per customer (a subscription, charge, or invoice) to be computed. If customers were imported but have no transaction history, scores will appear once new events arrive.

### I'm not receiving alert emails

1. Check your spam folder.
2. Verify the recipient address under *Alerts → [rule name] → Edit*.
3. Add `noreply@pulsescore.app` to your email allow-list.
4. Test-fire an alert by temporarily lowering the threshold below the score of a healthy customer, then restore the original threshold.

### I forgot my password

Click **Forgot password?** on the login screen at [https://pulsescore.app/login](https://pulsescore.app/login) and follow the reset email instructions.

---

## Getting help

| Channel | Details |
|---------|---------|
| **In-app chat** | Click the **?** icon in the bottom-right corner |
| **Email support** | support@pulsescore.app |
| **Status page** | https://status.pulsescore.app |
