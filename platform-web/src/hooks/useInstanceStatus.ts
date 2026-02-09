import { useState, useEffect } from "react";
import { api } from "../lib/api";

export interface InstanceStatus {
  deployment_mode: string;
  setup_complete: boolean;
}

export function useInstanceStatus() {
  const [status, setStatus] = useState<InstanceStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .getInstanceStatus()
      .then((data) => {
        if (!cancelled) {
          setStatus(data);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load instance status");
          setStatus(null);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return { status, loading, error };
}
