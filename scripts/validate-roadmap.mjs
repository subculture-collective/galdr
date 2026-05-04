import { readFileSync } from "node:fs";

const roadmap = readFileSync(new URL("../docs/roadmap.md", import.meta.url), "utf8");

const requiredText = [
  "# PulseScore Master Roadmap",
  "GitHub issue: #240",
  "Phase 1: MVP",
  "Phase 2: Expansion",
  "Phase 3: Defensibility",
  "Chronological Build Order",
  "Total epics: 23",
  "Total implementation issues: 206",
  "AI-Powered Insights",
  "Predictive Churn Models",
  "Integration Marketplace",
];

for (const text of requiredText) {
  if (!roadmap.includes(text)) {
    throw new Error(`roadmap missing required text: ${text}`);
  }
}

const issueRefs = [...roadmap.matchAll(/#\d+/g)].map(([match]) => Number(match.slice(1)));
for (let issue = 32; issue <= 240; issue += 1) {
  if (issue >= 104 && issue <= 105) {
    continue;
  }
  if (!issueRefs.includes(issue)) {
    throw new Error(`roadmap missing issue reference: #${issue}`);
  }
}

const epicHeadings = roadmap.match(/^### \d+\. /gm) ?? [];
if (epicHeadings.length !== 23) {
  throw new Error(`expected 23 epic headings, found ${epicHeadings.length}`);
}

console.log("roadmap validation passed");
