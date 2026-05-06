import { cleanup, render, screen, waitFor } from "@testing-library/react";
import "@testing-library/jest-dom/vitest";
import userEvent from "@testing-library/user-event";
import type { AxiosResponse } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import BenchmarkSettings from "@/components/settings/BenchmarkSettings";
import { ToastProvider } from "@/contexts/ToastContext";
import api from "@/lib/api";

vi.mock("@/lib/api", () => ({
  default: {
    patch: vi.fn(),
  },
}));

const mockedApi = vi.mocked(api);

function apiResponse<T>(data: T) {
  return { data } as AxiosResponse<T>;
}

describe("BenchmarkSettings", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mockedApi.patch.mockResolvedValue(
      apiResponse({ benchmarking_enabled: false, company_size: 25 }),
    );
  });

  afterEach(() => {
    cleanup();
  });

  it("explains default opt-out sharing and deletes contributions on opt-out", async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    const setSaving = vi.fn();

    render(
      <ToastProvider>
        <BenchmarkSettings
          org={{ benchmarking_enabled: true, company_size: 25 }}
          industry="SaaS"
          saving={false}
          setSaving={setSaving}
          onSaved={onSaved}
        />
      </ToastProvider>,
    );

    expect(screen.getByText("Shared data")).toBeInTheDocument();
    expect(screen.getByText("Data retention")).toBeInTheDocument();
    expect(
      screen.getByText(/Your organization is opted out by default/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Customer PII, customer IDs, names, emails/i),
    ).toBeInTheDocument();
    expect(screen.getByText("Benchmark data usage terms")).toHaveAttribute(
      "href",
      "/legal/benchmark-data-usage",
    );

    await user.click(screen.getByRole("button", { name: "Opt out" }));

    await waitFor(() => {
      expect(mockedApi.patch).toHaveBeenCalledWith("/organizations/current", {
        benchmarking_enabled: false,
        industry: "SaaS",
        company_size: 25,
      });
    });
    expect(onSaved).toHaveBeenCalledWith({
      benchmarking_enabled: false,
      company_size: 25,
    });
  });
});
