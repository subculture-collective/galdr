import { cleanup, render, screen, waitFor } from "@testing-library/react";
import "@testing-library/jest-dom/vitest";
import userEvent from "@testing-library/user-event";
import type { AxiosResponse } from "axios";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "@/contexts/ToastContext";
import api from "@/lib/api";
import AccountAssignment from "./AccountAssignment";

vi.mock("@/lib/api", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  },
}));

const mockedApi = vi.mocked(api);

function apiResponse<T>(data: T) {
  return { data } as AxiosResponse<T>;
}

describe("AccountAssignment", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mockedApi.get.mockImplementation((url: string) => {
      if (url === "/customers/customer-1/assignments") {
        return Promise.resolve(apiResponse({ assignments: [] }));
      }
      if (url === "/members") {
        return Promise.resolve(
          apiResponse({
            members: [
              {
                user_id: "user-1",
                email: "ada@example.com",
                first_name: "Ada",
                last_name: "Lovelace",
              },
            ],
          }),
        );
      }
      throw new Error(`unexpected GET ${url}`);
    });
    mockedApi.post.mockResolvedValue(
      apiResponse({
        customer_id: "customer-1",
        user_id: "user-1",
        assignee: {
          id: "user-1",
          name: "Ada Lovelace",
          email: "ada@example.com",
          avatar_url: "",
        },
        assigned_at: "2026-05-05T12:00:00Z",
        assigned_by: "owner-1",
      }),
    );
    mockedApi.delete.mockResolvedValue(apiResponse({}));
  });

  afterEach(() => {
    cleanup();
  });

  it("assigns and unassigns team members for a customer", async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
        <AccountAssignment customerId="customer-1" />
      </ToastProvider>,
    );

    expect(await screen.findByText("No assignees yet.")).toBeInTheDocument();
    await user.selectOptions(screen.getByRole("combobox"), "user-1");
    await user.click(screen.getByRole("button", { name: /assign/i }));

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        "/customers/customer-1/assignments",
        { user_id: "user-1" },
      );
    });
    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Remove assignee" }));

    await waitFor(() => {
      expect(mockedApi.delete).toHaveBeenCalledWith(
        "/customers/customer-1/assignments/user-1",
      );
    });
    expect(await screen.findByText("No assignees yet.")).toBeInTheDocument();
  });
});
