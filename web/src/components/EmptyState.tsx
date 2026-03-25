import type { ReactNode } from "react";
import { Inbox } from "lucide-react";

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;
  actionLabel?: string;
  onAction?: () => void;
}

export default function EmptyState({
  icon,
  title,
  description,
  actionLabel,
  onAction,
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <div className="mb-4 text-[var(--galdr-fg-muted)]" aria-hidden="true">
        {icon ?? <Inbox className="h-12 w-12" />}
      </div>
      <h3 className="text-lg font-medium text-[var(--galdr-fg)]">{title}</h3>
      {description && (
        <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
          {description}
        </p>
      )}
      {actionLabel && onAction && (
        <button
          onClick={onAction}
          className="galdr-button-primary mt-4 px-4 py-2 text-sm font-medium"
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}
