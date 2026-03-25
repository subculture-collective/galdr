import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import api, {
  authApi,
  type AuthResponse,
  type AuthUser,
  type AuthOrg,
} from "@/lib/api";

interface AuthState {
  user: AuthUser | null;
  organization: AuthOrg | null;
  accessToken: string | null;
}

interface AuthContextValue extends AuthState {
  isAuthenticated: boolean;
  loading: boolean;
  setSession: (data: AuthResponse) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}

// Store refresh token in memory (not localStorage, for XSS protection)
let refreshTokenStore: string | null = null;
let refreshInFlight: Promise<AuthResponse> | null = null;

async function refreshSessionSingleFlight(): Promise<AuthResponse> {
  if (!refreshTokenStore) {
    throw new Error("No refresh token available");
  }

  if (!refreshInFlight) {
    const token = refreshTokenStore;
    refreshInFlight = authApi
      .refresh(token)
      .then(({ data }) => data)
      .finally(() => {
        refreshInFlight = null;
      });
  }

  return refreshInFlight;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({
    user: null,
    organization: null,
    accessToken: null,
  });
  const [loading, setLoading] = useState(true);
  const refreshTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const applySessionRef = useRef<(data: AuthResponse) => void>(() => undefined);

  const clearSession = useCallback(() => {
    setState({ user: null, organization: null, accessToken: null });
    refreshTokenStore = null;
    refreshInFlight = null;
    if (refreshTimer.current) {
      clearTimeout(refreshTimer.current);
      refreshTimer.current = null;
    }
  }, []);

  const scheduleRefresh = useCallback(
    (accessToken: string) => {
      if (refreshTimer.current) clearTimeout(refreshTimer.current);

      // Decode JWT to get exp — schedule refresh 1 min before expiry
      try {
        const payload = JSON.parse(atob(accessToken.split(".")[1]));
        const expiresMs = payload.exp * 1000;
        const refreshAt = expiresMs - Date.now() - 60_000; // 1 min before

        if (refreshAt > 0) {
          refreshTimer.current = setTimeout(async () => {
            try {
              const data = await refreshSessionSingleFlight();
              applySessionRef.current(data);
            } catch {
              clearSession();
            }
          }, refreshAt);
        }
      } catch {
        // If JWT decoding fails, don't schedule
      }
    },
    [clearSession],
  );

  const applySession = useCallback(
    (data: AuthResponse) => {
      setState({
        user: data.user,
        organization: data.organization,
        accessToken: data.tokens.access_token,
      });
      refreshTokenStore = data.tokens.refresh_token;
      scheduleRefresh(data.tokens.access_token);
    },
    [scheduleRefresh],
  );

  useEffect(() => {
    applySessionRef.current = applySession;
  }, [applySession]);

  // On mount: attempt silent refresh to restore session
  useEffect(() => {
    async function tryRestore() {
      if (!refreshTokenStore) {
        setLoading(false);
        return;
      }
      try {
        const data = await refreshSessionSingleFlight();
        applySession(data);
      } catch {
        clearSession();
      } finally {
        setLoading(false);
      }
    }
    tryRestore();

    return () => {
      if (refreshTimer.current) clearTimeout(refreshTimer.current);
    };
  }, [applySession, clearSession]);

  // Axios interceptor: attach Authorization header
  useEffect(() => {
    const requestId = api.interceptors.request.use((config) => {
      if (state.accessToken) {
        config.headers.Authorization = `Bearer ${state.accessToken}`;
      }
      if (state.organization) {
        config.headers["X-Organization-ID"] = state.organization.id;
      }
      return config;
    });

    const responseId = api.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;

        if (
          error.response?.status === 401 &&
          refreshTokenStore &&
          originalRequest &&
          !originalRequest._retry
        ) {
          originalRequest._retry = true;
          try {
            const data = await refreshSessionSingleFlight();
            applySession(data);
            // Retry original request with new token
            originalRequest.headers = originalRequest.headers ?? {};
            originalRequest.headers.Authorization = `Bearer ${data.tokens.access_token}`;
            return api.request(originalRequest);
          } catch {
            clearSession();
          }
        }
        return Promise.reject(error);
      },
    );

    return () => {
      api.interceptors.request.eject(requestId);
      api.interceptors.response.eject(responseId);
    };
  }, [state.accessToken, state.organization, applySession, clearSession]);

  const value = useMemo<AuthContextValue>(
    () => ({
      ...state,
      isAuthenticated: !!state.accessToken,
      loading,
      setSession: applySession,
      logout: clearSession,
    }),
    [state, loading, applySession, clearSession],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
