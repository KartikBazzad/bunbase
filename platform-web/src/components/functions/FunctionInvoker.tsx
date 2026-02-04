import { useState } from "react";
import { api } from "../../lib/api";
import { Play, AlertCircle } from 'lucide-react';

interface FunctionInvokerProps {
  projectId: string;
  functionName: string;
}

export function FunctionInvoker({
  projectId,
  functionName,
}: FunctionInvokerProps) {
  const [method, setMethod] = useState("GET");
  const [body, setBody] = useState("{}");
  const [response, setResponse] = useState<{
    status?: number;
    headers?: Record<string, string>;
    body?: string;
    time?: number;
  } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleInvoke = async () => {
    setLoading(true);
    setError(null);
    setResponse(null);
    const startTime = Date.now();

    try {
      let parsedBody;
      if (method !== "GET" && method !== "HEAD") {
        try {
          parsedBody = JSON.parse(body);
        } catch (e) {
          setError("Invalid JSON body");
          setLoading(false);
          return;
        }
      }

      // We use the raw fetch here to get headers and status properly if the API client doesn't expose them all
      // But let's try using the client first. The client returns parsed JSON usually.
      // Our Invoke endpoint returns raw response body.
      // The current ApiClient.invokeFunction expects JSON response which might fail if function returns plain text.
      // Let's use direct fetch wrapper to handle text responses.

      const res = await api.invokeFunction(
        projectId,
        functionName,
        method,
        parsedBody,
      );

      // Since our ApiClient automatically parses JSON, 'res' is the body object.
      // Ideally we want status and headers too.
      // For now, let's just show the result.

      setResponse({
        body: typeof res === "string" ? res : JSON.stringify(res, null, 2),
        time: Date.now() - startTime,
      });
    } catch (err: any) {
      // If ApiClient throws, it likely parsed the error
      setError(err.message || "Invocation failed");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-base-200 p-4 rounded-md border border-base-300 mt-4">
      <h4 className="font-medium mb-3 text-sm uppercase tracking-wide text-base-content/50 flex items-center gap-2">
        <Zap className="w-3 h-3" />
        Test Function
      </h4>

      <div className="flex gap-4 mb-4">
        <div className="flex-1">
          <label className="block text-xs font-medium text-base-content mb-1">
            JSON Body (optional)
          </label>
          <textarea
            className="textarea textarea-bordered textarea-sm w-full font-mono text-sm h-[38px] py-1 resize-none focus:h-24 transition-all"
            value={body}
            onChange={(e) => setBody(e.target.value)}
            disabled={method === "GET"}
            placeholder="{}"
          />
        </div>
        <div className="flex items-end">
          <button
            className="btn btn-primary btn-sm"
            onClick={handleInvoke}
            disabled={loading}
          >
            {loading ? (
              <span className="loading loading-spinner loading-sm"></span>
            ) : (
              <>
                <Play className="w-3 h-3 mr-1" />
                Run
              </>
            )}
          </button>
        </div>
      </div>

      {error && (
        <div className="alert alert-error">
          <AlertCircle className="w-5 h-5 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {response && (
        <div className="bg-base-100 border border-base-300 rounded p-3">
          <div className="flex justify-between items-center mb-2 border-b border-base-300 pb-2">
            <span className="text-xs font-semibold text-base-content/50">
              RESPONSE
            </span>
            <span className="text-xs text-base-content/40">{response.time}ms</span>
          </div>
          <pre className="text-sm font-mono whitespace-pre-wrap overflow-x-auto max-h-60">
            {response.body}
          </pre>
        </div>
      )}
    </div>
  );
}
