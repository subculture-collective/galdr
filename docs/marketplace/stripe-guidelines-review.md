# Stripe App Marketplace Guidelines Review

## Stripe App Marketplace Readiness

- Listing includes an app name, short description, long description, categories, visual assets, support contact, and legal links.
- Short description is under the 80 character discovery limit from issue #176.
- Visual assets live under `docs/marketplace/assets/` and use marketplace-friendly dimensions.
- Screenshots cover the required product surfaces: dashboard, customer detail, health scores, and alerts.

## Permission Claims

- Copy states that Stripe OAuth is read-only.
- No payment write access is claimed or implied.
- Listing says PulseScore cannot create charges, issue refunds, modify subscriptions, or access banking data.
- Benefits focus on customer health analytics and customer management, not payment operations.

## Support Readiness

Support readiness is covered by the app URL, support contact, and legal links below.

- App URL: https://pulsescore.app
- Support URL: https://pulsescore.app/support
- Support email: support@pulsescore.app
- Privacy policy: https://pulsescore.app/privacy
- Terms: https://pulsescore.app/terms

## Review Notes

- Mockup screenshots are acceptable issue assets for this repository task because the issue allowed screenshots captured or mockups created.
- Final live submission should verify Stripe dashboard image-format constraints and replace SVG mockups with PNG exports if the Stripe submission form requires raster uploads.
- Final live submission should confirm the support URL route is deployed; current repo legal routes exist at `/privacy` and `/terms`, while support contact is already documented as `support@pulsescore.app`.
