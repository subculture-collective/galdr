import { useState } from "react";
import type { FormEvent } from "react";
import api from "@/lib/api";
import type { MarketplaceManifest } from "@/pages/MarketplacePage";

type SubmissionState = "idle" | "submitting" | "submitted" | "error";

interface SubmitConnectorPageViewProps {
  manifestText: string;
  sourceUrl: string;
  description: string;
  screenshots: string;
  status: SubmissionState;
  error: string;
  onManifestTextChange: (value: string) => void;
  onSourceUrlChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onScreenshotsChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

export function SubmitConnectorPageView({
  manifestText,
  sourceUrl,
  description,
  screenshots,
  status,
  error,
  onManifestTextChange,
  onSourceUrlChange,
  onDescriptionChange,
  onScreenshotsChange,
  onSubmit,
}: SubmitConnectorPageViewProps) {
  const submitting = status === "submitting";

  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-8 sm:px-6 lg:px-8">
      <section className="rounded-[2rem] border border-[var(--galdr-border)] bg-[var(--galdr-surface)] p-8 shadow-sm">
        <p className="text-sm font-semibold uppercase tracking-[0.24em] text-[var(--galdr-accent)]">
          Developer portal
        </p>
        <h1 className="mt-3 text-3xl font-semibold text-[var(--galdr-fg)]">
          Submit a community connector
        </h1>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-[var(--galdr-fg-muted)]">
          Upload a valid connector manifest, source URL, description, and
          optional screenshots. Submissions enter admin review before
          marketplace publish.
        </p>
      </section>

      <form
        className="grid gap-5 rounded-[2rem] border border-[var(--galdr-border)] bg-[var(--galdr-surface)] p-6 shadow-sm"
        onSubmit={onSubmit}
      >
        <label className="grid gap-2 text-sm font-medium text-[var(--galdr-fg)]">
          Manifest JSON
          <textarea
            className="min-h-56 rounded-2xl border border-[var(--galdr-border)] bg-[var(--galdr-bg)] p-4 font-mono text-sm text-[var(--galdr-fg)] outline-none focus:border-[var(--galdr-accent)]"
            value={manifestText}
            onChange={(event) => onManifestTextChange(event.target.value)}
            placeholder='{"id":"mock-crm","name":"Mock CRM","version":"1.0.0",...}'
            required
          />
        </label>

        <label className="grid gap-2 text-sm font-medium text-[var(--galdr-fg)]">
          Source URL
          <input
            className="rounded-2xl border border-[var(--galdr-border)] bg-[var(--galdr-bg)] px-4 py-3 text-sm text-[var(--galdr-fg)] outline-none focus:border-[var(--galdr-accent)]"
            value={sourceUrl}
            onChange={(event) => onSourceUrlChange(event.target.value)}
            placeholder="https://github.com/example/mock-crm-connector"
            type="url"
            required
          />
        </label>

        <label className="grid gap-2 text-sm font-medium text-[var(--galdr-fg)]">
          Review description
          <textarea
            className="min-h-28 rounded-2xl border border-[var(--galdr-border)] bg-[var(--galdr-bg)] p-4 text-sm text-[var(--galdr-fg)] outline-none focus:border-[var(--galdr-accent)]"
            value={description}
            onChange={(event) => onDescriptionChange(event.target.value)}
            placeholder="Explain data accessed, auth flow, and customer value."
            required
          />
        </label>

        <label className="grid gap-2 text-sm font-medium text-[var(--galdr-fg)]">
          Screenshot URLs
          <input
            className="rounded-2xl border border-[var(--galdr-border)] bg-[var(--galdr-bg)] px-4 py-3 text-sm text-[var(--galdr-fg)] outline-none focus:border-[var(--galdr-accent)]"
            value={screenshots}
            onChange={(event) => onScreenshotsChange(event.target.value)}
            placeholder="https://example.com/screenshot.png"
          />
        </label>

        {error ? (
          <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            {error}
          </div>
        ) : null}
        {status === "submitted" ? (
          <div className="rounded-2xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-700">
            Connector submitted for review. We will email the developer when the
            status changes.
          </div>
        ) : null}

        <button
          className="rounded-2xl bg-[var(--galdr-accent)] px-5 py-3 text-sm font-semibold text-white disabled:opacity-60"
          disabled={submitting}
          type="submit"
        >
          {submitting ? "Submitting..." : "Submit connector"}
        </button>
      </form>
    </div>
  );
}

export default function SubmitConnectorPage() {
  const [manifestText, setManifestText] = useState("");
  const [sourceUrl, setSourceUrl] = useState("");
  const [description, setDescription] = useState("");
  const [screenshots, setScreenshots] = useState("");
  const [status, setStatus] = useState<SubmissionState>("idle");
  const [error, setError] = useState("");

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setStatus("submitting");

    let manifest: MarketplaceManifest;
    try {
      manifest = JSON.parse(manifestText) as MarketplaceManifest;
    } catch {
      setStatus("error");
      setError("Manifest must be valid JSON.");
      return;
    }

    try {
      await api.post("/marketplace/connectors", {
        manifest,
        status: "submitted",
        source_url: sourceUrl,
        description,
        screenshots: screenshots
          .split(",")
          .map((value) => value.trim())
          .filter(Boolean),
      });
      setStatus("submitted");
    } catch (err) {
      setStatus("error");
      setError(err instanceof Error ? err.message : "Submission failed.");
    }
  }

  return (
    <SubmitConnectorPageView
      manifestText={manifestText}
      sourceUrl={sourceUrl}
      description={description}
      screenshots={screenshots}
      status={status}
      error={error}
      onManifestTextChange={setManifestText}
      onSourceUrlChange={setSourceUrl}
      onDescriptionChange={setDescription}
      onScreenshotsChange={setScreenshots}
      onSubmit={handleSubmit}
    />
  );
}
