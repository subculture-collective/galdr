import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { AuthProvider } from "@/contexts/AuthContext";
import { ThemeProvider } from "@/contexts/ThemeContext";
import { ToastProvider } from "@/contexts/ToastContext";
import ProtectedRoute from "@/components/ProtectedRoute";
import ErrorBoundary from "@/components/ErrorBoundary";
import AppLayout from "@/layouts/AppLayout";

const LandingPage = lazy(() => import("@/pages/LandingPage"));
const PricingPage = lazy(() => import("@/pages/PricingPage"));
const SeoHubPage = lazy(() => import("@/pages/seo/SeoHubPage"));
const SeoProgrammaticPage = lazy(
  () => import("@/pages/seo/SeoProgrammaticPage"),
);
const LoginPage = lazy(() => import("@/pages/auth/LoginPage"));
const RegisterPage = lazy(() => import("@/pages/auth/RegisterPage"));
const DashboardPage = lazy(() => import("@/pages/DashboardPage"));
const CustomersPage = lazy(() => import("@/pages/CustomersPage"));
const CustomerDetailPage = lazy(() => import("@/pages/CustomerDetailPage"));
const SettingsPage = lazy(() => import("@/pages/settings/SettingsPage"));
const OnboardingPage = lazy(() => import("@/pages/OnboardingPage"));
const PrivacyPage = lazy(() => import("@/pages/PrivacyPage"));
const TermsPage = lazy(() => import("@/pages/TermsPage"));
const NotFoundPage = lazy(() => import("@/pages/NotFoundPage"));

function RouteLoadingFallback() {
  return (
    <div className="flex min-h-[30vh] items-center justify-center text-sm text-[var(--galdr-fg-muted)]">
      Loading…
    </div>
  );
}

function App() {
  return (
    <BrowserRouter>
      <ThemeProvider>
        <AuthProvider>
          <ToastProvider>
            <ErrorBoundary>
              <Suspense fallback={<RouteLoadingFallback />}>
                <Routes>
                  {/* Public marketing + auth routes */}
                  <Route path="/" element={<LandingPage />} />
                  <Route path="/pricing" element={<PricingPage />} />
                  <Route
                    path="/templates"
                    element={<SeoHubPage family="templates" />}
                  />
                  <Route
                    path="/templates/:slug"
                    element={<SeoProgrammaticPage family="templates" />}
                  />
                  <Route
                    path="/integrations"
                    element={<SeoHubPage family="integrations" />}
                  />
                  <Route
                    path="/integrations/:slug"
                    element={<SeoProgrammaticPage family="integrations" />}
                  />
                  <Route
                    path="/for"
                    element={<SeoHubPage family="personas" />}
                  />
                  <Route
                    path="/for/:slug"
                    element={<SeoProgrammaticPage family="personas" />}
                  />
                  <Route
                    path="/compare"
                    element={<SeoHubPage family="comparisons" />}
                  />
                  <Route
                    path="/compare/:slug"
                    element={<SeoProgrammaticPage family="comparisons" />}
                  />
                  <Route
                    path="/glossary"
                    element={<SeoHubPage family="glossary" />}
                  />
                  <Route
                    path="/glossary/:slug"
                    element={<SeoProgrammaticPage family="glossary" />}
                  />
                  <Route
                    path="/examples"
                    element={<SeoHubPage family="examples" />}
                  />
                  <Route
                    path="/examples/:slug"
                    element={<SeoProgrammaticPage family="examples" />}
                  />
                  <Route
                    path="/best"
                    element={<SeoHubPage family="curation" />}
                  />
                  <Route
                    path="/best/:slug"
                    element={<SeoProgrammaticPage family="curation" />}
                  />
                  <Route path="/login" element={<LoginPage />} />
                  <Route path="/register" element={<RegisterPage />} />
                  <Route path="/privacy" element={<PrivacyPage />} />
                  <Route path="/terms" element={<TermsPage />} />

                  {/* Backward-compatible auth aliases */}
                  <Route
                    path="/auth/login"
                    element={<Navigate to="/login" replace />}
                  />
                  <Route
                    path="/auth/register"
                    element={<Navigate to="/register" replace />}
                  />

                  {/* Protected app routes */}
                  <Route
                    element={
                      <ProtectedRoute>
                        <AppLayout />
                      </ProtectedRoute>
                    }
                  >
                    <Route path="/dashboard" element={<DashboardPage />} />
                    <Route path="/onboarding" element={<OnboardingPage />} />
                    <Route path="/customers" element={<CustomersPage />} />
                    <Route
                      path="/customers/:id"
                      element={<CustomerDetailPage />}
                    />
                    <Route path="/settings/*" element={<SettingsPage />} />
                  </Route>

                  {/* Catch-all 404 */}
                  <Route path="*" element={<NotFoundPage />} />
                </Routes>
              </Suspense>
            </ErrorBoundary>
          </ToastProvider>
        </AuthProvider>
      </ThemeProvider>
    </BrowserRouter>
  );
}

export default App;
