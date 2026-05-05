import type { SavedViewFilters } from "@/lib/api";

const CUSTOMER_FILTER_PARAM_KEYS = [
  "search",
  "risk",
  "source",
  "sort",
  "order",
  "assignee",
  "tags",
] as const;

export function filtersFromSearchParams(
  params: URLSearchParams,
): SavedViewFilters {
  const tags = params
    .get("tags")
    ?.split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);

  return compactFilters({
    search: params.get("search") ?? undefined,
    risk_level: params.get("risk") ?? undefined,
    source: params.get("source") ?? undefined,
    sort: params.get("sort") ?? undefined,
    order: params.get("order") ?? undefined,
    assignee: params.get("assignee") ?? undefined,
    tags: tags && tags.length > 0 ? tags : undefined,
  });
}

export function applyFiltersToSearchParams(
  current: URLSearchParams,
  filters: SavedViewFilters,
): URLSearchParams {
  const next = new URLSearchParams(current);
  CUSTOMER_FILTER_PARAM_KEYS.forEach((key) => next.delete(key));
  next.delete("page");

  setIfPresent(next, "search", filters.search);
  setIfPresent(next, "risk", filters.risk_level);
  setIfPresent(next, "source", filters.source);
  setIfPresent(next, "sort", filters.sort);
  setIfPresent(next, "order", filters.order);
  setIfPresent(next, "assignee", filters.assignee);
  if (filters.tags && filters.tags.length > 0) {
    next.set("tags", filters.tags.join(","));
  }

  return next;
}

function compactFilters(filters: SavedViewFilters): SavedViewFilters {
  return Object.fromEntries(
    Object.entries(filters).filter(([, value]) => {
      if (Array.isArray(value)) return value.length > 0;
      return value !== undefined && value !== "";
    }),
  ) as SavedViewFilters;
}

function setIfPresent(
  params: URLSearchParams,
  key: string,
  value: string | undefined,
) {
  if (value) {
    params.set(key, value);
  }
}
