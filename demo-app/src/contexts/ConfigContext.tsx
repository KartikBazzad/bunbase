import {
  createContext,
  useContext,
  useState,
  useCallback,
  ReactNode,
} from "react";
import type { ClientConfig } from "@/lib/client";

const STORAGE_KEY_URL = "demo_base_url";
const STORAGE_KEY_API_KEY = "demo_api_key";

interface ConfigContextValue {
  baseUrl: string;
  apiKey: string;
  setConfig: (config: Partial<ClientConfig>) => void;
  isConfigured: boolean;
}

const defaultUrl =
  import.meta.env.VITE_BUNBASE_URL || "http://localhost:3001";

const ConfigContext = createContext<ConfigContextValue | null>(null);

export function ConfigProvider({ children }: { children: ReactNode }) {
  const [baseUrl, setBaseUrl] = useState(() => {
    return localStorage.getItem(STORAGE_KEY_URL) || defaultUrl;
  });
  const [apiKey, setApiKeyState] = useState(() => {
    return localStorage.getItem(STORAGE_KEY_API_KEY) || "";
  });

  const setConfig = useCallback((config: Partial<ClientConfig>) => {
    if (config.baseUrl != null) {
      localStorage.setItem(STORAGE_KEY_URL, config.baseUrl);
      setBaseUrl(config.baseUrl);
    }
    if (config.apiKey != null) {
      localStorage.setItem(STORAGE_KEY_API_KEY, config.apiKey);
      setApiKeyState(config.apiKey);
    }
  }, []);

  const value: ConfigContextValue = {
    baseUrl,
    apiKey,
    setConfig,
    isConfigured: Boolean(apiKey),
  };

  return (
    <ConfigContext.Provider value={value}>{children}</ConfigContext.Provider>
  );
}

export function useConfig() {
  const ctx = useContext(ConfigContext);
  if (!ctx) throw new Error("useConfig must be used within ConfigProvider");
  return ctx;
}
