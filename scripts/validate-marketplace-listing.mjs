import { existsSync, readFileSync } from "node:fs";
import path from "node:path";

const root = process.cwd();
const marketplaceDir = path.join(root, "docs", "marketplace");
const listingPath = path.join(marketplaceDir, "stripe-listing.json");
const contentPath = path.join(marketplaceDir, "stripe-listing.md");
const guidelinesPath = path.join(marketplaceDir, "stripe-guidelines-review.md");

function fail(message) {
  throw new Error(`Marketplace listing validation failed: ${message}`);
}

function assert(condition, message) {
  if (!condition) fail(message);
}

function readRequiredFile(filePath) {
  assert(existsSync(filePath), `missing ${path.relative(root, filePath)}`);
  return readFileSync(filePath, "utf8");
}

const listing = JSON.parse(readRequiredFile(listingPath));
const content = readRequiredFile(contentPath);
const guidelines = readRequiredFile(guidelinesPath);

assert(listing.app_name === "PulseScore", "app_name must be PulseScore");
assert(listing.issue_reference === "#176", "issue_reference must be #176");
assert(listing.parent_epic_reference === "#24", "parent_epic_reference must be #24");
assert(
  typeof listing.short_description === "string" &&
    listing.short_description.length > 0 &&
    listing.short_description.length <= 80,
  "short_description must be 1-80 chars",
);
assert(
  typeof listing.long_description === "string" &&
    listing.long_description.length >= 600,
  "long_description must be at least 600 chars",
);
assert(
  Array.isArray(listing.categories) &&
    listing.categories.includes("analytics") &&
    listing.categories.includes("customer management"),
  "categories must include analytics and customer management",
);

for (const field of ["app_url", "support_url", "privacy_policy_url", "terms_url"]) {
  assert(
    typeof listing[field] === "string" && listing[field].startsWith("https://"),
    `${field} must be an https URL`,
  );
}

assert(listing.support_email === "support@pulsescore.app", "support_email mismatch");
assert(
  listing.logo?.file && existsSync(path.join(marketplaceDir, listing.logo.file)),
  "logo file missing",
);
assert(
  listing.logo?.width === 1024 && listing.logo?.height === 1024,
  "logo dimensions must be 1024x1024",
);

const screenshots = listing.screenshots ?? [];
assert(screenshots.length >= 4, "at least four screenshots/mockups required");
for (const screenshot of screenshots) {
  assert(screenshot.width >= 1280, `${screenshot.file} width must be >=1280`);
  assert(screenshot.height >= 800, `${screenshot.file} height must be >=800`);
  assert(
    screenshot.file && existsSync(path.join(marketplaceDir, screenshot.file)),
    `${screenshot.file} missing`,
  );
  assert(typeof screenshot.caption === "string" && screenshot.caption.length > 10, `${screenshot.file} caption missing`);
}

for (const phrase of ["OAuth", "read-only", "health score", "alerts", "customer detail"]) {
  assert(content.includes(phrase), `listing markdown must mention ${phrase}`);
}

for (const phrase of ["Issue #176", "Parent Epic #24"]) {
  assert(content.includes(phrase), `listing markdown must mention ${phrase}`);
}

for (const phrase of ["Stripe App Marketplace", "No payment write access", "Support readiness"]) {
  assert(guidelines.includes(phrase), `guidelines review must mention ${phrase}`);
}

console.log("Stripe Marketplace listing content is complete");
