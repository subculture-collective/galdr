import { cleanup, render, screen, waitFor } from "@testing-library/react";
import "@testing-library/jest-dom/vitest";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { AxiosResponse } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import BenchmarkPage, {
  BenchmarkAccessPrompt,
  BenchmarkPageView,
} from "./BenchmarkPage";
import api, { benchmarksApi } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  default: {
    get: vi.fn(),
  },
  benchmarksApi: {
    compare: vi.fn(),
  },
}));

vi.mock("@/contexts/FeatureFlagContext", () => ({
  FEATURE_BENCHMARKS: "benchmarks",
  useFeatureFlag: vi.fn(() => ({
    allowed: true,
    current: null,
    currentPlan: "scale",
    limit: null,
    recommendedTier: null,
  })),
}));

const mockedApi = vi.mocked(api);
const mockedBenchmarksApi = vi.mocked(benchmarksApi);

function apiResponse<T>(data: T) {
  return { data } as AxiosResponse<T>;
}

describe("BenchmarkPage", () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("defaults missing benchmark participation to opted out", async () => {
    mockedApi.get.mockResolvedValue(
      apiResponse({ industry: "SaaS", company_size: 25 }),
    );

    render(<BenchmarkPage />);

    expect(
      await screen.findByText("Opt in to benchmarking"),
    ).toBeInTheDocument();
    expect(screen.getByText("Review settings")).toHaveAttribute(
      "href",
      "/settings/organization",
    );

    await waitFor(() => {
      expect(mockedBenchmarksApi.compare).not.toHaveBeenCalled();
    });
  });

  it("renders the scale upgrade prompt", () => {
    const html = renderToStaticMarkup(
      React.createElement(BenchmarkAccessPrompt, { recommendedTier: "scale" }),
    );

    expect(html).toContain("Upgrade to Scale to access Benchmarking");
    expect(html).toContain("/pricing?tier=scale");
    expect(html).toContain("Anonymized peer benchmarks are available on Scale");
  });

  it("renders the benchmark comparison dashboard shell", () => {
    const html = renderToStaticMarkup(
      <BenchmarkPageView
        industry="SaaS"
        size="51-200"
        participating={true}
        loading={false}
        error={false}
        metrics={[
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
        ]}
        calloutPercentile={78}
        onIndustryChange={() => undefined}
        onSizeChange={() => undefined}
        onRetry={() => undefined}
      />,
    );

    expect(html).toContain("Benchmark comparison");
    expect(html).toContain("Industry");
    expect(html).toContain("Company size");
    expect(html).toContain("51-200 customers");
    expect(html).toContain("You are at the 78th percentile");
    expect(html).toContain("Avg health score");
    expect(html).toContain("MRR/customer");
    expect(html).toContain("Churn rate");
    expect(html).toContain("Integration count");
  });
});
