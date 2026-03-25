import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import CustomerTable, { type Customer } from "@/components/CustomerTable";
import CustomerFilters from "@/components/CustomerFilters";
import TableSkeleton from "@/components/skeletons/TableSkeleton";
import EmptyState from "@/components/EmptyState";
import { ChevronLeft, ChevronRight, Users } from "lucide-react";

interface CustomersResponse {
  customers: Customer[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export default function CustomersPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [data, setData] = useState<CustomersResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const toast = useToast();

  const page = parseInt(searchParams.get("page") ?? "1", 10);
  const perPage = parseInt(searchParams.get("per_page") ?? "20", 10);
  const sort = searchParams.get("sort") ?? "name";
  const order = (searchParams.get("order") ?? "asc") as "asc" | "desc";
  const risk = searchParams.get("risk") ?? "";
  const search = searchParams.get("search") ?? "";
  const source = searchParams.get("source") ?? "";

  const fetchCustomers = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string | number> = {
        page,
        per_page: perPage,
        sort,
        order,
      };
      if (risk) params.risk = risk;
      if (search) params.search = search;
      if (source) params.source = source;

      const { data: res } = await api.get<CustomersResponse>("/customers", {
        params,
      });
      setData(res);
    } catch {
      toast.error("Failed to load customers");
    } finally {
      setLoading(false);
    }
  }, [page, perPage, sort, order, risk, search, source, toast]);

  useEffect(() => {
    fetchCustomers();
  }, [fetchCustomers]);

  function updateParam(key: string, value: string) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set(key, value);
      return next;
    });
  }

  function handleSort(field: string) {
    if (sort === field) {
      updateParam("order", order === "asc" ? "desc" : "asc");
    } else {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("sort", field);
        next.set("order", "asc");
        return next;
      });
    }
  }

  function goToPage(p: number) {
    updateParam("page", String(p));
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-[var(--galdr-fg)]">Customers</h1>

      <CustomerFilters />

      {loading ? (
        <TableSkeleton />
      ) : !data || data.customers.length === 0 ? (
        <EmptyState
          icon={<Users className="h-12 w-12" />}
          title="No customers found"
          description="Try adjusting your filters or connect an integration to sync customers."
        />
      ) : (
        <>
          <CustomerTable
            customers={data.customers}
            sort={sort}
            order={order}
            onSort={handleSort}
          />

          {/* Pagination */}
          <div className="flex items-center justify-between text-sm text-[var(--galdr-fg-muted)]">
            <span>
              Page {data.page} of {data.total_pages} ({data.total} total)
            </span>
            <div className="flex gap-2">
              <button
                onClick={() => goToPage(page - 1)}
                disabled={page <= 1}
                className="galdr-button-secondary inline-flex items-center gap-1 px-3 py-1.5 text-sm disabled:cursor-not-allowed disabled:opacity-50"
              >
                <ChevronLeft className="h-4 w-4" />
                Previous
              </button>
              <button
                onClick={() => goToPage(page + 1)}
                disabled={page >= (data.total_pages ?? 1)}
                className="galdr-button-secondary inline-flex items-center gap-1 px-3 py-1.5 text-sm disabled:cursor-not-allowed disabled:opacity-50"
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
