import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import Toast from "@/components/Toast";

type ToastType = "success" | "error" | "warning" | "info";

interface ToastItem {
  id: number;
  type: ToastType;
  message: string;
  autoClose: boolean;
}

interface ToastContextValue {
  success: (message: string) => void;
  error: (message: string) => void;
  warning: (message: string) => void;
  info: (message: string) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be used within ToastProvider");
  return ctx;
}

let nextId = 0;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const removeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const addToast = useCallback(
    (type: ToastType, message: string) => {
      const id = ++nextId;
      const autoClose = type !== "error";
      setToasts((prev) => {
        const next = [...prev, { id, type, message, autoClose }];
        // Max 3 visible
        return next.slice(-3);
      });
      if (autoClose) {
        setTimeout(() => removeToast(id), 5000);
      }
    },
    [removeToast],
  );

  const success = useCallback(
    (msg: string) => addToast("success", msg),
    [addToast],
  );
  const error = useCallback(
    (msg: string) => addToast("error", msg),
    [addToast],
  );
  const warning = useCallback(
    (msg: string) => addToast("warning", msg),
    [addToast],
  );
  const info = useCallback((msg: string) => addToast("info", msg), [addToast]);

  const value = useMemo<ToastContextValue>(
    () => ({ success, error, warning, info }),
    [success, error, warning, info],
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="fixed right-4 top-4 z-50 flex flex-col gap-2">
        {toasts.map((t) => (
          <Toast
            key={t.id}
            type={t.type}
            message={t.message}
            onClose={() => removeToast(t.id)}
          />
        ))}
      </div>
    </ToastContext.Provider>
  );
}
