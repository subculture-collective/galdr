import { existsSync, readFileSync } from "node:fs";
import path from "node:path";

const root = process.cwd();
const marketplaceDir = path.join(root, "docs", "marketplace");
const listingPath = path.join(marketplaceDir, "stripe-listing.json");
const contentPath = path.join(marketplaceDir, "stripe-listing.md");
const guidelinesPath = path.join(marketplaceDir, "stripe-guidelines-review.md");

const expectedListing = {
  appName: "PulseScore",
  issueReference: "#176",
  parentEpicReference: "#24",
  supportEmail: "support@pulsescore.app",
};

const httpsUrlFields = ["app_url", "support_url", "privacy_policy_url", "terms_url"];
const requiredListingPhrases = ["OAuth", "read-only", "health score", "alerts", "customer detail"];
const requiredReferencePhrases = ["Issue #176", "Parent Epic #24"];
const requiredGuidelinesPhrases = ["Stripe App Marketplace", "No payment write access", "Support readiness"];

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

function assetExists(assetPath) {
  return existsSync(path.join(marketplaceDir, assetPath));
}

const listing = JSON.parse(readRequiredFile(listingPath));
const content = readRequiredFile(contentPath);
const guidelines = readRequiredFile(guidelinesPath);

assert(listing.app_name === expectedListing.appName, "app_name must be PulseScore");
assert(listing.issue_reference === expectedListing.issueReference, "issue_reference must be #176");
assert(listing.parent_epic_reference === expectedListing.parentEpicReference, "parent_epic_reference must be #24");
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

for (const field of httpsUrlFields) {
  assert(
    typeof listing[field] === "string" && listing[field].startsWith("https://"),
    `${field} must be an https URL`,
  );
}

assert(listing.support_email === expectedListing.supportEmail, "support_email mismatch");
assert(listing.logo?.file && assetExists(listing.logo.file), "logo file missing");
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
    screenshot.file && assetExists(screenshot.file),
    `${screenshot.file} missing`,
  );
  assert(
    typeof screenshot.caption === "string" && screenshot.caption.length > 10,
    `${screenshot.file} caption missing`,
  );
}

for (const phrase of requiredListingPhrases) {
  assert(content.includes(phrase), `listing markdown must mention ${phrase}`);
}

for (const phrase of requiredReferencePhrases) {
  assert(content.includes(phrase), `listing markdown must mention ${phrase}`);
}

for (const phrase of requiredGuidelinesPhrases) {
  assert(guidelines.includes(phrase), `guidelines review must mention ${phrase}`);
}

console.log("Stripe Marketplace listing content is complete");
