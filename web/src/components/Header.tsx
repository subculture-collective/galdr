import { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@/contexts/AuthContext";
import { LogOut, User, Sun, Moon, Monitor, Menu } from "lucide-react";
import { useTheme } from "@/contexts/ThemeContext";
import NotificationBell from "@/components/NotificationBell";

interface HeaderProps {
  onOpenMobileNav?: () => void;
}

export default function Header({ onOpenMobileNav }: HeaderProps) {
  const { user, organization, logout } = useAuth();
  const { theme, setTheme } = useTheme();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  useEffect(() => {
    if (!menuOpen) return;
    function handleEscape(e: KeyboardEvent) {
      if (e.key === "Escape") setMenuOpen(false);
    }
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [menuOpen]);

  function handleLogout() {
    logout();
    navigate("/login");
  }

  function cycleTheme() {
    const next =
      theme === "light" ? "dark" : theme === "dark" ? "system" : "light";
    setTheme(next);
  }

  const themeIcon =
    theme === "dark" ? (
      <Moon className="h-4 w-4" />
    ) : theme === "light" ? (
      <Sun className="h-4 w-4" />
    ) : (
      <Monitor className="h-4 w-4" />
    );

  return (
    <header className="flex h-16 items-center justify-between border-b border-[var(--galdr-border)] bg-[color:rgb(18_18_26_/_0.88)] px-4 backdrop-blur sm:px-6">
      <div className="flex items-center gap-2">
        {onOpenMobileNav && (
          <button
            onClick={onOpenMobileNav}
            className="galdr-icon-button p-2 md:hidden"
            aria-label="Open navigation"
          >
            <Menu className="h-5 w-5" />
          </button>
        )}
        {organization && (
          <span className="text-sm font-medium text-[var(--galdr-fg)]">
            {organization.name}
          </span>
        )}
      </div>

      <div className="flex items-center gap-2">
        <NotificationBell />

        <button
          onClick={cycleTheme}
          className="galdr-icon-button p-2"
          title={`Theme: ${theme}`}
          aria-label={`Toggle theme, current: ${theme}`}
        >
          {themeIcon}
        </button>

        <div className="relative" ref={menuRef}>
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="galdr-icon-button flex items-center gap-2 px-3 py-2 text-sm font-medium"
            aria-label="User menu"
          >
            <div className="flex h-7 w-7 items-center justify-center rounded-full border border-[color:rgb(139_92_246_/_0.4)] bg-[color:rgb(139_92_246_/_0.16)] text-xs font-semibold text-[color:rgb(224_212_255)]">
              {user?.first_name?.[0]}
              {user?.last_name?.[0]}
            </div>
            <span className="hidden sm:block">
              {user?.first_name} {user?.last_name}
            </span>
          </button>

          {menuOpen && (
            <div className="galdr-panel absolute right-0 top-full mt-1 w-48 overflow-hidden py-1 shadow-lg">
              <button
                onClick={() => {
                  setMenuOpen(false);
                  navigate("/settings/profile");
                }}
                className="flex w-full items-center gap-2 px-4 py-2 text-sm text-[var(--galdr-fg)] transition-colors hover:bg-[color:rgb(139_92_246_/_0.12)]"
              >
                <User className="h-4 w-4" />
                Profile
              </button>
              <hr className="my-1 border-[var(--galdr-border)]" />
              <button
                onClick={handleLogout}
                className="flex w-full items-center gap-2 px-4 py-2 text-sm text-[var(--galdr-danger)] transition-colors hover:bg-[color:rgb(244_63_94_/_0.12)]"
              >
                <LogOut className="h-4 w-4" />
                Logout
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
