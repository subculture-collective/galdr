import type { ReactNode } from "react";

interface BaseLayoutProps {
  children: ReactNode;
}

export default function BaseLayout({ children }: BaseLayoutProps) {
  return (
    <div className="galdr-shell min-h-screen text-[var(--galdr-fg)]">
      <header className="border-b border-[var(--galdr-border)] bg-[color-mix(in_srgb,var(--galdr-surface)_85%,black_15%)]/90 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
          <h1 className="text-xl font-bold tracking-tight text-[var(--galdr-fg)]">
            PulseScore
          </h1>
          <nav className="flex gap-4 text-sm font-medium text-[var(--galdr-fg-muted)]">
            <a href="/" className="galdr-link">
              Dashboard
            </a>
            <a href="/customers" className="galdr-link">
              Customers
            </a>
            <a href="/settings" className="galdr-link">
              Settings
            </a>
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
        {children}
      </main>
    </div>
  );
}
