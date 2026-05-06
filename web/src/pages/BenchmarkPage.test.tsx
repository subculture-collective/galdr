import { cleanup, render, screen, waitFor } from "@testing-library/react";
import "@testing-library/jest-dom/vitest";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { AxiosResponse } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import BenchmarkPage, { BenchmarkAccessPrompt } from "./BenchmarkPage";
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
});
