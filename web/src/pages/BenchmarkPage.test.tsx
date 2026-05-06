import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import { BenchmarkAccessPrompt } from "./BenchmarkPage";

function assert(condition: boolean, message: string) {
  if (!condition) {
    throw new Error(message);
  }
}

const html = renderToStaticMarkup(
  React.createElement(BenchmarkAccessPrompt, { recommendedTier: "scale" }),
);

assert(
  html.includes("Upgrade to Scale to access Benchmarking"),
  "benchmark page should show scale upgrade prompt",
);
assert(
  html.includes("/pricing?tier=scale"),
  "benchmark upgrade CTA should link to scale pricing",
);
assert(
  html.includes("Anonymized peer benchmarks are available on Scale"),
  "benchmark prompt should explain the gated feature",
);

console.log("BenchmarkPage gated state renders correctly");
