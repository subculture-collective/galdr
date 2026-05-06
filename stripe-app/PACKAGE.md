# Stripe App Package

Package root: `stripe-app/`

Parent epic: #24
Implementation issue: #174

## OAuth Alignment

The manifest uses Stripe Apps OAuth and the same callback path as the existing Stripe integration:

`https://pulsescore.app/api/v1/integrations/stripe/callback`

Production must set:

`STRIPE_OAUTH_REDIRECT_URL=https://pulsescore.app/api/v1/integrations/stripe/callback`

The post-install action sends users to onboarding:

`https://pulsescore.app/onboarding?source=stripe-app`

## Packaging Commands

Run from this directory after installing the Stripe CLI and Apps plugin:

```bash
stripe plugin install apps
stripe apps validate
stripe apps upload
```

For local extension testing before upload:

```bash
stripe apps start
```

Distribution is `private` initially. Change `distribution_type` to `public` only after marketplace review assets and approval are ready.

## Review Checklist

- OAuth redirect URI matches `STRIPE_OAUTH_REDIRECT_URL`.
- Requested permissions stay read-only: customers, subscriptions, charges, invoices.
- Sandbox installs remain enabled for test-mode verification.
- Post-install redirect lands in the onboarding wizard.
