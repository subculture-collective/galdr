import { useState, useRef, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Bell } from "lucide-react";
import { notificationsApi, type AppNotification } from "@/lib/api";
import { relativeTime } from "@/lib/format";

const POLL_INTERVAL = 30_000;

export default function NotificationBell() {
  const [unreadCount, setUnreadCount] = useState(0);
  const [notifications, setNotifications] = useState<AppNotification[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  const fetchCount = useCallback(async () => {
    try {
      const { data } = await notificationsApi.unreadCount();
      setUnreadCount(data.count);
    } catch {
      // Silently fail — don't block UI
    }
  }, []);

  // Poll unread count
  useEffect(() => {
    fetchCount();
    const timer = setInterval(fetchCount, POLL_INTERVAL);
    return () => clearInterval(timer);
  }, [fetchCount]);

  // Close on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    function handleEscape(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open]);

  async function toggleOpen() {
    if (!open) {
      setLoading(true);
      try {
        const { data } = await notificationsApi.list({ limit: 10 });
        setNotifications(data.notifications ?? []);
      } catch {
        setNotifications([]);
      } finally {
        setLoading(false);
      }
    }
    setOpen(!open);
  }

  async function handleMarkRead(id: string) {
    await notificationsApi.markRead(id);
    setNotifications((prev) =>
      prev.map((n) =>
        n.id === id ? { ...n, read_at: new Date().toISOString() } : n,
      ),
    );
    setUnreadCount((c) => Math.max(0, c - 1));
  }

  async function handleMarkAllRead() {
    await notificationsApi.markAllRead();
    setNotifications((prev) =>
      prev.map((n) => ({
        ...n,
        read_at: n.read_at ?? new Date().toISOString(),
      })),
    );
    setUnreadCount(0);
  }

  function handleNotificationClick(n: AppNotification) {
    if (!n.read_at) handleMarkRead(n.id);
    const customerId = (n.data as Record<string, string>)?.customer_id;
    if (customerId) {
      navigate(`/customers/${customerId}`);
      setOpen(false);
    }
  }

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={toggleOpen}
        className="galdr-icon-button relative p-2"
        aria-label="Notifications"
      >
        <Bell className="h-5 w-5" />
        {unreadCount > 0 && (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-[var(--galdr-danger)] px-1 text-[10px] font-bold text-white">
            {unreadCount > 99 ? "99+" : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="galdr-panel absolute right-0 top-full z-50 mt-2 w-80 shadow-lg">
          <div className="flex items-center justify-between border-b border-[var(--galdr-border)] px-4 py-3">
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              Notifications
            </h3>
            {unreadCount > 0 && (
              <button
                onClick={handleMarkAllRead}
                className="galdr-link text-xs"
              >
                Mark all read
              </button>
            )}
          </div>

          <div className="max-h-80 overflow-y-auto">
            {loading ? (
              <div className="px-4 py-6 text-center text-sm text-[var(--galdr-fg-muted)]">
                Loading...
              </div>
            ) : notifications.length === 0 ? (
              <div className="px-4 py-6 text-center text-sm text-[var(--galdr-fg-muted)]">
                No notifications
              </div>
            ) : (
              notifications.map((n) => (
                <button
                  key={n.id}
                  onClick={() => handleNotificationClick(n)}
                  className={`block w-full px-4 py-3 text-left transition-colors hover:bg-[color:rgb(139_92_246_/_0.08)] ${
                    !n.read_at ? "bg-[color:rgb(139_92_246_/_0.1)]" : ""
                  }`}
                >
                  <div className="flex items-start gap-2">
                    {!n.read_at && (
                      <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-[var(--galdr-accent)]" />
                    )}
                    <div className={!n.read_at ? "" : "ml-4"}>
                      <p className="text-sm font-medium text-[var(--galdr-fg)]">
                        {n.title}
                      </p>
                      <p className="mt-0.5 text-xs text-[var(--galdr-fg-muted)] line-clamp-2">
                        {n.message}
                      </p>
                      <p className="mt-1 text-[10px] text-[color:rgb(168_168_188_/_0.86)]">
                        {relativeTime(n.created_at)}
                      </p>
                    </div>
                  </div>
                </button>
              ))
            )}
          </div>

          <div className="border-t border-[var(--galdr-border)]">
            <button
              onClick={() => {
                setOpen(false);
                navigate("/settings/notifications");
              }}
              className="galdr-link block w-full px-4 py-2.5 text-center text-xs hover:bg-[color:rgb(139_92_246_/_0.08)]"
            >
              Notification settings
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
