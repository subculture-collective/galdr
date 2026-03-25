import { type FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { authApi } from "@/lib/api";
import { useAuth } from "@/contexts/AuthContext";
import { AxiosError } from "axios";
import SeoMeta from "@/components/SeoMeta";

export default function LoginPage() {
  const navigate = useNavigate();
  const { setSession } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");

    if (!email || !password) {
      setError("Email and password are required.");
      return;
    }

    setLoading(true);
    try {
      const { data } = await authApi.login({ email, password });
      setSession(data);
      navigate("/dashboard");
    } catch (err) {
      if (err instanceof AxiosError && err.response?.data?.error) {
        setError(err.response.data.error);
      } else {
        setError("An unexpected error occurred. Please try again.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="galdr-noise flex min-h-screen items-center justify-center bg-[var(--galdr-bg)] px-4 py-10 text-[var(--galdr-fg)]">
      <SeoMeta
        title="Sign in | PulseScore"
        description="Sign in to your PulseScore workspace."
        path="/login"
        noIndex
      />
      <div className="galdr-card w-full max-w-md p-8">
        <h1 className="mb-1 text-center text-2xl font-bold text-[var(--galdr-fg)]">
          Sign in to Galdr
        </h1>
        <p className="mb-6 text-center text-sm text-[var(--galdr-fg-muted)]">
          Enter your credentials to access your account
        </p>

        {error && (
          <div className="galdr-alert-danger mb-4 px-4 py-3 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="email"
              className="block text-sm font-medium text-[var(--galdr-fg-muted)]"
            >
              Email
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="galdr-input mt-1 block w-full px-3 py-2 text-sm"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-[var(--galdr-fg-muted)]"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="galdr-input mt-1 block w-full px-3 py-2 text-sm"
              placeholder="••••••••"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="galdr-button-primary w-full px-4 py-2 text-sm font-medium disabled:opacity-50"
          >
            {loading ? "Signing in..." : "Sign in"}
          </button>
        </form>

        <div className="mt-4 flex items-center justify-between text-sm">
          <Link to="/auth/forgot-password" className="galdr-link">
            Forgot password?
          </Link>
          <Link to="/register" className="galdr-link">
            Create an account
          </Link>
        </div>
      </div>
    </div>
  );
}
