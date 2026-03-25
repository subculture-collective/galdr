import { Link, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  Users,
  Plug,
  Settings,
  ChevronLeft,
  ChevronRight,
  X,
} from "lucide-react";

const navItems = [
  { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { to: "/customers", label: "Customers", icon: Users },
  { to: "/settings/integrations", label: "Integrations", icon: Plug },
  { to: "/settings", label: "Settings", icon: Settings, end: true },
];

interface SidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
  mobileOpen: boolean;
  onCloseMobile: () => void;
}

export default function Sidebar({
  collapsed,
  onToggleCollapse,
  mobileOpen,
  onCloseMobile,
}: SidebarProps) {
  const location = useLocation();

  function isActive(to: string, end?: boolean) {
    if (end) return location.pathname === to;
    if (to === "/dashboard") return location.pathname === "/dashboard";
    return location.pathname.startsWith(to);
  }

  const nav = (
    <nav className="flex flex-1 flex-col gap-1 px-3 py-4">
      {navItems.map((item) => {
        const active = isActive(item.to, item.end);
        const Icon = item.icon;
        return (
          <Link
            key={item.to}
            to={item.to}
            onClick={onCloseMobile}
            title={collapsed ? item.label : undefined}
            className={`galdr-nav-item flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium ${active ? "galdr-nav-item-active" : ""} ${collapsed ? "justify-center" : ""}`}
          >
            <Icon className="h-5 w-5 shrink-0" />
            {!collapsed && <span>{item.label}</span>}
          </Link>
        );
      })}
    </nav>
  );

  return (
    <>
      {/* Mobile overlay */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/60 md:hidden"
          onClick={onCloseMobile}
        />
      )}

      {/* Mobile drawer */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-[var(--galdr-border)] bg-[color:rgb(18_18_26_/_0.98)] transition-transform duration-300 md:hidden ${
          mobileOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <div className="flex h-16 items-center justify-between border-b border-[var(--galdr-border)] px-4">
          <span className="text-lg font-bold tracking-tight text-[var(--galdr-fg)]">
            Galdr
          </span>
          <button
            onClick={onCloseMobile}
            className="galdr-icon-button p-1.5"
            aria-label="Close navigation"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        {nav}
      </aside>

      {/* Desktop sidebar */}
      <aside
        className={`hidden flex-col border-r border-[var(--galdr-border)] bg-[color:rgb(18_18_26_/_0.98)] transition-all duration-300 md:flex ${
          collapsed ? "w-16" : "w-64"
        }`}
      >
        <div
          className={`flex h-16 items-center border-b border-[var(--galdr-border)] ${
            collapsed ? "justify-center px-2" : "px-4"
          }`}
        >
          {!collapsed && (
            <span className="text-lg font-bold tracking-tight text-[var(--galdr-fg)]">
              Galdr
            </span>
          )}
          {collapsed && (
            <span className="text-lg font-bold tracking-tight text-[var(--galdr-fg)]">
              GL
            </span>
          )}
        </div>
        {nav}
        <div className="border-t border-[var(--galdr-border)] p-3">
          <button
            onClick={onToggleCollapse}
            className="galdr-icon-button flex w-full items-center justify-center p-2"
            title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <ChevronLeft className="h-4 w-4" />
            )}
          </button>
        </div>
      </aside>
    </>
  );
}
