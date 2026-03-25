import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangle } from "lucide-react";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("ErrorBoundary caught an error:", error, errorInfo);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="galdr-shell flex min-h-[400px] items-center justify-center p-8">
          <div className="galdr-card w-full max-w-lg p-8 text-center">
            <AlertTriangle className="mx-auto mb-4 h-12 w-12 text-[var(--galdr-danger)]" />
            <h2 className="text-lg font-semibold text-[var(--galdr-fg)]">
              Something went wrong
            </h2>
            <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
              An unexpected error occurred. Please try again.
            </p>
            <button
              onClick={this.handleReset}
              className="galdr-button-primary mt-4 px-4 py-2 text-sm font-medium"
            >
              Try Again
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
