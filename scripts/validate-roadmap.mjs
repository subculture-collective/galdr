import { readFileSync } from "node:fs";

const roadmapPath = new URL("../docs/roadmap.md", import.meta.url);
const roadmap = readFileSync(roadmapPath, "utf8");

const firstImplementationIssue = 32;
const lastRoadmapIssue = 240;
const omittedIssueRefs = new Set([104, 105]);
const expectedEpicCount = 23;

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

const issueRefs = new Set([...roadmap.matchAll(/#\d+/g)].map(([match]) => Number(match.slice(1))));
for (let issue = firstImplementationIssue; issue <= lastRoadmapIssue; issue += 1) {
  if (omittedIssueRefs.has(issue)) {
    continue;
  }
  if (!issueRefs.has(issue)) {
    throw new Error(`roadmap missing issue reference: #${issue}`);
  }
}

const epicHeadings = roadmap.match(/^### \d+\. /gm) ?? [];
if (epicHeadings.length !== expectedEpicCount) {
  throw new Error(`expected ${expectedEpicCount} epic headings, found ${epicHeadings.length}`);
}

console.log("roadmap validation passed");
