import { useEffect, useState } from "react";
import { CheckCircle, XCircle, AlertTriangle, Info, X } from "lucide-react";

interface ToastProps {
  type: "success" | "error" | "warning" | "info";
  message: string;
  onClose: () => void;
}

const styles = {
  success: {
    tone: "galdr-alert-success",
    text: "text-[var(--galdr-success)]",
    Icon: CheckCircle,
  },
  error: {
    tone: "galdr-alert-danger",
    text: "text-[var(--galdr-danger)]",
    Icon: XCircle,
  },
  warning: {
    tone: "galdr-alert-warning",
    text: "text-[var(--galdr-at-risk)]",
    Icon: AlertTriangle,
  },
  info: {
    tone: "galdr-alert-info",
    text: "text-[var(--galdr-accent-2)]",
    Icon: Info,
  },
};

export default function Toast({ type, message, onClose }: ToastProps) {
  const [visible, setVisible] = useState(false);
  const { tone, text, Icon } = styles[type];

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true));
  }, []);

  function handleClose() {
    setVisible(false);
    setTimeout(onClose, 200);
  }

  return (
    <div
      className={`flex w-80 items-start gap-3 p-4 shadow-lg transition-all duration-200 ${tone} ${
        visible ? "translate-x-0 opacity-100" : "translate-x-full opacity-0"
      }`}
    >
      <Icon className={`h-5 w-5 shrink-0 ${text}`} />
      <p className={`flex-1 text-sm font-medium ${text}`}>{message}</p>
      <button
        onClick={handleClose}
        className={`shrink-0 ${text} hover:opacity-70`}
        aria-label="Close notification"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
