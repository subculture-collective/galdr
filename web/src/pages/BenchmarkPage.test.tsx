import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { BenchmarkPageView } from "./BenchmarkPage";

function assert(condition: boolean, message: string) {
  if (!condition) {
    throw new Error(message);
  }
}

function render(component: React.ReactElement) {
  return renderToStaticMarkup(component);
}

function testBenchmarkDashboardShell() {
  const html = render(
    React.createElement(BenchmarkPageView, {
      industry: "SaaS",
      size: "51-200",
      participating: true,
      loading: false,
      error: false,
      metrics: [
        {
          key: "health_score",
          label: "Avg health score",
          unit: "score",
          yourValue: 82,
          percentile: 78,
          benchmarks: { p25: 61, p50: 70, p75: 79 },
          sampleCount: 42,
        },
        {
          key: "mrr_per_customer",
          label: "MRR/customer",
          unit: "currency",
          yourValue: 42000,
          percentile: 64,
          benchmarks: { p25: 20000, p50: 35000, p75: 50000 },
          sampleCount: 42,
        },
        {
          key: "churn_rate",
          label: "Churn rate",
          unit: "percent",
          yourValue: 0.04,
          percentile: 36,
          benchmarks: { p25: 0.02, p50: 0.05, p75: 0.09 },
          sampleCount: 42,
        },
        {
          key: "integration_usage",
          label: "Integration count",
          unit: "count",
          yourValue: 3,
          percentile: 72,
          benchmarks: { p25: 1, p50: 2, p75: 4 },
          sampleCount: 42,
        },
      ],
      calloutPercentile: 78,
      onIndustryChange: () => undefined,
      onSizeChange: () => undefined,
      onRetry: () => undefined,
    }),
  );

  assert(html.includes("Benchmark comparison"), "title should render");
  assert(html.includes("Industry"), "industry selector should render");
  assert(html.includes("Company size"), "size selector should render");
  assert(html.includes("51-200 customers"), "selected size segment should render");
  assert(
    html.includes("You are at the 78th percentile"),
    "percentile callout should render",
  );
  assert(html.includes("Avg health score"), "benchmark metric should render");
  assert(html.includes("MRR/customer"), "MRR metric should render");
  assert(html.includes("Churn rate"), "churn metric should render");
  assert(html.includes("Integration count"), "integration metric should render");
}

function testOptInPromptAndMissingData() {
  const html = render(
    React.createElement(BenchmarkPageView, {
      industry: "SaaS",
      size: "51-200",
      participating: false,
      loading: false,
      error: false,
      metrics: [],
      calloutPercentile: null,
      onIndustryChange: () => undefined,
      onSizeChange: () => undefined,
      onRetry: () => undefined,
    }),
  );

  assert(html.includes("Opt in to benchmarking"), "opt-in prompt should render");
  assert(html.includes("No benchmark data yet"), "empty state should render");
}

testBenchmarkDashboardShell();
testOptInPromptAndMissingData();

console.log("BenchmarkPage tests passed");
