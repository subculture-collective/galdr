import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";

const appDir = new URL("../stripe-app/", import.meta.url);
const manifestPath = new URL("stripe-app.json", appDir);
const packageGuidePath = new URL("PACKAGE.md", appDir);
const redirectUri = "https://pulsescore.app/api/v1/integrations/stripe/callback";
const postInstallUrl = "https://pulsescore.app/onboarding?source=stripe-app";

const expectedManifest = {
  id: "com.pulsescore.customer-health",
  name: "PulseScore",
  distributionType: "private",
  sandboxInstallCompatible: true,
  stripeApiAccessType: "oauth",
  allowedRedirectUris: [redirectUri],
  postInstallAction: {
    type: "external",
    url: postInstallUrl,
  },
};

const expectedPermissions = new Map([
  ["customer_read", "import Stripe customers"],
  ["subscription_read", "monitor subscription status"],
  ["charge_read", "analyze successful and failed payments"],
  ["invoice_read", "reconcile invoice payment events"],
]);

assert.ok(existsSync(manifestPath), "stripe-app/stripe-app.json must exist");
assert.ok(existsSync(packageGuidePath), "stripe-app/PACKAGE.md must exist");

const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));

assert.equal(manifest.id, expectedManifest.id);
assert.equal(manifest.name, expectedManifest.name);
assert.equal(manifest.distribution_type, expectedManifest.distributionType);
assert.equal(manifest.sandbox_install_compatible, expectedManifest.sandboxInstallCompatible);
assert.equal(manifest.stripe_api_access_type, expectedManifest.stripeApiAccessType);
assert.deepEqual(manifest.allowed_redirect_uris, expectedManifest.allowedRedirectUris);
assert.deepEqual(manifest.post_install_action, expectedManifest.postInstallAction);

const permissions = new Map(
  manifest.permissions.map(({ permission, purpose }) => [permission, purpose]),
);
assert.deepEqual([...permissions.keys()].sort(), [...expectedPermissions.keys()].sort());
for (const [permission, phrase] of expectedPermissions) {
  assert.match(permissions.get(permission), new RegExp(phrase, "i"));
}

assert.match(manifest.icon, /^\.\/assets\/.*\.png$/);
assert.ok(existsSync(new URL(manifest.icon.replace(/^\.\//, ""), appDir)), "icon must exist");

const packageGuide = readFileSync(packageGuidePath, "utf8");
assert.match(packageGuide, /stripe plugin install apps/);
assert.match(packageGuide, /stripe apps validate/);
assert.match(packageGuide, /stripe apps upload/);
assert.match(packageGuide, /STRIPE_OAUTH_REDIRECT_URL/);
assert.ok(packageGuide.includes(redirectUri), "package guide must document OAuth redirect URI");
assert.ok(packageGuide.includes(postInstallUrl), "package guide must document post-install URL");

console.log("stripe app package validation passed");
