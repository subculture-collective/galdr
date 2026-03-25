import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Search, X } from "lucide-react";

const riskOptions = [
  { label: "All", value: "" },
  { label: "Green", value: "green" },
  { label: "Yellow", value: "yellow" },
  { label: "Red", value: "red" },
];

const sourceOptions = [
  { label: "All Sources", value: "" },
  { label: "Stripe", value: "stripe" },
  { label: "HubSpot", value: "hubspot" },
  { label: "Intercom", value: "intercom" },
];

export default function CustomerFilters() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [searchInput, setSearchInput] = useState(
    searchParams.get("search") ?? "",
  );
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const risk = searchParams.get("risk") ?? "";
  const source = searchParams.get("source") ?? "";

  function updateParam(key: string, value: string) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (value) {
        next.set(key, value);
      } else {
        next.delete(key);
      }
      next.delete("page"); // reset page on filter change
      return next;
    });
  }

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      updateParam("search", searchInput);
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchInput]);

  function clearAll() {
    setSearchInput("");
    setSearchParams(new URLSearchParams());
  }

  const hasFilters = !!(risk || source || searchInput);

  return (
    <div className="flex flex-wrap items-center gap-3">
      {/* Search */}
      <div className="relative flex-1 sm:max-w-xs">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--galdr-fg-muted)]" />
        <input
          type="text"
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          placeholder="Search customers..."
          className="galdr-input w-full py-2 pl-9 pr-3 text-sm"
        />
      </div>

      {/* Risk filter */}
      <div className="flex gap-1">
        {riskOptions.map((opt) => (
          <button
            key={opt.value}
            onClick={() => updateParam("risk", opt.value)}
            className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
              risk === opt.value
                ? "border border-[color:rgb(139_92_246_/_0.45)] bg-[color:rgb(139_92_246_/_0.2)] text-[var(--galdr-fg)]"
                : "galdr-pill hover:border-[color:rgb(139_92_246_/_0.3)] hover:text-[var(--galdr-fg)]"
            }`}
          >
            {opt.label}
          </button>
        ))}
      </div>

      {/* Source filter */}
      <select
        value={source}
        onChange={(e) => updateParam("source", e.target.value)}
        className="galdr-input px-3 py-1.5 text-sm"
      >
        {sourceOptions.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>

      {/* Clear all */}
      {hasFilters && (
        <button
          onClick={clearAll}
          className="galdr-link flex items-center gap-1 text-xs"
        >
          <X className="h-3 w-3" />
          Clear all
        </button>
      )}
    </div>
  );
}
