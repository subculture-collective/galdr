import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import UpgradePrompt from "./UpgradePrompt";

function assertMatch(input: string, pattern: RegExp) {
  if (!pattern.test(input)) {
    throw new Error(`Expected ${pattern} to match ${input}`);
  }
}

const markup = renderToStaticMarkup(
  React.createElement(UpgradePrompt, {
    featureName: "Full dashboard",
    recommendedTier: "growth",
    description: "Unlock health score charts and at-risk customer views.",
  }),
);

assertMatch(markup, /Upgrade to Growth to access Full dashboard/);
assertMatch(markup, /Unlock health score charts/);
assertMatch(markup, /href="\/pricing\?tier=growth"/);

console.log("UpgradePrompt renders upgrade CTA");
