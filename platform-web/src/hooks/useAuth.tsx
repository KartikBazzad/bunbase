import {
  useState,
  useEffect,
  createContext,
  useContext,
  ReactNode,
} from "react";
import { api } from "../lib/api";

export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
}

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [eventSource, setEventSource] = useState<EventSource | null>(null);

  const refresh = async () => {
    try {
      const userData = await api.getMe();
      setUser(userData as User);
    } catch (_error) {
      setUser(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // Initial rehydration from cookie
    refresh();

    // Start SSE auth stream once on mount
    const es = new EventSource(
      `${import.meta.env.VITE_API_URL || "http://localhost:3001/v1"}/auth/stream`,
      { withCredentials: true } as any,
    );

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.status === "expired") {
          setUser(null);
        }
      } catch {
        // ignore malformed events
      }
    };

    setEventSource(es);

    return () => {
      es.close();
    };
  }, []);

  const login = async (email: string, password: string) => {
    const userData = await api.login(email, password);
    setUser(userData as User);
  };

  const register = async (email: string, password: string, name: string) => {
    const userData = await api.register(email, password, name);
    setUser(userData as User);
  };

  const logout = async () => {
    await api.logout();
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{ user, loading, login, register, logout, refresh }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
