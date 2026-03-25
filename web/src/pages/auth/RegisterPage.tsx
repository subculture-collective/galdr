import { type FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { authApi } from "@/lib/api";
import { useAuth } from "@/contexts/AuthContext";
import { AxiosError } from "axios";
import SeoMeta from "@/components/SeoMeta";

export default function RegisterPage() {
  const navigate = useNavigate();
  const { setSession } = useAuth();

  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [orgName, setOrgName] = useState("");
  const [error, setError] = useState("");
  const [fieldError, setFieldError] = useState("");
  const [loading, setLoading] = useState(false);

  function validate(): boolean {
    if (!email || !password || !orgName) {
      setError("Email, password, and organization name are required.");
      return false;
    }
    if (password.length < 8) {
      setFieldError("Password must be at least 8 characters.");
      return false;
    }
    if (password !== confirmPassword) {
      setFieldError("Passwords do not match.");
      return false;
    }
    return true;
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setFieldError("");

    if (!validate()) return;

    setLoading(true);
    try {
      const { data } = await authApi.register({
        email,
        password,
        first_name: firstName,
        last_name: lastName,
        org_name: orgName,
      });
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
        title="Create account | PulseScore"
        description="Create your PulseScore account and start tracking customer health."
        path="/register"
        noIndex
      />
      <div className="galdr-card w-full max-w-md p-8">
        <h1 className="mb-1 text-center text-2xl font-bold text-[var(--galdr-fg)]">
          Create your account
        </h1>
        <p className="mb-6 text-center text-sm text-[var(--galdr-fg-muted)]">
          Get started with Galdr for your team
        </p>

        {error && (
          <div className="galdr-alert-danger mb-4 px-4 py-3 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label
                htmlFor="firstName"
                className="block text-sm font-medium text-[var(--galdr-fg-muted)]"
              >
                First name
              </label>
              <input
                id="firstName"
                type="text"
                autoComplete="given-name"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                className="galdr-input mt-1 block w-full px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label
                htmlFor="lastName"
                className="block text-sm font-medium text-[var(--galdr-fg-muted)]"
              >
                Last name
              </label>
              <input
                id="lastName"
                type="text"
                autoComplete="family-name"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                className="galdr-input mt-1 block w-full px-3 py-2 text-sm"
              />
            </div>
          </div>

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
              htmlFor="orgName"
              className="block text-sm font-medium text-[var(--galdr-fg-muted)]"
            >
              Organization name
            </label>
            <input
              id="orgName"
              type="text"
              required
              value={orgName}
              onChange={(e) => setOrgName(e.target.value)}
              className="galdr-input mt-1 block w-full px-3 py-2 text-sm"
              placeholder="Acme Inc."
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
              autoComplete="new-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="galdr-input mt-1 block w-full px-3 py-2 text-sm"
              placeholder="••••••••"
            />
            <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
              Min 8 characters with uppercase, lowercase, and a digit
            </p>
          </div>

          <div>
            <label
              htmlFor="confirmPassword"
              className="block text-sm font-medium text-[var(--galdr-fg-muted)]"
            >
              Confirm password
            </label>
            <input
              id="confirmPassword"
              type="password"
              autoComplete="new-password"
              required
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="galdr-input mt-1 block w-full px-3 py-2 text-sm"
              placeholder="••••••••"
            />
            {fieldError && (
              <p className="mt-1 text-xs text-[var(--galdr-danger)]">
                {fieldError}
              </p>
            )}
          </div>

          <button
            type="submit"
            disabled={loading}
            className="galdr-button-primary w-full px-4 py-2 text-sm font-medium disabled:opacity-50"
          >
            {loading ? "Creating account..." : "Create account"}
          </button>
        </form>

        <p className="mt-4 text-center text-sm text-[var(--galdr-fg-muted)]">
          Already have an account?{" "}
          <Link to="/login" className="galdr-link">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
