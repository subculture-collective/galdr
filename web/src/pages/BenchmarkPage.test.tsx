import { cleanup, render, screen, waitFor } from "@testing-library/react";
import "@testing-library/jest-dom/vitest";
import type { AxiosResponse } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import BenchmarkPage from "./BenchmarkPage";
import api, { benchmarksApi } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  default: {
    get: vi.fn(),
  },
  benchmarksApi: {
    compare: vi.fn(),
  },
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
});
