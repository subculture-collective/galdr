import { Link } from "react-router-dom";
import { FileQuestion } from "lucide-react";

export default function NotFoundPage() {
  return (
    <div className="galdr-shell flex min-h-screen items-center justify-center p-8">
      <div className="galdr-card w-full max-w-xl p-8 text-center">
        <FileQuestion className="mx-auto mb-6 h-16 w-16 text-[var(--galdr-fg-muted)]" />
        <h1 className="text-3xl font-bold text-[var(--galdr-fg)]">
          Page not found
        </h1>
        <p className="mt-2 text-[var(--galdr-fg-muted)]">
          The page you're looking for doesn't exist or has been moved.
        </p>
        <Link
          to="/"
          className="galdr-button-primary mt-6 inline-flex px-4 py-2 text-sm font-medium"
        >
          Back to home
        </Link>
      </div>
    </div>
  );
}
