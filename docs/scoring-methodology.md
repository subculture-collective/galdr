# PulseScore Scoring Methodology

This document explains how PulseScore computes health scores — the algorithm, the five scoring factors, their default weights, risk level thresholds, and how you can customize all of these to fit your business.

---

## Overview

A customer's health score is a single number between **0 and 100**. It is a weighted average of five independent scoring factors, each of which examines a different dimension of the customer relationship. The final integer is mapped to one of three risk levels:

| Risk Level | Score Range | Colour |
|------------|-------------|--------|
| Green      | 70 – 100    | 🟢     |
| Yellow     | 40 – 69     | 🟡     |
| Red        | 0 – 39      | 🔴     |

All thresholds are configurable per organisation (see [Customization](#customization)).

---

## Algorithm

### Step 1 — Calculate each factor

For every customer, PulseScore independently evaluates each of the five factors listed below. Each factor returns a normalized score in the range **0.0 – 1.0**, or `nil` when the necessary data is not yet available (e.g. no payment history at all). Factors that return `nil` are skipped and their weight is redistributed proportionally to the remaining factors.

### Step 2 — Weighted aggregation

```
overall_score = round( sum( factor_score × adjusted_weight ) × 100 )
```

Where `adjusted_weight` is the factor's configured weight rescaled so the weights of all *present* factors still sum to 1.0:

```
adjusted_weight[i] = configured_weight[i] / sum(configured_weight for present factors)
```

The result is rounded to the nearest integer and clamped to [0, 100].

### Step 3 — Risk level assignment

The integer score is compared against the configured thresholds (default: green ≥ 70, yellow ≥ 40) to produce the risk level label (`green`, `yellow`, or `red`).

---

## Scoring Factors

### 1. Payment Recency (`payment_recency`) — default weight 30%

**What it measures:** How recently and reliably a customer has made successful payments.

**Data sources:** Stripe payment records stored in `stripe_payments`.

**How it's calculated:** An underlying `PaymentRecencyService` produces a 0–100 score based on days since the last successful payment and payment consistency. This is normalized to 0.0–1.0. If the customer has no payment history at all (first-time customer), a neutral score of **0.5** is used.

**Score interpretation:**

| Score (0.0–1.0) | Meaning |
|-----------------|---------|
| 1.0             | Very recent, consistent payments |
| 0.5             | No payment history yet (new customer) |
| → 0.0           | Long overdue or no payments |

---

### 2. MRR Trend (`mrr_trend`) — default weight 20%

**What it measures:** Whether the customer's Monthly Recurring Revenue is growing, stable, or declining.

**Data sources:** `mrr.changed` events from the `customer_events` table; current MRR from the `customers` table.

**How it's calculated:** Three time windows are compared (oldest MRR event in window → current MRR):

| Window | Weight within factor |
|--------|---------------------|
| 30 days | 50% |
| 60 days | 30% |
| 90 days | 20% |

The weighted percentage change is converted to a score:

| Trend             | Score range |
|-------------------|-------------|
| > +5% (growing)   | 0.8 – 1.0   |
| −5% to +5% (stable) | 0.5 – 0.7 |
| −50% to −5% (declining) | 0.1 – 0.4 |
| < −50% (severe decline) | 0.0     |

If no historical MRR events exist, a neutral score of **0.5** is returned.

---

### 3. Failed Payments (`failed_payments`) — default weight 20%

**What it measures:** The volume and recency of payment failures.

**Data sources:** Stripe payment records, supplemented by the `PaymentHealthService` for consecutive failure tracking.

**How it's calculated:**

| Condition | Score |
|-----------|-------|
| No failures in last 90 days | 1.0 |
| Single failure, already resolved | 0.75 |
| 1 consecutive unresolved failure | 0.25 |
| 2 consecutive unresolved failures | 0.15 |
| ≥ 3 consecutive unresolved failures | 0.0 |
| Multiple failures but resolved (proportional) | 0.1 – 1.0 based on failure rate |

An additional penalty of **−0.1 per failure** is applied for any failures in the most recent 7 days.

---

### 4. Support Tickets (`support_tickets`) — default weight 15%

**What it measures:** The customer's support ticket volume relative to the organisation median — fewer tickets than average signals a healthier, lower-friction experience.

**Data sources:** `ticket.opened` and `ticket.resolved` events from the `customer_events` table (90-day window).

**How it's calculated:**

| Volume vs. org median | Score range |
|-----------------------|-------------|
| ≤ 50% of median (low) | 0.7 – 1.0 |
| 50% – 150% of median (average) | 0.4 – 0.7 |
| > 150% of median (high) | 0.0 – 0.4 |

An additional penalty of **−0.1 per unresolved ticket** is applied on top of the volume score.

If no ticket data exists for the organisation, this factor is skipped and its weight is redistributed.

---

### 5. Engagement (`engagement`) — default weight 15%

**What it measures:** How actively the customer uses the product, relative to the organisation median.

**Data sources:** `login`, `feature_use`, and `api_call` events from `customer_events` (30-day window for volume, 7-day window for recency).

**How it's calculated:**

| Activity vs. org median | Score range |
|-------------------------|-------------|
| ≥ 150% of median (highly active) | 0.8 – 1.0 |
| 50% – 150% of median (average) | 0.4 – 0.8 |
| < 50% of median (low activity) | 0.0 – 0.4 |

A **recency bonus of +0.02 per event** (capped at +0.1) is added for any activity recorded in the last 7 days.

If no activity data exists for the organisation, this factor is skipped and its weight is redistributed.

---

## Default Weights

| Factor | Default Weight | Rationale |
|--------|---------------|-----------|
| `payment_recency` | **30%** | Payment health is the strongest predictor of churn; recency captures both reliability and engagement. |
| `mrr_trend` | **20%** | Revenue trajectory reveals expansion/contraction before it fully materialises. |
| `failed_payments` | **20%** | Hard failures are direct signals of billing risk and potential involuntary churn. |
| `support_tickets` | **15%** | High ticket volume correlates with friction and dissatisfaction, but is a secondary signal. |
| `engagement` | **15%** | Product usage indicates value realisation, complementing the financial signals. |

All weights sum to **1.0** (100%).

---

## Risk Level Thresholds

| Risk Level | Default Minimum Score | Meaning |
|------------|-----------------------|---------|
| 🟢 Green   | 70                    | Healthy — low churn risk |
| 🟡 Yellow  | 40                    | At risk — warrants attention |
| 🔴 Red     | 0                     | Critical — high churn risk |

Thresholds are enforced as `green > yellow > 0` and `green ≤ 100`. Scores are compared with `>=`, so a score of exactly 70 is Green, and exactly 40 is Yellow.

---

## Score Recalculation

Scores are kept current through two mechanisms:

### Periodic batch recalculation

A background scheduler recalculates every customer's score across all active organisations on a configurable interval (default: **every 6 hours**). Up to **5 workers** run in parallel for throughput.

### Event-triggered recalculation

Scores can be recalculated immediately for a single customer or for all customers in an org when a relevant event occurs. This is used, for example, after a Stripe webhook arrives (new payment, subscription change) or after an organisation's scoring config is updated.

### Change detection

After each recalculation, PulseScore compares the new score to the previous one and records change events:

| Event | Trigger |
|-------|---------|
| `score.initial` | First score ever computed for a customer |
| `score.changed` | Absolute delta ≥ 10 points |
| `risk_level.changed` | Risk level transitions (e.g. green → yellow) |

These events are stored in `customer_events` and used to drive alerts.

---

## Customization

Each organisation can override the default weights and thresholds through **Settings → Scoring** or via the API.

### Changing factor weights

Weights must satisfy two constraints:

1. Every weight is in the range **[0.0, 1.0]**.
2. All weights must **sum to exactly 1.0** (tolerance ±0.001).

Example: if your product has no meaningful support-ticket signal, you can redistribute that weight to `payment_recency`:

```json
{
  "weights": {
    "payment_recency": 0.40,
    "mrr_trend":       0.20,
    "failed_payments": 0.20,
    "support_tickets": 0.05,
    "engagement":      0.15
  }
}
```

### Changing risk level thresholds

Thresholds must satisfy: `green > yellow > 0` and `green ≤ 100`.

Example: stricter thresholds for a high-touch enterprise product:

```json
{
  "thresholds": {
    "green":  80,
    "yellow": 55
  }
}
```

### API endpoint

```http
PATCH /api/v1/scoring/config
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "weights": { ... },
  "thresholds": { ... }
}
```

After saving, PulseScore automatically triggers a full recalculation of all customer scores for the organisation.

---

## Worked Example

Consider **Acme Corp**, a customer with the following signals:

| Factor | Raw data | Factor score (0.0–1.0) | Weight |
|--------|----------|------------------------|--------|
| `payment_recency` | Last payment 5 days ago, consistent history | **0.95** | 0.30 |
| `mrr_trend` | MRR grew from $800 → $1,000 over 30 days (+25%) | **0.90** | 0.20 |
| `failed_payments` | 1 failure 45 days ago, resolved | **0.75** | 0.20 |
| `support_tickets` | 2 tickets vs. org median of 4 (50% of median) | **0.70** | 0.15 |
| `engagement` | 120 events vs. org median of 80 (150% of median) | **0.80** | 0.15 |

All five factors are present, so no weight redistribution is needed.

```
weighted_sum = (0.95 × 0.30) + (0.90 × 0.20) + (0.75 × 0.20) + (0.70 × 0.15) + (0.80 × 0.15)
             = 0.285 + 0.180 + 0.150 + 0.105 + 0.120
             = 0.840

overall_score = round(0.840 × 100) = 84
risk_level    = "green"   (84 ≥ 70)
```

**Result:** Health score **84 / 100** 🟢 Green.

---

### Example with a missing factor

Now consider **Beta LLC**, a new customer with no engagement data yet:

| Factor | Factor score | Weight |
|--------|--------------|--------|
| `payment_recency` | 0.50 (no history — neutral) | 0.30 |
| `mrr_trend` | 0.50 (no history — neutral) | 0.20 |
| `failed_payments` | 1.00 (no failures) | 0.20 |
| `support_tickets` | 1.00 (no tickets) | 0.15 |
| `engagement` | *skipped (nil)* | — |

With `engagement` skipped, the remaining weights sum to **0.85**. Each weight is rescaled:

| Factor | Configured weight | Adjusted weight |
|--------|------------------|-----------------|
| `payment_recency` | 0.30 | 0.30 / 0.85 ≈ 0.353 |
| `mrr_trend` | 0.20 | 0.20 / 0.85 ≈ 0.235 |
| `failed_payments` | 0.20 | 0.20 / 0.85 ≈ 0.235 |
| `support_tickets` | 0.15 | 0.15 / 0.85 ≈ 0.176 |

```
weighted_sum = (0.50 × 0.353) + (0.50 × 0.235) + (1.00 × 0.235) + (1.00 × 0.176)
             ≈ 0.176 + 0.118 + 0.235 + 0.176
             ≈ 0.705

overall_score = round(0.705 × 100) = 71
risk_level    = "green"   (71 ≥ 70)
```

**Result:** Health score **71 / 100** 🟢 Green — a healthy new customer, with the missing engagement signal automatically excluded from the calculation.

---

## Related documentation

- [Quickstart Guide](./quickstart.md) — Get up and running with PulseScore in 5 minutes.
- [API Reference](./api-reference.md) — Full REST API documentation.
