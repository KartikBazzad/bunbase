import {
  createContext,
  useContext,
  useState,
  useCallback,
  useEffect,
  ReactNode,
} from "react";
import { useConfig } from "./ConfigContext";

export interface ProjectUser {
  id: string;
  user_id?: string;
  email: string;
  project_id: string;
}

interface AuthContextValue {
  user: ProjectUser | null;
  isLoggedIn: boolean;
  isLoadingSession: boolean;
  login: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string, name?: string) => Promise<void>;
  logout: () => Promise<void>;
  error: string | null;
  clearError: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const { baseUrl, apiKey, isConfigured } = useConfig();
  const [user, setUser] = useState<ProjectUser | null>(null);
  const [isLoadingSession, setIsLoadingSession] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Helper to get the API URL - use relative URL in dev (via Vite proxy) for cookies to work
  const getApiUrl = useCallback(
    (path: string) => {
      // In development, if baseUrl points to localhost:3001, use relative URL for Vite proxy
      // This ensures cookies work properly
      if (
        typeof window !== "undefined" &&
        baseUrl &&
        (baseUrl.includes("localhost:3001") ||
          baseUrl.includes("127.0.0.1:3001"))
      ) {
        return path;
      }
      return `${baseUrl}${path}`;
    },
    [baseUrl],
  );

  const checkSession = useCallback(async () => {
    if (!baseUrl) {
      setIsLoadingSession(false);
      return;
    }
    try {
      // Cookie is automatically sent with credentials: "include"
      // Use relative URL in dev so Vite proxy handles cookies correctly
      const res = await fetch(getApiUrl("/v1/auth/session"), {
        method: "GET",
        credentials: "include",
      });
      if (res.ok) {
        const data = await res.json();
        setUser(data.user ?? null);
      } else {
        setUser(null);
      }
    } catch {
      setUser(null);
    } finally {
      setIsLoadingSession(false);
    }
  }, [baseUrl, getApiUrl]);

  useEffect(() => {
    checkSession();
  }, [checkSession]);

  const login = useCallback(
    async (email: string, password: string) => {
      setError(null);
      if (!baseUrl || !apiKey) {
        throw new Error(
          "API key and base URL are required. Set them in Settings.",
        );
      }
      const res = await fetch(getApiUrl("/v1/auth/project/login"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Bunbase-Client-Key": apiKey,
        },
        credentials: "include",
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || "Login failed");
      }
      const data = await res.json();
      // Cookie is automatically set by server response
      if (data.user) setUser(data.user);
    },
    [baseUrl, apiKey, getApiUrl],
  );

  const signUp = useCallback(
    async (email: string, password: string, _name?: string) => {
      setError(null);
      if (!baseUrl || !apiKey) {
        throw new Error(
          "API key and base URL are required. Set them in Settings.",
        );
      }
      const res = await fetch(getApiUrl("/v1/auth/project/register"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Bunbase-Client-Key": apiKey,
        },
        credentials: "include",
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || "Sign up failed");
      }
      const created = await res.json();
      setUser(created);
      // Optionally auto-login so user gets a token
      try {
        await login(email, password);
      } catch {
        // User was created; they can log in on the login page
      }
    },
    [baseUrl, apiKey, login, getApiUrl],
  );

  const logout = useCallback(async () => {
    if (!baseUrl || !apiKey) {
      setUser(null);
      return;
    }
    try {
      // Call logout endpoint to clear cookie server-side
      await fetch(getApiUrl("/v1/auth/project/logout"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Bunbase-Client-Key": apiKey,
        },
        credentials: "include",
      });
    } catch {
      // Ignore errors, still clear user state
    }
    setUser(null);
  }, [baseUrl, apiKey, getApiUrl]);

  const value: AuthContextValue = {
    user,
    isLoggedIn: Boolean(user),
    isLoadingSession,
    login,
    signUp,
    logout,
    error,
    clearError: () => setError(null),
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
