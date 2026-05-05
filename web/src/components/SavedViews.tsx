import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Bookmark, Save, Share2, Trash2 } from "lucide-react";
import { savedViewsApi, type SavedView } from "@/lib/api";
import {
  applyFiltersToSearchParams,
  filtersFromSearchParams,
} from "@/lib/savedViewFilters";
import { useToast } from "@/contexts/ToastContext";

export default function SavedViews() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [views, setViews] = useState<SavedView[]>([]);
  const [selectedViewId, setSelectedViewId] = useState("");
  const [name, setName] = useState("");
  const [isShared, setIsShared] = useState(false);
  const [loading, setLoading] = useState(false);
  const toast = useToast();

  const loadViews = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await savedViewsApi.list();
      setViews(data.views ?? []);
    } catch {
      toast.error("Failed to load saved views");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    loadViews();
  }, [loadViews]);

  const selectedView = views.find((view) => view.id === selectedViewId);

  function selectView(id: string) {
    setSelectedViewId(id);
    const view = views.find((item) => item.id === id);
    if (!view) {
      return;
    }
    setName(view.name);
    setIsShared(view.is_shared);
    setSearchParams((prev) => applyFiltersToSearchParams(prev, view.filters));
  }

  async function createView() {
    const trimmedName = name.trim();
    if (!trimmedName) {
      toast.warning("Name the saved view first");
      return;
    }
    try {
      const { data } = await savedViewsApi.create({
        name: trimmedName,
        filters: filtersFromSearchParams(searchParams),
        is_shared: isShared,
      });
      setViews((prev) => [data, ...prev]);
      setSelectedViewId(data.id);
      toast.success("Saved view created");
    } catch {
      toast.error("Failed to save view");
    }
  }

  async function updateView() {
    if (!selectedView) {
      return;
    }

    const trimmedName = name.trim();
    try {
      const { data } = await savedViewsApi.update(selectedView.id, {
        name: trimmedName || selectedView.name,
        filters: filtersFromSearchParams(searchParams),
        is_shared: isShared,
      });
      setViews((prev) =>
        prev.map((view) => (view.id === data.id ? data : view)),
      );
      toast.success("Saved view updated");
    } catch {
      toast.error("Only the owner can update this view");
    }
  }

  async function deleteView() {
    if (!selectedView) {
      return;
    }

    try {
      await savedViewsApi.delete(selectedView.id);
      setViews((prev) => prev.filter((view) => view.id !== selectedView.id));
      setSelectedViewId("");
      setName("");
      setIsShared(false);
      toast.success("Saved view deleted");
    } catch {
      toast.error("Only the owner can delete this view");
    }
  }

  return (
    <section className="galdr-card space-y-4 p-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div className="flex items-center gap-2 text-sm font-semibold text-[var(--galdr-fg)]">
            <Bookmark className="h-4 w-4 text-[var(--galdr-accent)]" />
            Saved views
          </div>
          <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
            Save this customer filter setup, then share it with the team.
          </p>
        </div>

        <div className="grid gap-2 md:grid-cols-[minmax(180px,1fr)_minmax(180px,1fr)_auto_auto] md:items-center">
          <select
            value={selectedViewId}
            onChange={(event) => selectView(event.target.value)}
            disabled={loading}
            className="galdr-input text-sm"
            aria-label="Load saved view"
          >
            <option value="">
              {loading ? "Loading views..." : "Load view"}
            </option>
            {views.map((view) => (
              <option key={view.id} value={view.id}>
                {view.name} {view.is_shared ? "(shared)" : "(private)"}
              </option>
            ))}
          </select>

          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            className="galdr-input text-sm"
            placeholder="View name"
            maxLength={120}
          />

          <label className="inline-flex items-center gap-2 text-sm text-[var(--galdr-fg-muted)]">
            <input
              type="checkbox"
              checked={isShared}
              onChange={(event) => setIsShared(event.target.checked)}
              className="h-4 w-4 rounded border-[var(--galdr-border)]"
            />
            <Share2 className="h-4 w-4" />
            Share
          </label>

          <div className="flex gap-2">
            <button
              type="button"
              onClick={createView}
              className="galdr-button-primary inline-flex items-center gap-1 px-3 py-2 text-sm"
            >
              <Save className="h-4 w-4" />
              Save
            </button>
            <button
              type="button"
              onClick={updateView}
              disabled={!selectedView}
              className="galdr-button-secondary px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50"
            >
              Update
            </button>
            <button
              type="button"
              onClick={deleteView}
              disabled={!selectedView}
              className="galdr-button-secondary inline-flex items-center gap-1 px-3 py-2 text-sm text-red-600 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </button>
          </div>
        </div>
      </div>
    </section>
  );
}
