import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import BenchmarkChart, { type BenchmarkMetric } from "./BenchmarkChart";

function assert(condition: boolean, message: string) {
  if (!condition) {
    throw new Error(message);
  }
}

const metric: BenchmarkMetric = {
  key: "health_score",
  label: "Avg health score",
  unit: "score",
  yourValue: 82,
  percentile: 78,
  benchmarks: {
    p25: 61,
    p50: 70,
    p75: 79,
  },
  sampleCount: 42,
};

function render(component: React.ReactElement) {
  return renderToStaticMarkup(component);
}

function testBenchmarkBarsRender() {
  const html = render(React.createElement(BenchmarkChart, { metric }));

  assert(html.includes("Avg health score"), "metric title should render");
  assert(html.includes("78th percentile"), "percentile callout should render");
  assert(html.includes("Your value"), "org value bar should render");
  assert(html.includes("P25"), "P25 bar should render");
  assert(html.includes("P50"), "P50 bar should render");
  assert(html.includes("P75"), "P75 bar should render");
  assert(html.includes("n=42"), "sample count should render");
}

function testMissingBenchmarksRenderGracefully() {
  const html = render(
    React.createElement(BenchmarkChart, {
      metric: {
        ...metric,
        benchmarks: null,
        percentile: null,
        sampleCount: 0,
      },
    }),
  );

  assert(
    html.includes("Benchmark data unavailable"),
    "missing benchmark message should render",
  );
}

testBenchmarkBarsRender();
testMissingBenchmarksRenderGracefully();

console.log("BenchmarkChart tests passed");
