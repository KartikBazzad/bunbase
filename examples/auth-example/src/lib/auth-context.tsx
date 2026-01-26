/**
 * Authentication Context and Hook
 * 
 * Manages authentication state, session persistence, and provides auth methods
 */

import { createContext, useContext, useState, useEffect, ReactNode } from "react";
import { client } from "./client";
import type { AuthUser, AuthSession } from "@bunbase/js-sdk";

interface AuthContextType {
  user: AuthUser | null;
  session: AuthSession | null;
  loading: boolean;
  error: string | null;
  signUp: (email: string, password: string, name: string) => Promise<void>;
  signIn: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  getUser: () => Promise<void>;
  clearError: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const SESSION_STORAGE_KEY = "bunbase_session";
const USER_STORAGE_KEY = "bunbase_user";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [session, setSession] = useState<AuthSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load session from localStorage on mount
  useEffect(() => {
    const loadSession = async () => {
      try {
        const storedSession = localStorage.getItem(SESSION_STORAGE_KEY);
        const storedUser = localStorage.getItem(USER_STORAGE_KEY);

        if (storedSession && storedUser) {
          const parsedSession = JSON.parse(storedSession);
          const parsedUser = JSON.parse(storedUser);
          
          // Convert date strings back to Date objects
          parsedSession.expiresAt = new Date(parsedSession.expiresAt);
          parsedSession.createdAt = new Date(parsedSession.createdAt);
          parsedUser.createdAt = new Date(parsedUser.createdAt);
          parsedUser.updatedAt = new Date(parsedUser.updatedAt);

          setSession(parsedSession);
          setUser(parsedUser);

          // Verify session is still valid by fetching current user
          try {
            const currentUser = await client.auth.getUser();
            setUser(currentUser);
            localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(currentUser));
          } catch (err) {
            // Session expired or invalid, clear storage
            localStorage.removeItem(SESSION_STORAGE_KEY);
            localStorage.removeItem(USER_STORAGE_KEY);
            setSession(null);
            setUser(null);
          }
        }
      } catch (err) {
        console.error("Error loading session:", err);
        localStorage.removeItem(SESSION_STORAGE_KEY);
        localStorage.removeItem(USER_STORAGE_KEY);
      } finally {
        setLoading(false);
      }
    };

    loadSession();
  }, []);

  const saveSession = (userData: AuthUser, sessionData: AuthSession | null) => {
    if (sessionData) {
      localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(sessionData));
      setSession(sessionData);
    } else {
      // Session might be null when email verification is required
      localStorage.removeItem(SESSION_STORAGE_KEY);
      setSession(null);
    }
    localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(userData));
    setUser(userData);
  };

  const clearSession = () => {
    localStorage.removeItem(SESSION_STORAGE_KEY);
    localStorage.removeItem(USER_STORAGE_KEY);
    setUser(null);
    setSession(null);
  };

  const signUp = async (email: string, password: string, name: string) => {
    try {
      setError(null);
      setLoading(true);
      const result = await client.auth.signUp(email, password, name);
      saveSession(result.user, result.session);
    } catch (err: any) {
      const errorMessage = err.message || "Failed to sign up";
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  };

  const signIn = async (email: string, password: string) => {
    try {
      setError(null);
      setLoading(true);
      const result = await client.auth.signIn(email, password);
      saveSession(result.user, result.session);
    } catch (err: any) {
      const errorMessage = err.message || "Failed to sign in";
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  };

  const signOut = async () => {
    try {
      setError(null);
      setLoading(true);
      await client.auth.signOut();
      clearSession();
    } catch (err: any) {
      const errorMessage = err.message || "Failed to sign out";
      setError(errorMessage);
      // Clear session anyway
      clearSession();
    } finally {
      setLoading(false);
    }
  };

  const getUser = async () => {
    try {
      setError(null);
      const userData = await client.auth.getUser();
      setUser(userData);
      localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(userData));
    } catch (err: any) {
      const errorMessage = err.message || "Failed to get user";
      setError(errorMessage);
      // If getting user fails, session might be invalid
      clearSession();
      throw err;
    }
  };

  const clearError = () => {
    setError(null);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        session,
        loading,
        error,
        signUp,
        signIn,
        signOut,
        getUser,
        clearError,
      }}
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
